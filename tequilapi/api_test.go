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

package tequilapi

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type tequilapiTestSuite struct {
	suite.Suite
	server APIServer
	client TestClient
}

func (testSuite *tequilapiTestSuite) SetupSuite() {
	testSuite.server = NewServer("localhost", 0, NewAPIRouter())

	assert.NoError(testSuite.T(), testSuite.server.StartServing())
	port, err := testSuite.server.Port()
	assert.NoError(testSuite.T(), err)
	testSuite.client = NewTestClient(testSuite.T(), port)
}

func (testSuite *tequilapiTestSuite) TestHealthCheckReturnsExpectedResponse() {
	resp := testSuite.client.Get("/healthcheck")

	expectJSONStatus200(testSuite.T(), resp, 200)

	var jsonMap map[string]interface{}
	parseResponseAsJSON(testSuite.T(), resp, &jsonMap)
	assert.NotEmpty(testSuite.T(), jsonMap["uptime"])
}

func (testSuite *tequilapiTestSuite) TearDownSuite() {
	testSuite.server.Stop()
	testSuite.server.Wait()
}

func TestTequilapiTestSuite(t *testing.T) {
	suite.Run(t, new(tequilapiTestSuite))
}
