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

package p2p

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/pb"
	nats_lib "github.com/nats-io/go-nats"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

// brokerConnector connects to broker.
type brokerConnector interface {
	Connect(serverURIs ...string) (nats.Connection, error)
}

const pingMaxPorts = 10

// NewManager creates new p2p communication manager.
func NewManager(broker brokerConnector, address string, signer identity.SignerFactory, ipResolver ip.Resolver, natPinger traversal.NATPinger) *Manager {
	return &Manager{
		broker:         broker,
		brokerAddress:  address,
		pendingConfigs: map[PublicKey]*p2pConnectConfig{},
		ipResolver:     ipResolver,
		signer:         signer,
		verifier:       identity.NewVerifierSigned(),
		portPool:       port.NewPool(),
		pinger:         natPinger,
	}
}

// Manager knows how to exchange p2p keys and encrypted configuration and creates ready to use p2p channels.
type Manager struct {
	portPool      *port.Pool
	broker        brokerConnector
	pinger        traversal.NATPinger
	signer        identity.SignerFactory
	verifier      identity.Verifier
	ipResolver    ip.Resolver
	brokerAddress string

	// Keys holds pendingConfigs temporary configs for provider side since it
	// need to handle key exchange in two steps.
	pendingConfigs   map[PublicKey]*p2pConnectConfig
	pendingConfigsMu sync.Mutex
}

type p2pConnectConfig struct {
	peerPublicIP string
	peerPorts    []int
	localPorts   []int
	privateKey   PrivateKey
	peerPubKey   PublicKey
}

// CreateChannel exchanges p2p configuration via broker, performs NAT pinging if needed
// and create p2p channel which is ready for communication.
func (m *Manager) CreateChannel(consumerID, providerID identity.Identity, timeout time.Duration) (*Channel, error) {
	config, err := m.exchangeConsumerConfig(consumerID, providerID, timeout)
	if err != nil {
		return nil, fmt.Errorf("could not exchange config: %w", err)
	}

	var remotePort, localPort int
	var serviceConn *net.UDPConn
	if len(config.peerPorts) == 1 {
		localPort = config.localPorts[0]
		remotePort = config.peerPorts[0]
	} else {
		log.Debug().Msgf("Pinging provider %s with public IP %s using ports %v:%v", providerID.Address, config.peerPublicIP, config.localPorts, config.peerPorts)
		conns, err := m.pinger.PingProviderPeer(config.peerPublicIP, config.localPorts, config.peerPorts, 128, 2)
		if err != nil {
			return nil, fmt.Errorf("could not ping peer: %w", err)
		}
		conn0 := conns[0]
		localPort = conn0.LocalAddr().(*net.UDPAddr).Port
		remotePort = conn0.RemoteAddr().(*net.UDPAddr).Port
		conn0.Close()
		serviceConn = conns[1]
	}

	log.Debug().Msgf("Creating channel with listen port: %d, peer port: %d", localPort, remotePort)
	peer := Peer{
		Addr:      &net.UDPAddr{IP: net.ParseIP(config.peerPublicIP), Port: remotePort},
		PublicKey: config.peerPubKey,
	}
	channel, err := NewChannel(localPort, config.privateKey, &peer)
	if err != nil {
		return nil, fmt.Errorf("could not create p2p channel: %w", err)
	}
	channel.serviceConn = serviceConn
	return channel, nil
}

