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

package agent

import "horus/log"

// PingCollector is a prometheus collector
type PingCollector struct {
	*PromCollector
}

// Push converts a ping measure to prometheus samples and pushes them to the sample queue.
func (c *PingCollector) Push(meas PingMeasure) {
	log.Debug2f(">> posting ping measures for %s at %v", meas.IpAddr, meas.Stamp)
	var hostId string
	if len(meas.Hostname) >= 4 {
		hostId = meas.Hostname[:4]
	}
	ping_min := PromSample{
		Name:  "ping_min_duration_seconds",
		Desc:  "min ping RTT time on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"id":            hostId,
			"host":          meas.Hostname,
			"ip_address":    meas.IpAddr,
			"device_type":   meas.Category,
			"device_vendor": meas.Vendor,
			"device_model":  meas.Model,
		},
		Value: meas.Min,
	}
	c.promSamples <- &ping_min

	ping_max := PromSample{
		Name:  "ping_max_duration_seconds",
		Desc:  "max ping RTT time on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"id":            hostId,
			"host":          meas.Hostname,
			"ip_address":    meas.IpAddr,
			"device_type":   meas.Category,
			"device_vendor": meas.Vendor,
			"device_model":  meas.Model,
		},
		Value: meas.Max,
	}
	c.promSamples <- &ping_max

	ping_avg := PromSample{
		Name:  "ping_avg_duration_seconds",
		Desc:  "average ping RTT time on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"id":            hostId,
			"host":          meas.Hostname,
			"ip_address":    meas.IpAddr,
			"device_type":   meas.Category,
			"device_vendor": meas.Vendor,
			"device_model":  meas.Model,
		},
		Value: meas.Avg,
	}
	c.promSamples <- &ping_avg

	ping_loss := PromSample{
		Name:  "ping_loss_ratio",
		Desc:  "ping packet loss ratio on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"id":            hostId,
			"host":          meas.Hostname,
			"ip_address":    meas.IpAddr,
			"device_type":   meas.Category,
			"device_vendor": meas.Vendor,
			"device_model":  meas.Model,
		},
		Value: meas.Loss,
	}
	c.promSamples <- &ping_loss
}
