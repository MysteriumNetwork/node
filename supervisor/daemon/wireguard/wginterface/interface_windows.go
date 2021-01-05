/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package wginterface

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"

	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
)

func createTunnel(interfaceName string) (tunnel tun.Device, _ string, err error) {
	log.Info().Msg("Creating Wintun interface")
	wintun, err := tun.CreateTUN(interfaceName, 0)
	if err != nil {
		return nil, interfaceName, fmt.Errorf("could not create Wintun tunnel: %w", err)
	}
	nativeTun := wintun.(*tun.NativeTun)
	wintunVersion, ndisVersion, err := nativeTun.Version()
	if err != nil {
		log.Warn().Err(err).Msg("Unable to determine Wintun version")
	} else {
		log.Info().Msgf("Using Wintun/%s (NDIS %s)", wintunVersion, ndisVersion)
	}
	return wintun, interfaceName, nil
}

func newUAPIListener(interfaceName string) (listener net.Listener, err error) {
	uapi, err := ipc.UAPIListen(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("could not listen for UAPI wg configuration: %w", err)
	}
	return uapi, nil
}

func applySocketPermissions(_ string, _ string) error {
	return nil
}
