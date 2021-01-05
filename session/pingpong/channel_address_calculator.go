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

package pingpong

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/crypto"
)

// ChannelAddressCalculator calculates the channel addresses for consumer.
type ChannelAddressCalculator struct {
	hermesAddress         string
	channelImplementation string
	registryAddress       string
}

// NewChannelAddressCalculator returns a new instance of channel address calculator.
func NewChannelAddressCalculator(hermesSCAddress, channelImplementation, registryAddress string) *ChannelAddressCalculator {
	return &ChannelAddressCalculator{
		hermesAddress:         hermesSCAddress,
		channelImplementation: channelImplementation,
		registryAddress:       registryAddress,
	}
}

// GetChannelAddress returns channel id.
func (cac *ChannelAddressCalculator) GetChannelAddress(id identity.Identity) (common.Address, error) {
	addr, err := crypto.GenerateChannelAddress(id.Address, cac.hermesAddress, cac.registryAddress, cac.channelImplementation)
	return common.HexToAddress(addr), err
}
