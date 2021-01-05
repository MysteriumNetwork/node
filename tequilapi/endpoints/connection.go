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

package endpoints

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	// statusConnectCancelled indicates that connect request was cancelled by user. Since there is no such concept in REST
	// operations, custom client error code is defined. Maybe in later times a better idea will come how to handle these situations
	statusConnectCancelled = 499
)

var (
	errNoProposal = errors.New("provider has no service proposals")
)

// ProposalGetter defines interface to fetch currently active service proposal by id
type ProposalGetter interface {
	GetProposal(id market.ProposalID) (*market.ServiceProposal, error)
}

type identityRegistry interface {
	GetRegistrationStatus(int64, identity.Identity) (registry.RegistrationStatus, error)
}

// ConnectionEndpoint struct represents /connection resource and it's subresources
type ConnectionEndpoint struct {
	manager       connection.Manager
	publisher     eventbus.Publisher
	stateProvider stateProvider
	//TODO connection should use concrete proposal from connection params and avoid going to marketplace
	proposalRepository proposal.Repository
	identityRegistry   identityRegistry
}

// NewConnectionEndpoint creates and returns connection endpoint
func NewConnectionEndpoint(manager connection.Manager, stateProvider stateProvider, proposalRepository proposal.Repository, identityRegistry identityRegistry, publisher eventbus.Publisher) *ConnectionEndpoint {
	return &ConnectionEndpoint{
		manager:            manager,
		publisher:          publisher,
		stateProvider:      stateProvider,
		proposalRepository: proposalRepository,
		identityRegistry:   identityRegistry,
	}
}

// Status returns status of connection
// swagger:operation GET /connection Connection connectionStatus
// ---
// summary: Returns connection status
// description: Returns status of current connection
// responses:
//   200:
//     description: Status
//     schema:
//       "$ref": "#/definitions/ConnectionInfoDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ce *ConnectionEndpoint) Status(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	status := ce.manager.Status()
	statusResponse := contract.NewConnectionInfoDTO(status)
	utils.WriteAsJSON(statusResponse, resp)
}

// Create starts new connection
// swagger:operation PUT /connection Connection connectionCreate
// ---
// summary: Starts new connection
// description: Consumer opens connection to provider
// parameters:
//   - in: body
//     name: body
//     description: Parameters in body (consumer_id, provider_id, service_type) required for creating new connection
//     schema:
//       $ref: "#/definitions/ConnectionCreateRequestDTO"
// responses:
//   201:
//     description: Connection started
//     schema:
//       "$ref": "#/definitions/ConnectionInfoDTO"
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   409:
//     description: Conflict. Connection already exists
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   422:
//     description: Parameters validation error
//     schema:
//       "$ref": "#/definitions/ValidationErrorDTO"
//   499:
//     description: Connection was cancelled
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ce *ConnectionEndpoint) Create(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	cr, err := toConnectionRequest(req)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		ce.publisher.Publish(quality.AppTopicConnectionEvents, (&contract.ConnectionCreateRequest{}).Event(quality.StagePraseRequest, err.Error()))
		return
	}

	if errorMap := cr.Validate(); errorMap.HasErrors() {
		if out, err := errorMap.MarshalJSON(); err != nil {
			log.Error().Err(err).Msg("Failed to marshal error map")
		} else {
			ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageValidateRequest, string(out)))
		}

		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}

	consumerID := identity.FromAddress(cr.ConsumerID)
	status, err := ce.identityRegistry.GetRegistrationStatus(config.GetInt64(config.FlagChainID), consumerID)
	if err != nil {
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageRegistrationGetStatus, err.Error()))
		log.Error().Err(err).Stack().Msg("could not check registration status")
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	switch status {
	case registry.Unregistered, registry.RegistrationError:
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageRegistrationUnregistered, ""))
		log.Error().Msgf("identity %q is not registered, aborting...", cr.ConsumerID)
		utils.SendErrorMessage(resp, fmt.Sprintf("identity %q is not registered. Please register the identity first", cr.ConsumerID), http.StatusExpectationFailed)
		return
	case registry.InProgress:
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageRegistrationInProgress, ""))
		log.Info().Msgf("identity %q registration is in progress, continuing...", cr.ConsumerID)
	case registry.Registered:
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageRegistrationRegistered, ""))
		log.Info().Msgf("identity %q is registered, continuing...", cr.ConsumerID)
	default:
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageRegistrationUnknown, ""))
		log.Error().Msgf("identity %q has unknown status, aborting...", cr.ConsumerID)
		utils.SendErrorMessage(resp, fmt.Sprintf("identity %q has unknown status. aborting", cr.ConsumerID), http.StatusExpectationFailed)
		return
	}

	// TODO Pass proposal ID directly in request
	proposal, err := ce.proposalRepository.Proposal(market.ProposalID{
		ProviderID:  cr.ProviderID,
		ServiceType: cr.ServiceType,
	})
	if err != nil {
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageGetProposal, err.Error()))
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	if proposal == nil {
		ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageNoProposal, errNoProposal.Error()))
		utils.SendError(resp, errNoProposal, http.StatusBadRequest)
		return
	}

	err = ce.manager.Connect(consumerID, common.HexToAddress(cr.HermesID), *proposal, getConnectOptions(cr))

	if err != nil {
		switch err {
		case connection.ErrAlreadyExists:
			ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageConnectionAlreadyExists, err.Error()))
			utils.SendError(resp, err, http.StatusConflict)
		case connection.ErrConnectionCancelled:
			ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageConnectionCanceled, err.Error()))
			utils.SendError(resp, err, statusConnectCancelled)
		default:
			ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageConnectionUnknownError, err.Error()))
			log.Error().Err(err).Msg("Failed to connect")
			utils.SendError(resp, err, http.StatusInternalServerError)
		}
		return
	}

	ce.publisher.Publish(quality.AppTopicConnectionEvents, cr.Event(quality.StageConnectionOK, ""))
	resp.WriteHeader(http.StatusCreated)
	ce.Status(resp, req, params)
}

