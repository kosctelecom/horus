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
	"errors"
	"strings"
)

// Profile represents the device profile. A profile is composed of
// a unique (model, vendor, category) tuple and have a list of scalar and
// tabular measures attached to it.
type Profile struct {
	// ID is the device profile id.
	ID int `db:"profile_id" json:"-"`

	// Category is the device category (router, switch, dslam, etc.)
	Category string `db:"category" json:"category"`

	// Vendor is the device vendor.
	Vendor string `db:"vendor" json:"vendor"`

	// Model is the device model.
	Model string `db:"model" json:"model"`
}

// UnmarshalJSON implements json Unmarhsaler interface for a Profile.
// Validates that the Category, Vendor and Model fields are not empty.
func (prof *Profile) UnmarshalJSON(data []byte) error {
	type P Profile
	var p P

	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	p.Category, p.Vendor, p.Model = strings.TrimSpace(p.Category), strings.TrimSpace(p.Vendor), strings.TrimSpace(p.Model)
	if p.Category == "" || p.Vendor == "" || p.Model == "" {
		return errors.New("invalid profile: category, vendor and model are required")
	}
	*prof = Profile(p)
	return nil
}
