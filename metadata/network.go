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

package metadata

// NetworkDefinition structure holds all parameters which describe particular network
type NetworkDefinition struct {
	MysteriumAPIAddress       string
	AccessPolicyOracleAddress string
	BrokerAddresses           []string
	TransactorAddress         string
	TransactorIdentity        string
	Chain1                    ChainDefinition
	Chain2                    ChainDefinition
	MMNAddress                string
	MMNAPIAddress             string
	PilvytisAddress           string
	DNSMap                    map[string][]string
	DefaultChainID            int64
	DefaultCurrency           string
	LocationAddress           string
	Payments                  Payments
}

// ChainDefinition defines the configuration for the chain.
type ChainDefinition struct {
	RegistryAddress    string
	HermesID           string
	ChannelImplAddress string
	ChainID            int64
	MystAddress        string
	EtherClientRPC     []string
}

// Payments defines payments configuration
type Payments struct {
	Consumer Consumer
}

// Consumer defines consumer side settings
type Consumer struct {
	DataLeewayMegabytes uint64
	PriceGiBMax         string
	PriceHourMax        string
	EtherClientRPC      string
}

// Testnet3Definition defines parameters for testnet3 network (currently default network)
var Testnet3Definition = NetworkDefinition{
	MysteriumAPIAddress:       "https://testnet3-discovery.mysterium.network/api/v3",
	AccessPolicyOracleAddress: "https://testnet3-trust.mysterium.network/api/v1/access-policies/",
	BrokerAddresses:           []string{"nats://testnet3-broker.mysterium.network"},
	TransactorIdentity:        "0x7d72db0c2db675ea5107caba80acac2154ca362b",
	TransactorAddress:         "https://testnet3-transactor.mysterium.network/api/v1",
	Chain1: ChainDefinition{
		RegistryAddress:    "0xDFAB03C9fbDbef66dA105B88776B35bfd7743D39",
		ChannelImplAddress: "0x1aDF7Ef34b9d48DCc8EBC47D989bfdE55933B6ea",
		HermesID:           "0x7119442C7E627438deb0ec59291e31378F88DD06",
		ChainID:            5,
		MystAddress:        "0xf74a5ca65E4552CfF0f13b116113cCb493c580C5",
		EtherClientRPC: []string{
			"wss://goerli.infura.io/ws/v3/c2c7da73fcc84ec5885a7bb0eb3c3637",
		},
	},
	Chain2: ChainDefinition{
		RegistryAddress:    "0xDFAB03C9fbDbef66dA105B88776B35bfd7743D39",
		ChannelImplAddress: "0xf8982Ba93D3d9182D095B892DE2A7963eF9807ee",
		HermesID:           "0x7119442C7E627438deb0ec59291e31378F88DD06",
		ChainID:            80001,
		MystAddress:        "0xB923b52b60E247E34f9afE6B3fa5aCcBAea829E8",
		EtherClientRPC: []string{
			"wss://rpc-mumbai.maticvigil.com/ws/v1/8b19c8a6bfbeaee32af34d8df479b1c9558571a5",
		},
	},
	MMNAddress:      "https://mmn3.mysterium.network/",
	MMNAPIAddress:   "https://mmn3.mysterium.network/api/v1",
	PilvytisAddress: "https://testnet3-pilvytis.mysterium.network/api/v1",
	DNSMap: map[string][]string{
		"testnet3-discovery.mysterium.network":  {"167.233.11.60"},
		"testnet3-trust.mysterium.network":      {"167.233.11.60"},
		"testnet3-broker.mysterium.network":     {"167.233.11.60"},
		"testnet3-transactor.mysterium.network": {"167.233.11.60"},
		"testnet3-pilvytis.mysterium.network":   {"167.233.11.60"},
	},
	DefaultChainID:  80001,
	DefaultCurrency: "MYSTT",
	LocationAddress: "https://testnet3-location.mysterium.network/api/v1/location",
	Payments: Payments{
		Consumer: Consumer{
			DataLeewayMegabytes: 20,
			PriceGiBMax:         "500000000000000000", // 0.5 MYSTT
			PriceHourMax:        "180000000000000",    // 0.0018 MYSTT
		},
	},
}

// LocalnetDefinition defines parameters for local network
// Expects discovery, broker and morqa services on localhost
var LocalnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "http://localhost:8001/v1",
	AccessPolicyOracleAddress: "https://localhost:8081/api/v1/access-policies/",
	BrokerAddresses:           []string{"localhost"},
	MMNAddress:                "http://localhost/",
	MMNAPIAddress:             "http://localhost/api/v1",
	PilvytisAddress:           "http://localhost:8002/api/v1",
	DNSMap: map[string][]string{
		"localhost": {"127.0.0.1"},
	},
	DefaultChainID: 1,
}

// DefaultNetwork defines default network values when no runtime parameters are given
var DefaultNetwork = Testnet3Definition
