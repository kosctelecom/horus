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
	"database/sql"
	"database/sql/driver"
	"encoding/json"
)

// NullInt64 is a sql.NullInt64 with custom json marshaller/unmarshaller.
type NullInt64 sql.NullInt64

// UnmarshalJSON implements the json.Unmarshaler interface with a special
// case for json `null` (unquoted) converted as a null int without error.
func (n *NullInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Int64, n.Valid = 0, false
		return nil
	}

	var i int64
	err := json.Unmarshal(data, &i)
	if err != nil {
		n.Int64, n.Valid = 0, false
	} else {
		n.Int64, n.Valid = i, true
	}
	return err
}

// MarshalJSON implements the json.Marshaler interface with invalid values
// converted to json `null`.
func (n NullInt64) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Int64)
	}
	return json.Marshal(nil)
}

// Scan implements the sql.Scanner interface.
func (n *NullInt64) Scan(value interface{}) error {
	nn := new(sql.NullInt64)
	err := nn.Scan(value)
	n.Int64, n.Valid = nn.Int64, nn.Valid
	return err
}

// Value implements the driver sql.Valuer interface.
func (n NullInt64) Value() (driver.Value, error) {
	return sql.NullInt64(n).Value()
}
