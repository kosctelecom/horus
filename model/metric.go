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
	"fmt"
	"regexp"
	"strings"
)

// Metric represents a single snmp OID to poll.
type Metric struct {
	// ID is the metric db ID.
	ID int `db:"id"`

	// Name is the metric name.
	Name string `db:"name"`

	// Oid is the metric OID.
	Oid OID `db:"oid"`

	// Description is the metric description.
	Description string `db:"description"`

	// Active indicates if this metric is actually polled (all inactive metrics are ignored).
	Active bool `db:"active"`

	// ExportAsLabel tells if this metric is exported as a prometheus label (instead of value).
	ExportAsLabel bool `db:"export_as_label"`

	// ToKafka is a flag telling if the results should be exported to Kafka.
	ToKafka bool `db:"to_kafka"`

	// ToProm tells if the results are kept for Prometheus scraping.
	ToProm bool `db:"to_prometheus"`

	// ToInflux is a flag telling if the results should be exported to InfluxDB.
	ToInflux bool `db:"to_influx"`

	// RunningIfaceOnly tells to retrieve this metric only for running ifaces (for indexed metrics).
	RunningIfaceOnly bool `db:"running_if_only"`

	// IndexPattern is the regex with subexpression used to extract index from tabular Oids.
	IndexPattern string `json:",omitempty" db:"index_pattern"`

	// IndexRegex is the compiled IndexPattern regexp.
	IndexRegex *regexp.Regexp `json:"-" db:"-"`
}

// UnmarshalJSON unserializes a Metric. Checks specifically if the index pattern
// is valid and contains at least one sub-expression.
func (metric *Metric) UnmarshalJSON(data []byte) error {
	type M Metric
	var metr M

	if err := json.Unmarshal(data, &metr); err != nil {
		return err
	}
	if metr.IndexPattern != "" {
		escaped := strings.Replace(metr.IndexPattern, `.`, `\.`, -1)
		metr.IndexPattern = strings.Replace(escaped, `\\.`, `\.`, -1)
		if !strings.HasPrefix(metr.IndexPattern, strings.Replace(string(metr.Oid), `.`, `\.`, -1)) {
			return fmt.Errorf("index_pattern `%s` must start with oid `%s`", metr.IndexPattern, metr.Oid)
		}
		var err error
		if metr.IndexRegex, err = regexp.Compile(metr.IndexPattern); err != nil {
			return fmt.Errorf("invalid index pattern: %v", err)
		}
		if metr.IndexRegex.NumSubexp() < 1 {
			return fmt.Errorf("index_pattern `%s` must contain at least one capture group for the index", metr.IndexPattern)
		}
	}
	*metric = Metric(metr)
	return nil
}

// Names returns the names of the metric list in an array.
func Names(metrics []Metric) []string {
	res := make([]string, len(metrics))
	for i, m := range metrics {
		res[i] = m.Name
	}
	return res
}

// GroupByOid returns a list of an array of metrics grouped by base OID.
func GroupByOid(metrics []Metric) [][]Metric {
	var res [][]Metric

	grouped := make(map[OID][]Metric)
	for _, m := range metrics {
		// group by oid
		grouped[m.Oid] = append(grouped[m.Oid], m)
	}
	for _, m := range metrics {
		// keep same oid order in output
		if _, ok := grouped[m.Oid]; ok {
			res = append(res, grouped[m.Oid])
			delete(grouped, m.Oid)
		}
	}
	return res
}
