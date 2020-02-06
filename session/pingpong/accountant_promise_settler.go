/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package pingpong

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/payments/bindings"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type transactor interface {
	FetchSettleFees() (registry.FeesResponse, error)
	SettleAndRebalance(id string, promise crypto.Promise) error
}

type promiseStorage interface {
	Get(providerID, accountantID identity.Identity) (AccountantPromise, error)
}

type receivedPromise struct {
	provider identity.Identity
	promise  crypto.Promise
}

// AccountantPromiseSettler is responsible for settling the accountant promises.
type AccountantPromiseSettler struct {
	bc                         providerChannelStatusProvider
	config                     AccountantPromiseSettlerConfig
	lock                       sync.Mutex
	accountantPromiseGetter    accountantPromiseGetter
	registrationStatusProvider registrationStatusProvider
	ks                         ks
	transactor                 transactor
	promiseStorage             promiseStorage

	currentState map[identity.Identity]state
	settleQueue  chan receivedPromise
	stop         chan struct{}
	once         sync.Once
}

// AccountantPromiseSettlerConfig configures the accountant promise settler accordingly.
type AccountantPromiseSettlerConfig struct {
	AccountantAddress    common.Address
	Threshold            float64
	MaxWaitForSettlement time.Duration
}

// NewAccountantPromiseSettler creates a new instance of accountant promise settler.
func NewAccountantPromiseSettler(transactor transactor, promiseStorage promiseStorage, providerChannelStatusProvider providerChannelStatusProvider, registrationStatusProvider registrationStatusProvider, ks ks, accountantPromiseGetter accountantPromiseGetter, config AccountantPromiseSettlerConfig) *AccountantPromiseSettler {
	return &AccountantPromiseSettler{
		bc:                         providerChannelStatusProvider,
		accountantPromiseGetter:    accountantPromiseGetter,
		ks:                         ks,
		registrationStatusProvider: registrationStatusProvider,
		config:                     config,
		currentState:               make(map[identity.Identity]state),
		promiseStorage:             promiseStorage,

		// defaulting to a queue of 5, in case we have a few active identities.
		settleQueue: make(chan receivedPromise, 5),
		stop:        make(chan struct{}),
		transactor:  transactor,
	}
}

// loadInitialState loads the initial state for the given identity. Inteded to be called on service start.
func (aps *AccountantPromiseSettler) loadInitialState(addr identity.Identity) error {
	aps.lock.Lock()
	defer aps.lock.Unlock()

	if _, ok := aps.currentState[addr]; ok {
		log.Info().Msgf("State for %v already loaded, skipping", addr)
		return nil
	}

	status, err := aps.registrationStatusProvider.GetRegistrationStatus(addr)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not check registration status for %v", addr))
	}

	if status != registry.RegisteredProvider {
		log.Info().Msgf("Provider %v not registered, skipping", addr)
		return nil
	}

	return aps.resyncState(addr)
}

func (aps *AccountantPromiseSettler) resyncState(addr identity.Identity) error {
	res, err := aps.bc.GetProviderChannel(aps.config.AccountantAddress, addr.ToCommonAddress())
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not get provider channel for %v", addr))
	}

	accountantPromise, err := aps.accountantPromiseGetter.Get(addr, identity.FromAddress(aps.config.AccountantAddress.Hex()))
	if err != nil && err != ErrNotFound {
		return errors.Wrap(err, fmt.Sprintf("could not get accountant promise for %v", addr))
	}

	availableBalance := res.Balance.Uint64() + res.Settled.Uint64()
	currentBalance := availableBalance - accountantPromise.Promise.Amount
	s := state{
		balance:          currentBalance,
		availableBalance: availableBalance,
		lastPromise:      accountantPromise.Promise,
		registered:       true,
	}

	aps.currentState[addr] = s
	log.Info().Msgf("Loaded state for provider %q: balance %v, available balance %v ", addr, s.balance, s.availableBalance)

	return nil
}

// Subscribe subscribes the accountant promise settler to the appropriate events
func (aps *AccountantPromiseSettler) Subscribe(sub eventbus.Subscriber) error {
	err := sub.SubscribeAsync(nodevent.AppTopicNode, aps.handleNodeEvent)
	if err != nil {
		return errors.Wrap(err, "could not subscribe to node status event")
	}

	err = sub.SubscribeAsync(registry.AppTopicRegistration, aps.handleRegistrationEvent)
	if err != nil {
		return errors.Wrap(err, "could not subscribe to registration event")
	}

	err = sub.SubscribeAsync(service.AppTopicServiceStatus, aps.handleServiceEvent)
	if err != nil {
		return errors.Wrap(err, "could not subscribe to service status event")
	}

	err = sub.SubscribeAsync(AppTopicAccountantPromise, aps.handleAccountantPromiseReceived)
	return errors.Wrap(err, "could not subscribe to accountant promise event")
}

