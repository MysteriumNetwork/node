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

package endpoints

import (
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/tequilapi/utils"
	"github.com/mysterium/node/version"
	"net/http"
	"time"
)

type healthCheckData struct {
	Uptime  string    `json:"uptime"`
	Process int       `json:"process"`
	Version buildInfo `json:"version"`
}

type buildInfo struct {
	Commit      string `json:"commit"`
	Branch      string `json:"branch"`
	BuildNumber string `json:"buildNumber"`
}

type healthCheckEndpoint struct {
	startTime       time.Time
	currentTimeFunc func() time.Time
	processNumber   int
	versionInfo     *version.Info
}

/*
HealthCheckEndpointFactory creates a structure with single HealthCheck method for healthcheck serving as http,
currentTimeFunc is injected for easier testing
*/
func HealthCheckEndpointFactory(currentTimeFunc func() time.Time, procID func() int, versionInfo *version.Info) *healthCheckEndpoint {
	startTime := currentTimeFunc()
	return &healthCheckEndpoint{
		startTime,
		currentTimeFunc,
		procID(),
		versionInfo,
	}
}

func (hce *healthCheckEndpoint) HealthCheck(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	status := healthCheckData{
		Uptime:  hce.currentTimeFunc().Sub(hce.startTime).String(),
		Process: hce.processNumber,
		Version: buildInfo{
			hce.versionInfo.Commit,
			hce.versionInfo.Branch,
			hce.versionInfo.BuildNumber,
		},
	}
	utils.WriteAsJSON(status, writer)
}
