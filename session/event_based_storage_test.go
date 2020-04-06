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

package session

import (
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/mocks"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	"github.com/stretchr/testify/assert"
)

func TestEventBasedStorage_PublishesEventsOnCreate(t *testing.T) {
	session := expectedSession
	mp := mocks.NewEventBus()
	sessionStore := NewEventBasedStorage(mp, NewStorageMemory())

	sessionStore.Add(session)

	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.Created), 2*time.Second, 10*time.Millisecond)
}

func TestEventBasedStorage_PublishesEventsOnDelete(t *testing.T) {
	session := expectedSession
	mp := mocks.NewEventBus()
	sessionStore := NewEventBasedStorage(mp, NewStorageMemory())
	sessionStore.Add(session)
	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.Created), 1*time.Second, 5*time.Millisecond)

	sessionStore.Remove(session.ID)

	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.Removed), 2*time.Second, 10*time.Millisecond)
}

func TestEventBasedStorage_PublishesEventsOnDataTransferUpdate(t *testing.T) {
	session := expectedSession
	mp := mocks.NewEventBus()
	sessionStore := NewEventBasedStorage(mp, NewStorageMemory())
	sessionStore.Add(session)

	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.Created), 1*time.Second, 5*time.Millisecond)

	sessionStore.UpdateDataTransfer(session.ID, 1, 2)

	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.Updated), 2*time.Second, 10*time.Millisecond)
}

func TestNewEventBasedStorage_HandlesAppEventTokensEarned(t *testing.T) {
	// given
	session := expectedSession
	mp := mocks.NewEventBus()
	sessionStore := NewEventBasedStorage(mp, NewStorageMemory())
	sessionStore.Add(session)

	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.Created), 1*time.Second, 5*time.Millisecond)

	storedSession, ok := sessionStore.Find(session.ID)
	assert.True(t, ok)
	assert.Zero(t, storedSession.TokensEarned)

	// when
	sessionStore.consumeTokensEarnedEvent(sessionEvent.AppEventSessionTokensEarned{
		ProviderID: identity.FromAddress("0x1"),
		SessionID:  string(session.ID),
		Total:      500,
	})
	// then
	storedSession, ok = sessionStore.Find(session.ID)
	assert.True(t, ok)
	assert.EqualValues(t, 500, storedSession.TokensEarned)
	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.Updated), 2*time.Second, 10*time.Millisecond)
}

func TestEventBasedStorage_PublishesEventsOnRemoveForService(t *testing.T) {
	session := expectedSession
	mp := mocks.NewEventBus()
	sessionStore := NewEventBasedStorage(mp, NewStorageMemory())
	sessionStore.Add(session)
	assert.Eventually(t, lastEventMatches(mp, session.ID, sessionEvent.Created), 1*time.Second, 5*time.Millisecond)

	sessionStore.RemoveForService("whatever")

	assert.Eventually(t, lastEventMatches(mp, "", sessionEvent.Removed), 2*time.Second, 10*time.Millisecond)
}

func lastEventMatches(mp *mocks.EventBus, id ID, action sessionEvent.Action) func() bool {
	return func() bool {
		last := mp.Pop()
		evt, ok := last.(sessionEvent.Payload)
		if !ok {
			return false
		}
		return evt.ID == string(id) && evt.Action == action
	}
}
