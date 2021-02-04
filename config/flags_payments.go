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

package config

import (
	"time"

	"github.com/mysteriumnetwork/node/metadata"

	"github.com/urfave/cli/v2"
)

var (
	// FlagPaymentsMaxHermesFee represents the max hermes fee.
	FlagPaymentsMaxHermesFee = cli.IntFlag{
		Name:  "payments.hermes.max.fee",
		Value: 1500,
		Usage: "The max fee that we'll accept from an hermes. In percentiles. 1500 means 15%",
	}
	// FlagPaymentsBCTimeout represents the BC call timeout.
	FlagPaymentsBCTimeout = cli.DurationFlag{
		Name:  "payments.bc.timeout",
		Value: time.Second * 30,
		Usage: "The duration we'll wait before timing out BC calls.",
	}
	// FlagPaymentsHermesPromiseSettleThreshold represents the percentage of balance left when we go for promise settling.
	FlagPaymentsHermesPromiseSettleThreshold = cli.Float64Flag{
		Name:  "payments.hermes.promise.threshold",
		Value: 0.1,
		Usage: "The percentage of balance before we settle promises",
	}
	// FlagPaymentsHermesPromiseSettleTimeout represents the time we wait for confirmation of the promise settlement.
	FlagPaymentsHermesPromiseSettleTimeout = cli.DurationFlag{
		Name:  "payments.hermes.promise.timeout",
		Value: time.Hour * 2,
		Usage: "The duration we'll wait before timing out our wait for promise settle.",
	}
	// FlagPaymentsProviderInvoiceFrequency determines how often the provider sends invoices.
	FlagPaymentsProviderInvoiceFrequency = cli.DurationFlag{
		Name:  "payments.provider.invoice-frequency",
		Value: time.Minute,
		Usage: "Determines how often the provider sends invoices.",
	}
	// FlagPaymentsConsumerPricePerMinuteUpperBound sets the upper price bound per minute to a set value.
	FlagPaymentsConsumerPricePerMinuteUpperBound = cli.StringFlag{
		Name:  "payments.consumer.price-perminute-max",
		Usage: "Sets the maximum price of the service per minute. All proposals with a price above this bound will be filtered out and not visible.",
		Value: metadata.Testnet2Definition.Payments.Consumer.PricePerMinuteMax,
	}
	// FlagPaymentsConsumerPricePerMinuteLowerBound sets the lower price bound per minute to a set value.
	FlagPaymentsConsumerPricePerMinuteLowerBound = cli.StringFlag{
		Name:  "payments.consumer.price-perminute-min",
		Usage: "Sets the minimum price of the service per minute. All proposals with a below above this bound will be filtered out and not visible.",
		Value: metadata.Testnet2Definition.Payments.Consumer.PricePerMinuteMin,
	}
	// FlagPaymentsConsumerPricePerGBUpperBound sets the upper price bound per GiB to a set value.
	FlagPaymentsConsumerPricePerGBUpperBound = cli.StringFlag{
		Name:  "payments.consumer.price-pergib-max",
		Usage: "Sets the maximum price of the service per gb. All proposals with a price above this bound will be filtered out and not visible.",
		Value: metadata.Testnet2Definition.Payments.Consumer.PricePerGIBMax,
	}
	// FlagPaymentsConsumerPricePerGBLowerBound sets the lower price bound per GiB to a set value.
	FlagPaymentsConsumerPricePerGBLowerBound = cli.StringFlag{
		Name:  "payments.consumer.price-pergib-min",
		Usage: "Sets the minimum price of the service per gb. All proposals with a below above this bound will be filtered out and not visible.",
		Value: metadata.Testnet2Definition.Payments.Consumer.PricePerGIBMin,
	}
	// FlagPaymentsConsumerDataLeewayMegabytes sets the data amount the consumer agrees to pay before establishing a session
	FlagPaymentsConsumerDataLeewayMegabytes = cli.Uint64Flag{
		Name:  "payments.consumer.data-leeway-megabytes",
		Usage: "sets the data amount the consumer agrees to pay before establishing a session",
		Value: metadata.Testnet2Definition.Payments.Consumer.DataLeewayMegabytes,
	}
	// FlagPaymentsMaxUnpaidInvoiceValue sets the upper limit of session payment value before forcing an invoice
	FlagPaymentsMaxUnpaidInvoiceValue = cli.StringFlag{
		Name:  "payments.provider.max-unpaid-invoice-value",
		Usage: "sets the upper limit of session payment value before forcing an invoice. If this value is exceeded before a payment interval is reached, an invoice is sent.",
		Value: "30000000000000000",
	}
)

// RegisterFlagsPayments function register payments flags to flag list.
func RegisterFlagsPayments(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagPaymentsMaxHermesFee,
		&FlagPaymentsBCTimeout,
		&FlagPaymentsHermesPromiseSettleThreshold,
		&FlagPaymentsHermesPromiseSettleTimeout,
		&FlagPaymentsProviderInvoiceFrequency,
		&FlagPaymentsConsumerPricePerMinuteUpperBound,
		&FlagPaymentsConsumerPricePerMinuteLowerBound,
		&FlagPaymentsConsumerPricePerGBUpperBound,
		&FlagPaymentsConsumerPricePerGBLowerBound,
		&FlagPaymentsConsumerDataLeewayMegabytes,
		&FlagPaymentsMaxUnpaidInvoiceValue,
	)
}

// ParseFlagsPayments function fills in payments options from CLI context.
func ParseFlagsPayments(ctx *cli.Context) {
	Current.ParseIntFlag(ctx, FlagPaymentsMaxHermesFee)
	Current.ParseDurationFlag(ctx, FlagPaymentsBCTimeout)
	Current.ParseFloat64Flag(ctx, FlagPaymentsHermesPromiseSettleThreshold)
	Current.ParseDurationFlag(ctx, FlagPaymentsHermesPromiseSettleTimeout)
	Current.ParseDurationFlag(ctx, FlagPaymentsProviderInvoiceFrequency)
	Current.ParseStringFlag(ctx, FlagPaymentsConsumerPricePerMinuteUpperBound)
	Current.ParseStringFlag(ctx, FlagPaymentsConsumerPricePerMinuteLowerBound)
	Current.ParseStringFlag(ctx, FlagPaymentsConsumerPricePerGBUpperBound)
	Current.ParseStringFlag(ctx, FlagPaymentsConsumerPricePerGBLowerBound)
	Current.ParseUInt64Flag(ctx, FlagPaymentsConsumerDataLeewayMegabytes)
	Current.ParseStringFlag(ctx, FlagPaymentsMaxUnpaidInvoiceValue)
}