// SubscribeChannel subscribers to the provider communication channel and handles incoming requests.
func (m *Manager) SubscribeChannel(providerID identity.Identity, channelHandler func(ch *Channel)) error {
	brokerConn, err := m.broker.Connect(m.brokerAddress)
	if err != nil {
		return err
	}
	// TODO: Expose func to close broker conn.

	_, err = brokerConn.Subscribe(fmt.Sprintf("%s.p2p-config-exchange", providerID.Address), func(msg *nats_lib.Msg) {
		if err := m.providerStartConfigExchange(brokerConn, providerID, msg); err != nil {
			log.Err(err).Msg("Could not handle initial exchange")
			return
		}
	})

	_, err = brokerConn.Subscribe(fmt.Sprintf("%s.p2p-config-exchange-ack", providerID.Address), func(msg *nats_lib.Msg) {
		config, err := m.providerAckConfigExchange(msg)
		if err != nil {
			log.Err(err).Msg("Could not handle exchange ack")
			return
		}

		// Send ack in separate goroutine and start pinging.
		// It is important that provider starts sending pings first otherwise
		// providers router can think that consumer is sending DDoS packets.
		go func(reply string) {
			if err := brokerConn.Publish(reply, []byte("OK")); err != nil {
				log.Err(err).Msg("Could publish exchange ack")
			}
		}(msg.Reply)

		var remotePort, localPort int
		var serviceConn *net.UDPConn
		if len(config.peerPorts) == 1 {
			localPort = config.localPorts[0]
			remotePort = config.peerPorts[0]
		} else {
			log.Debug().Msgf("Pinging consumer with public IP %s using ports %v:%v", config.peerPublicIP, config.localPorts, config.peerPorts)
			conns, err := m.pinger.PingConsumerPeer(config.peerPublicIP, config.localPorts, config.peerPorts, 2, 2)
			if err != nil {
				log.Err(err).Msg("Could not ping peer")
				return
			}
			conn0 := conns[0]
			localPort = conn0.LocalAddr().(*net.UDPAddr).Port
			remotePort = conn0.RemoteAddr().(*net.UDPAddr).Port
			conn0.Close()
			serviceConn = conns[1]
		}

		log.Debug().Msgf("Creating channel with listen port: %d, peer port: %d", localPort, remotePort)
		peer := Peer{
			Addr:      &net.UDPAddr{IP: net.ParseIP(config.peerPublicIP), Port: remotePort},
			PublicKey: config.peerPubKey,
		}
		channel, err := NewChannel(localPort, config.privateKey, &peer)
		if err != nil {
			log.Err(err).Msg("Could not create channel")
			return
		}
		channel.serviceConn = serviceConn

		channelHandler(channel)
	})
	return err
}

func (m *Manager) providerStartConfigExchange(brokerConn nats.Connection, signerID identity.Identity, msg *nats_lib.Msg) error {
	pubKey, privateKey, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("could not generate provider p2p keys: %w", err)
	}

	// Get initial peer exchange with it's public key.
	signedMsg, err := m.unpackSignedMsg(msg.Data)
	if err != nil {
		return err
	}
	var peerExchangeMsg pb.P2PConfigExchangeMsg
	if err := proto.Unmarshal(signedMsg.Data, &peerExchangeMsg); err != nil {
		return err
	}
	peerPubKey, err := DecodePublicKey(peerExchangeMsg.PublicKey)
	if err != nil {
		return err
	}
	log.Debug().Msgf("Received consumer public key %s", peerPubKey.Hex())

	// Send reply with encrypted exchange config.
	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return err
	}
	localPorts, err := m.acquireLocalPorts()
	if err != nil {
		return err
	}
	config := pb.P2PConnectConfig{
		PublicIP: publicIP,
		Ports:    intToInt32Slice(localPorts),
	}
	configCiphertext, err := encryptConnConfigMsg(&config, privateKey, peerPubKey)
	if err != nil {
		return err
	}
	exchangeMsg := pb.P2PConfigExchangeMsg{
		PublicKey:        pubKey.Hex(),
		ConfigCiphertext: configCiphertext,
	}
	log.Debug().Msgf("Sending reply with public key %s and encrypted config to consumer", exchangeMsg.PublicKey)
	packedMsg, err := m.packSignedMsg(signerID, &exchangeMsg)
	if err != nil {
		return err
	}
	err = brokerConn.Publish(msg.Reply, packedMsg)
	if err != nil {
		return err
	}

	m.pendingConfigsMu.Lock()
	m.pendingConfigs[peerPubKey] = &p2pConnectConfig{
		localPorts: localPorts,
		privateKey: privateKey,
		peerPubKey: peerPubKey,
	}
	m.pendingConfigsMu.Unlock()
	return nil
}

