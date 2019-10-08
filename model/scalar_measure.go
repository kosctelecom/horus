package model

import (
	"horus-core/log"
)

// ScalarMeasure is a scalar measure with its list
// of scalar metrics like sysInfo, sysUsage...
type ScalarMeasure struct {
	// ID is the measure id
	ID int `db:"id"`

	// Name is the name of the scalar measure
	Name string `db:"name"`

	// Description is the description of the scalar metric
	Description string `db:"description"`

	// PollingFrequency is the measures polling frequency.
	// It is only taken in account if greater than device polling frequency.
	PollingFrequency int `db:"polling_frequency"`

	// LastPolledAt is the measure last poll time
	LastPolledAt NullTime `db:"last_polled_at"`

	// Metrics is the list of metrics of this scalar measure
	Metrics []Metric
}

// RemoveInactive filters out all metrics of this scalar measure marked as inactive.
func (scalar *ScalarMeasure) RemoveInactive() {
	var filtered []Metric

	for _, metric := range scalar.Metrics {
		if !metric.Active {
			filtered = append(filtered, metric)
		}
	}
	log.Debug3f("metrics before = %v", Names(scalar.Metrics))
	scalar.Metrics = filtered
	log.Debug3f("metrics after = %v", Names(filtered))
}
