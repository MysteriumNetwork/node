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

package dialog

import (
	"github.com/mysterium/node/communication"
)

type dialogCreateConsumer struct {
	Callback func(request *dialogCreateRequest) (*dialogCreateResponse, error)
}

func (consumer *dialogCreateConsumer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointDialogCreate
}

func (consumer *dialogCreateConsumer) NewRequest() (requestPtr interface{}) {
	return &dialogCreateRequest{}
}

func (consumer *dialogCreateConsumer) Consume(requestPtr interface{}) (responsePtr interface{}, err error) {
	return consumer.Callback(requestPtr.(*dialogCreateRequest))
}
