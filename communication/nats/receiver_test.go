/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReceiverInterface(t *testing.T) {
	var _ communication.Receiver = &receiverNATS{}
}

func TestReceiverNew(t *testing.T) {
	connection := &connectionFake{}
	codec := communication.NewCodecFake()

	assert.Equal(
		t,
		&receiverNATS{
			connection:   connection,
			codec:        codec,
			messageTopic: "custom.",
		},
		NewReceiver(connection, codec, "custom"),
	)
}
