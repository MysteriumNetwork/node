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

package mysterium

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/requests"
)

// MysteriumAPI provides access to Mysterium owned central discovery service.
type MysteriumAPI struct {
	httpClient          *requests.HTTPClient
	discoveryAPIAddress string
}

// NewClient creates a Discovery client.
func NewClient(httpClient *requests.HTTPClient, discoveryAPIAddress string) *MysteriumAPI {
	return &MysteriumAPI{
		httpClient:          httpClient,
		discoveryAPIAddress: discoveryAPIAddress,
	}
}

// QueryProposals returns active service proposals.
func (mApi *MysteriumAPI) QueryProposals(query ProposalsQuery) ([]market.ServiceProposal, error) {
	req, err := requests.NewGetRequest(mApi.discoveryAPIAddress, "proposals", query.ToURLValues())
	if err != nil {
		return nil, err
	}

	res, err := mApi.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "cannot fetch proposals")
	}
	defer res.Body.Close()

	if err := requests.ParseResponseError(res); err != nil {
		return nil, err
	}

	var proposals []market.ServiceProposal
	if err := requests.ParseResponseJSON(res, &proposals); err != nil {
		return nil, errors.Wrap(err, "cannot parse proposals response")
	}

	total := len(proposals)
	supported := supportedProposalsOnly(proposals)
	log.Debug().Msgf("Total proposals: %d supported: %d", total, len(supported))
	return supported, nil
}

func supportedProposalsOnly(proposals []market.ServiceProposal) (supported []market.ServiceProposal) {
	for _, proposal := range proposals {
		if proposal.Validate() == nil && proposal.IsSupported() {
			supported = append(supported, proposal)
		}
	}
	return
}