func (aps *AccountantPromiseSettler) handleServiceEvent(event service.EventPayload) {
	switch event.Status {
	case string(service.Running):
		err := aps.loadInitialState(identity.FromAddress(event.ProviderID))
		// TODO: should we retry? should we signal that we need to cancel and abort?
		// In any case, if we start exceeding our balances, the accountant will let us know.
		// Sessions will be aborted, node *should* stop, indicating something went wrong.
		// On restart, a rebalance then should follow almost immediately.
		// But we'll get punished for that, won't we?
		if err != nil {
			log.Error().Err(err).Msgf("could not load initial state for provider %v", event.ProviderID)
		}
	default:
		log.Debug().Msgf("Ignoring service event with status %v", event.Status)
	}
}

func (aps *AccountantPromiseSettler) handleNodeEvent(payload nodevent.Payload) {
	if payload.Status == nodevent.StatusStarted {
		aps.handleNodeStart()
		return
	}

	if payload.Status == nodevent.StatusStopped {
		aps.handleNodeStop()
		return
	}
}

func (aps *AccountantPromiseSettler) handleRegistrationEvent(payload registry.RegistrationEventPayload) {
	aps.lock.Lock()
	defer aps.lock.Unlock()

	if payload.Status != registry.RegisteredProvider {
		log.Debug().Msgf("Ignoring event %v for provider %q", payload.Status.String(), payload.ID)
		return
	}
	log.Info().Msgf("Identity registration event received for provider %q", payload.ID)

	err := aps.resyncState(payload.ID)
	if err != nil {
		// TODO: should we retry? should we signal that we need to cancel and abort?
		// In any case, if we start exceeding our balances, the accountant will let us know.
		// Sessions will be aborted, node *should* stop, indicating something went wrong.
		// On restart, a rebalance then should follow almost immediately.
		// But we'll get punished for that, won't we?
		log.Error().Err(err).Msgf("Could not resync state for provider %v", payload.ID)
		return
	}

	log.Info().Msgf("Identity registration event handled for provider %q", payload.ID)
}

func (aps *AccountantPromiseSettler) handleAccountantPromiseReceived(apep AccountantPromiseEventPayload) {
	aps.lock.Lock()
	defer aps.lock.Unlock()

	log.Info().Msgf("Received accountant promise for %q", apep.ProviderID)

	v, ok := aps.currentState[apep.ProviderID]
	if !ok {
		log.Error().Msgf("Have no info on provider %q, skipping", apep.ProviderID)
		return
	}

	if !v.registered {
		log.Error().Msgf("provider %q not registered, skipping", apep.ProviderID)
		return
	}

	newState := v.updateWithNewPromise(apep.Promise)
	aps.currentState[apep.ProviderID] = newState
	log.Info().Msgf("Accountant promise state updated for provider %q", apep.ProviderID)

	if newState.needsSettling(aps.config.Threshold) {
		aps.settleQueue <- receivedPromise{
			provider: apep.ProviderID,
			promise:  apep.Promise,
		}
	}
}

func (aps *AccountantPromiseSettler) listenForSettlementRequests() {
	log.Info().Msg("Listening for settlement events")
	defer func() {
		log.Info().Msg("Stopped listening for settlement events")
	}()

	for {
		select {
		case <-aps.stop:
			return
		case p := <-aps.settleQueue:
			go aps.settle(p)
		}
	}
}

// ErrNothingToSettle indicates that there is nothing to settle.
var ErrNothingToSettle = errors.New("nothing to settle for the given provider")

// ForceSettle forces the settlement for a provider
func (aps *AccountantPromiseSettler) ForceSettle(providerID, accountantID identity.Identity) error {
	promise, err := aps.promiseStorage.Get(providerID, accountantID)
	if err == ErrNotFound {
		return ErrNothingToSettle
	}
	if err != nil {
		return errors.Wrap(err, "could not get promise from storage")
	}

	hexR, err := hex.DecodeString(promise.R)
	if err != nil {
		return errors.Wrap(err, "could not decode R")
	}

	promise.Promise.R = hexR
	return aps.settle(receivedPromise{
		promise:  promise.Promise,
		provider: providerID,
	})
}

// ErrSettleTimeout indicates that the settlement has timed out
var ErrSettleTimeout = errors.New("settle timeout")

