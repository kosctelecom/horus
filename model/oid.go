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
	"errors"
	"fmt"
	"regexp"
)

// OID represents a dotted OID string.
type OID string

// oidPattern is the regexp pattern of a valid OID.
var oidPattern = regexp.MustCompile(`^\.?(\d+\.)+\d+$`)

// MarshalJSON implements the json Marshaler interface for the OID.
func (o OID) MarshalJSON() ([]byte, error) {
	if !oidPattern.MatchString(string(o)) {
		return nil, fmt.Errorf("MarshalJSON: bad OID format `%s`", o)
	}
	return []byte(`"` + o + `"`), nil
}

// UnmarshalJSON implements the json Unmarshaler interface for the OID
// Validates the correct oid format and adds leading dot if needed.
func (o *OID) UnmarshalJSON(value []byte) error {
	if len(value) < 2 {
		return errors.New("UnmarshalJSON: bad OID")
	}
	sval := string(value)[1 : len(value)-1] // strip quotes
	if !oidPattern.MatchString(sval) {
		return fmt.Errorf("UnmarshalJSON: bad OID `%s`", sval)
	}
	if sval[:1] != "." {
		// add leading dot
		sval = "." + sval
	}
	*o = OID(sval)
	return nil
}