func (m *Manager) providerAckConfigExchange(msg *nats_lib.Msg) (*p2pConnectConfig, error) {
	signedMsg, err := m.unpackSignedMsg(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("could not unpack signed msg: %w", err)
	}
	var peerExchangeMsg pb.P2PConfigExchangeMsg
	if err := proto.Unmarshal(signedMsg.Data, &peerExchangeMsg); err != nil {
		return nil, fmt.Errorf("could not unmarshal exchange msg: %w", err)
	}
	peerPubKey, err := DecodePublicKey(peerExchangeMsg.PublicKey)
	if err != nil {
		return nil, err
	}

	defer func() {
		m.pendingConfigsMu.Lock()
		delete(m.pendingConfigs, peerPubKey)
		m.pendingConfigsMu.Unlock()
	}()

	m.pendingConfigsMu.Lock()
	config, ok := m.pendingConfigs[peerPubKey]
	m.pendingConfigsMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("pending config not found for key %s", peerPubKey.Hex())
	}

	peerConfig, err := decryptConnConfigMsg(peerExchangeMsg.ConfigCiphertext, config.privateKey, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt peer conn config: %w", err)
	}

	log.Debug().Msgf("Decrypted consumer config: %v", peerConfig)

	return &p2pConnectConfig{
		privateKey:   config.privateKey,
		localPorts:   config.localPorts,
		peerPubKey:   config.peerPubKey,
		peerPublicIP: peerConfig.PublicIP,
		peerPorts:    int32ToIntSlice(peerConfig.Ports),
	}, nil
}

func (m *Manager) exchangeConsumerConfig(consumerID, providerID identity.Identity, timeout time.Duration) (*p2pConnectConfig, error) {
	pubKey, privateKey, err := GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("could not generate consumer p2p keys: %w", err)
	}

	brokerConn, err := m.broker.Connect(m.brokerAddress)
	if err != nil {
		return nil, fmt.Errorf("could not open broker conn: %w", err)
	}
	defer brokerConn.Close()

	// Send initial exchange with signed consumer public key.
	beginExchangeMsg := &pb.P2PConfigExchangeMsg{
		PublicKey: pubKey.Hex(),
	}
	log.Debug().Msgf("Consumer %s sending public key %s to provider %s", consumerID.Address, beginExchangeMsg.PublicKey, providerID.Address)
	exchangeMsgBrokerReply, err := m.sendSignedMsg(brokerConn, fmt.Sprintf("%s.p2p-config-exchange", providerID.Address), consumerID, beginExchangeMsg, timeout)
	if err != nil {
		return nil, fmt.Errorf("could not send signed message: %w", err)
	}

	// Parse provider response with public key and encrypted and signed connection config.
	exchangeMsgReplySignedMsg, err := m.unpackSignedMsg(exchangeMsgBrokerReply)
	if err != nil {
		return nil, fmt.Errorf("could not unpack peer siged message: %w", err)
	}
	var exchangeMsgReply pb.P2PConfigExchangeMsg
	if err := proto.Unmarshal(exchangeMsgReplySignedMsg.Data, &exchangeMsgReply); err != nil {
		return nil, fmt.Errorf("could not unmarshal peer signed message payload: %w", err)
	}
	peerPubKey, err := DecodePublicKey(exchangeMsgReply.PublicKey)
	if err != nil {
		return nil, err
	}
	peerConnConfig, err := decryptConnConfigMsg(exchangeMsgReply.ConfigCiphertext, privateKey, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt peer conn config: %w", err)
	}
	log.Debug().Msgf("Consumer %s received provider %s with config: %v", consumerID.Address, providerID.Address, peerConnConfig)

	// Finally send consumer encrypted and signed connect config in ack message.
	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return nil, err
	}
	localPorts, err := m.acquireLocalPorts()
	if err != nil {
		return nil, err
	}
	connConfig := &pb.P2PConnectConfig{
		PublicIP: publicIP,
		Ports:    intToInt32Slice(localPorts),
	}
	connConfigCiphertext, err := encryptConnConfigMsg(connConfig, privateKey, peerPubKey)
	if err != nil {
		return nil, err
	}
	endExchangeMsg := &pb.P2PConfigExchangeMsg{
		PublicKey:        pubKey.Hex(),
		ConfigCiphertext: connConfigCiphertext,
	}
	log.Debug().Msgf("Consumer %s sending ack with encrypted config to provider %s", consumerID.Address, providerID.Address)
	_, err = m.sendSignedMsg(brokerConn, fmt.Sprintf("%s.p2p-config-exchange-ack", providerID.Address), consumerID, endExchangeMsg, timeout)
	if err != nil {
		return nil, err
	}

	return &p2pConnectConfig{
		privateKey:   privateKey,
		localPorts:   localPorts,
		peerPubKey:   peerPubKey,
		peerPublicIP: peerConnConfig.PublicIP,
		peerPorts:    int32ToIntSlice(peerConnConfig.Ports),
	}, nil
}

