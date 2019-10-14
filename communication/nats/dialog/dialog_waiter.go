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

package dialog

import (
	"fmt"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/gofrs/uuid"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/pkg/errors"
)

type validator func(peerID identity.Identity) error

// NewDialogWaiter constructs new DialogWaiter which works through NATS connection.
func NewDialogWaiter(address *discovery.AddressNATS, signer identity.Signer, validators ...validator) *dialogWaiter {
	return &dialogWaiter{
		address:    address,
		signer:     signer,
		dialogs:    make([]communication.Dialog, 0),
		validators: validators,
	}
}

const waiterLogPrefix = "[NATS.DialogWaiter] "

type dialogWaiter struct {
	address    *discovery.AddressNATS
	signer     identity.Signer
	dialogs    []communication.Dialog
	validators []validator

	sync.RWMutex
}

// Start registers dialogWaiter with broker (NATS) service
func (waiter *dialogWaiter) Start() (market.Contact, error) {
	log.Info(waiterLogPrefix, "Connecting to: ", waiter.address.GetContact())

	err := waiter.address.Connect()
	if err != nil {
		return market.Contact{}, errors.Errorf("failed to start my connection with: %v", waiter.address.GetContact())
	}

	return waiter.address.GetContact(), nil
}

// Stop disconnects dialogWaiter from broker (NATS) service
func (waiter *dialogWaiter) Stop() error {
	waiter.RLock()
	defer waiter.RUnlock()

	for _, dialog := range waiter.dialogs {
		dialog.Close()
	}
	waiter.address.Disconnect()
	return nil
}

// ServeDialogs starts accepting dialogs initiated by peers
func (waiter *dialogWaiter) ServeDialogs(dialogHandler communication.DialogHandler) error {
	createDialog := func(request *dialogCreateRequest) (*dialogCreateResponse, error) {
		err := waiter.validateDialogRequest(request)
		if err != nil {
			log.Error(waiterLogPrefix, "Validation check failed: ", err.Error())
			return &responseInvalidIdentity, nil
		}

		uid, err := uuid.NewV4()
		if err != nil {
			log.Error(waiterLogPrefix, "Failed to generate unique topic: ", err)
			return &responseInternalError, errors.Wrap(err, "failed to generate unique topic")
		}

		peerID := identity.FromAddress(request.PeerID)
		topic := uid.String()
		if len(request.Version) == 0 {
			// TODO this is a compatibility check. It should be removed once all consumers will migrate to the newer version.
			topic = waiter.address.GetTopic() + "." + peerID.Address
		}
		dialog := waiter.newDialogToPeer(peerID, waiter.newCodecForPeer(peerID), topic)
		err = dialogHandler.Handle(dialog)
		if err != nil {
			log.Error(waiterLogPrefix, fmt.Sprintf("Failed dialog from: '%s'. %s", request.PeerID, err))
			return &responseInternalError, nil
		}

		waiter.Lock()
		waiter.dialogs = append(waiter.dialogs, dialog)
		waiter.Unlock()

		log.Info(waiterLogPrefix, fmt.Sprintf("Accepted dialog from: '%s'", request.PeerID))
		return &dialogCreateResponse{
			Reason:        responseOK.Reason,
			ReasonMessage: responseOK.ReasonMessage,
			Topic:         topic,
		}, nil
	}
	codec := NewCodecSecured(communication.NewCodecJSON(), waiter.signer, identity.NewVerifierSigned())
	receiver := nats.NewReceiver(waiter.address.GetConnection(), codec, waiter.address.GetTopic())
	return receiver.Respond(&dialogCreateConsumer{Callback: createDialog})
}

func (waiter *dialogWaiter) newCodecForPeer(peerID identity.Identity) *codecSecured {
	return NewCodecSecured(
		communication.NewCodecJSON(),
		waiter.signer,
		identity.NewVerifierIdentity(peerID),
	)
}

func (waiter *dialogWaiter) newDialogToPeer(peerID identity.Identity, peerCodec *codecSecured, topic string) *dialog {
	return &dialog{
		peerID:   peerID,
		Sender:   nats.NewSender(waiter.address.GetConnection(), peerCodec, topic),
		Receiver: nats.NewReceiver(waiter.address.GetConnection(), peerCodec, topic),
	}
}

func (waiter *dialogWaiter) validateDialogRequest(request *dialogCreateRequest) error {
	if request.PeerID == "" {
		return errors.New("no identity provided")
	}

	for _, f := range waiter.validators {
		if err := f(identity.FromAddress(request.PeerID)); err != nil {
			return errors.Wrap(err, "failed to validate dialog request")
		}
	}

	return nil
}
