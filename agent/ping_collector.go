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

import (
	"strconv"

	"github.com/kosctelecom/horus/log"
)

// PingCollector is a prometheus collector
type PingCollector struct {
	*PromCollector
}

// Push converts a ping measure to prometheus samples and pushes them to the sample queue.
func (c *PingCollector) Push(meas PingMeasure) {
	if c == nil {
		log.Debugf("Push called on nil pingcollector")
		return
	}

	log.Debug2f(">> posting ping measures for %s at %v", meas.IPAddr, meas.Stamp)
	pingMin := PromSample{
		Name:  "ping_min_duration_seconds",
		Desc:  "min ping RTT time on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"id":         strconv.Itoa(meas.HostID),
			"host":       meas.Hostname,
			"ip_address": meas.IPAddr,
			"category":   meas.Category,
			"vendor":     meas.Vendor,
			"model":      meas.Model,
		},
		Value: meas.Min,
	}
	c.promSamples <- &pingMin

	pingMax := PromSample{
		Name:  "ping_max_duration_seconds",
		Desc:  "max ping RTT time on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"id":         strconv.Itoa(meas.HostID),
			"host":       meas.Hostname,
			"ip_address": meas.IPAddr,
			"category":   meas.Category,
			"vendor":     meas.Vendor,
			"model":      meas.Model,
		},
		Value: meas.Max,
	}
	c.promSamples <- &pingMax

	pingAvg := PromSample{
		Name:  "ping_avg_duration_seconds",
		Desc:  "average ping RTT time on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"id":         strconv.Itoa(meas.HostID),
			"host":       meas.Hostname,
			"ip_address": meas.IPAddr,
			"category":   meas.Category,
			"vendor":     meas.Vendor,
			"model":      meas.Model,
		},
		Value: meas.Avg,
	}
	c.promSamples <- &pingAvg

	pingLoss := PromSample{
		Name:  "ping_loss_ratio",
		Desc:  "ping packet loss ratio on this measure",
		Stamp: meas.Stamp,
		Labels: map[string]string{
			"id":         strconv.Itoa(meas.HostID),
			"host":       meas.Hostname,
			"ip_address": meas.IPAddr,
			"category":   meas.Category,
			"vendor":     meas.Vendor,
			"model":      meas.Model,
		},
		Value: meas.Loss,
	}
	c.promSamples <- &pingLoss
}
