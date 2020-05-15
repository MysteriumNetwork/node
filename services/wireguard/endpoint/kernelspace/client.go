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

package kernelspace

import (
	"encoding/base64"
	"fmt"
	"net"
	"time"

	"github.com/jackpal/gateway"
	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/mysteriumnetwork/node/utils/cmdutil"
	"github.com/pkg/errors"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type client struct {
	iface    string
	wgClient *wgctrl.Client
}

// NewWireguardClient creates new wireguard kernel space client.
func NewWireguardClient() (*client, error) {
	wgClient, err := wgctrl.New()
	if err != nil {
		return nil, err
	}
	return &client{wgClient: wgClient}, nil
}

func (c *client) ConfigureDevice(config wg.DeviceConfig) error {
	var deviceConfig wgtypes.Config
	port := config.ListenPort
	privateKey, err := stringToKey(config.PrivateKey)
	if err != nil {
		return err
	}
	deviceConfig.PrivateKey = &privateKey
	deviceConfig.ListenPort = &port
	if err := c.up(config.IfaceName, config.Subnet); err != nil {
		return err
	}
	c.iface = config.IfaceName
	return c.wgClient.ConfigureDevice(c.iface, deviceConfig)
}

func (c *client) AddPeer(iface string, peer wg.Peer) error {
	endpoint := peer.Endpoint
	publicKey, err := stringToKey(peer.PublicKey)
	if err != nil {
		return errors.Wrap(err, "could not convert string key to wgtypes.Key")
	}

	// Apply keep alive interval
	var keepAliveInterval *time.Duration
	if peer.KeepAlivePeriodSeconds > 0 {
		interval := time.Duration(peer.KeepAlivePeriodSeconds) * time.Second
		keepAliveInterval = &interval
	}

	// Apply allowed IPs network
	var allowedIPs []net.IPNet
	for _, ip := range peer.AllowedIPs {
		_, network, err := net.ParseCIDR(ip)
		if err != nil {
			return fmt.Errorf("could not parse IP %q: %v", ip, err)
		}
		allowedIPs = append(allowedIPs, *network)
	}

	var deviceConfig wgtypes.Config
	deviceConfig.Peers = []wgtypes.PeerConfig{{
		Endpoint:                    endpoint,
		PublicKey:                   publicKey,
		AllowedIPs:                  allowedIPs,
		PersistentKeepaliveInterval: keepAliveInterval,
	}}
	return c.wgClient.ConfigureDevice(iface, deviceConfig)
}

func (c *client) RemovePeer(iface string, publicKey string) error {
	key, err := stringToKey(publicKey)
	if err != nil {
		return err
	}

	return c.wgClient.ConfigureDevice(iface, wgtypes.Config{Peers: []wgtypes.PeerConfig{{
		PublicKey: key,
		Remove:    true,
	}}})
}

func (c *client) PeerStats(string) (*wg.Stats, error) {
	d, err := c.wgClient.Device(c.iface)
	if err != nil {
		return nil, err
	}

	if len(d.Peers) != 1 {
		return nil, errors.New("kernelspace: exactly 1 peer expected")
	}

	return &wg.Stats{
		BytesReceived: uint64(d.Peers[0].ReceiveBytes),
		BytesSent:     uint64(d.Peers[0].TransmitBytes),
		LastHandshake: d.Peers[0].LastHandshakeTime,
	}, nil
}

func (c *client) DestroyDevice(name string) error {
	return cmdutil.SudoExec("ip", "link", "del", "dev", name)
}

func (c *client) up(iface string, ipAddr net.IPNet) error {
	if d, err := c.wgClient.Device(iface); err != nil || d.Name != iface {
		if err := cmdutil.SudoExec("ip", "link", "add", "dev", iface, "type", "wireguard"); err != nil {
			return err
		}
	}

	if err := cmdutil.SudoExec("ip", "address", "replace", "dev", iface, ipAddr.String()); err != nil {
		return err
	}

	return cmdutil.SudoExec("ip", "link", "set", "dev", iface, "up")
}

func (c *client) ConfigureRoutes(iface string, ip net.IP) error {
	if err := excludeRoute(ip); err != nil {
		return err
	}
	return addDefaultRoute(iface)
}

func excludeRoute(ip net.IP) error {
	gw, err := gateway.DiscoverGateway()
	if err != nil {
		return err
	}

	return cmdutil.SudoExec("ip", "route", "replace", ip.String(), "via", gw.String())
}

func addDefaultRoute(iface string) error {
	if err := cmdutil.SudoExec("ip", "route", "replace", "0.0.0.0/1", "dev", iface); err != nil {
		return err
	}
	return cmdutil.SudoExec("ip", "route", "replace", "128.0.0.0/1", "dev", iface)
}

func (c *client) Close() (err error) {
	errs := utils.ErrorCollection{}
	if err := c.DestroyDevice(c.iface); err != nil {
		errs.Add(err)
	}
	if err := c.wgClient.Close(); err != nil {
		errs.Add(err)
	}
	if err := errs.Error(); err != nil {
		return fmt.Errorf("could not close client: %w", err)
	}
	return nil
}

func stringToKey(key string) (wgtypes.Key, error) {
	k, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return wgtypes.Key{}, err
	}
	return wgtypes.NewKey(k)
}
