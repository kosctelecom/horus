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
	"context"
	"errors"
	"fmt"
	"horus/log"
	"horus/model"
	"strings"
	"sync"
	"time"

	"github.com/vma/gosnmp"
)

// SnmpRequest is a model.SnmpRequest with a snmp connection handler and a logger.
type SnmpRequest struct {
	model.SnmpRequest

	// resultCache is a cache for walk results to avoid rewalking
	// the same oids (of same community). Only unique base oids are cached.
	resultCache   map[string]TabularResults
	resultCacheMu sync.RWMutex

	// snmpClis is an array of gosnmp connections
	snmpClis []*gosnmp.GoSNMP

	// logger is the internal gosnmp compatible glog Logger.
	log.Logger
}

type snmpgetResult struct {
	metric model.Metric
	pkt    *gosnmp.SnmpPacket
	err    error
}

type snmpwalkResult struct {
	oid model.OID
	tab TabularResults
	err error
}

func (s *SnmpRequest) UnmarshalJSON(data []byte) error {
	var r model.SnmpRequest

	err := r.UnmarshalJSON(data)
	if err != nil {
		return err
	}
	s.SnmpRequest = r
	s.Logger = log.WithPrefix(s.UID)

	var secParams gosnmp.UsmSecurityParameters
	var msgFlag gosnmp.SnmpV3MsgFlags
	var authProto gosnmp.SnmpV3AuthProtocol
	var privProto gosnmp.SnmpV3PrivProtocol
	if s.Device.Version == model.Version3 {
		switch s.Device.AuthProto {
		case "MD5":
			authProto = gosnmp.MD5
		case "SHA":
			authProto = gosnmp.SHA
		default:
			authProto = gosnmp.NoAuth
		}
		switch s.Device.PrivProto {
		case "DES":
			privProto = gosnmp.DES
		case "AES":
			privProto = gosnmp.AES
		default:
			privProto = gosnmp.NoPriv
		}
		switch s.Device.SecLevel {
		case "NoAuthNoPriv":
			msgFlag = gosnmp.NoAuthNoPriv
			secParams = gosnmp.UsmSecurityParameters{
				UserName:               s.Device.AuthUser,
				AuthenticationProtocol: gosnmp.NoAuth,
				PrivacyProtocol:        gosnmp.NoPriv,
			}
		case "AuthNoPriv":
			msgFlag = gosnmp.AuthNoPriv
			secParams = gosnmp.UsmSecurityParameters{
				UserName:                 s.Device.AuthUser,
				AuthenticationProtocol:   authProto,
				AuthenticationPassphrase: s.Device.AuthPasswd,
				PrivacyProtocol:          gosnmp.NoPriv,
			}
		case "AuthPriv":
			msgFlag = gosnmp.AuthPriv
			secParams = gosnmp.UsmSecurityParameters{
				UserName:                 s.Device.AuthUser,
				AuthenticationProtocol:   authProto,
				AuthenticationPassphrase: s.Device.AuthPasswd,
				PrivacyProtocol:          privProto,
				PrivacyPassphrase:        s.Device.PrivPasswd,
			}
		default:
			return errors.New("invalid snmpv3 security level")
		}
	}

	s.resultCache = make(map[string]TabularResults)
	snmpParams := s.Device.SnmpParams
	s.snmpClis = make([]*gosnmp.GoSNMP, snmpParams.ConnectionCount)
	for i := 0; i < snmpParams.ConnectionCount; i++ {
		cli := &gosnmp.GoSNMP{
			Target:    snmpParams.IPAddress,
			Port:      uint16(snmpParams.Port),
			Community: snmpParams.Community,
			Version:   snmpParams.GoSnmpVersion(),
			Timeout:   time.Duration(snmpParams.Timeout) * time.Second,
			Retries:   snmpParams.Retries,
			Logger:    s.Logger,
		}
		if snmpParams.Version == model.Version3 {
			cli.SecurityModel = gosnmp.UserSecurityModel
			cli.MsgFlags = msgFlag
			cli.SecurityParameters = &secParams
		}
		s.snmpClis[i] = cli
	}

	var allMetrics []model.Metric
	for _, scalar := range s.ScalarMeasures {
		allMetrics = append(allMetrics, scalar.Metrics...)
	}
	for i, indexed := range s.IndexedMeasures {
		indexed.RemoveInactive()
		s.IndexedMeasures[i] = indexed
		allMetrics = append(allMetrics, indexed.Metrics...)
	}
	s.Debugf(2, "requested metrics: %v", model.Names(allMetrics))
	return nil
}

