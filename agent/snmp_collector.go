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
	errCount := PromSample{
		Name:   "snmp_poll_error_count",
		Desc:   "snmp poll errors",
		Stamp:  pollRes.stamp,
		Labels: make(map[string]string),
		Value:  float64(0),
	}
	if pollRes.pollErr != nil {
		errCount.Value = 1
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
		errCount.Labels[k] = v
		dur.Labels[k] = v
		mcount.Labels[k] = v
	}
	c.promSamples <- &errCount
	c.promSamples <- &dur
	c.promSamples <- &mcount

	for _, scalar := range pollRes.Scalar {
		for _, res := range scalar.Results {
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
				if res.AsLabel {
					resAsLabels[res.Name] = fmt.Sprintf("%v", res.Value)
				}
			}
			for _, res := range indexedRes {
				if res.AsLabel {
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
