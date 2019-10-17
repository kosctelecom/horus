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
	"hash/fnv"
	"horus/log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PromSample is a prometheus metric.
type PromSample struct {
	// Name is the prometheus metric name in the form of <snmp measurement name>_<snmp metric name>.
	Name string

	// Desc is the metric description (usually the snmp oid).
	Desc string

	// Value is the metric value.
	Value float64

	// Labels is the metric label map.
	Labels map[string]string

	// Stamp is the metric timestamp (the snmp poll start time).
	Stamp time.Time
}

// PromCollector represents a prometheus collector
type PromCollector struct {
	// Samples is the map of last samples kept in memory.
	Samples map[uint64]*PromSample

	// MaxResultAge is the max time a sample is kept in memory.
	MaxResultAge time.Duration

	// SweepFreq is the cleanup goroutine frequency to remove old metrics.
	SweepFreq time.Duration

	scrapeCount    int
	scrapeDuration time.Duration
	promSamples    chan *PromSample
	sync.Mutex
}

var (
	workersCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agent_worker_count",
		Help: "Number of max workers for this agent.",
	})
	currSampleCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agent_sample_count",
		Help: "Number of prom samples currently in memory of the agent.",
	})
	ongoingPollCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agent_snmp_poll_count",
		Help: "Number of currently ongoing snmp polls on this agent.",
	})
	heapMem = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agent_heap_memory_bytes",
		Help: "Heap memory usage for this agent.",
	})
	sysMem = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agent_system_memory_bytes",
		Help: "System memory for this agent.",
	})
	snmpScrapes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agent_snmp_scrape_total",
		Help: "Number of total prometheus snmp scrapes count.",
	})
	snmpScrapeDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "agent_snmp_scrape_duration_seconds",
		Help: "snmp scrape duration.",
	})
)

var (
	snmpCollector *SnmpCollector
	pingCollector *PingCollector
)

// InitCollectors initializes the snmp and ping collectors with retention time and cleanup frequency.
// We have three collectors:
// - /metrics for internal poll related metrics
// - /snmpmetrics for snmp polling results
// - /pingmetrics for ping results
func InitCollectors(maxResAge, sweepFreq int) error {
	workersCount.Set(float64(MaxRequests))
	sysMem.Set(float64(sysTotalMemory()))
	prometheus.MustRegister(currSampleCount)
	prometheus.MustRegister(ongoingPollCount)
	prometheus.MustRegister(workersCount)
	prometheus.MustRegister(heapMem)
	prometheus.MustRegister(sysMem)
	http.Handle("/metrics", promhttp.Handler())

	snmpColl, err := NewCollector(maxResAge, sweepFreq, "/snmpmetrics")
	if err != nil {
		return fmt.Errorf("snmp collector: %v", err)
	}
	snmpCollector = &SnmpCollector{PromCollector: snmpColl}
	prometheus.MustRegister(snmpScrapes)
	prometheus.MustRegister(snmpScrapeDuration)

	pingColl, err := NewCollector(maxResAge, sweepFreq, "/pingmetrics")
	if err != nil {
		return fmt.Errorf("ping collector: %v", err)
	}
	pingCollector = &PingCollector{PromCollector: pingColl}
	return nil
}

// NewCollector creates a new prometheus collector
func NewCollector(maxResAge, sweepFreq int, endpoint string) (*PromCollector, error) {
	if maxResAge <= 0 || sweepFreq <= 0 {
		return nil, fmt.Errorf("max_result_age or sweep_frequency must be set")
	}

	collector := &PromCollector{
		Samples:      make(map[uint64]*PromSample),
		MaxResultAge: time.Duration(maxResAge) * time.Second,
		SweepFreq:    time.Duration(sweepFreq) * time.Second,
		promSamples:  make(chan *PromSample),
	}
	http.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()
		registry.MustRegister(collector)
		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})
	go collector.processSamples()
	return collector, nil
}

// processSamples processes each new sample popped from the channel
// and regularly cleans older results
func (c *PromCollector) processSamples() {
	sweepTick := time.NewTicker(c.SweepFreq).C
	for {
		select {
		case s := <-c.promSamples:
			id := computeKey(*s)
			c.Lock()
			c.Samples[id] = s
			c.Unlock()
		case <-sweepTick:
			minStamp := time.Now().Add(-c.MaxResultAge)
			c.Lock()
			var outdatedCount int
			for id, res := range c.Samples {
				if res.Stamp.Before(minStamp) {
					delete(c.Samples, id)
					outdatedCount++
				}
			}
			if len(c.Samples) == 0 {
				// recreate map to solve go mem leak issue (https://github.com/golang/go/issues/20135)
				c.Samples = make(map[uint64]*PromSample)
			}
			c.Unlock()
			log.Debugf("%d prom samples after cleanup, %d outdated samples deleted", len(c.Samples), outdatedCount)
		}
	}
}

// Describe implements Prometheus.Collector
func (c *PromCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

// Collect implements Prometheus.Collector
func (c *PromCollector) Collect(ch chan<- prometheus.Metric) {
	samples := make([]*PromSample, 0, len(c.Samples))
	start := time.Now()
	c.Lock()
	for _, s := range c.Samples {
		// make current samples copy first
		samples = append(samples, s)
	}
	c.Unlock()

	for _, sample := range samples {
		id := sample.Labels["id"]
		log.Debug3f("scraping sample %s@%s[%s] ts=%d (%s)", sample.Name, id, sample.Labels["ifName"],
			sample.Stamp.Unix(), sample.Stamp.Format(time.RFC3339))
		desc := prometheus.NewDesc(sample.Name, sample.Desc, nil, sample.Labels)
		metr, err := prometheus.NewConstMetric(desc, prometheus.UntypedValue, sample.Value)
		if err != nil {
			log.Errorf("collect: NewConstMetric: %v (sample: %+v)", err, sample)
			continue
		}
		ch <- prometheus.NewMetricWithTimestamp(sample.Stamp, metr)
	}
	log.Debugf("scrape done in %dms (%d samples)", time.Since(start)/time.Millisecond, len(samples))
	c.scrapeCount++
	c.scrapeDuration = time.Since(start)
}

// computeKey calculates a consistent hash for the sample. It is used as the
// samples map key instead of the `sid` string for memory efficiency.
func computeKey(sample PromSample) uint64 {
	lnames := make([]string, 0, len(sample.Labels))
	for k := range sample.Labels {
		lnames = append(lnames, k)
	}
	sort.Strings(lnames)
	sid := sample.Name
	for _, label := range lnames {
		sid += label + sample.Labels[label]
	}
	h := fnv.New64a()
	h.Write([]byte(sid))
	return h.Sum64()
}
