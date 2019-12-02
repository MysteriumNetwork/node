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

package quality

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/mysteriumnetwork/metrics"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/logconfig/httptrace"
)

const (
	mysteriumMorqaAgentName = "goclient-v0.1"
)

// MysteriumMORQA HTTP client for Mysterium QualityOracle - MORQA
type MysteriumMORQA struct {
	// http    HTTPClient
	baseURL string

	client        *retryablehttp.Client
	clientMu      sync.Mutex
	clientFactory func() *retryablehttp.Client
}

// NewMorqaClient creates Mysterium Morqa client with a real communication
func NewMorqaClient(srcIP, baseURL string, timeout time.Duration) *MysteriumMORQA {
	traceLog := &httptrace.HTTPTraceLog{}
	morqa := &MysteriumMORQA{
		baseURL: baseURL,
		clientFactory: func() *retryablehttp.Client {
			return &retryablehttp.Client{
				HTTPClient: &http.Client{
					Timeout:   timeout,
					Transport: requests.GetDefaultTransport(srcIP),
				},
				Logger:          traceLog,
				RequestLogHook:  traceLog.LogRequest,
				ResponseLogHook: traceLog.LogResponse,
				RetryWaitMin:    timeout,
				RetryWaitMax:    10 * timeout,
				RetryMax:        10,
				CheckRetry:      retryablehttp.DefaultRetryPolicy,
				Backoff:         retryablehttp.DefaultBackoff,
			}
		},
	}
	morqa.client = morqa.clientFactory()
	return morqa
}

// ProposalsMetrics returns a list of proposals connection metrics
func (m *MysteriumMORQA) ProposalsMetrics() []ConnectMetric {
	request, err := m.newRequestJSON(http.MethodGet, "providers/sessions", nil)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create proposals metrics request")
		return nil
	}

	response, err := m.client.Do(request)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to request or parse proposals metrics")
		return nil
	}
	defer response.Body.Close()

	var metricsResponse ServiceMetricsResponse
	if err = parseResponseJSON(response, &metricsResponse); err != nil {
		log.Warn().Err(err).Msg("Failed to request or parse proposals metrics")
		return nil
	}

	return metricsResponse.Connects
}

// SendMetric submits new metric
func (m *MysteriumMORQA) SendMetric(event *metrics.Event) error {
	request, err := m.newRequestBinary(http.MethodPost, "metrics", event)
	if err != nil {
		return err
	}

	request.Close = true

	response, err := m.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return parseResponseError(response)
}

// Reconnect creates new instance of underlying retryable HTTP client.
func (m *MysteriumMORQA) Reconnect() {
	m.clientMu.Lock()
	defer m.clientMu.Unlock()
	m.client.HTTPClient.CloseIdleConnections()
	m.client = m.clientFactory()
}

func (m *MysteriumMORQA) resolveClient() *retryablehttp.Client {
	m.clientMu.Lock()
	defer m.clientMu.Unlock()
	if m.client != nil {
		return m.client
	}
	m.client = m.clientFactory()
	return m.client
}

func (m *MysteriumMORQA) newRequest(method, path string, body []byte) (*retryablehttp.Request, error) {
	url := m.baseURL
	if len(path) > 0 {
		url = fmt.Sprintf("%v/%v", url, path)
	}

	request, err := retryablehttp.NewRequest(method, url, bytes.NewBuffer(body))
	request.Header.Set("User-Agent", mysteriumMorqaAgentName)
	request.Header.Set("Accept", "application/json")
	return request, err
}

func (m *MysteriumMORQA) newRequestJSON(method, path string, payload interface{}) (*retryablehttp.Request, error) {
	payloadBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := m.newRequest(method, path, payloadBody)
	req.Header.Set("Accept", "application/json")
	return req, err
}

func (m *MysteriumMORQA) newRequestBinary(method, path string, payload proto.Message) (*retryablehttp.Request, error) {
	payloadBody, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	request, err := m.newRequest(method, path, payloadBody)
	request.Header.Set("Content-Type", "application/octet-stream")
	return request, err
}

func parseResponseJSON(response *http.Response, dto interface{}) error {
	responseJSON, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(responseJSON, dto)
}

type errorDTO struct {
	Message string `json:"message"`
}

// Sometimes we can get json message with single "message" field which represents error - try to get that
func parseResponseError(response *http.Response) error {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}

	var parsedBody errorDTO
	var message string
	err := parseResponseJSON(response, &parsedBody)
	if err != nil {
		message = err.Error()
	} else {
		message = parsedBody.Message
	}
	return fmt.Errorf("server response invalid: %s (%s). Possible error: %s", response.Status, response.Request.URL, message)
}
