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
	"reflect"
	"testing"
)

func TestDevice(t *testing.T) {
	tests := []struct {
		in    string
		out   Device
		valid bool
	}{
		{
			`{
				"id": 9,
				"active": true,
				"hostname": "9.kosc.net",
				"polling_frequency": 300,
				"ping_frequency": 60,
				"tags": "",
				"category": "c1",
				"vendor": "v1",
				"model": "m1",
				"to_kafka": true,
				"ip_address": "10.2.0.9",
				"snmp_port": 163,
				"snmp_version": "2c",
				"snmp_community": "snmpxxx9",
				"snmpv3_auth_user": "snmpv3auth"
			}`,
			Device{
				ID:               9,
				Active:           true,
				Hostname:         "9.kosc.net",
				PollingFrequency: 300,
				PingFrequency:    60,
				Tags:             "{}",
				SnmpParams: SnmpParams{
					IPAddress:       "10.2.0.9",
					Port:            163,
					Version:         Version2c,
					Community:       "snmpxxx9",
					Timeout:         10,
					Retries:         1,
					ConnectionCount: 1,
					AuthUser:        "snmpv3auth",
				},
				Profile: Profile{
					Category: "c1",
					Vendor:   "v1",
					Model:    "m1",
				},
			},
			true,
		},
		{
			`{
				"id": 2,
				"active": true,
				"hostname": "2.kosc.net",
				"polling_frequency": 10,
				"tags": "",
				"category": "c1",
				"vendor": "v1",
				"model": "m1"
			}`,
			Device{},
			false, // no snmp fields
		},
		{
			`{
				"id": 3,
				"active": true,
				"hostname": "3.kosc.net",
				"polling_frequency": 10,
				"tags": ""
			}`,
			Device{},
			false, // no profile & snmp fields
		},
	}

	for i, tt := range tests {
		var d Device
		err := json.Unmarshal([]byte(tt.in), &d)
		valid := err == nil
		if !valid && testing.Verbose() {
			t.Logf("Device#%d: unmarshal: %v", i, err)
		}
		if valid != tt.valid {
			t.Errorf("Device#%d: expected validity: %v, got %v (err: %v)", i, tt.valid, valid, err)
		}
		if tt.valid && !reflect.DeepEqual(tt.out, d) {
			t.Errorf("Device#%d: expected:\n%+v\ngot:\n%+v\n", i, tt.out, d)
		}
	}
}