func (m *Manager) acquireLocalPorts() ([]int, error) {
	ports, err := m.portPool.AcquireMultiple(pingMaxPorts)
	if err != nil {
		return nil, err
	}
	var res []int
	for _, p := range ports {
		res = append(res, p.Num())
	}
	return res, nil
}

func (m *Manager) sendSignedMsg(brokerConn nats.Connection, subject string, senderID identity.Identity, msg *pb.P2PConfigExchangeMsg, timeout time.Duration) ([]byte, error) {
	packedMsg, err := m.packSignedMsg(senderID, msg)
	if err != nil {
		return nil, fmt.Errorf("could not pack signed message: %v", err)
	}
	reply, err := brokerConn.Request(subject, packedMsg, timeout)
	if err != nil {
		return nil, fmt.Errorf("could send broker request to subject %s: %v", subject, err)
	}
	return reply.Data, nil
}

// packSignedMsg marshals, signs and returns ready to send bytes.
func (m *Manager) packSignedMsg(signerID identity.Identity, msg *pb.P2PConfigExchangeMsg) ([]byte, error) {
	protoBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	signature, err := m.signer(signerID).Sign(protoBytes)
	if err != nil {
		return nil, err
	}
	signedMsg := &pb.P2PSignedMsg{Data: protoBytes, Signature: signature.Bytes()}
	signedMsgProtoBytes, err := proto.Marshal(signedMsg)
	if err != nil {
		return nil, err
	}
	return signedMsgProtoBytes, nil
}

func (m *Manager) unpackSignedMsg(b []byte) (*pb.P2PSignedMsg, error) {
	var signedMsg pb.P2PSignedMsg
	if err := proto.Unmarshal(b, &signedMsg); err != nil {
		return nil, err
	}
	if ok := m.verifier.Verify(signedMsg.Data, identity.SignatureBytes(signedMsg.Signature)); !ok {
		return nil, errors.New("message signature is invalid")
	}
	return &signedMsg, nil
}

// encryptConnConfigMsg encrypts proto message and returns bytes.
func encryptConnConfigMsg(msg *pb.P2PConnectConfig, privateKey PrivateKey, peerPubKey PublicKey) ([]byte, error) {
	protoBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	ciphertext, err := privateKey.Encrypt(peerPubKey, protoBytes)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

func decryptConnConfigMsg(ciphertext []byte, privateKey PrivateKey, peerPubKey PublicKey) (*pb.P2PConnectConfig, error) {
	peerConnectConfigProtoBytes, err := privateKey.Decrypt(peerPubKey, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt config to proto bytes: %w", err)
	}
	var peerProtoConnectConfig pb.P2PConnectConfig
	if err := proto.Unmarshal(peerConnectConfigProtoBytes, &peerProtoConnectConfig); err != nil {
		return nil, fmt.Errorf("could not unmarshal decrypted conn config: %w", err)
	}
	return &peerProtoConnectConfig, nil
}

func int32ToIntSlice(arr []int32) []int {
	var res []int
	for _, v := range arr {
		res = append(res, int(v))
	}
	return res
}

func intToInt32Slice(arr []int) []int32 {
	var res []int32
	for _, v := range arr {
		res = append(res, int32(v))
	}
	return res
}
