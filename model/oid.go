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
