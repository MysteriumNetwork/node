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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/stretchr/testify/assert"
)

var (
	mockServiceID             = service.ID("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	mockAccessPolicyServiceID = service.ID("6ba7b810-9dad-11d1-80b4-00c04fd430c9")
	mockServiceOptions        = fancyServiceOptions{
		Foo: "bar",
	}
	mockAccessPolicyEndpoint = "https://some.domain/api/v1/lists/"
	mockProposal             = market.ServiceProposal{
		ID:                1,
		ServiceType:       "testprotocol",
		ServiceDefinition: TestServiceDefinition{},
		ProviderID:        "0xProviderId",
		PaymentMethodType: mocks.DefaultPaymentMethodType,
		PaymentMethod:     mocks.DefaultPaymentMethod(),
	}
	ap = []market.AccessPolicy{
		{
			ID:     "verified-traffic",
			Source: fmt.Sprintf("%v%v", mockAccessPolicyEndpoint, "verified-traffic"),
		},
		{
			ID:     "0x0000000000000001",
			Source: fmt.Sprintf("%v%v", mockAccessPolicyEndpoint, "0x0000000000000001"),
		},
		{
			ID:     "dvpn-traffic",
			Source: fmt.Sprintf("%v%v", mockAccessPolicyEndpoint, "dvpn-traffic"),
		},
		{
			ID:     "12312312332132",
			Source: fmt.Sprintf("%v%v", mockAccessPolicyEndpoint, "12312312332132"),
		},
	}
	serviceTypeWithAccessPolicy  = "mockAccessPolicyService"
	mockProposalWithAccessPolicy = market.ServiceProposal{
		ID:                1,
		ServiceType:       serviceTypeWithAccessPolicy,
		ServiceDefinition: TestServiceDefinition{},
		ProviderID:        "0xProviderId",
		AccessPolicies:    &ap,
		PaymentMethodType: mocks.DefaultPaymentMethodType,
		PaymentMethod:     mocks.DefaultPaymentMethod(),
	}
	mockServiceRunning                 = service.NewInstance(mockServiceOptions, servicestate.Running, nil, mockProposal, nil, nil, nil)
	mockServiceStopped                 = service.NewInstance(mockServiceOptions, servicestate.NotRunning, nil, mockProposal, nil, nil, nil)
	mockServiceRunningWithAccessPolicy = service.NewInstance(mockServiceOptions, servicestate.Running, nil, mockProposalWithAccessPolicy, nil, nil, nil)
)

type fancyServiceOptions struct {
	Foo string `json:"foo"`
}

type mockServiceManager struct{}

func (sm *mockServiceManager) Start(providerID identity.Identity, serviceType string, policyIDs []string, options service.Options) (service.ID, error) {
	if serviceType == serviceTypeWithAccessPolicy {
		return mockAccessPolicyServiceID, nil
	}
	return mockServiceID, nil
}
func (sm *mockServiceManager) Stop(id service.ID) error { return nil }
func (sm *mockServiceManager) Service(id service.ID) *service.Instance {
	if id == "6ba7b810-9dad-11d1-80b4-00c04fd430c8" {
		return mockServiceRunning
	}
	if id == mockAccessPolicyServiceID {
		return mockServiceRunningWithAccessPolicy
	}
	return nil
}
func (sm *mockServiceManager) List() map[service.ID]*service.Instance {
	return map[service.ID]*service.Instance{
		"11111111-9dad-11d1-80b4-00c04fd430c0": mockServiceStopped,
	}
}
func (sm *mockServiceManager) Kill() error { return nil }

var fakeOptionsParser = map[string]ServiceOptionsParser{
	"testprotocol": func(opts *json.RawMessage) (service.Options, error) {
		return nil, nil
	},
	serviceTypeWithAccessPolicy: func(opts *json.RawMessage) (service.Options, error) {
		return nil, nil
	},
	"errorprotocol": func(opts *json.RawMessage) (service.Options, error) {
		return nil, errors.New("error")
	},
}

func Test_AddRoutesForServiceAddsRoutes(t *testing.T) {
	router := httprouter.New()
	AddRoutesForService(router, &mockServiceManager{}, fakeOptionsParser)

	tests := []struct {
		method         string
		path           string
		body           string
		expectedStatus int
		expectedJSON   string
	}{
		{
			http.MethodGet,
			"/services",
			"",
			http.StatusOK,
			`[{
				"id": "11111111-9dad-11d1-80b4-00c04fd430c0",
				"provider_id": "0xProviderId",
				"type": "testprotocol",
				"options": {"foo": "bar"},
				"status": "NotRunning",
				"proposal": {
					"id": 1,
					"provider_id": "0xProviderId",
					"service_type": "testprotocol",
					"service_definition": {
						"location_originate": {"asn": 123, "country": "Lithuania", "city": "Vilnius"}
					},
					"payment_method": {
						"type": "BYTES_TRANSFERRED_WITH_TIME",
						"price": {
							"amount":50000,
							"currency":"MYST"
						},
						"rate":{
							"per_seconds":60,
							"per_bytes":7669584
						}
					}
				}
			}]`,
		},
		{
			http.MethodPost,
			"/services",
			`{"provider_id": "node1", "type": "testprotocol"}`,
			http.StatusCreated,
			`{
				"id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
				"provider_id": "0xProviderId",
				"type": "testprotocol",
				"options": {"foo": "bar"},
				"status": "Running",
				"proposal": {
					"id": 1,
					"provider_id": "0xProviderId",
					"service_type": "testprotocol",
					"service_definition": {
						"location_originate": {"asn": 123, "country": "Lithuania", "city": "Vilnius"}
					},
					"payment_method": {
						"type": "BYTES_TRANSFERRED_WITH_TIME",
						"price": {
							"amount":50000,
							"currency":"MYST"
						},
						"rate":{
							"per_seconds":60,
							"per_bytes":7669584
						}
					}
				}
			}`,
		},
		{
			http.MethodGet,
			"/services/6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			"",
			http.StatusOK,
			`{
				"id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
				"provider_id": "0xProviderId",
				"type": "testprotocol",
				"options": {"foo": "bar"},
				"status": "Running",
				"proposal": {
					"id": 1,
					"provider_id": "0xProviderId",
					"service_type": "testprotocol",
					"service_definition": {
						"location_originate": {"asn": 123, "country": "Lithuania", "city": "Vilnius"}
					},
					"payment_method": {
						"type": "BYTES_TRANSFERRED_WITH_TIME",
						"price": {
							"amount":50000,
							"currency":"MYST"
						},
						"rate":{
							"per_seconds":60,
							"per_bytes":7669584
						}
					}
				}
			}`,
		},
		{
			http.MethodDelete, "/services/6ba7b810-9dad-11d1-80b4-00c04fd430c8", "",
			http.StatusAccepted, "",
		},
		{
			http.MethodDelete, "/services/00000000-9dad-11d1-80b4-00c04fd43000", "",
			http.StatusNotFound, `{"message":"Service not found"}`,
		},
	}

	for _, test := range tests {
		req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, test.expectedStatus, resp.Code)
		if test.expectedJSON != "" {
			assert.JSONEq(t, test.expectedJSON, resp.Body.String())
		} else {
			assert.Equal(t, "", resp.Body.String())
		}
	}
}

