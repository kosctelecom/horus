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

package agent

import (
	"fmt"
)

// SnmpCollector is a prometheus collector for snmp datas.
type SnmpCollector struct {
	*PromCollector
}

// Push convert a poll result to prometheus samples and push them to the sample queue.
func (c *SnmpCollector) Push(pollRes *PollResult) {
	pollTimeout := PromSample{
		Name:   "snmp_poll_timeout_count",
		Desc:   "current snmp poll failed due to timeout",
		Stamp:  pollRes.stamp,
		Labels: make(map[string]string),
		Value:  float64(0),
	}
	if ErrIsTimeout(pollRes.pollErr) {
		pollTimeout.Value = 1
	}
	pollRefused := PromSample{
		Name:   "snmp_poll_refused_count",
		Desc:   "current snmp poll failed due to connection refused",
		Stamp:  pollRes.stamp,
		Labels: make(map[string]string),
		Value:  float64(0),
	}
	if ErrIsRefused(pollRes.pollErr) {
		pollRefused.Value = 1
	}
	dur := PromSample{
		Name:   "snmp_poll_duration_ms",
		Desc:   "snmp polling duration",
		Stamp:  pollRes.stamp,
		Labels: make(map[string]string),
		Value:  float64(pollRes.Duration),
	}
	mcount := PromSample{
		Name:   "snmp_poll_metric_count",
		Desc:   "number of snmp metrics in poll result",
		Stamp:  pollRes.stamp,
		Labels: make(map[string]string),
		Value:  float64(pollRes.metricCount),
	}
	for k, v := range pollRes.Tags {
		pollTimeout.Labels[k] = v
		pollRefused.Labels[k] = v
		dur.Labels[k] = v
		mcount.Labels[k] = v
	}
	c.promSamples <- &pollTimeout
	c.promSamples <- &pollRefused
	c.promSamples <- &dur
	c.promSamples <- &mcount

	for _, scalar := range pollRes.Scalar {
		for _, res := range scalar.Results {
			if !res.toProm {
				continue
			}
			var sample PromSample
			if res.AsLabel {
				sample = PromSample{
					Name:   scalar.Name + "_" + res.Name,
					Value:  1,
					Stamp:  pollRes.stamp,
					Labels: make(map[string]string),
				}
				sample.Labels[res.Name] = fmt.Sprintf("%v", res.Value)
			} else {
				var value float64
				switch v := res.Value.(type) {
				case float64:
					value = v
				case int64:
					value = float64(v)
				case int:
					value = float64(v)
				case uint:
					value = float64(v)
				case bool:
					if v {
						value = 1
					}
				default:
					continue
				}
				sample = PromSample{
					Name:   scalar.Name + "_" + res.Name,
					Value:  value,
					Stamp:  pollRes.stamp,
					Labels: make(map[string]string),
				}
			}
			for k, v := range pollRes.Tags {
				sample.Labels[k] = v
			}
			sample.Labels["oid"] = res.Oid
			c.promSamples <- &sample
		}
	}

	for _, indexed := range pollRes.Indexed {
		for _, indexedRes := range indexed.Results {
			resAsLabels := make(map[string]string)
			for _, res := range indexedRes {
				if res.toProm && res.AsLabel {
					resAsLabels[res.Name] = fmt.Sprintf("%v", res.Value)
				}
			}
			for _, res := range indexedRes {
				if !res.toProm || res.AsLabel {
					continue
				}
				labels := make(map[string]string)
				for k, v := range pollRes.Tags {
					labels[k] = v
				}
				for k, v := range resAsLabels {
					labels[k] = v
				}
				var value float64
				switch v := res.Value.(type) {
				case float64:
					value = v
				case int64:
					value = float64(v)
				case int:
					value = float64(v)
				case uint:
					value = float64(v)
				case bool:
					value = 0.0
					if v {
						value = 1.0
					}
				default:
					continue
				}
				labels["oid"] = res.Oid
				sample := PromSample{
					Name:   indexed.Name + "_" + res.Name,
					Value:  value,
					Stamp:  pollRes.stamp,
					Labels: labels,
				}
				c.promSamples <- &sample
			}
		}
	}
}

// SnmpScrapeCount returns the number of prometheus snmp scrapes.
// Returns 0 if the collector is not initialized.
func SnmpScrapeCount() int {
	if snmpCollector == nil {
		return 0
	}
	return snmpCollector.scrapeCount
}
