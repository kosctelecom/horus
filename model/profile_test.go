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

func TestProfile(t *testing.T) {
	tests := []struct {
		in    string
		out   Profile
		valid bool
	}{
		{
			`{
				"category": "C",
				"vendor": "V",
				"model": "M"
			}`,
			Profile{
				Category: "C",
				Vendor:   "V",
				Model:    "M",
			},
			true,
		},
		{
			`{
				"category": " C ",
				"vendor": "  V",
				"model": "M  "
			}`,
			Profile{
				Category: "C",
				Vendor:   "V",
				Model:    "M",
			},
			true,
		},
		{
			`{
				"category": "C",
				"vendor": "",
				"model": "M"
			}`,
			Profile{},
			false,
		},
	}
	for i, tt := range tests {
		var p Profile
		err := json.Unmarshal([]byte(tt.in), &p)
		valid := err == nil
		if !valid && testing.Verbose() {
			t.Logf("Profile#%d: unmarshal: %v", i, err)
		}
		if valid != tt.valid {
			t.Errorf("Profile#%d: expected validity: %v, got %v (err: %v)", i, tt.valid, valid, err)
		}
		if tt.valid && !reflect.DeepEqual(tt.out, p) {
			t.Errorf("Profile#%d: expected:\n%+v\ngot:\n%+v\n", i, tt.out, p)
		}
	}
}
