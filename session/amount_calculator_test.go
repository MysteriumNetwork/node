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

package session

import (
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
	"github.com/stretchr/testify/assert"
)

func Test_CorrectMoneyValueIsReturnedForTotalAmount(t *testing.T) {
	aCalc := AmountCalc{
		PaymentDef: dto.PaymentPerTime{
			Duration: time.Minute,
			Price: money.Money{
				Amount:   100,
				Currency: money.CurrencyMyst,
			},
		},
	}

	elapsed := 3*time.Minute + 25*time.Second

	totalAmount := aCalc.TotalAmount(elapsed)

	assert.Equal(t, uint64(300), totalAmount.Amount)
}
