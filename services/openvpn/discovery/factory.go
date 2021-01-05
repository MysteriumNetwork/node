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

package discovery

import (
	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
)

// NewServiceProposalWithLocation creates service proposal description for openvpn service
func NewServiceProposalWithLocation(
	loc locationstate.Location,
	protocol string,
) market.ServiceProposal {
	serviceLocation := market.Location{
		Continent: loc.Continent,
		Country:   loc.Country,
		City:      loc.City,
		ASN:       loc.ASN,
		ISP:       loc.ISP,
		NodeType:  loc.NodeType,
	}

	return market.ServiceProposal{
		ServiceType: openvpn.ServiceType,
		ServiceDefinition: dto.ServiceDefinition{
			Location:          serviceLocation,
			LocationOriginate: serviceLocation,
			SessionBandwidth:  dto.Bandwidth(10 * datasize.MiB),
			Protocol:          protocol,
		},
	}
}
