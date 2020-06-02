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
	"bufio"
	"fmt"
	"log"
	"strings"

	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
)

// New creates new WgInterface instance.
func New(cfg wgcfg.DeviceConfig, uid string) (*WgInterface, error) {
	log.Println("Creating Wintun interface")

	wintun, err := tun.CreateTUN(cfg.IfaceName, 0)
	if err != nil {
		return nil, fmt.Errorf("could not create wintun: %w", err)
	}
	nativeTun := wintun.(*tun.NativeTun)
	wintunVersion, ndisVersion, err := nativeTun.Version()
	if err != nil {
		log.Printf("Warning: unable to determine Wintun version: %v", err)
	} else {
		log.Printf("Using Wintun/%s (NDIS %s)", wintunVersion, ndisVersion)
	}

	log.Println("Creating interface instance")
	// TODO: Use ring logger?
	logger := device.NewLogger(device.LogLevelDebug, fmt.Sprintf("(%s) ", cfg.IfaceName))
	logger.Info.Println("Starting wireguard-go version", device.WireGuardGoVersion)
	wgDevice := device.NewDevice(wintun, logger)

	log.Println("Setting interface configuration")
	uapi, err := ipc.UAPIListen(cfg.IfaceName)
	if err != nil {
		return nil, fmt.Errorf("could not listen for user API wg configuration: %w", err)
	}
	if err := wgDevice.IpcSetOperation(bufio.NewReader(strings.NewReader(cfg.Encode()))); err != nil {
		return nil, fmt.Errorf("could not set device uapi config: %w", err)
	}

	log.Println("Bringing peers up")
	wgDevice.Up()

	log.Println("Configuring network")
	if err := configureNetwork(cfg); err != nil {
		return nil, fmt.Errorf("could not setup network: %w", err)
	}

	wgInterface := &WgInterface{
		Name:   cfg.IfaceName,
		Device: wgDevice,
		uapi:   uapi,
	}
	go wgInterface.handleUAPI()

	return wgInterface, nil
}

// handleUAPI listens for WireGuard configuration changes via user space socket.
func (a *WgInterface) handleUAPI() {
	for {
		conn, err := a.uapi.Accept()
		if err != nil {
			log.Println("Closing UAPI listener, err:", err)
			return
		}
		go a.Device.IpcHandle(conn)
	}
}

// Down closes device and user space api socket.
func (a *WgInterface) Down() {
	if err := a.uapi.Close(); err != nil {
		log.Printf("could not close uapi socket: %w", err)
	}
	a.Device.Close()
}

func configureNetwork(cfg wgcfg.DeviceConfig) error {
	if err := netutil.AssignIP(cfg.IfaceName, cfg.Subnet); err != nil {
		return fmt.Errorf("failed to assign IP address: %w", err)
	}

	if cfg.Peer.Endpoint != nil {
		if err := netutil.ExcludeRoute(cfg.Peer.Endpoint.IP); err != nil {
			return fmt.Errorf("could not exclude route %s: %w", cfg.Peer.Endpoint.IP.String(), err)
		}
		if err := netutil.AddDefaultRoute(cfg.IfaceName); err != nil {
			return fmt.Errorf("could not add default route for %s: %w", cfg.IfaceName, err)
		}
	}
	return nil
}
