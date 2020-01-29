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

import "testing"

func TestUnmarshalOID(t *testing.T) {
	tests := []struct {
		in    string
		o     OID
		valid bool
	}{
		{`".1.3.6.1.2.1.1.1.0"`, OID(".1.3.6.1.2.1.1.1.0"), true},
		{`"1.3.6.1.2.1.1.3.0"`, OID(".1.3.6.1.2.1.1.3.0"), true},
	}

	for _, tt := range tests {
		var o OID
		err := o.UnmarshalJSON([]byte(tt.in))
		valid := (err == nil)
		if tt.valid != valid {
			t.Fatalf("UnmarshalJSON: %s valid? expected %v, got %v (err: %v)", tt.in, tt.valid, valid, err)
		}
		if valid && o != tt.o {
			t.Fatalf("UnmarshalJSON: input %s: expected %v, got %v", tt.in, tt.o, o)
		}
	}
}
