package model

import (
	"encoding/json"
	"errors"
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

	// HonorRunningOnly tells if we must take into account the metric RunningIfOnly flag.
	HonorRunningOnly bool `db:"honor_running_only" json:"honor_running_only,omitempty"`
}

// UnmarshalJSON implements json Unmarhsaler interface for a Profile.
// Validates that the Category, Vendor and Model fields are not empty.
func (prof *Profile) UnmarshalJSON(data []byte) error {
	type P Profile
	var p P

	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	if p.Category == "" || p.Vendor == "" || p.Model == "" {
		return errors.New("invalid profile: category, vendor and model are required")
	}
	*prof = Profile(p)
	return nil
}
