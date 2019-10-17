package model

import (
	"encoding/json"
	"fmt"
	"horus/log"
	"regexp"
)

// IndexedMeasure is a group of tabular metrics indexed by the first one.
type IndexedMeasure struct {
	// ID is the measure db id.
	ID int `db:"id"`

	// Name is the name of the indexed measure.
	Name string `db:"name"`

	// Description is the description of the indexed measure.
	Description string `db:"description"`

	// Metrics is the list of metrics forming this measure.
	Metrics []Metric

	// IndexMetricID is the id of the metric used as index.
	IndexMetricID int `db:"index_metric_id"`

	// IndexPos is the position of the index metric in the Metrics array.
	IndexPos int `db:"-"`

	// PollingFrequency is the measure specific polling frequency.
	// It is only taken in account if greater than the device polling frequency.
	PollingFrequency int `db:"polling_frequency"`

	// LastPolledAt is the measure last poll time.
	LastPolledAt NullTime `db:"last_polled_at"`

	// FilterPattern is the regex pattern used to filter the IndexedResults of this metric group.
	// It can be used to only keep results from interesting interfaces.
	FilterPattern string `db:"filter_pattern"`

	// FilterMetricID is the id of the metric on which the filter is applied.
	FilterMetricID NullInt64 `db:"filter_metric_id"`

	// FilterPos is the index of the filter metric in the Metrics array.
	FilterPos int `db:"-"`

	// InvertFilterMatch negates the match result of the FilterPattern.
	InvertFilterMatch bool `db:"invert_filter_match"`

	// FilterRegex is the compiled FilterPattern pattern.
	FilterRegex *regexp.Regexp `db:"-" json:"-"`
}

// UnmarshalJSON unserializes data into an IndexedMetric.
// Checks specifically if the filter index and pattern are valid.
func (x *IndexedMeasure) UnmarshalJSON(data []byte) error {
	type IM IndexedMeasure
	var im IM

	if err := json.Unmarshal(data, &im); err != nil {
		return err
	}
	im.IndexPos = -1
	for i, metric := range im.Metrics {
		if metric.ID == im.IndexMetricID {
			im.IndexPos = i
			break
		}
	}
	if im.IndexPos == -1 {
		return fmt.Errorf("indexed measure %s: IndexMetricID %d not found in metric list", im.Name, im.IndexMetricID)
	}
	if im.FilterPattern != "" && !im.FilterMetricID.Valid {
		return fmt.Errorf("indexed measure %s: FilterMetricID cannot be null when FilterPattern is defined", im.Name)
	}
	if im.FilterPattern == "" && im.FilterMetricID.Valid {
		return fmt.Errorf("indexed measure %s: FilterPattern cannot be empty when FilterMetricID is defined", im.Name)
	}
	im.FilterPos = -1
	if im.FilterPattern != "" {
		for i, metric := range im.Metrics {
			if int64(metric.ID) == im.FilterMetricID.Int64 {
				im.FilterPos = i
				break
			}
		}
		if im.FilterPos == -1 {
			return fmt.Errorf("indexed measure: invalid FilterMetricID %d, not in metric list", im.FilterMetricID.Int64)
		}
		var err error
		if im.FilterRegex, err = regexp.Compile(im.FilterPattern); err != nil {
			return fmt.Errorf("invalid filter regexp: %v", err)
		}
	}
	*x = IndexedMeasure(im)
	return nil
}

// RemoveInactive filters out all metrics of this indexed measure that are marked as inactive.
func (x *IndexedMeasure) RemoveInactive() {
	var filtered []Metric

	for _, metric := range x.Metrics {
		if metric.Active {
			filtered = append(filtered, metric)
		}
	}
	log.Debug3f("metrics before = %v", Names(x.Metrics))
	x.Metrics = filtered
	log.Debug3f("metrics after = %v", Names(filtered))
}

// HasMetricWithRunningOnly returns true if this measure has a metric with
// RunningIfaceOnly flag is true.
func (x IndexedMeasure) HasMetricWithRunningOnly() bool {
	for _, metr := range x.Metrics {
		if metr.RunningIfaceOnly {
			return true
		}
	}
	return false
}
