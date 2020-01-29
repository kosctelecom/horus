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

func TestScalarMeasure(t *testing.T) {
	tests := []struct {
		in    string
		out   ScalarMeasure
		valid bool
	}{
		{
			`{
			"Name": "sysUsage",
			"Metrics": [
				{"Name":"sysName", "Oid":".1.3.6.1.2.1.1.5.0", "Active":true, "PollingFrequency": 300}
			]
		}`,
			ScalarMeasure{
				Name: "sysUsage",
				Metrics: []Metric{
					Metric{
						Name:             "sysName",
						Oid:              ".1.3.6.1.2.1.1.5.0",
						Active:           true,
						PollingFrequency: 300,
					},
				},
			},
			true,
		},
	}

	for i, tt := range tests {
		var sm ScalarMeasure
		err := json.Unmarshal([]byte(tt.in), &sm)
		valid := err == nil
		if !valid && testing.Verbose() {
			t.Logf("ScalarMeasure#%d: unmarshal: %v", i, err)
		}
		if valid != tt.valid {
			t.Errorf("ScalarMeasure#%d: expected validity: %v, got %v", i, tt.valid, valid)
		}
		if valid && !reflect.DeepEqual(sm, tt.out) {
			t.Errorf("ScalarMeasure#%d: expected:\n%+v\ngot:\n%+v\n", i, tt.out, sm)
		}
	}
}