// Dial opens all the needed snmp connections to the device.
func (r *SnmpRequest) Dial(ctx context.Context) error {
	var wg sync.WaitGroup
	errs := make(chan error, len(r.snmpClis))
	for i, cli := range r.snmpClis {
		r.Debugf(2, "dial: initiating conn #%d", i)
		wg.Add(1)
		go func(i int, cli *gosnmp.GoSNMP) {
			defer wg.Done()
			if err := cli.DialWithCtx(ctx); err != nil {
				r.Warningf("dial: snmp cli #%d: %v", i, err)
				errs <- err
			}
		}(i, cli)
	}
	wg.Wait()
	r.Debug(2, "dial: done with all connections")
	if len(errs) == len(r.snmpClis) {
		return fmt.Errorf("dial: unable to get any snmp conn: %v", <-errs)
	}
	return nil
}

// Close closes all the opened snmp connections.
func (r *SnmpRequest) Close() {
	r.Debugf(2, "closing all snmp cons...")
	for _, cli := range r.snmpClis {
		cli.Conn.Close()
	}
}

// Get fetches all the scalar measures results.
// Returns the last non-nil error from gosnmp.
func (r *SnmpRequest) Get(ctx context.Context) (results []ScalarResults, err error) {
	for _, scalar := range r.ScalarMeasures {
		r.Debugf(1, "polling scalar measure %s", scalar.Name)
		var res []Result
		res, err = r.getMeasure(ctx, scalar)
		if err != nil {
			if ErrIsUnreachable(err) {
				r.Errorf("Get %s: device unreachable (%v), stopping poll", scalar.Name, err)
				return
			}
			r.Warningf("Get %s: %v, skipping result", scalar.Name, err)
			continue
		}
		sres := ScalarResults{
			Name:     scalar.Name,
			Results:  res,
			ToProm:   scalar.ToProm,
			ToKafka:  scalar.ToKafka,
			ToInflux: scalar.ToInflux,
		}
		results = append(results, sres)
	}
	return
}

// getMeasure gets a scalar measure using all available connections simultaneously.
// Each oid is fetched in a separate gosnmp Get call to avoid cascading errors.
// If one of the Get call results in an error, the last non-nil error is returned.
func (r *SnmpRequest) getMeasure(ctx context.Context, meas model.ScalarMeasure) ([]Result, error) {
	metrics := make(chan model.Metric, len(meas.Metrics))
	defer close(metrics) // needed for async range loop below
	for _, metric := range meas.Metrics {
		metrics <- metric
	}

	snmpResults := make(chan snmpgetResult)
	for i, cli := range r.snmpClis {
		go func(i int, cli *gosnmp.GoSNMP) {
			for metric := range metrics {
				if meas.UseAlternateCommunity && r.Device.AlternateCommunity != "" {
					cli.Community = r.Device.AlternateCommunity
				} else {
					cli.Community = r.Device.Community
				}
				oid := string(metric.Oid)
				r.Debugf(1, "con#%d: getting scalar oid %s (%s)", i, oid, metric.Name)
				pkt, err := cli.GetWithCtx(ctx, []string{oid})
				r.Debugf(2, "con#%d oid %s: got snmp reply, pushing...", i, oid)
				r.Debugf(3, ">> pkt=%+v, err=%v", pkt, err)
				snmpResults <- snmpgetResult{metric, pkt, err}
				r.Debugf(2, "con#%d oid %s: pushed", i, oid)
				if ErrIsUnreachable(err) {
					// device unrechable, do not continue
					break
				}
			}
			r.Debugf(3, "con#%d: measure %s: metric loop terminated", i, meas.Name)
		}(i, cli)
	}

	var results []Result
	var snmpErr error
	for range meas.Metrics {
		snmpres := <-snmpResults // we cannot range over snmpResults as it is never closed
		metric := snmpres.metric
		if snmpres.err != nil {
			snmpErr = fmt.Errorf("get %s: %v", metric.Name, snmpres.err)
		}
		if ErrIsUnreachable(snmpres.err) {
			// escape from the blocking chan read
			break
		}
		if snmpres.pkt == nil {
			continue
		}
		for _, pdu := range snmpres.pkt.Variables {
			r.Debugf(2, "pdu = %#v", pdu)
			res, err := MakeResult(pdu, metric)
			if err != nil {
				r.Warningf("get %s: make result: %v", metric.Name, err)
				continue
			}
			results = append(results, res)
		}
	}
	return results, snmpErr
}

