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
	EtherClientRPC            string
	TransactorAddress         string
	TransactorIdentity        string
	RegistryAddress           string
	HermesID                  string
	ChannelImplAddress        string
	MMNAddress                string
	MMNAPIAddress             string
	DAIAddress                string
	WETHAddress               string
	PilvytisAddress           string
	DNSMap                    map[string][]string
	DefaultChainID            int64
}

// TestnetDefinition defines parameters for test network (currently default network)
var TestnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "https://testnet-api.mysterium.network/v1",
	AccessPolicyOracleAddress: "https://testnet-trust.mysterium.network/api/v1/access-policies/",
	BrokerAddresses:           []string{"nats://testnet-broker.mysterium.network"},
	EtherClientRPC:            "wss://goerli.infura.io/ws/v3/c2c7da73fcc84ec5885a7bb0eb3c3637",
	TransactorAddress:         "https://testnet-transactor.mysterium.network/api/v1",
	TransactorIdentity:        "0x0828d0386c1122e565f07dd28c7d1340ed5b3315",
	RegistryAddress:           "0x3dD81545F3149538EdCb6691A4FfEE1898Bd2ef0",
	ChannelImplAddress:        "0x3026eB9622e2C5bdC157C6b117F7f4aC2C2Db3b5",
	HermesID:                  "0x0214281cf15C1a66b51990e2E65e1f7b7C363318",
	MMNAddress:                "https://my.mysterium.network/",
	MMNAPIAddress:             "https://my.mysterium.network/api/v1",
	DAIAddress:                "0xC496Bae7780C92281F19626F233b1B11f52D38A3",
	WETHAddress:               "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
	DNSMap: map[string][]string{
		"testnet-api.mysterium.network":        {"78.47.176.149"},
		"testnet-trust.mysterium.network":      {"82.196.15.9"},
		"testnet-broker.mysterium.network":     {"82.196.15.9"},
		"testnet-transactor.mysterium.network": {"116.203.17.150"},
		"my.mysterium.network":                 {"168.119.183.173"},
	},
	DefaultChainID: 5,
}

// BetanetDefinition defines parameters for Betanet network (currently default network)
var BetanetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "https://betanet-api.mysterium.network/v1",
	AccessPolicyOracleAddress: "https://betanet-trust.mysterium.network/api/v1/access-policies/",
	BrokerAddresses:           []string{"nats://betanet-broker.mysterium.network"},
	EtherClientRPC:            "wss://goerli.infura.io/ws/v3/c2c7da73fcc84ec5885a7bb0eb3c3637",
	TransactorAddress:         "https://betanet-transactor.mysterium.network/api/v1",
	TransactorIdentity:        "0x45b224f0cd64ed5179502da42ed4e32228485b3b",
	RegistryAddress:           "0x15B1281F4e58215b2c3243d864BdF8b9ddDc0DA2",
	ChannelImplAddress:        "0xc49B987fB8701a41ae65Cf934a811FeA15bCC6E4",
	HermesID:                  "0xD5d2f5729D4581dfacEBedF46C7014DeFda43585",
	MMNAddress:                "https://betanet.mysterium.network/",
	MMNAPIAddress:             "https://betanet.mysterium.network/api/v1",
	DAIAddress:                "0xC496Bae7780C92281F19626F233b1B11f52D38A3",
	WETHAddress:               "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
	PilvytisAddress:           "https://betanet-pilvytis.mysterium.network/api/v1",
	DNSMap: map[string][]string{
		"betanet-api.mysterium.network":        {"78.47.55.197"},
		"betanet-trust.mysterium.network":      {"95.216.204.232"},
		"betanet-broker.mysterium.network":     {"95.216.204.232"},
		"betanet-transactor.mysterium.network": {"135.181.82.67"},
		"betanet.mysterium.network":            {"138.201.244.63"},
	},
	DefaultChainID: 5,
}

// LocalnetDefinition defines parameters for local network
// Expects discovery, broker and morqa services on localhost
var LocalnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "http://localhost:8001/v1",
	AccessPolicyOracleAddress: "https://localhost:8081/api/v1/access-policies/",
	BrokerAddresses:           []string{"localhost"},
	EtherClientRPC:            "http://localhost:8545",
	MMNAddress:                "http://localhost/",
	MMNAPIAddress:             "http://localhost/api/v1",
	PilvytisAddress:           "http://localhost:8002/api/v1",
	DNSMap: map[string][]string{
		"localhost": {"127.0.0.1"},
	},
	DefaultChainID: 1,
}

// DefaultNetwork defines default network values when no runtime parameters are given
var DefaultNetwork = BetanetDefinition
