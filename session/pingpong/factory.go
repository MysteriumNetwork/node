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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/mbtime"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

const (
	// PaymentForDataWithTime is a payment method type that is used for both data transfer and time.
	PaymentForDataWithTime = "BYTES_TRANSFERRED_WITH_TIME"

	// PromiseWaitTimeout is the time that the provider waits for the promise to arrive
	PromiseWaitTimeout = time.Second * 50

	// InvoiceSendPeriod is how often the provider will send invoice messages to the consumer
	InvoiceSendPeriod = time.Second * 60

	// DefaultAccountantFailureCount defines how many times we're allowed to fail to reach accountant in a row before announcing the failure.
	DefaultAccountantFailureCount uint64 = 10

	gb       = 1024 * 1024 * 1024
	accuracy = 50000
)

// NewPaymentMethod returns the the default payment method of time + bytes.
func NewPaymentMethod(pricePerGB, pricePerMinute uint64) PaymentMethod {
	if pricePerGB > 0 {
		pricePerGB = gb * accuracy / pricePerGB
	}
	if pricePerMinute > 0 {
		pricePerMinute = uint64(time.Minute) * accuracy / pricePerMinute
	}

	return PaymentMethod{
		Price:    money.NewMoney(accuracy, money.CurrencyMyst),
		Duration: time.Duration(pricePerMinute),
		Type:     PaymentForDataWithTime,
		Bytes:    pricePerGB,
	}
}

// PaymentMethod represents a payment method
type PaymentMethod struct {
	Price    money.Money   `json:"price"`
	Duration time.Duration `json:"duration"`
	Bytes    uint64        `json:"bytes"`
	Type     string        `json:"type"`
}

// GetPrice returns the payment methods price
func (pm PaymentMethod) GetPrice() money.Money {
	return pm.Price
}

// GetType gets the payment methods type
func (pm PaymentMethod) GetType() string {
	return pm.Type
}

// GetRate returns the payment rate for the method
func (pm PaymentMethod) GetRate() market.PaymentRate {
	return market.PaymentRate{PerByte: pm.Bytes, PerTime: pm.Duration}
}

// InvoiceFactoryCreator returns a payment engine factory.
func InvoiceFactoryCreator(
	channel p2p.Channel,
	balanceSendPeriod, promiseTimeout time.Duration,
	invoiceStorage providerInvoiceStorage,
	registryAddress string,
	channelImplementationAddress string,
	maxAccountantFailureCount uint64,
	maxAllowedAccountantFee uint16,
	maxUnpaidInvoiceValue uint64,
	blockchainHelper bcHelper,
	eventBus eventbus.EventBus,
	proposal market.ServiceProposal,
	promiseHandler promiseHandler,
	providersAccountant common.Address,
) func(identity.Identity, identity.Identity, common.Address, string) (session.PaymentEngine, error) {
	return func(providerID, consumerID identity.Identity, accountantID common.Address, sessionID string) (session.PaymentEngine, error) {
		exchangeChan, err := exchangeMessageReceiver(channel)
		if err != nil {
			return nil, err
		}
		timeTracker := session.NewTracker(mbtime.Now)
		deps := InvoiceTrackerDeps{
			Proposal:                   proposal,
			Peer:                       consumerID,
			PeerInvoiceSender:          NewInvoiceSender(channel),
			InvoiceStorage:             invoiceStorage,
			TimeTracker:                &timeTracker,
			ChargePeriod:               balanceSendPeriod,
			ChargePeriodLeeway:         2 * time.Minute,
			ExchangeMessageChan:        exchangeChan,
			ExchangeMessageWaitTimeout: promiseTimeout,
			FirstInvoiceSendTimeout:    10 * time.Second,
			FirstInvoiceSendDuration:   1 * time.Second,
			ProviderID:                 providerID,
			ConsumersAccountantID:      accountantID,
			ProvidersAccountantID:      providersAccountant,
			Registry:                   registryAddress,
			MaxAccountantFailureCount:  maxAccountantFailureCount,
			MaxAllowedAccountantFee:    maxAllowedAccountantFee,
			BlockchainHelper:           blockchainHelper,
			EventBus:                   eventBus,
			SessionID:                  sessionID,
			PromiseHandler:             promiseHandler,
			ChannelAddressCalculator:   NewChannelAddressCalculator(accountantID.Hex(), channelImplementationAddress, registryAddress),
			MaxNotPaidInvoice:          maxUnpaidInvoiceValue,
		}
		paymentEngine := NewInvoiceTracker(deps)
		return paymentEngine, nil
	}
}

// ExchangeFactoryFunc returns a exchange factory.
func ExchangeFactoryFunc(
	keystore hashSigner,
	signer identity.SignerFactory,
	totalStorage consumerTotalsStorage,
	channelImplementation string,
	registryAddress string,
	eventBus eventbus.EventBus,
	dataLeewayMegabytes uint64) func(channel p2p.Channel, consumer, provider identity.Identity, accountant common.Address, proposal market.ServiceProposal) (connection.PaymentIssuer, error) {
	return func(channel p2p.Channel, consumer, provider identity.Identity, accountant common.Address, proposal market.ServiceProposal) (connection.PaymentIssuer, error) {
		invoices, err := invoiceReceiver(channel)
		if err != nil {
			return nil, err
		}
		timeTracker := session.NewTracker(mbtime.Now)
		deps := InvoicePayerDeps{
			InvoiceChan:               invoices,
			PeerExchangeMessageSender: NewExchangeSender(channel),
			ConsumerTotalsStorage:     totalStorage,
			TimeTracker:               &timeTracker,
			Ks:                        keystore,
			Identity:                  consumer,
			Peer:                      provider,
			Proposal:                  proposal,
			ChannelAddressCalculator:  NewChannelAddressCalculator(accountant.Hex(), channelImplementation, registryAddress),
			EventBus:                  eventBus,
			AccountantAddress:         accountant,
			DataLeeway:                datasize.MiB * datasize.BitSize(dataLeewayMegabytes),
		}
		return NewInvoicePayer(deps), nil
	}
}

func invoiceReceiver(channel p2p.ChannelHandler) (chan crypto.Invoice, error) {
	invoices := make(chan crypto.Invoice)

	channel.Handle(p2p.TopicPaymentInvoice, func(c p2p.Context) error {
		var msg pb.Invoice
		if err := c.Request().UnmarshalProto(&msg); err != nil {
			return err
		}
		log.Debug().Msgf("Received P2P message for %q: %s", p2p.TopicPaymentInvoice, msg.String())

		invoices <- crypto.Invoice{
			AgreementID:    msg.GetAgreementID(),
			AgreementTotal: msg.GetAgreementTotal(),
			TransactorFee:  msg.GetTransactorFee(),
			Hashlock:       msg.GetHashlock(),
			Provider:       msg.GetProvider(),
		}

		return nil
	})

	return invoices, nil
}
