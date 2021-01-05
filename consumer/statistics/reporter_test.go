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

package statistics

import (
	"errors"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/market/mysterium"
	"github.com/mysteriumnetwork/node/session"
	"github.com/stretchr/testify/assert"
)

var mockSessionEvent = connectionstate.AppEventConnectionSession{
	Status: connectionstate.SessionCreatedStatus,
	SessionInfo: connectionstate.Status{
		ConsumerID: identity.FromAddress("0x000"),
		SessionID:  session.ID("test"),
		Proposal: market.ServiceProposal{
			ServiceType: "just a test",
		},
	},
}

func mockSignerFactory(_ identity.Identity) identity.Signer {
	return &identity.SignerFake{}
}

func TestStatisticsReporterStartsAndStops(t *testing.T) {
	mockSender := newMockRemoteSender()
	reporter := NewSessionStatisticsReporter(mockSender, mockSignerFactory, time.Minute)

	reporter.consumeSessionEvent(mockSessionEvent)

	reporter.start(mockSessionEvent.SessionInfo)
	reporter.stop()

	assert.NoError(t, waitForChannel(mockSender.called, time.Millisecond*200))
	assert.False(t, reporter.started)
}

func TestStatisticsReporterInterval(t *testing.T) {
	mockSender := newMockRemoteSender()
	reporter := NewSessionStatisticsReporter(mockSender, mockSignerFactory, time.Nanosecond)

	reporter.consumeSessionEvent(mockSessionEvent)

	reporter.start(mockSessionEvent.SessionInfo)
	assert.NoError(t, waitForChannel(mockSender.called, time.Millisecond*200))

	reporter.stop()
}

func TestStatisticsReporterConsumeSessionEvent(t *testing.T) {
	mockSender := newMockRemoteSender()
	reporter := NewSessionStatisticsReporter(mockSender, mockSignerFactory, time.Nanosecond)
	reporter.consumeSessionEvent(mockSessionEvent)
	<-mockSender.called
	assert.True(t, reporter.started)
	copy := mockSessionEvent
	copy.Status = connectionstate.SessionEndedStatus
	reporter.consumeSessionEvent(copy)
	assert.False(t, reporter.started)
}

func waitForChannel(ch chan bool, duration time.Duration) error {
	select {
	case <-ch:
		return nil
	case <-time.After(duration):
		return errors.New("timed out waiting for channel")
	}
}

type mockRemoteSender struct {
	called chan bool
}

func (mrs *mockRemoteSender) SendSessionStats(id session.ID, stats mysterium.SessionStats, signer identity.Signer) error {
	mrs.called <- true
	return nil
}

func newMockRemoteSender() *mockRemoteSender {
	return &mockRemoteSender{
		called: make(chan bool),
	}
}

var _ Reporter = &mockRemoteSender{}
