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
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/mocks"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/stretchr/testify/assert"
)

const identityUrl = "/irrelevant"

var (
	existingIdentities = []identity.Identity{
		{Address: "0x000000000000000000000000000000000000000a"},
		{Address: "0x000000000000000000000000000000000000beef"},
	}
	newIdentity = identity.Identity{Address: "0x000000000000000000000000000000000000aaac"}
)

type selectorFake struct {
}

func (hf *selectorFake) UseOrCreate(address, _ string, _ int64) (identity.Identity, error) {
	if len(address) > 0 {
		return identity.Identity{Address: address}, nil
	}

	return identity.Identity{Address: "0x000000"}, nil
}

func TestCurrentIdentitySuccess(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(`{"passphrase": "mypassphrase"}`),
	)
	params := httprouter.Params{{Key: "id", Value: "current"}}
	assert.Nil(t, err)

	endpoint := &identitiesAPI{
		idm:      mockIdm,
		selector: &selectorFake{},
	}
	endpoint.Current(resp, req, params)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(
		t,
		`{
			"id": "0x000000"
		}`,
		resp.Body.String(),
	)
}

func TestUnlockIdentitySuccess(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(`{"passphrase": "mypassphrase"}`),
	)
	params := httprouter.Params{{Key: "id", Value: "0x000000000000000000000000000000000000000a"}}
	assert.Nil(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Unlock(resp, req, params)

	assert.Equal(t, http.StatusAccepted, resp.Code)

	assert.Equal(t, "0x000000000000000000000000000000000000000a", mockIdm.LastUnlockAddress)
	assert.Equal(t, "mypassphrase", mockIdm.LastUnlockPassphrase)
	assert.Equal(t, int64(0), mockIdm.LastUnlockChainID)
}

func TestUnlockIdentityWithInvalidJSON(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(`{invalid json}`),
	)
	params := httprouter.Params{{Key: "id", Value: "0x000000000000000000000000000000000000000a"}}
	assert.Nil(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Unlock(resp, req, params)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestUnlockIdentityWithNoPassphrase(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPost,
		identityUrl,
		bytes.NewBufferString(`{}`),
	)
	params := httprouter.Params{{Key: "id", Value: "0x000000000000000000000000000000000000000a"}}
	assert.NoError(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Unlock(resp, req, params)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "validation_error",
			"errors" : {
				"passphrase": [ {"code" : "required" , "message" : "Field is required" } ]
			}
		}`,
		resp.Body.String(),
	)
}

func TestUnlockFailure(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(`{"passphrase": "mypassphrase"}`),
	)
	params := httprouter.Params{{Key: "id", Value: "0x000000000000000000000000000000000000000a"}}
	assert.Nil(t, err)

	mockIdm.MarkUnlockToFail()

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Unlock(resp, req, params)

	assert.Equal(t, http.StatusForbidden, resp.Code)

	assert.Equal(t, "0x000000000000000000000000000000000000000a", mockIdm.LastUnlockAddress)
	assert.Equal(t, "mypassphrase", mockIdm.LastUnlockPassphrase)
	assert.Equal(t, int64(0), mockIdm.LastUnlockChainID)
}

func TestCreateNewIdentityEmptyPassphrase(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{"passphrase": ""}`),
	)
	assert.Nil(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Create(resp, req, nil)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestCreateNewIdentityNoPassphrase(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{}`),
	)
	assert.Nil(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Create(resp, req, nil)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.JSONEq(
		t,
		`{
			"message": "validation_error",
			"errors" : {
				"passphrase": [ {"code" : "required" , "message" : "Field is required" } ]
			}
		}`,
		resp.Body.String(),
	)
}

func TestCreateNewIdentity(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(`{"passphrase": "mypass"}`),
	)
	assert.Nil(t, err)

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.Create(resp, req, nil)

	assert.JSONEq(
		t,
		`{
            "id": "0x000000000000000000000000000000000000aaac"
        }`,
		resp.Body.String(),
	)
}

func TestListIdentities(t *testing.T) {
	mockIdm := identity.NewIdentityManagerFake(existingIdentities, newIdentity)
	req := httptest.NewRequest("GET", "/irrelevant", nil)
	resp := httptest.NewRecorder()

	endpoint := &identitiesAPI{idm: mockIdm}
	endpoint.List(resp, req, nil)

	assert.JSONEq(
		t,
		`{
            "identities": [
                {"id": "0x000000000000000000000000000000000000000a"},
                {"id": "0x000000000000000000000000000000000000beef"}
            ]
        }`,
		resp.Body.String(),
	)
}

func Test_ReferralTokenGet(t *testing.T) {
	router := httprouter.New()

	server := newTestTransactorServer(http.StatusAccepted, `{"token":"yay-free-myst"}`)
	tr := registry.NewTransactor(requests.NewHTTPClient(server.URL, requests.DefaultTimeout), server.URL, "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", "0xbe180c8CA53F280C7BE8669596fF7939d933AA10", fakeSignerFactory, mocks.NewEventBus(), nil)
	endpoint := &identitiesAPI{transactor: tr}
	router.GET("/identities/:id/referral", endpoint.GetReferralToken)

	tokenRequest := `{"identity": "0x0"}`
	req, err := http.NewRequest(
		http.MethodGet,
		"/identities/0x0/referral",
		bytes.NewBufferString(tokenRequest),
	)
	assert.Nil(t, err)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{"token":"yay-free-myst"}`, resp.Body.String())
}
