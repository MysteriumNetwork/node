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
	"time"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

// TimeGetter function returns current time
type TimeGetter func() time.Time

// SessionStatisticsTracker keeps the session stats safe and sound
type SessionStatisticsTracker struct {
	lastInvoice  crypto.Invoice
	lastStats    connection.Statistics
	sessionStats connection.Statistics
	timeGetter   TimeGetter
	sessionStart *time.Time
}

// NewSessionStatisticsTracker returns new session stats statisticsTracker with given timeGetter function
func NewSessionStatisticsTracker(timeGetter TimeGetter) *SessionStatisticsTracker {
	return &SessionStatisticsTracker{timeGetter: timeGetter}
}

// GetDataStats returns session data stats
func (sst *SessionStatisticsTracker) GetDataStats() connection.Statistics {
	return sst.sessionStats
}

// GetDuration returns elapsed time from marked session start
func (sst *SessionStatisticsTracker) GetDuration() time.Duration {
	if sst.sessionStart == nil {
		return time.Duration(0)
	}
	duration := sst.timeGetter().Sub(*sst.sessionStart)
	return duration
}

// GetInvoice retrieves session payment stats
func (sst *SessionStatisticsTracker) GetInvoice() crypto.Invoice {
	return sst.lastInvoice
}

// MarkSessionStart marks current time as session start time for statistics
func (sst *SessionStatisticsTracker) markSessionStart() {
	time := sst.timeGetter()
	sst.sessionStart = &time
	// reset the stats in preparation for a new session
	sst.sessionStats = connection.Statistics{}
}

// MarkSessionEnd stops counting session duration
func (sst *SessionStatisticsTracker) markSessionEnd() {
	sst.sessionStart = nil
}

// ConsumeStatisticsEvent handles the connection statistics changes
func (sst *SessionStatisticsTracker) ConsumeStatisticsEvent(e connection.SessionStatsEvent) {
	sst.sessionStats = sst.sessionStats.Plus(sst.lastStats.Diff(e.Stats))
	sst.lastStats = e.Stats
	log.Trace().Msg(sst.sessionStats.String())
}

// ConsumeSessionEvent handles the session state changes
func (sst *SessionStatisticsTracker) ConsumeSessionEvent(sessionEvent connection.SessionEvent) {
	switch sessionEvent.Status {
	case connection.SessionEndedStatus:
		sst.markSessionEnd()
	case connection.SessionCreatedStatus:
		sst.markSessionStart()
	}
}

// ConsumeInvoiceEvent handles the connection statistics changes
func (sst *SessionStatisticsTracker) ConsumeInvoiceEvent(e pingpong.AppEventInvoicePaid) {
	sst.lastInvoice = e.Invoice
}