func (aps *AccountantPromiseSettler) settle(p receivedPromise) error {
	if aps.isSettling(p.provider) {
		return errors.New("provider already has settlement in progress")
	}

	aps.setSettling(p.provider, true)
	log.Info().Msgf("Marked provider %v as requesting setlement", p.provider)
	sink, cancel, err := aps.bc.SubscribeToPromiseSettledEvent(p.provider.ToCommonAddress(), aps.config.AccountantAddress)
	if err != nil {
		aps.setSettling(p.provider, false)
		log.Error().Err(err).Msg("Could not subscribe to promise settlement")
		return err
	}

	errCh := make(chan error)
	go func() {
		defer cancel()
		defer aps.setSettling(p.provider, false)
		defer close(errCh)
		select {
		case <-aps.stop:
			return
		case _, more := <-sink:
			if !more {
				break
			}

			log.Info().Msgf("Settling complete for provider %v", p.provider)

			err := aps.resyncState(p.provider)
			if err != nil {
				// This will get retried so we do not need to explicitly retry
				// TODO: maybe add a sane limit of retries
				log.Error().Err(err).Msgf("Resync failed for provider %v", p.provider)
			} else {
				log.Info().Msgf("Resync success for provider %v", p.provider)
			}
			return
		case <-time.After(aps.config.MaxWaitForSettlement):
			log.Info().Msgf("Settle timeout for %v", p.provider)

			// send a signal to waiter that the settlement has timed out
			errCh <- ErrSettleTimeout
			return
		}
	}()

	err = aps.transactor.SettleAndRebalance(aps.config.AccountantAddress.Hex(), p.promise)
	if err != nil {
		cancel()
		log.Error().Err(err).Msgf("Could not settle promise for %v", p.provider.Address)
		return err
	}

	return <-errCh
}

func (aps *AccountantPromiseSettler) isSettling(id identity.Identity) bool {
	aps.lock.Lock()
	defer aps.lock.Unlock()
	v, ok := aps.currentState[id]
	if !ok {
		return false
	}

	return v.settleInProgress
}

func (aps *AccountantPromiseSettler) setSettling(id identity.Identity, settling bool) {
	aps.lock.Lock()
	defer aps.lock.Unlock()
	v := aps.currentState[id]
	v.settleInProgress = settling
	aps.currentState[id] = v
}

func (aps *AccountantPromiseSettler) handleNodeStart() {
	go aps.listenForSettlementRequests()

	for _, v := range aps.ks.Accounts() {
		addr := identity.FromAddress(v.Address.Hex())
		go func(address identity.Identity) {
			err := aps.loadInitialState(address)
			if err != nil {
				// TODO: should we retry? should we signal that we need to cancel and abort?
				// In any case, if we start exceeding our balances, the accountant will let us know.
				// Sessions will be aborted, node *should* stop, indicating something went wrong.
				// On restart, a rebalance then should follow almost immediately.
				// But we'll get punished for that, won't we?
				log.Error().Err(err).Msgf("could not load initial state for %v", addr)
			}
		}(addr)
	}
}

func (aps *AccountantPromiseSettler) handleNodeStop() {
	aps.once.Do(func() {
		close(aps.stop)
	})
}

type providerChannelStatusProvider interface {
	SubscribeToPromiseSettledEvent(providerID, accountantID common.Address) (sink chan *bindings.AccountantImplementationPromiseSettled, cancel func(), err error)
	GetProviderChannel(accountantAddress common.Address, addressToCheck common.Address) (ProviderChannel, error)
}

type accountantPromiseGetter interface {
	Get(providerID, accountantID identity.Identity) (AccountantPromise, error)
}

type state struct {
	settleInProgress bool
	balance          uint64
	availableBalance uint64
	registered       bool
	lastPromise      crypto.Promise
}

func (s state) needsSettling(threshold float64) bool {
	if !s.registered {
		return false
	}

	if s.settleInProgress {
		return false
	}

	if float64(s.balance) <= 0 {
		return true
	}

	if float64(s.balance) <= threshold*float64(s.availableBalance) {
		return true
	}

	return false
}

func (s state) updateWithNewPromise(promise crypto.Promise) state {
	diff := promise.Amount - s.lastPromise.Amount
	return state{
		settleInProgress: s.settleInProgress,
		balance:          s.balance - diff,
		availableBalance: s.availableBalance,
		registered:       s.registered,
		lastPromise:      promise,
	}
}

type ks interface {
	Accounts() []accounts.Account
}

type registrationStatusProvider interface {
	GetRegistrationStatus(id identity.Identity) (registry.RegistrationStatus, error)
}