func Test_ServiceStartInvalidType(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(
		http.MethodGet,
		"/irrelevant",
		strings.NewReader(`{
			"type": "openvpn",
			"provider_id": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "validation_error",
			"errors": {
				"type": [ {"code": "invalid", "message": "Invalid service type"} ]
			}
		}`,
		resp.Body.String(),
	)
}

func Test_ServiceStart_InvalidType(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(
		http.MethodGet,
		"/irrelevant",
		strings.NewReader(`{
			"type": "openvpn",
			"provider_id": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "validation_error",
			"errors": {
				"type": [ {"code": "invalid", "message": "Invalid service type"} ]
			}
		}`,
		resp.Body.String(),
	)
}

func Test_ServiceStart_InvalidOptions(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(
		http.MethodGet,
		"/irrelevant",
		strings.NewReader(`{
			"type": "errorprotocol",
			"provider_id": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "validation_error",
			"errors": {
				"options": [ {"code": "invalid", "message": "Invalid options" } ]
			}
		}`,
		resp.Body.String(),
	)
}

func Test_ServiceStartAlreadyRunning(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(
		http.MethodGet,
		"/irrelevant",
		strings.NewReader(`{
			"type": "testprotocol",
			"provider_id": "0xProviderId",
			"options": {}
		}`),
	)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.JSONEq(
		t,
		`{"message":"Service already running"}`,
		resp.Body.String(),
	)
}

func Test_ServiceStatus_NotFoundIsReturnedWhenNotStarted(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(http.MethodGet, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceGet(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func Test_ServiceGetReturnsServiceInfo(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(http.MethodGet, "/irrelevant", nil)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceGet(resp, req, httprouter.Params{{Key: "id", Value: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"}})

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"id":"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			"provider_id": "0xProviderId",
			"type": "testprotocol",
			"options": {"foo": "bar"},
			"status": "Running",
			"proposal": {
				"id": 1,
				"provider_id": "0xProviderId",
				"service_type": "testprotocol",
				"service_definition": {
					"location_originate": {
						"asn": 123,
						"country": "Lithuania",
						"city": "Vilnius"
					}
				},
				"payment_method": {
					"type": "BYTES_TRANSFERRED_WITH_TIME",
					"price": {
						"amount":50000,
						"currency":"MYST"
					},
					"rate":{
						"per_seconds":60,
						"per_bytes":7669584
					}
				}
			}
		}`,
		resp.Body.String(),
	)
}
func Test_ServiceCreate_Returns400ErrorIfRequestBodyIsNotJSON(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(http.MethodPut, "/irrelevant", strings.NewReader("a"))
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "invalid character 'a' looking for beginning of value"
		}`,
		resp.Body.String(),
	)
}

func Test_ServiceCreate_Returns422ErrorIfRequestBodyIsMissingFieldValues(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(http.MethodPut, "/irrelevant", strings.NewReader("{}"))
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(t,
		`{
			"message": "validation_error",
			"errors": {
				"provider_id": [ {"code": "required", "message": "Field is required"} ],
				"type": [ {"code": "required", "message": "Field is required"} ]
			}
		}`,
		resp.Body.String(),
	)
}

func Test_ServiceStart_WithAccessPolicy(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(
		http.MethodGet,
		"/irrelevant",
		strings.NewReader(`{
			"type": "mockAccessPolicyService",
			"provider_id": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"access_policies": {
				"ids": ["verified-traffic", "dvpn-traffic", "12312312332132", "0x0000000000000001"]
			}
		}`),
	)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.JSONEq(
		t,
		`{
			"id": "6ba7b810-9dad-11d1-80b4-00c04fd430c9",
			"provider_id": "0xProviderId",
			"type": "mockAccessPolicyService",
			"options": {"foo": "bar"},
			"status": "Running",
			"proposal": {
				"id": 1,
				"provider_id": "0xProviderId",
				"service_type": "mockAccessPolicyService",
				"service_definition": {
					"location_originate": {"asn": 123, "country": "Lithuania", "city": "Vilnius"}
				},
				"payment_method": {
					"type": "BYTES_TRANSFERRED_WITH_TIME",
					"price": {
						"amount":50000,
						"currency":"MYST"
					},
					"rate":{
						"per_seconds":60,
						"per_bytes":7669584
					}
				},
				"access_policies": [
					{
						"id":"verified-traffic",
						"source": "https://some.domain/api/v1/lists/verified-traffic"
					},
					{
						"id":"0x0000000000000001",
						"source": "https://some.domain/api/v1/lists/0x0000000000000001"
					},
					{
						"id":"dvpn-traffic",
						"source": "https://some.domain/api/v1/lists/dvpn-traffic"
					},
					{
						"id":"12312312332132",
						"source": "https://some.domain/api/v1/lists/12312312332132"
					}
				]
			}
		}`,
		resp.Body.String(),
	)
}

func Test_ServiceStart_ReturnsBadRequest_WithUnknownParams(t *testing.T) {
	serviceEndpoint := NewServiceEndpoint(&mockServiceManager{}, fakeOptionsParser)

	req := httptest.NewRequest(
		http.MethodGet,
		"/irrelevant",
		strings.NewReader(`{
			"type": "mockAccessPolicyService",
			"provider_id": "0x9edf75f870d87d2d1a69f0d950a99984ae955ee0",
			"access_policy": {
				"ids": ["verified-traffic", "dvpn-traffic", "12312312332132", "0x0000000000000001"]
			}
		}`),
	)
	resp := httptest.NewRecorder()

	serviceEndpoint.ServiceStart(resp, req, httprouter.Params{})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.JSONEq(
		t,
		`{"message": "json: unknown field \"access_policy\""}`,
		resp.Body.String(),
	)
}
