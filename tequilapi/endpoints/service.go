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

package endpoints

import (
	"encoding/json"
	"math/big"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/services"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
	"github.com/rs/zerolog/log"
)

// ServiceEndpoint struct represents management of service resource and it's sub-resources
type ServiceEndpoint struct {
	serviceManager ServiceManager
	optionsParser  map[string]services.ServiceOptionsParser
}

var (
	// serviceTypeInvalid represents service type which is unknown to node
	serviceTypeInvalid = "<unknown>"
	// serviceOptionsInvalid represents service options which is unknown to node (i.e. invalid structure for given type)
	serviceOptionsInvalid struct{}
)

// NewServiceEndpoint creates and returns service endpoint
func NewServiceEndpoint(serviceManager ServiceManager, optionsParser map[string]services.ServiceOptionsParser) *ServiceEndpoint {
	return &ServiceEndpoint{
		serviceManager: serviceManager,
		optionsParser:  optionsParser,
	}
}

// ServiceList provides a list of running services on the node.
// swagger:operation GET /services Service ServiceListResponse
// ---
// summary: List of services
// description: ServiceList provides a list of running services on the node.
// responses:
//   200:
//     description: List of running services
//     schema:
//       "$ref": "#/definitions/ServiceListResponse"
func (se *ServiceEndpoint) ServiceList(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	instances := se.serviceManager.List()

	statusResponse := toServiceListResponse(instances)
	utils.WriteAsJSON(statusResponse, resp)
}

// ServiceGet provides info for requested service on the node.
// swagger:operation GET /services/:id Service serviceGet
// ---
// summary: Information about service
// description: ServiceGet provides info for requested service on the node.
// responses:
//   200:
//     description: Service detailed information
//     schema:
//       "$ref": "#/definitions/ServiceInfoDTO"
//   404:
//     description: Service not found
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (se *ServiceEndpoint) ServiceGet(resp http.ResponseWriter, _ *http.Request, params httprouter.Params) {
	id := service.ID(params.ByName("id"))

	instance := se.serviceManager.Service(id)
	if instance == nil {
		utils.SendErrorMessage(resp, "Requested service not found", http.StatusNotFound)
		return
	}

	statusResponse := toServiceInfoResponse(id, instance)
	utils.WriteAsJSON(statusResponse, resp)
}

// ServiceStart starts requested service on the node.
// swagger:operation POST /services Service serviceStart
// ---
// summary: Starts service
// description: Provider starts serving new service to consumers
// parameters:
//   - in: body
//     name: body
//     description: Parameters in body (providerID) required for starting new service
//     schema:
//       $ref: "#/definitions/ServiceStartRequestDTO"
// responses:
//   201:
//     description: Initiates service start
//     schema:
//       "$ref": "#/definitions/ServiceInfoDTO"
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   409:
//     description: Conflict. Service is already running
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   422:
//     description: Parameters validation error
//     schema:
//       "$ref": "#/definitions/ValidationErrorDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (se *ServiceEndpoint) ServiceStart(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	sr, err := se.toServiceRequest(req)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	errorMap := validateServiceRequest(sr)
	if errorMap.HasErrors() {
		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}

	if se.isAlreadyRunning(sr) {
		utils.SendErrorMessage(resp, "Service already running", http.StatusConflict)
		return
	}

	log.Info().Msgf("Service start options: %+v", sr)
	id, err := se.serviceManager.Start(
		identity.FromAddress(sr.ProviderID),
		sr.Type,
		sr.AccessPolicies.IDs,
		sr.Options,
		pingpong.NewPrice(new(big.Int).SetUint64(sr.Price.PerGiB), new(big.Int).SetUint64(sr.Price.PerHour)),
	)
	if err == service.ErrorLocation {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	} else if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	instance := se.serviceManager.Service(id)

	resp.WriteHeader(http.StatusCreated)
	statusResponse := toServiceInfoResponse(id, instance)
	utils.WriteAsJSON(statusResponse, resp)
}

