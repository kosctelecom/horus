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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kosctelecom/horus/log"
	"github.com/kosctelecom/horus/model"
	"github.com/mitchellh/copystructure"
	"github.com/vma/glog"
	"github.com/vma/gosnmp"
)

// Result represents a single snmp result
type Result struct {
	// Oid is the metric OID as returned by the device.
	Oid string `json:"oid"`

	// Name is the metric name (from SNMP MIB usually).
	Name string `json:"name"`

	// ExportedName is the name of the exported metric.
	ExportedName string `json:"exported_name"`

	// Description is the metric description copied from request.
	Description string `json:"description,omitempty"`

	// Value is the metric value converted to the corresponding Go type.
	Value interface{} `json:"value"`

	// AsLabel tells if the result is exported as a prometheus label.
	AsLabel bool `json:"as_label,omitempty"`

	snmpType gosnmp.Asn1BER
	rawValue interface{}
	suffix   string
}

// TabularResults is a map of Result array containing all values for a given indexed oid.
// The map key is the result index extracted from the result oid: if IndexRegex is not defined,
// its the suffix of the base oid; otherwise, its the concatenation of all parenthesized
// subexpressions extracted from the result oid. For example, if a walk result of `oid` returns
// oid.i1->res1 oid.i1.i12->res11, oid.i1.i13->res12, oid.i2->res2, oid.i3.xxx->res3,...
// with i1,i2... as the index and i12,i13 the sub-index, the corresponding TabularResults is
// {i1=>[res1], i1.i12=>[res11], i1.i13=>[res12], i2=>[res2], i3=>[res3], ...}
type TabularResults map[string][]Result

// ScalarResults is a scalar measure results.
type ScalarResults struct {
	// Name is the name of the result group
	Name string `json:"name"`

	// Results is the list of results of this measure
	Results []Result `json:"metrics"`

	// ToInflux tells wether this measure is exported to influxDB
	ToInflux bool `json:"to_influx,omitempty"`

	// ToKafka tells wether this measure is exported to kafka
	ToKafka bool `json:"to_kafka,omitempty"`

	// ToProm tells wether this measure is exported to prometheus
	ToProm bool `json:"to_prom,omitempty"`
}

// IndexedResults is an indexed measure results.
type IndexedResults struct {
	// Name is the measure name.
	Name string `json:"name"`

	// Results is an 2-dimensional array of all results for this indexed measure
	// with the index as first dimension and the oid as second dimension.
	Results [][]Result `json:"metrics"`

	// ToInflux tells wether this measure is exported to influxDB
	ToInflux bool `json:"to_influx,omitempty"`

	// ToKafka tells wether this measure is exported to kafka
	ToKafka bool `json:"to_kafka,omitempty"`

	// ToProm tells wether this measure is exported to prometheus
	ToProm bool `json:"to_prom,omitempty"`

	// LabelsOnly tells wether the measure is label-only
	LabelsOnly bool `json:"labels_only,omitempty"`
}

// PollResult is the complete result set of a polling job
type PollResult struct {
	// RequestID is the polling job id
	RequestID string `json:"request_id"`

	// AgentID is the poller agent id
	AgentID int `json:"agent_id"`

	// IPAddr is the polled device IP address
	IPAddr string `json:"device_ipaddr"`

	// Scalar is the set of scalar measures results
	Scalar []ScalarResults `json:"scalar_measures,omitempty"`

	// Indexed is the set of indexed measures results
	Indexed []IndexedResults `json:"indexed_measures,omitempty"`

	// PollStart is the poll starting time
	PollStart time.Time `json:"poll_start"`

	// Duration is the total polling duration in ms
	Duration int64 `json:"poll_duration"`

	// PollErr is the error message returned by the poll request
	PollErr string `json:"poll_error,omitempty"`

	// Tags is the tag map associated with the result
	Tags map[string]string `json:"tags,omitempty"`

	// IsPartial tells if the result is partial due to a mid-request snmp timeout.
	IsPartial bool `json:"is_partial,omitempty"`

	stamp       time.Time
	reportURL   string
	metricCount int
	pollErr     error
}

