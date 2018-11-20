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

package stats

import (
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/server"
	"github.com/mysteriumnetwork/node/server/dto"
	"github.com/mysteriumnetwork/node/session"
)

const statsSenderLogPrefix = "[session-stats-sender] "

// RemoteStatsSender sends session stats to remote API server with a fixed sendInterval.
// Extra one send will be done on session disconnect.
type RemoteStatsSender struct {
	sessionID       session.ID
	providerID      identity.Identity
	consumerCountry string
	serviceType     string

	signer          identity.Signer
	statsKeeper     SessionStatsKeeper
	mysteriumClient server.Client

	sendInterval time.Duration
	done         chan struct{}

	opLock  sync.Mutex
	started bool
}

// NewRemoteStatsSender function creates new session stats sender by given options
func NewRemoteStatsSender(statsKeeper SessionStatsKeeper, mysteriumClient server.Client, sessionID session.ID, providerID identity.Identity, serviceType string, signer identity.Signer, consumerCountry string, interval time.Duration) *RemoteStatsSender {
	return &RemoteStatsSender{
		sessionID:       sessionID,
		providerID:      providerID,
		consumerCountry: consumerCountry,
		serviceType:     serviceType,

		signer:          signer,
		statsKeeper:     statsKeeper,
		mysteriumClient: mysteriumClient,

		sendInterval: interval,
		done:         make(chan struct{}),
	}
}

// Start starts the sending of stats
func (rss *RemoteStatsSender) Start() {
	rss.opLock.Lock()
	defer rss.opLock.Unlock()

	if rss.started {
		return
	}

	rss.done = make(chan struct{})
	go rss.intervalSend()
	rss.started = true
}

// Stop stops the sending of stats
func (rss *RemoteStatsSender) Stop() {
	rss.opLock.Lock()
	defer rss.opLock.Unlock()

	if !rss.started {
		return
	}

	close(rss.done)
	rss.started = false
}

func (rss *RemoteStatsSender) intervalSend() {
	for {
		select {
		case <-rss.done:
			if err := rss.send(); err != nil {
				log.Error(statsSenderLogPrefix, "Failed to send session stats to the remote service: ", err)
			}
			return
		case <-time.After(rss.sendInterval):
			if err := rss.send(); err != nil {
				log.Error(statsSenderLogPrefix, "Failed to send session stats to the remote service: ", err)
			}
		}
	}
}

func (rss *RemoteStatsSender) send() error {
	sessionStats := rss.statsKeeper.Retrieve()
	return rss.mysteriumClient.SendSessionStats(
		rss.sessionID,
		dto.SessionStats{
			ServiceType:     rss.serviceType,
			BytesSent:       sessionStats.BytesSent,
			BytesReceived:   sessionStats.BytesReceived,
			ProviderID:      rss.providerID.Address,
			ConsumerCountry: rss.consumerCountry,
		},
		rss.signer,
	)
}
