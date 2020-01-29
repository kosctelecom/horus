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
	"regexp"
	"testing"
)

func TestIndexedMeas(t *testing.T) {
	tests := []struct {
		in    string
		out   IndexedMeasure
		valid bool
	}{
		{
			`{
				"Name":"ifStatus",
				"Metrics": [
					{"ID":8, "Name":"ifIndex", "Oid":".1.3.6.1.2.1.2.2.1.1", "Active":true, "ExportAsLabel":true},
					{"ID":9, "Name":"ifDescr", "Oid":".1.3.6.1.2.1.2.2.1.2", "Active":true, "ExportAsLabel":false},
					{"ID":10, "Name":"ifAdminStatus", "Oid":".1.3.6.1.2.1.2.2.1.7", "Active":true, "ExportAsLabel":false},
					{"ID":11, "Name":"ifOperStatus", "Oid":".1.3.6.1.2.1.2.2.1.8", "Active":true, "ExportAsLabel":false}
				],
				"IndexMetricID": 8,
				"FilterMetricID": 9,
				"FilterPattern": "DSL"
			}`,
			IndexedMeasure{
				Name:           "ifStatus",
				IndexMetricID:  8,
				FilterMetricID: NullInt64{9, true},
				FilterPattern:  "DSL",
				FilterRegex:    regexp.MustCompile("DSL"),
				IndexPos:       0,
				FilterPos:      1,
				Metrics: []Metric{
					Metric{
						ID:            8,
						Name:          "ifIndex",
						Oid:           ".1.3.6.1.2.1.2.2.1.1",
						Active:        true,
						ExportAsLabel: true,
					},
					Metric{
						ID:            9,
						Name:          "ifDescr",
						Oid:           ".1.3.6.1.2.1.2.2.1.2",
						Active:        true,
						ExportAsLabel: false,
					},
					Metric{
						ID:            10,
						Name:          "ifAdminStatus",
						Oid:           ".1.3.6.1.2.1.2.2.1.7",
						Active:        true,
						ExportAsLabel: false,
					},
					Metric{
						ID:            11,
						Name:          "ifOperStatus",
						Oid:           ".1.3.6.1.2.1.2.2.1.8",
						Active:        true,
						ExportAsLabel: false,
					},
				},
			},
			true,
		},
		{
			`{
				"Name":"ifStatus",
				"Metrics": [
					{"ID":8, "Name":"ifIndex", "Oid":".1.3.6.1.2.1.2.2.1.1", "Active":true, "ExportAsLabel":true},
					{"ID":9, "Name":"ifDescr", "Oid":".1.3.6.1.2.1.2.2.1.2", "Active":true, "ExportAsLabel":false}
				],
				"IndexMetricID": 8,
				"FilterPattern": "DSL"
			}`,
			IndexedMeasure{},
			false, // no FilterMetricId
		},
		{
			`{
				"Name":"ifStatus",
				"Metrics": [
					{"ID":8, "Name":"ifIndex", "Oid":".1.3.6.1.2.1.2.2.1.1", "Active":true, "ExportAsLabel":true},
					{"ID":9, "Name":"ifDescr", "Oid":".1.3.6.1.2.1.2.2.1.2", "Active":true, "ExportAsLabel":false}
				],
				"IndexMetricID": 8,
				"FilterMetricId": 1,
				"FilterPattern": ""
			}`,
			IndexedMeasure{},
			false, // no FilterPattern
		},
		{
			`{
				"Name":"ifStatus",
				"Metrics": [
					{"ID":8, "Name":"ifIndex", "Oid":".1.3.6.1.2.1.2.2.1.1", "Active":true, "ExportAsLabel":true},
					{"ID":9, "Name":"ifDescr", "Oid":".1.3.6.1.2.1.2.2.1.2", "Active":true, "ExportAsLabel":false}
				],
				"IndexMetricID": 88,
				"FilterMetricId": 9,
				"FilterPattern": "DSL"
			}`,
			IndexedMeasure{},
			false, // invalid IndexMetricID
		},
		{
			`{
				"Name":"ifStatus",
				"Metrics": [
					{"ID":8, "Name":"ifIndex", "Oid":".1.3.6.1.2.1.2.2.1.1", "Active":true, "ExportAsLabel":true},
					{"ID":9, "Name":"ifDescr", "Oid":".1.3.6.1.2.1.2.2.1.2", "Active":true, "ExportAsLabel":false}
				],
				"IndexMetricID": 8,
				"FilterMetricId": 99,
				"FilterPattern": "DSL"
			}`,
			IndexedMeasure{},
			false, // invalid FilterMetricID
		},
	}

	for i, tt := range tests {
		var im IndexedMeasure
		err := json.Unmarshal([]byte(tt.in), &im)
		valid := err == nil
		if !valid && testing.Verbose() {
			t.Logf("IndexedMeasure#%d: unmarshal: %v", i, err)
		}
		if valid != tt.valid {
			t.Errorf("IndexedMeasure#%d: expected validity: %v, got %v", i, tt.valid, valid)
		}
		if valid && !reflect.DeepEqual(im, tt.out) {
			t.Errorf("IndexedMeasure#%d: expected:\n%+v\ngot:\n%+v\n", i, tt.out, im)
		}
	}
}
