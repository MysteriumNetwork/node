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
	"time"

	"github.com/gofrs/uuid"
	"github.com/mysteriumnetwork/node/identity"
)

// ID represents session id type.
type ID string

// PaymentEngine is responsible for interacting with the consumer in regard to payments.
type PaymentEngine interface {
	Start() error
	Stop()
}

// DataTransferred represents the data transferred on each session.
type DataTransferred struct {
	Up, Down uint64
}

// Session structure holds all required information about current session between service consumer and provider.
type Session struct {
	ID              ID
	ConsumerID      identity.Identity
	Config          ServiceConfiguration
	ServiceID       string
	ServiceType     string
	CreatedAt       time.Time
	DataTransferred DataTransferred
	TokensEarned    uint64
	Last            bool
	done            chan struct{}
}

// Done returns readonly done channel.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

// NewSession creates a blank new session with an ID.
func NewSession() (*Session, error) {
	uid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	return &Session{ID: ID(uid.String())}, nil
}

// ServiceConfiguration defines service configuration from underlying transport mechanism to be passed to remote party
// should be serializable to json format.
type ServiceConfiguration interface{}
