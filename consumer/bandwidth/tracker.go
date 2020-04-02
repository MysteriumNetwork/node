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

package bandwidth

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/rs/zerolog/log"
)

const bitsInByte = 8

// Throughput represents the throughput
type Throughput struct {
	BitsPerSecond float64
}

// String returns human readable form of the throughput
func (t Throughput) String() string {
	return datasize.BitSize(t.BitsPerSecond).String() + "/s"
}

// CurrentSpeed represents the current(moment) download and upload speeds in bits per second
type CurrentSpeed struct {
	Up, Down Throughput
}

// Tracker keeps track of current speed
type Tracker struct {
	previous     connection.Statistics
	currentSpeed CurrentSpeed
	lock         sync.RWMutex
}

// Get returns the current upload and download speeds in bits per second
func (t *Tracker) Get() CurrentSpeed {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.currentSpeed
}

const consumeCooldown = 500 * time.Millisecond

// ConsumeStatisticsEvent handles the connection statistics changes
func (t *Tracker) ConsumeStatisticsEvent(evt connection.SessionStatsEvent) {
	t.lock.Lock()
	defer func() {
		t.lock.Unlock()
	}()

	// Skip speed calculation on the very first event.
	if t.previous.At.IsZero() {
		t.previous = evt.Stats
		return
	}

	secondsSince := evt.Stats.At.Sub(t.previous.At).Seconds()
	if secondsSince < consumeCooldown.Seconds() {
		log.Trace().Msgf("%fs passed since the last consumption, ignoring the event", secondsSince)
		return
	}

	byteDownDiff := evt.Stats.BytesReceived - t.previous.BytesReceived
	byteUpDiff := evt.Stats.BytesSent - t.previous.BytesSent

	t.currentSpeed = CurrentSpeed{
		Up:   Throughput{BitsPerSecond: float64(byteUpDiff) / secondsSince * bitsInByte},
		Down: Throughput{BitsPerSecond: float64(byteDownDiff) / secondsSince * bitsInByte},
	}
	t.previous = evt.Stats

	log.Trace().Msgf("Download speed: %s", t.currentSpeed.Down)
	log.Trace().Msgf("Upload speed: %s", t.currentSpeed.Up)
}

// ConsumeSessionEvent handles the session state changes
func (t *Tracker) ConsumeSessionEvent(sessionEvent connection.SessionEvent) {
	t.lock.Lock()
	defer t.lock.Unlock()
	switch sessionEvent.Status {
	case connection.SessionEndedStatus, connection.SessionCreatedStatus:
		t.previous = connection.Statistics{}
		t.currentSpeed = CurrentSpeed{}
	}
}
