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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

type hermesAddressProvider func(hermes common.Address) (common.Address, error)

// HermesOwnerAddressCache gets and caches hermes addresses
type HermesOwnerAddressCache struct {
	hermesOwners          map[common.Address]common.Address
	lock                  sync.Mutex
	hermesAddressProvider hermesAddressProvider
}

// NewHermesOwnerAddressCache returns a new instance of hermes owner address cache
func NewHermesOwnerAddressCache(hermesAddressProvider hermesAddressProvider) *HermesOwnerAddressCache {
	return &HermesOwnerAddressCache{
		hermesOwners:          make(map[common.Address]common.Address),
		hermesAddressProvider: hermesAddressProvider,
	}
}

// GetHermesOwnerAddress gets the hermes address and keeps it in cache
func (aoac *HermesOwnerAddressCache) GetHermesOwnerAddress(id common.Address) (common.Address, error) {
	aoac.lock.Lock()
	defer aoac.lock.Unlock()
	if v, ok := aoac.hermesOwners[id]; ok {
		return v, nil
	}

	addr, err := aoac.hermesAddressProvider(id)
	if err != nil {
		return common.Address{}, errors.Wrap(err, "could not get hermes address")
	}

	aoac.hermesOwners[id] = addr
	return addr, nil
}
