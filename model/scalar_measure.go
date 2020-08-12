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

import "github.com/kosctelecom/horus/log"

// ScalarMeasure is a scalar measure with its list
// of scalar metrics like sysInfo, sysUsage...
type ScalarMeasure struct {
	// ID is the measure id
	ID int `db:"id"`

	// Name is the name of the scalar measure
	Name string `db:"name"`

	// Description is the description of the scalar metric
	Description string `db:"description"`

	// Metrics is the list of metrics of this scalar measure
	Metrics []Metric

	// UseAlternateCommunity tells wether to use the alternate community for all metrics of this measure.
	UseAlternateCommunity bool `db:"use_alternate_community"`

	// ToKafka is a flag telling if the results are exported to Kafka.
	ToKafka bool `db:"to_kafka"`

	// ToProm tells if the results are kept for Prometheus scraping.
	ToProm bool `db:"to_prometheus"`

	// ToInflux is a flag telling if the results are exported to InfluxDB.
	ToInflux bool `db:"to_influx"`

	// ToNats tells if the results are exported to NATS.
	ToNats bool `db:"to_nats"`
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
