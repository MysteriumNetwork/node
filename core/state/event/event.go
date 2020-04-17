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

package event

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/consumer/bandwidth"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/payments/crypto"
)

// AppTopicState is the topic that we use to announce state changes to via the event bus
const AppTopicState = "State change"

// State represents the node state at the current moment. It's a read only object, used only to display data.
type State struct {
	NATStatus  NATStatus
	Services   []ServiceInfo
	Sessions   []ServiceSession
	Connection Connection
	Identities []Identity
}

// Identity represents identity and its status.
type Identity struct {
	Address            string
	RegistrationStatus registry.RegistrationStatus
	ChannelAddress     common.Address
	Balance            uint64
	Earnings           uint64
	EarningsTotal      uint64
}

// Connection represents consumer connection state.
type Connection struct {
	Session    connection.Status
	Statistics connection.Statistics
	Throughput bandwidth.Throughput
	Invoice    crypto.Invoice
}

func (c Connection) String() string {
	return fmt.Sprintf(
		"ID %s %s duration: %s data: %s/%s, throughput: %s/%s, spent: %s",
		c.Session.SessionID,
		c.Session.State,
		c.Session.Duration(),
		datasize.FromBytes(c.Statistics.BytesReceived),
		datasize.FromBytes(c.Statistics.BytesSent),
		c.Throughput.Down,
		c.Throughput.Up,
		money.NewMoney(c.Invoice.AgreementTotal, money.CurrencyMyst),
	)
}

// NATStatus stores the nat status related information
// swagger:model NATStatusDTO
type NATStatus struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

// ConnectionStatistics shows the successful and attempted connection count
type ConnectionStatistics struct {
	Attempted  int `json:"attempted"`
	Successful int `json:"successful"`
}

// ServiceInfo stores the information about a service
type ServiceInfo struct {
	ID                   string                 `json:"id"`
	ProviderID           string                 `json:"provider_id"`
	Type                 string                 `json:"type"`
	Options              interface{}            `json:"options"`
	Status               string                 `json:"status"`
	Proposal             market.ServiceProposal `json:"proposal"`
	AccessPolicies       *[]market.AccessPolicy `json:"access_policies,omitempty"`
	Sessions             []ServiceSession       `json:"service_session,omitempty"`
	ConnectionStatistics ConnectionStatistics   `json:"connection_statistics"`
}

// ServiceSession represents the session object
// swagger:model ServiceSessionDTO
type ServiceSession struct {
	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	ID string `json:"id"`
	// example: 0x0000000000000000000000000000000000000001
	ConsumerID string `json:"consumer_id"`
	// example: 2019-06-06T11:04:43.910035Z
	CreatedAt time.Time `json:"created_at"`
	// example: 12345
	BytesOut uint64 `json:"bytes_out"`
	// example: 23451
	BytesIn uint64 `json:"bytes_in"`
	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	ServiceID string `json:"service_id"`
	// example: wireguard
	ServiceType string `json:"service_type"`
	// example: 500000
	TokensEarned uint64 `json:"tokens_earned"`
}
