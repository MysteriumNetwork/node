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

package traversal

import (
	"context"
	"net"
)

// NoopPinger does nothing
type NoopPinger struct{}

// NewNoopPinger returns noop nat pinger
func NewNoopPinger() *NoopPinger {
	return &NoopPinger{}
}

// PingProviderPeer does nothing.
func (np *NoopPinger) PingProviderPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
	return []*net.UDPConn{}, nil
}

// PingConsumerPeer does nothing.
func (np *NoopPinger) PingConsumerPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
	return []*net.UDPConn{}, nil
}

// StopNATProxy does nothing
func (np *NoopPinger) StopNATProxy() {}

// SetProtectSocketCallback does nothing
func (np *NoopPinger) SetProtectSocketCallback(socketProtect func(socket int) bool) {}

// Stop does nothing
func (np *NoopPinger) Stop() {}

// BindServicePort does nothing
func (np *NoopPinger) BindServicePort(key string, port int) {}

// PingPeer does nothing.
func (np *NoopPinger) PingPeer(ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error) {
	return nil, nil
}
