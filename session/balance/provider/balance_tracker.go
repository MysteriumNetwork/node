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

package provider

import (
	"time"

	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/session/balance"
)

// PeerSender knows how to send a balance message to the peer
type PeerSender interface {
	Send(balance.Message) error
}

// TimeKeeper keeps track of time for payments
type TimeKeeper interface {
	StartTracking()
	Elapsed() time.Duration
}

// AmountCalculator is able to deduce the amount required for payment from a given duration
type AmountCalculator interface {
	TotalAmount(duration time.Duration) money.Money
}

// BalanceTracker is responsible for tracking the balance on the provider side
type BalanceTracker struct {
	timeKeeper       TimeKeeper
	amountCalculator AmountCalculator

	totalPromised uint64
	balance       uint64
	stop          chan struct{}
}

// NewBalanceTracker returns a new instance of the providerBalanceTracker
func NewBalanceTracker(timeKeeper TimeKeeper, amountCalculator AmountCalculator, initialBalance uint64) *BalanceTracker {
	return &BalanceTracker{
		timeKeeper:       timeKeeper,
		amountCalculator: amountCalculator,
		totalPromised:    initialBalance,

		stop: make(chan struct{}),
	}
}

func (bt *BalanceTracker) calculateBalance() {
	cost := bt.amountCalculator.TotalAmount(bt.timeKeeper.Elapsed())
	bt.balance = bt.totalPromised - cost.Amount
}

// GetBalance returns the balance message
func (bt *BalanceTracker) GetBalance() balance.Message {
	bt.calculateBalance()
	// TODO: sequence ID should come here, somehow
	return balance.Message{
		SequenceID: 0,
		Balance:    bt.balance,
	}
}
