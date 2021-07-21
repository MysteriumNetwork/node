/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package bcutil

import (
	"github.com/mysteriumnetwork/payments/client"
	"github.com/rs/zerolog/log"
)

// ManageMultiClient will read from the given `ch` channel and re-order
// the clients putting the working ones first and pushing the non working ones back.
func ManageMultiClient(mc *client.EthMultiClient, ch chan string) {
	for rpc := range ch {
		log.Warn().Msgf("received a notification for blockchain client down: %s", rpc)

		currentOrder := mc.CurrentClientOrder()
		if len(currentOrder) == 1 {
			continue
		}

		for i, current := range currentOrder {
			if current != rpc {
				continue
			}

			if len(currentOrder)-1 == i {
				break
			}

			newOrder := append(currentOrder[:i], currentOrder[i+1:]...)
			if err := mc.ReorderClients(append(newOrder, rpc)); err != nil {
				log.Err(err).Msg("failed to re-order the RPC client slice")
			}

			break
		}

	}
}
