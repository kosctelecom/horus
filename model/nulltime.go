package model

import (
	"database/sql/driver"
	"time"
)

// NullTime represents a nullable time.Time.
type NullTime struct {
	Time  time.Time
	Valid bool
}

// Scan implements the sql Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	nt.Time, nt.Valid = value.(time.Time)
	return nil
}

// Value implements the sql driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

// MarshalJSON implements json.Marshaler.
func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	return nt.Time.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler. Accepts
// either `null` (unquoted) or a quoted string in RFC 3339 format.
func (nt *NullTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nt.Valid = false
		return nil
	}
	err := nt.Time.UnmarshalJSON(data)
	nt.Valid = (err == nil)
	return err
}
