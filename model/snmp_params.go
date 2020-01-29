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

	"github.com/vma/gosnmp"
)

const (
	// Version1 is for snmp v1
	Version1 = "1"

	// Version2c is for snmp v2c
	Version2c = "2c"

	// Version3 is for snmp v3
	Version3 = "3"
)

// SnmpParams represents the snmp config params.
type SnmpParams struct {
	// IPAddress is the device's ip address for snmp polling.
	IPAddress string `db:"ip_address" json:"ip_address"`

	// Port is the device's snmp port.
	Port int `db:"snmp_port" json:"snmp_port"`

	// Version is the snmp version available for the device.
	Version string `db:"snmp_version" json:"snmp_version"`

	// Community is the device's snmp community.
	Community string `db:"snmp_community" json:"snmp_community"`

	// AlternateCommunity is an alternate snmp community used for querying some metrics.
	AlternateCommunity string `db:"snmp_alternate_community" json:"snmp_alternate_community"`

	// Timeout is the snmp query timeout (default 10s).
	Timeout int `db:"snmp_timeout" json:"snmp_timeout"`

	// Retries is the number of retries to attempt on timeout (default 1).
	Retries int `db:"snmp_retries" json:"snmp_retries"`

	// DisableBulk is a flag that disables snmp bulk requests (automatic for snmp v1).
	DisableBulk bool `db:"snmp_disable_bulk" json:"snmp_disable_bulk,omitempty"`

	// ConnectionCount is the number of possible simultaneous snmp queries
	// to the device (defaults to 1).
	ConnectionCount int `db:"snmp_connection_count" json:"snmp_connection_count"`

	// SecLevel is the security level for snmpv3: "NoAuthNoPriv", "AuthNoPriv" or "AuthPriv".
	SecLevel string `db:"snmpv3_security_level" json:"snmpv3_security_level,omitempty"`

	// AuthUser is the authentication username for snmpv3.
	AuthUser string `db:"snmpv3_auth_user" json:"snmpv3_auth_user,omitempty"`

	// AuthProto is the authentication protocol for snmpv3: "MD5" or "SHA".
	AuthProto string `db:"snmpv3_auth_proto" json:"snmpv3_auth_proto,omitempty"`

	// AuthPasswd is the authentication password for snmpv3.
	AuthPasswd string `db:"snmpv3_auth_passwd" json:"snmpv3_auth_passwd,omitempty"`

	// PrivProto is the privacy protocol for snmpv3: "DES" or "AES".
	PrivProto string `db:"snmpv3_privacy_proto" json:"snmpv3_privacy_proto,omitempty"`

	// PrivPasswd is the privacy passphrase for snmpv3.
	PrivPasswd string `db:"snmpv3_privacy_passwd" json:"snmpv3_privacy_passwd,omitempty"`
}

// UnmarshalJSON implements the json Unmarshaler interface.
// Does some additional checks and sets default values for fields.
func (s *SnmpParams) UnmarshalJSON(data []byte) error {
	type S SnmpParams
	var params S

	if err := json.Unmarshal(data, &params); err != nil {
		return err
	}
	if params.IPAddress == "" {
		return errors.New("invalid snmp params: ip_address cannot be empty")
	}
	if params.Community == "" {
		return errors.New("invalid snmp params: community cannot be empty")
	}
	if params.Port == 0 {
		params.Port = 161
	}
	if params.Version == "" {
		params.Version = Version2c
	}
	if params.Version != Version1 && params.Version != Version2c && params.Version != Version3 {
		return errors.New("invalid version " + params.Version + ": must be either `1`, or `2c`, or `3`")
	}
	if params.Timeout == 0 {
		params.Timeout = 10
	}
	if params.Retries == 0 {
		params.Retries = 1
	}
	if params.Version == Version1 {
		params.DisableBulk = true
	}
	if params.ConnectionCount == 0 {
		params.ConnectionCount = 1
	}
	if params.Version == Version3 {
		if !strInList(params.SecLevel, "NoAuthNoPriv", "AuthNoPriv", "AuthPriv") {
			return errors.New("invalid snmp params: snmpv3_security_level must be either NoAuthNoPriv, AuthNoPriv or AuthPriv")
		}
		if params.SecLevel != "NoAuthNoPriv" && params.AuthUser == "" {
			return errors.New("invalid snmp params: snmpv3_auth_user cannot be empty with this v3_security_level")
		}
		if !strInList(params.AuthProto, "", "MD5", "SHA") {
			return errors.New("invalid inmp params: snmpv3_auth_proto must be either empty, MD5 or SHA")
		}
		if !strInList(params.PrivProto, "", "DES", "AES") {
			return errors.New("invalid snmp params: snmpv3_privacy_proto must be either empty, DES or AES")
		}
	}
	*s = SnmpParams(params)
	return nil
}

// GoSnmpVersion converts the snmp version to a gosnmp version.
func (s SnmpParams) GoSnmpVersion() gosnmp.SnmpVersion {
	switch s.Version {
	case Version1:
		return gosnmp.Version1
	case Version2c:
		return gosnmp.Version2c
	default:
		return gosnmp.Version3
	}
}

// strInList tells wether elem is part of list.
func strInList(elem string, list ...string) bool {
	for _, s := range list {
		if elem == s {
			return true
		}
	}
	return false
}
