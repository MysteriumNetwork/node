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

package session

import (
	"github.com/mysterium/node/communication"
)

// NewDialogHandler constructs handler which gets all incoming dialogs and starts handling them
func NewDialogHandler(proposalId int, sessionManager Manager) *handler {
	return &handler{
		CurrentProposalID: proposalId,
		SessionManager:    sessionManager,
	}
}

type handler struct {
	CurrentProposalID int
	SessionManager    Manager
}

// Handle starts serving services in given Dialog instance
func (handler *handler) Handle(dialog communication.Dialog) error {
	subscribeError := dialog.Respond(
		&SessionCreateConsumer{
			CurrentProposalID: handler.CurrentProposalID,
			SessionManager:    handler.SessionManager,
			PeerID:            dialog.PeerID(),
		},
	)
	if subscribeError != nil {
		return subscribeError
	}

	return nil
}
