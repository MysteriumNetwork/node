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

// Package payment is responsible for ensuring that the consumer can fullfil his obligation to provider.
// It contains all the orchestration required for value transfer from consumer to provider.
package payment

import (
	"errors"
	"time"

	"github.com/mysteriumnetwork/node/session/balance"
	"github.com/mysteriumnetwork/node/session/promise"
)

// BalanceTracker keeps track of current balance
type BalanceTracker interface {
	GetBalance() balance.Message
}

// PromiseValidator validates given promise
type PromiseValidator interface {
	Validate(promise.Message) bool
}

// PeerBalanceSender knows how to send a balance message to the peer
type PeerBalanceSender interface {
	Send(balance.Message) error
}

// ErrPromiseWaitTimeout indicates that we waited for a promise long enough, but with no result
var ErrPromiseWaitTimeout = errors.New("did not get a new promise")

// ErrPromiseValidationFailed indicates that an invalid promise was sent
var ErrPromiseValidationFailed = errors.New("promise validation failed")

// SessionBalance orchestrates the ping pong of balance sent to consumer -> promise received from consumer flow
type SessionBalance struct {
	stop               chan struct{}
	peerBalanceSender  PeerBalanceSender
	balanceTracker     BalanceTracker
	promiseChan        chan promise.Message
	chargePeriod       time.Duration
	promiseWaitTimeout time.Duration
	promiseValidator   PromiseValidator
}

// NewSessionBalance creates a new instance of provider payment orchestrator
func NewSessionBalance(
	peerBalanceSender PeerBalanceSender,
	balanceTracker BalanceTracker,
	promiseChan chan promise.Message,
	chargePeriod time.Duration,
	promiseWaitTimeout time.Duration,
	promiseValidator PromiseValidator) *SessionBalance {
	return &SessionBalance{
		stop:               make(chan struct{}),
		peerBalanceSender:  peerBalanceSender,
		balanceTracker:     balanceTracker,
		promiseChan:        promiseChan,
		chargePeriod:       chargePeriod,
		promiseWaitTimeout: promiseWaitTimeout,
		promiseValidator:   promiseValidator,
	}
}

// Start starts the payment orchestrator. Blocks.
func (ppo *SessionBalance) Start() error {
	for {
		select {
		case <-ppo.stop:
			return nil
		case <-time.After(ppo.chargePeriod):
			err := ppo.sendBalance()
			if err != nil {
				return err
			}
			err = ppo.receivePromiseOrTimeout()
			if err != nil {
				return err
			}
		}
	}
}

func (ppo *SessionBalance) sendBalance() error {
	balance := ppo.balanceTracker.GetBalance()
	return ppo.peerBalanceSender.Send(balance)
}

func (ppo *SessionBalance) receivePromiseOrTimeout() error {
	select {
	case pm := <-ppo.promiseChan:
		if !ppo.promiseValidator.Validate(pm) {
			return ErrPromiseValidationFailed
		}
		// TODO: Save the promise
		// TODO: Change balance
	case <-time.After(ppo.promiseWaitTimeout):
		return ErrPromiseWaitTimeout
	}
	return nil
}

// Stop stops the payment orchestrator
func (ppo *SessionBalance) Stop() {
	close(ppo.stop)
}