// ServiceStop stops service on the node.
// swagger:operation DELETE /services/:id Service serviceStop
// ---
// summary: Stops service
// description: Initiates service stop
// responses:
//   202:
//     description: Service Stop initiated
//   404:
//     description: No service exists
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (se *ServiceEndpoint) ServiceStop(resp http.ResponseWriter, _ *http.Request, params httprouter.Params) {
	id := service.ID(params.ByName("id"))

	instance := se.serviceManager.Service(id)
	if instance == nil {
		utils.SendErrorMessage(resp, "Service not found", http.StatusNotFound)
		return
	}

	if err := se.serviceManager.Stop(id); err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

func (se *ServiceEndpoint) isAlreadyRunning(sr contract.ServiceStartRequest) bool {
	for _, instance := range se.serviceManager.List() {
		if instance.ProviderID.Address == sr.ProviderID && instance.Type == sr.Type {
			return true
		}
	}
	return false
}

// AddRoutesForService adds service routes to given router
func AddRoutesForService(router *httprouter.Router, serviceManager ServiceManager, optionsParser map[string]services.ServiceOptionsParser) {
	serviceEndpoint := NewServiceEndpoint(serviceManager, optionsParser)

	router.GET("/services", serviceEndpoint.ServiceList)
	router.POST("/services", serviceEndpoint.ServiceStart)
	router.GET("/services/:id", serviceEndpoint.ServiceGet)
	router.DELETE("/services/:id", serviceEndpoint.ServiceStop)
}

func (se *ServiceEndpoint) toServiceRequest(req *http.Request) (contract.ServiceStartRequest, error) {
	var jsonData struct {
		ProviderID     string                          `json:"provider_id"`
		Type           string                          `json:"type"`
		Options        *json.RawMessage                `json:"options"`
		Price          contract.Price                  `json:"price"`
		AccessPolicies *contract.ServiceAccessPolicies `json:"access_policies"`
	}
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&jsonData); err != nil {
		return contract.ServiceStartRequest{}, err
	}

	serviceOpts, _ := services.GetStartOptions(jsonData.Type)
	sr := contract.ServiceStartRequest{
		ProviderID: jsonData.ProviderID,
		Type:       se.toServiceType(jsonData.Type),
		Options:    se.toServiceOptions(jsonData.Type, jsonData.Options),
		Price:      jsonData.Price,
		AccessPolicies: contract.ServiceAccessPolicies{
			IDs: serviceOpts.AccessPolicyList,
		},
	}
	if jsonData.AccessPolicies != nil {
		sr.AccessPolicies = *jsonData.AccessPolicies
	}
	return sr, nil
}

func (se *ServiceEndpoint) toServiceType(value string) string {
	if value == "" {
		return ""
	}

	_, ok := se.optionsParser[value]
	if !ok {
		return serviceTypeInvalid
	}

	return value
}

func (se *ServiceEndpoint) toServiceOptions(serviceType string, value *json.RawMessage) service.Options {
	optionsParser, ok := se.optionsParser[serviceType]
	if !ok {
		return nil
	}

	options, err := optionsParser(value)
	if err != nil {
		return serviceOptionsInvalid
	}

	return options
}

func toServiceInfoResponse(id service.ID, instance *service.Instance) contract.ServiceInfoDTO {
	return contract.ServiceInfoDTO{
		ID:         string(id),
		ProviderID: instance.ProviderID.Address,
		Type:       instance.Type,
		Options:    instance.Options,
		Status:     string(instance.State()),
		Proposal:   contract.NewProposalDTO(instance.Proposal),
	}
}

func toServiceListResponse(instances map[service.ID]*service.Instance) contract.ServiceListResponse {
	res := make([]contract.ServiceInfoDTO, 0)
	for id, instance := range instances {
		res = append(res, toServiceInfoResponse(id, instance))
	}
	return res
}

func validateServiceRequest(sr contract.ServiceStartRequest) *validation.FieldErrorMap {
	errors := validation.NewErrorMap()
	if len(sr.ProviderID) == 0 {
		errors.ForField("provider_id").AddError("required", "Field is required")
	}
	if sr.Type == "" {
		errors.ForField("type").AddError("required", "Field is required")
	}
	if sr.Type == serviceTypeInvalid {
		errors.ForField("type").AddError("invalid", "Invalid service type")
	}
	if sr.Options == serviceOptionsInvalid {
		errors.ForField("options").AddError("invalid", "Invalid options")
	}
	return errors
}

// ServiceManager represents service manager that is used for services management.
type ServiceManager interface {
	Start(providerID identity.Identity, serviceType string, policies []string, options service.Options, price market.Price) (service.ID, error)
	Stop(id service.ID) error
	Service(id service.ID) *service.Instance
	Kill() error
	List() map[service.ID]*service.Instance
}
