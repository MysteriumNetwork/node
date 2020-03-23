/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package session

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/nat/traversal"
	sevent "github.com/mysteriumnetwork/node/session/event"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	// ErrorInvalidProposal is validation error then invalid proposal requested for session creation
	ErrorInvalidProposal = errors.New("proposal does not exist")
	// ErrorSessionNotExists returned when consumer tries to destroy session that does not exists
	ErrorSessionNotExists = errors.New("session does not exists")
	// ErrorWrongSessionOwner returned when consumer tries to destroy session that does not belongs to him
	ErrorWrongSessionOwner = errors.New("wrong session owner")
)

// IDGenerator defines method for session id generation
type IDGenerator func() (ID, error)

// ConfigParams session configuration parameters
type ConfigParams struct {
	SessionServiceConfig   ServiceConfiguration
	SessionDestroyCallback DestroyCallback
	TraversalParams        traversal.Params
}

type publisher interface {
	Publish(topic string, data interface{})
}

// ConfigProvider is able to handle config negotiations
type ConfigProvider interface {
	ProvideConfig(sessionID string, sessionConfig json.RawMessage) (*ConfigParams, error)
}

// DestroyCallback cleanups session
type DestroyCallback func()

// PromiseProcessor processes promises at provider side.
// Provider checks promises from consumer and signs them also.
// Provider clears promises from consumer.
type PromiseProcessor interface {
	Start(proposal market.ServiceProposal) error
	Stop() error
}

// Storage interface to session storage
type Storage interface {
	Add(sessionInstance Session)
	Find(id ID) (Session, bool)
	Remove(id ID)
}

// PaymentEngineFactory creates a new instance of payment engine
type PaymentEngineFactory func(providerID, accountantID identity.Identity, sessionID string) (PaymentEngine, error)

// NATEventGetter lets us access the last known traversal event
type NATEventGetter interface {
	LastEvent() *event.Event
}

// NewManager returns new session Manager
func NewManager(
	currentProposal market.ServiceProposal,
	sessionStorage Storage,
	paymentEngineFactory PaymentEngineFactory,
	natPinger traversal.NATPinger,
	natEventGetter NATEventGetter,
	serviceId string,
	publisher publisher,
) *Manager {
	return &Manager{
		currentProposal:      currentProposal,
		sessionStorage:       sessionStorage,
		natPinger:            natPinger,
		natEventGetter:       natEventGetter,
		serviceId:            serviceId,
		publisher:            publisher,
		paymentEngineFactory: paymentEngineFactory,
		creationLock:         sync.Mutex{},
	}
}

// Manager knows how to start and provision session
type Manager struct {
	currentProposal      market.ServiceProposal
	sessionStorage       Storage
	paymentEngineFactory PaymentEngineFactory
	natPinger            traversal.NATPinger
	natEventGetter       NATEventGetter
	serviceId            string
	publisher            publisher
	creationLock         sync.Mutex
}

// Start starts a session on the provider side for the given consumer.
// Multiple sessions per peerID is possible in case different services are used
func (manager *Manager) Start(session *Session, consumerID identity.Identity, consumerInfo ConsumerInfo, proposalID int, config ServiceConfiguration, pingerParams traversal.Params) (err error) {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()

	if manager.currentProposal.ID != proposalID {
		err = ErrorInvalidProposal
		return
	}

	session.ServiceType = manager.currentProposal.ServiceType
	session.ServiceID = manager.serviceId
	session.ConsumerID = consumerID
	session.Done = make(chan struct{})
	session.Config = config
	session.CreatedAt = time.Now().UTC()

	log.Info().Msg("Using new payments")
	engine, err := manager.paymentEngineFactory(identity.FromAddress(manager.currentProposal.ProviderID), consumerInfo.AccountantID, string(session.ID))
	if err != nil {
		return err
	}

	// stop the balance tracker once the session is finished
	go func() {
		<-session.Done
		engine.Stop()
	}()

	go func() {
		err := engine.Start()
		if err != nil {
			log.Error().Err(err).Msg("Payment engine error")
			destroyErr := manager.Destroy(consumerID, string(session.ID))
			if destroyErr != nil {
				log.Error().Err(err).Msg("Session cleanup failed")
			}
		}
	}()

	go manager.natPinger.PingConsumer(pingerParams.IP, pingerParams.LocalPorts, pingerParams.RemotePorts, pingerParams.ProxyPortMappingKey)
	manager.sessionStorage.Add(*session)
	return nil
}

// Acknowledge marks the session as successfully established as far as the consumer is concerned.
func (manager *Manager) Acknowledge(consumerID identity.Identity, sessionID string) error {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()
	session, found := manager.sessionStorage.Find(ID(sessionID))

	if !found {
		return ErrorSessionNotExists
	}

	if session.ConsumerID != consumerID {
		return ErrorWrongSessionOwner
	}

	manager.publisher.Publish(sevent.AppTopicSession, sevent.Payload{
		Action: sevent.Acknowledged,
		ID:     sessionID,
	})

	return nil
}

// Destroy destroys session by given sessionID
func (manager *Manager) Destroy(consumerID identity.Identity, sessionID string) error {
	manager.creationLock.Lock()
	defer manager.creationLock.Unlock()

	session, found := manager.sessionStorage.Find(ID(sessionID))

	if !found {
		return ErrorSessionNotExists
	}

	if session.ConsumerID != consumerID {
		return ErrorWrongSessionOwner
	}

	manager.sessionStorage.Remove(ID(sessionID))
	close(session.Done)

	return nil
}