// Kill stops connection
// swagger:operation DELETE /connection Connection connectionCancel
// ---
// summary: Stops connection
// description: Stops current connection
// responses:
//   202:
//     description: Connection Stopped
//   409:
//     description: Conflict. No connection exists
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ce *ConnectionEndpoint) Kill(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	err := ce.manager.Disconnect()
	if err != nil {
		switch err {
		case connection.ErrNoConnection:
			utils.SendError(resp, err, http.StatusConflict)
		default:
			utils.SendError(resp, err, http.StatusInternalServerError)
		}
		return
	}
	resp.WriteHeader(http.StatusAccepted)
}

// GetStatistics returns statistics about current connection
// swagger:operation GET /connection/statistics Connection connectionStatistics
// ---
// summary: Returns connection statistics
// description: Returns statistics about current connection
// responses:
//   200:
//     description: Connection statistics
//     schema:
//       "$ref": "#/definitions/ConnectionStatisticsDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (ce *ConnectionEndpoint) GetStatistics(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	connection := ce.stateProvider.GetState().Connection
	response := contract.NewConnectionStatisticsDTO(connection.Session, connection.Statistics, connection.Throughput, connection.Invoice)

	utils.WriteAsJSON(response, writer)
}

// AddRoutesForConnection adds connections routes to given router
func AddRoutesForConnection(router *httprouter.Router, manager connection.Manager,
	stateProvider stateProvider, proposalRepository proposal.Repository, identityRegistry identityRegistry, publisher eventbus.Publisher) {
	connectionEndpoint := NewConnectionEndpoint(manager, stateProvider, proposalRepository, identityRegistry, publisher)
	router.GET("/connection", connectionEndpoint.Status)
	router.PUT("/connection", connectionEndpoint.Create)
	router.DELETE("/connection", connectionEndpoint.Kill)
	router.GET("/connection/statistics", connectionEndpoint.GetStatistics)
}

func toConnectionRequest(req *http.Request) (*contract.ConnectionCreateRequest, error) {
	var connectionRequest = contract.ConnectionCreateRequest{
		ConnectOptions: contract.ConnectOptions{
			DisableKillSwitch: false,
			DNS:               connection.DNSOptionAuto,
		},
		HermesID: config.GetString(config.FlagHermesID),
	}
	err := json.NewDecoder(req.Body).Decode(&connectionRequest)
	if err != nil {
		return nil, err
	}
	return &connectionRequest, nil
}

func getConnectOptions(cr *contract.ConnectionCreateRequest) connection.ConnectParams {
	dns := connection.DNSOptionAuto
	if cr.ConnectOptions.DNS != "" {
		dns = cr.ConnectOptions.DNS
	}

	return connection.ConnectParams{
		DisableKillSwitch: cr.ConnectOptions.DisableKillSwitch,
		DNS:               dns,
	}
}