// MakePollResult builds a PollResult from an SnmpRequest.
func MakePollResult(req SnmpRequest) PollResult {
	tags := make(map[string]string)
	tags["id"] = strconv.Itoa(req.Device.ID)
	tags["host"] = req.Device.Hostname
	tags["vendor"] = req.Device.Vendor
	tags["model"] = req.Device.Model
	tags["category"] = req.Device.Category
	if req.Device.Tags != "" {
		var reqTags map[string]interface{}
		if err := json.Unmarshal([]byte(req.Device.Tags), &reqTags); err != nil {
			log.Errorf("json tag unmarshal: %v", err)
		} else {
			for k, v := range reqTags {
				tags[k] = fmt.Sprint(v)
			}
		}
	}
	return PollResult{
		RequestID: req.UID,
		AgentID:   req.AgentID,
		IPAddr:    req.Device.IPAddress,
		PollStart: time.Now(),
		Tags:      tags,
		reportURL: req.ReportURL,
	}
}

func (p PollResult) Copy() PollResult {
	cp, err := copystructure.Copy(p)
	if err != nil {
		log.Errorf("copy PollResult: %v", err)
		return PollResult{}
	}
	cpy := cp.(PollResult)
	cpy.stamp = p.stamp
	cpy.reportURL = p.reportURL
	cpy.metricCount = p.metricCount
	cpy.pollErr = p.pollErr
	return cpy
}

// PruneForKafka prunes PollResult to keep only metrics to be exported to kafka.
func (p *PollResult) PruneForKafka() {
	ps := p.Scalar[:0]
	for _, s := range p.Scalar {
		if s.ToKafka {
			ps = append(ps, s)
		}
	}
	p.Scalar = ps

	pi := p.Indexed[:0]
	for _, indexed := range p.Indexed {
		if indexed.ToKafka {
			pi = append(pi, indexed)
		}
	}
	p.Indexed = pi
}

// MakeResult builds a Result from a gosnmp PDU. The value is casted to its
// corresponding Go type when necessary. In particular, Counter64 values
// are converted to float as influx does not support them out of the box.
// Returns an error on snmp NoSuchObject reply or nil value.
func MakeResult(pdu gosnmp.SnmpPDU, metric model.Metric) (Result, error) {
	res := Result{
		Name:         metric.Name,
		Description:  metric.Description,
		Oid:          string(metric.Oid),
		AsLabel:      metric.ExportAsLabel,
		ExportedName: metric.ExportedName,
		snmpType:     pdu.Type,
		rawValue:     pdu.Value,
	}
	if len(pdu.Name) > len(metric.Oid) {
		res.suffix = pdu.Name[len(metric.Oid)+1:]
	}
	switch pdu.Type {
	case gosnmp.NoSuchObject:
		return res, fmt.Errorf("oid %s: NoSuchObject", pdu.Name)
	case gosnmp.OctetString:
		if len(metric.PostProcessors) == 0 {
			// default processor
			metric.PostProcessors = []string{"trim"}
		}
		res.Value = pdu.Value.([]byte)
		for _, pp := range metric.PostProcessors {
			val := res.Value.([]byte)
			switch pp {
			case "parse-hex-be":
				n, err := bigEndianUint(val)
				if err != nil {
					return res, fmt.Errorf("parse `%+v`: %v", val, err)
				}
				log.Debug3f("%s: parsing `%x` as big endian num => %v", res.Name, string(val), n)
				res.Value = float64(n)
			case "parse-hex-le":
				n, err := littleEndianUint(val)
				if err != nil {
					return res, fmt.Errorf("parse `%+v`: %v", val, err)
				}
				log.Debug3f("%s: parsing `%x` as little endian num => %v", res.Name, string(val), n)
				res.Value = float64(n)
			case "parse-int":
				sv := string(val)
				v, err := strconv.Atoi(sv)
				if err != nil {
					return res, fmt.Errorf("%s: invalid int value %s: %v", res.Name, sv, err)
				}
				res.Value = float64(v)
			case "trim":
				res.Value = strings.TrimSpace(string(val))
			default:
				return res, fmt.Errorf("%s: invalid post-processor %s", res.Name, pp)
			}
		}
	case gosnmp.Counter64:
		// 64 bit counters are automatically wrapped by 2^53 to avoid precision loss due
		// to rounding (https://en.wikipedia.org/wiki/Double-precision_floating-point_format)
		res.Value = float64(gosnmp.ToBigInt(pdu.Value).Uint64() % (1 << 53))
	case gosnmp.OpaqueFloat:
		res.Value = float64(pdu.Value.(float32))
	case gosnmp.OpaqueDouble:
		res.Value = pdu.Value.(float64)
	default:
		res.Value = pdu.Value
	}
	if pdu.Value == nil {
		return res, fmt.Errorf("oid %s: nil value", pdu.Name)
	}
	return res, nil
}