// walkMetric walks an oid and puts the resulting metrics in a TabularResults.
// The grouped parameter is a list of metrics with the same base oid but different
// suffix or index position (typically for composite index.)
// If a cached result is available for this request, the Oid is not requested again.
// The results of single-metric requests are put in local cache map.
// All metrics with the same base oid but different index position are extracted at once.
func (r *SnmpRequest) walkMetric(ctx context.Context, grouped []model.Metric, conIdx int, useAltCommunity bool) (TabularResults, error) {
	oid := grouped[0].Oid
	r.resultCacheMu.RLock()
	cached, ok := r.resultCache[oid.CacheKey(useAltCommunity)]
	r.resultCacheMu.RUnlock()
	if ok {
		r.Debugf(1, "con#%d: returning cached res map for oid %s", conIdx, oid)
		return cached, nil
	}

	tabResult := make(TabularResults)
	pduWalker := func(pdu gosnmp.SnmpPDU) error {
		if len(pdu.Name) < len(oid) {
			return fmt.Errorf("child oid (%s) smaller than base oid (%s)", pdu.Name, oid)
		}
		if pdu.Value != nil {
			for _, metric := range grouped {
				res, err := MakeResult(pdu, metric)
				if err != nil {
					r.Warningf("walk %s: make result: %v", metric.Name, err)
					continue
				}
				idx := pdu.Name[len(oid)+1:]
				if metric.IndexRegex != nil {
					submatches := metric.IndexRegex.FindStringSubmatch(pdu.Name)
					if len(submatches) < 2 {
						// no match, skip
						continue
					}
					idx = strings.Join(submatches[1:], ".") // starts at 1 to skip the entire oid match
					r.Debugf(3, "con#%d: %s - idx `%s` extracted from oid %s", conIdx, metric.Name, idx, pdu.Name)
				}
				res.Index = idx
				tabResult[idx] = append(tabResult[idx], res)
			}
		}
		return nil
	}
	r.Debugf(2, "con#%d: walking indexed metric %s, alternate community: %v", conIdx, oid, useAltCommunity)
	cli := r.snmpClis[conIdx]
	if useAltCommunity && r.Device.AlternateCommunity != "" {
		cli.Community = r.Device.AlternateCommunity
	} else {
		cli.Community = r.Device.Community
	}
	var err error
	if r.Device.Version == model.Version1 || r.Device.DisableBulk {
		err = cli.WalkWithCtx(ctx, string(oid), pduWalker)
	} else {
		err = cli.BulkWalkWithCtx(ctx, string(oid), pduWalker)
	}
	if err != nil {
		return tabResult, fmt.Errorf("Walk: %v", err)
	}
	r.resultCacheMu.Lock()
	if _, ok := r.resultCache[oid.CacheKey(useAltCommunity)]; !ok && len(grouped) == 1 && grouped[0].IndexRegex == nil {
		// cache only non-grouped metrics with no index-pattern
		r.resultCache[oid.CacheKey(useAltCommunity)] = tabResult
	}
	r.resultCacheMu.Unlock()
	r.Debugf(3, "con#%d: res map for group indexed oid %s: %d metrics", conIdx, oid, len(tabResult))
	return tabResult, nil
}

// walkSingleMetric is a simplified walkMetric when there is only one metric and
// one snmp connection and no alternate community.
func (r *SnmpRequest) walkSingleMetric(ctx context.Context, metr model.Metric) (TabularResults, error) {
	return r.walkMetric(ctx, []model.Metric{metr}, 0, false)
}

