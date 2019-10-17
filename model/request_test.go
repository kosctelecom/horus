// Copyright 2019 Kosc Telecom.
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

func TestRequest(t *testing.T) {
	tests := []struct {
		in    string
		out   SnmpRequest
		valid bool
	}{
		{
			`{
				"uid": "000",
				"agent_id": 1,
				"report_url": "localhost/report",
				"device": {
					"id": 1500,
					"hostname": "dsl1500.kosc.net",
					"polling_frequency": 300,
					"category": "c",
					"vendor": "v",
					"model": "m",
					"ip_address": "10.2.0.9",
					"snmp_timeout": 20,
					"snmp_version": "2c",
					"snmp_community": "snmpxxx9",
					"to_kafka": true
				}
			}`,
			SnmpRequest{
				UID:       "000",
				AgentID:   1,
				ReportURL: "http://localhost/report",
				Device: Device{
					ID:               1500,
					Hostname:         "dsl1500.kosc.net",
					PollingFrequency: 300,
					ToKafka:          true,
					SnmpParams: SnmpParams{
						IPAddress:       "10.2.0.9",
						Port:            161,
						Version:         Version2c,
						Community:       "snmpxxx9",
						Timeout:         20,
						Retries:         1,
						ConnectionCount: 1,
					},
					Profile: Profile{
						Category: "c",
						Vendor:   "v",
						Model:    "m",
					},
				},
			},
			true,
		},
		{
			`{
				"uid": "001"
			}`,
			SnmpRequest{},
			false, // empty device
		},
		{
			`{
				"uid": "002",
				"agent_id": 1,
				"report_url": "localhost/report",
				"device": {
					"id": 1,
					"hostname": "10.2.0.9",
					"polling_frequency": 300,
					"to_kafka": true,
					"category": "c1",
					"vendor": "v1",
					"model": "m1",
					"ip_address": "10.2.0.9",
					"snmp_timeout": 20,
					"snmp_version": "2c",
					"snmp_community": "snmpxxx3"
				}
			}`,
			SnmpRequest{
				UID:       "002",
				AgentID:   1,
				ReportURL: "http://localhost/report",
				Device: Device{
					ID:               1,
					Hostname:         "10.2.0.9",
					PollingFrequency: 300,
					ToKafka:          true,
					SnmpParams: SnmpParams{
						IPAddress:       "10.2.0.9",
						Port:            161,
						Version:         Version2c,
						Community:       "snmpxxx3",
						Timeout:         20,
						Retries:         1,
						ConnectionCount: 1,
					},
					Profile: Profile{
						Category: "c1",
						Vendor:   "v1",
						Model:    "m1",
					},
				},
			},
			true,
		},
		{
			`{
				"uid": "003",
				"device": {
					"id": 1492,
					"hostname": "1492.kosc.net",
					"polling_frequency": 300,
					"category": "c1",
					"vendor": "v1",
					"model": "m1",
					"to_kafka":true,
					"ip_address": "10.2.0.92",
					"snmp_timeout": 20,
					"snmp_version": "2c",
					"snmp_community": "snmpxxx2"
				},
				"ScalarMeasures": [
					{"Name": "sysUsage", "PollingFrequency": 300, "Metrics":[
						{"Name":"sysName", "Oid":".1.3.6.1.2.1.1.5.0", "Active":true}
					]
				}],
				"IndexedMeasures": [{
					"Name":"ifStatus", "Metrics":[
						{"ID":8, "Name":"ifIndex", "Oid":".1.3.6.1.2.1.2.2.1.1", "Active":true, "ExportAsLabel":true}
					],
					"IndexMetricID": 8
				}]
			}`,
			SnmpRequest{
				UID: "003",
				Device: Device{
					ID:               1492,
					Hostname:         "1492.kosc.net",
					PollingFrequency: 300,
					ToKafka:          true,
					SnmpParams: SnmpParams{
						IPAddress:       "10.2.0.92",
						Port:            161,
						Version:         Version2c,
						Community:       "snmpxxx2",
						Timeout:         20,
						Retries:         1,
						ConnectionCount: 1,
					},
					Profile: Profile{
						Category: "c1",
						Vendor:   "v1",
						Model:    "m1",
					},
				},
				ScalarMeasures: []ScalarMeasure{
					ScalarMeasure{
						Name:             "sysUsage",
						PollingFrequency: 300,
						Metrics: []Metric{
							Metric{
								Name:          "sysName",
								Oid:           ".1.3.6.1.2.1.1.5.0",
								Active:        true,
								ExportAsLabel: false,
							},
						},
					},
				},
				IndexedMeasures: []IndexedMeasure{
					IndexedMeasure{
						Name:             "ifStatus",
						PollingFrequency: 0,
						Metrics: []Metric{
							Metric{
								ID:            8,
								Name:          "ifIndex",
								Oid:           ".1.3.6.1.2.1.2.2.1.1",
								Active:        true,
								ExportAsLabel: true,
							},
						},
						IndexMetricID: 8,
						IndexPos:      0,
						FilterPos:     -1,
					},
				},
			},
			true,
		},
	}

	for i, tt := range tests {
		var r SnmpRequest
		err := json.Unmarshal([]byte(tt.in), &r)
		valid := (err == nil)
		if !valid && testing.Verbose() {
			t.Logf("unmarshal Request#%d: %v", i, err)
		}
		if valid != tt.valid {
			t.Errorf("Request#%d: expected validity: %v, got %v", i, tt.valid, valid)
		}
		if valid && !reflect.DeepEqual(r, tt.out) {
			t.Errorf("Request#%d: expected:\n%+v\ngot:\n%+v\n", i, tt.out, r)
		}
	}
}
