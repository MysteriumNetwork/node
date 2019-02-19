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

package ip

// NewResolverFake returns fakeResolver which resolves statically entered IP
func NewResolverFake(ipAddress string) Resolver {
	return &fakeResolver{
		ipAddress: ipAddress,
		error:     nil,
	}
}

// NewResolverFakeFailing returns fakeResolver with entered error
func NewResolverFakeFailing(err error) Resolver {
	return &fakeResolver{
		ipAddress: "",
		error:     err,
	}
}

type fakeResolver struct {
	ipAddress string
	error     error
}

func (client *fakeResolver) GetPublicIP() (string, error) {
	return client.ipAddress, client.error
}

func (client *fakeResolver) GetOutboundIP() (string, error) {
	return client.ipAddress, client.error
}