// walkMeasure queries an indexed measure and returns the corresponding indexed results.
// Makes multiple parallel snmp queries and gathers the results at the end.
// If one or more of the walk requests resulted in an error, the last one is returned.
func (r *SnmpRequest) walkMeasure(ctx context.Context, measure model.IndexedMeasure) (IndexedResults, error) {
	var tabResults []TabularResults
	r.Debugf(1, "getting indexed measure %s", measure.Name)
	if len(measure.Metrics) == 0 {
		r.Errorf("walk indexed: measure %s: metric list empty", measure.Name)
		return IndexedResults{}, nil
	}

	byOid := model.GroupByOid(measure.Metrics)
	groupedMetrics := make(chan []model.Metric, len(byOid))
	defer close(groupedMetrics)
	walkResults := make(chan snmpwalkResult)
	for _, grouped := range byOid {
		groupedMetrics <- grouped
	}
	r.Debugf(2, "grouped metric count: %d", len(groupedMetrics))
	for conIdx := range r.snmpClis {
		go func(conIdx int) {
			for grouped := range groupedMetrics {
				oid, name := grouped[0].Oid, grouped[0].Name
				start := time.Now()
				r.Debugf(2, "con#%d: start walking indexed oid %s [%s], %d metric(s)", conIdx, oid, name, len(grouped))
				groupedRes, err := r.walkMetric(ctx, grouped, conIdx, measure.UseAlternateCommunity)
				walkResults <- snmpwalkResult{oid, groupedRes, err}
				r.Debugf(1, "con#%d: done walking indexed oid %s [%s]: took %v", conIdx, oid, name, time.Since(start).Truncate(time.Millisecond))
			}
			r.Debugf(2, "con#%d: measure %s: oid loop terminated", conIdx, measure.Name)
		}(conIdx)
	}

	var walkErr error
	for i, grouped := range byOid {
		res := <-walkResults
		if res.err != nil {
			walkErr = fmt.Errorf("walk oid %s: %v", res.oid, res.err)
			continue
		}
		if len(res.tab) > 0 {
			tabResults = append(tabResults, res.tab)
		} else {
			r.Debugf(2, "walkMetric %s: skipping empty tabular result", res.oid)
		}
		if measure.IndexMetricID.Valid && int64(grouped[0].ID) == measure.IndexMetricID.Int64 {
			// recompute index result position on tabResults
			measure.IndexPos = i
		}
	}
	indexed := MakeIndexed(r.UID, measure, tabResults)
	r.Debugf(2, "walkMeasure: full index results count: %d", len(indexed.Results))
	indexed.Filter(measure)
	r.Debugf(2, "walkMeasure: filtered index results count: %d", len(indexed.Results))
	return indexed, walkErr
}

// Walk polls all the indexed measures and returns an array of IndexedResults
// in the same order as each indexed measure.
// On error, a partial result is still returned.
func (r *SnmpRequest) Walk(ctx context.Context) ([]IndexedResults, error) {
	var results []IndexedResults
	var err error

	for _, meas := range r.IndexedMeasures {
		var indexed IndexedResults
		indexed, err = r.walkMeasure(ctx, meas)
		if err != nil {
			r.Errorf("Walk %s: %v", meas.Name, err)
		}
		if len(indexed.Results) == 0 {
			r.Debugf(2, "skipping indexed measure %s with no result", meas.Name)
			continue
		}
		results = append(results, indexed)
	}
	return results, err
}

// Poll queries all metrics of the request and returns them in a PollResult.
// If there was a timeout while getting scalar results, we stop there, there is
// no Walk attempted to get the indexed results.
func (r *SnmpRequest) Poll(ctx context.Context) PollResult {
	res := r.MakePollResult()
	res.Scalar, res.pollErr = r.Get(ctx)
	if ErrIsUnreachable(res.pollErr) {
		res.PollErr = res.pollErr.Error()
		res.Duration = int64(time.Since(res.PollStart) / time.Millisecond)
		r.Warningf("poll: %v", res.pollErr)
		res.IsPartial = len(res.Scalar) > 0
		return res
	}
	res.Indexed, res.pollErr = r.Walk(ctx)
	res.Duration = int64(time.Since(res.PollStart) / time.Millisecond)
	if res.pollErr != nil {
		r.Warningf("poll: %v", res.pollErr)
		res.PollErr = res.pollErr.Error()
		res.IsPartial = len(res.Scalar)+len(res.Indexed) > 0
	}
	return res
}

// ErrIsTimeout tells whether the error is an snmp timeout error.
func ErrIsTimeout(err error) bool {
	return err != nil && strings.Contains(err.Error(), "timeout")
}

// ErrIsRefused tells whether the error is an snmp connection refused error.
func ErrIsRefused(err error) bool {
	return err != nil && strings.Contains(err.Error(), "connection refused")
}

// ErrIsUnreachable tells whether the error is an snmp timeout or connection refused.
func ErrIsUnreachable(err error) bool {
	return ErrIsTimeout(err) || ErrIsRefused(err)
}
