package model

import (
	"encoding/json"
	"errors"
)

// Device represents an snmp device.
type Device struct {
	// ID is the device id.
	ID int `db:"id" json:"id"`

	// Active tells whether the device can be polled.
	Active bool `db:"active" json:"active"`

	// Hostname is the device's FQDN.
	Hostname string `db:"hostname" json:"hostname"`

	// PollingFrequency is the device's snmp polling frequency.
	PollingFrequency int `db:"polling_frequency" json:"polling_frequency"`

	// Tags is the influx tags (and prometheus labels) added to
	// each measurement of this device.
	Tags string `db:"tags" json:"tags,omitempty"`

	// ToInflux is a flag telling if the results should be exported to InfluxDB.
	// This will only work if the agent actually has an influxdb connection.
	ToInflux bool `db:"to_influx" json:"to_influx"`

	// ToKafka is a flag telling if the results should be exported to Kafka.
	ToKafka bool `db:"to_kafka" json:"to_kafka"`

	// ToProm tells if the results are kept for Prometheus scraping.
	ToProm bool `db:"to_prometheus" json:"to_prometheus"`

	// SnmpParams is the device snmp config.
	SnmpParams

	// Profile is the device profile.
	Profile
}

// UnmarshalJSON implements the json Unmarshaler interface for Device type.
// Takes a flat json and builds a Device with embedded Profile and SnmpParams.
// Note: the standard Marshaler also outputs a flat json document.
func (dev *Device) UnmarshalJSON(data []byte) error {
	type D Device
	var d D

	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	if d.ID == 0 {
		return errors.New("invalid device: id cannot be empty")
	}
	if d.Hostname == "" {
		return errors.New("invalid device: hostname cannot be empty")
	}
	if d.PollingFrequency == 0 {
		return errors.New("invalid device: polling_frequency cannot be zero or empty")
	}
	if !d.ToProm && !d.ToInflux && !d.ToKafka {
		return errors.New("invalid device: either to_kafka or to_influx or to_prometheus must be set")
	}
	var t map[string]interface{}
	if d.Tags != "" && json.Unmarshal([]byte(d.Tags), &t) != nil {
		return errors.New("invalid device: tags must be a valid json map")
	}
	if err := json.Unmarshal(data, &d.Profile); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &d.SnmpParams); err != nil {
		return err
	}
	*dev = Device(d)
	return nil
}