// String returns a string representation of a Result.
func (res Result) String() string {
	if res.Oid == "" {
		return ""
	}
	return fmt.Sprintf("<name:%s oid:%s suffix:%s snmptype:%#x val:%v>", res.Name, res.Oid, res.suffix, res.snmpType, res.Value)
}

// String returns a string representation of an IndexedResults.
func (i IndexedResults) String() string {
	str := i.Name + " = [\n"
	for _, ir := range i.Results {
		str += "  [\n"
		for _, r := range ir {
			str += "  " + r.String() + ",\n"
		}
		str += "  ]\n"
	}
	str += "]\n"
	return str
}

// MakeIndexed builds an indexed results set from a TabularResults array.
// All results at the same key are grouped together.
// Note: tabResults[i] is an array of results for a given oid on all indexes
// and tabResults is a list of these results for all oids.
func MakeIndexed(uid string, meas model.IndexedMeasure, tabResults []TabularResults) IndexedResults {
	indexed := IndexedResults{
		Name:       meas.Name,
		ToKafka:    meas.ToKafka,
		ToProm:     meas.ToProm,
		ToInflux:   meas.ToInflux,
		LabelsOnly: meas.LabelsOnly,
	}
	if len(tabResults) == 0 {
		log.Errorf("%s - makeIndexed: measure %s: result list empty...", uid, meas.Name)
		return indexed
	}
	if meas.IndexPos >= len(tabResults) {
		log.Errorf("%s - makeIndexed: measure %s index #%d bigger than tabResults", uid, meas.Name, meas.IndexPos)
		return indexed
	}
	for index := range tabResults[meas.IndexPos] {
		var results []Result
		for {
			for _, tabRes := range tabResults {
				if metr, ok := tabRes[index]; ok {
					results = append(results, metr...)
				}
			}
			// groups together metrics with composite indexes i.e.
			// oid1.i1 metric will be grouped with oid2.i1.s1 and oid3.i1.s1.s2
			lastDot := strings.LastIndex(index, ".")
			if lastDot <= 0 {
				break
			}
			index = index[:lastDot]
		}
		var labelCount int
		for _, r := range results {
			if r.AsLabel {
				labelCount++
			}
		}
		if len(results) <= 1 || (labelCount == len(results) && !meas.LabelsOnly) {
			// skip empty results, those with index only, and
			// label-only results on non label-only measure
			log.Debug2f(">>> %s - filtering empty or label-only results (%+v) from non label-only measure %s", uid, results, meas.Name)
			continue
		}
		indexed.Results = append(indexed.Results, results)
	}
	return indexed
}

// DedupDesc strips the description field from all entries of an
// indexed result, except the first one.
// This is essential to reduce the size of the json pushed to kafka.
func (indexed *IndexedResults) DedupDesc() {
	found := make(map[string]bool)
	for i, ir := range indexed.Results {
		for j := range ir {
			if _, ok := found[ir[j].Name]; ok {
				indexed.Results[i][j].Description = ""
			} else {
				found[ir[j].Name] = true
			}
		}
	}
}

// Filter filters the indexed result against the regex filter..
func (indexed *IndexedResults) Filter(meas model.IndexedMeasure) {
	if meas.FilterPos == -1 {
		return
	}
	if meas.FilterRegex == nil {
		glog.Errorf("Filter (idx=%d): nil regexp", meas.FilterPos)
		return
	}
	if meas.FilterPos < 0 {
		glog.Error("Filter: invalid index with non-nil filter")
		return
	}
	filtered := indexed.Results[:0]
	for _, ir := range indexed.Results {
		val := fmt.Sprint(ir[meas.FilterPos].Value)
		match := meas.FilterRegex.MatchString(val)
		if (match && !meas.InvertFilterMatch) || (!match && meas.InvertFilterMatch) {
			filtered = append(filtered, ir)
		}
	}
	indexed.Results = filtered
	if len(filtered) == 0 {
		glog.Warning("Filter: empty indexed result after filtering...")
	}
}

