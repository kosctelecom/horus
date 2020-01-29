// Copyright 2019-2020 Kosc Telecom.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// SnmpRequest represents a snmp poll request.
type SnmpRequest struct {
	// UID is the request unique id
	UID string `json:"uid"`

	// AgentID is the agent id
	AgentID int `json:"agent_id"`

	// ScalarMeasures is a list of scalar measures to poll
	ScalarMeasures []ScalarMeasure `json:",omitempty"`

	// IndexedMeasures is a list of tabular measures to poll
	IndexedMeasures []IndexedMeasure `json:",omitempty"`

	// ReportURL is the url where the polling result report is sent
	ReportURL string `json:"report_url"`

	// Device is the network device to poll.
	Device Device `json:"device"`
}

// OngoingPolls is the result to the OngoingURI api request.
type OngoingPolls struct {
	// Requests is the current polling requests IDs.
	Requests []string `json:"ongoing"`

	// Load is the current load of the agent.
	Load float64 `json:"load"`
}

// PingHost is a host to ping.
type PingHost struct {
	// Name is the target hostname
	Name string `db:"hostname" json:"hostname"`

	// IpAddr is the target ip address
	IpAddr string `db:"ip_address" json:"ip_address"`

	// Category is the equipment category (for profile identification)
	Category string `db:"category" json:"category"`

	// Vendor is the equipment vendor (for profile identification)
	Vendor string `db:"vendor" json:"vendor"`

	// Model is the equipment model (for profile identification)
	Model string `db:"model" json:"model"`
}

// PingRequest is a ping job sent to an agent.
type PingRequest struct {
	// UID is the request unique ID
	UID string `json:"uid"`

	// Hosts is the list of hosts to ping
	Hosts []PingHost `json:"hosts"`

	// Stamp is the ping metric timestamp
	Stamp time.Time `json:"-"`
}

const (
	// SnmpJobURI is the agent uri for snmp poll requests
	SnmpJobURI = "/r/poll"

	// CheckURI is the agent keep-alive uri
	CheckURI = "/r/check"

	// PingURI is the agent uri for ping jobs
	PingJobURI = "/r/ping"

	// OngoingURI is the agent current ongoing request list uri endpoint
	OngoingURI = "/r/ongoing"

	// ReportURI is the controller report callback uri
	ReportURI = "/r/report"
)

// UnmarshalJSON validates the json input and unmarshals it to and SnmpRequest.
func (r *SnmpRequest) UnmarshalJSON(data []byte) error {
	type R SnmpRequest
	var req R

	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}
	if req.UID == "" {
		return errors.New("invalid request: request_id cannot be empty")
	}
	if req.ReportURL != "" && !strings.HasPrefix(req.ReportURL, "http://") && !strings.HasPrefix(req.ReportURL, "https://") {
		req.ReportURL = "http://" + req.ReportURL
	}
	if req.Device.ID == 0 {
		return errors.New("invalid request: missing device")
	}
	*r = SnmpRequest(req)
	return nil
}

// Targets returns the list of host targets for this ping request.
func (r PingRequest) Targets() []string {
	res := make([]string, len(r.Hosts))
	for i, h := range r.Hosts {
		res[i] = h.IpAddr
	}
	return res
}
