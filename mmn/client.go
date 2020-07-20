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

package mmn

import (
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
)

// NodeInformationDto contains node information to be sent to MMN
type NodeInformationDto struct {
	// local IP is used to give quick access to WebUI from MMN
	LocalIP     string `json:"local_ip"`
	Identity    string `json:"identity"`
	APIKey      string `json:"api_key"`
	VendorID    string `json:"vendor_id"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	NodeVersion string `json:"node_version"`
}

// NewClient returns MMN API Client
func NewClient(httpClient *requests.HTTPClient, mmnAddress string, signer identity.SignerFactory) *Client {
	return &Client{
		httpClient: httpClient,
		mmnAddress: mmnAddress,
		signer:     signer,
	}
}

type Client struct {
	httpClient *requests.HTTPClient
	mmnAddress string
	signer     identity.SignerFactory
}

func (m *Client) RegisterNode(info *NodeInformationDto) error {
	log.Debug().Msgf("Registering node to MMN: %+v", *info)

	id := identity.FromAddress(info.Identity)
	req, err := requests.NewSignedPostRequest(m.mmnAddress, "node", info, m.signer(id))
	if err != nil {
		return err
	}

	return m.httpClient.DoRequest(req)
}