// handlePollResults exports asynchronously each new result
// to each active receiver (influx, kafka or prometheus).
func handlePollResults() {
	for res := range pollResults {
		res.stamp = time.Now()
		ongoingMu.Lock()
		delete(ongoingReqs, res.RequestID)
		ongoingMu.Unlock()
		if res.pollErr != nil {
			log.Debugf("%s - poll failed: %s, partial result? %v", res.RequestID, res.PollErr, res.IsPartial)
		}

		for _, s := range res.Scalar {
			res.metricCount += len(s.Results)
		}
		for _, x := range res.Indexed {
			for _, xr := range x.Results {
				res.metricCount += len(xr)
			}
		}

		go kafkaCli.Push(res.Copy())
		go snmpCollector.Push(res.Copy())
		go influxCli.Push(res.Copy())
		res.sendReport()
	}
}

// sendReport sends the poll report to the url in a get request with the following params
// - request_id: the request id
// - agent_id: the agent db id
// - poll_duration_ms: the snmp polling duration in ms
// - poll_error: the polling error if any
// - current_load: current agent load (current_jobs/total_capacity)
func (p *PollResult) sendReport() {
	log.Debugf("report: id=%s agent_id=%d poll_err=%q poll_dur=%dms metric_count=%d",
		p.RequestID, p.AgentID, p.PollErr, p.Duration, p.metricCount)
	if p.reportURL == "" {
		glog.Warningf("no report url for req %s", p.RequestID)
		return
	}
	req, err := http.NewRequest("GET", p.reportURL, nil)
	if err != nil {
		glog.Errorf("sendReport: %v", err)
		return
	}
	q := req.URL.Query()
	q.Add("request_id", p.RequestID)
	q.Add("agent_id", strconv.Itoa(p.AgentID))
	q.Add("poll_duration_ms", strconv.FormatInt(p.Duration, 10))
	q.Add("poll_error", p.PollErr)
	q.Add("metric_count", strconv.Itoa(p.metricCount))
	q.Add("current_load", fmt.Sprintf("%.4f", CurrentSNMPLoad()))
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 3 * time.Second}
	for i := 0; i < 3; i++ {
		if i > 0 {
			time.Sleep(time.Duration(1<<uint(i-1)) * 3 * time.Second)
		}
		log.Debug2f("%s - posting report, try #%d/3", p.RequestID, i+1)
		resp, err := client.Do(req)
		if err != nil {
			glog.Errorf("send report, try #%d/3: %v", i+1, err)
			continue
		}
		log.Debug2f("%s - report posted at try #%d/3, status: %s", p.RequestID, i+1, resp.Status)
		resp.Body.Close()
		break
	}
}

// bigEndianUint converts byte slice to big-endian int64, taking its size in account.
func bigEndianUint(b []byte) (uint64, error) {
	var res uint64
	switch len(b) {
	case 8:
		res = binary.BigEndian.Uint64(b)
	case 4:
		res = uint64(binary.BigEndian.Uint32(b))
	case 2:
		res = uint64(binary.BigEndian.Uint16(b))
	case 0:
		res = 0
	default:
		return 0, fmt.Errorf("bigEndianUint: invalid slice size %d", len(b))
	}
	return res, nil
}

// littleEndianUint converts byte slice to little-endian int64, taking its size in account.
func littleEndianUint(b []byte) (uint64, error) {
	var res uint64
	switch len(b) {
	case 8:
		res = binary.LittleEndian.Uint64(b)
	case 4:
		res = uint64(binary.LittleEndian.Uint32(b))
	case 2:
		res = uint64(binary.LittleEndian.Uint16(b))
	case 0:
		res = 0
	default:
		return 0, fmt.Errorf("littleEndianUint: invalid slice size %d", len(b))
	}
	return res, nil
}
