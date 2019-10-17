package agent

import (
	"context"
	"horus/model"
)

// maxGetOids is the max number of oids to query in a single snmp GET.
// It is lower than gosnmp's maxOids.
const maxGetOids = 50

var (
	ifOperStatus = model.Metric{
		Name:   "ifOperStatus",
		Oid:    ".1.3.6.1.2.1.2.2.1.8",
		Active: true,
	}
)

// walkRunningMeasure retrieves given indexed measure for operational ports only.
// It retrieves first the ifOperStatus of all ports and for all metrics specified for
// running interfaces only, does a snmpget on the oid+index. And for all other metrics,
// does a walk as usual.
func (r *SnmpRequest) walkRunningMeasure(ctx context.Context, meas model.IndexedMeasure) (IndexedResults, error) {
	operStatus, err := r.walkSingleMetric(ctx, ifOperStatus)
	if err != nil {
		r.Errorf("walk running meas: ifOperStatus: %v", err)
		return IndexedResults{}, err
	}
	var upIndexes []string
	for idx, pdu := range operStatus {
		if pdu[0].Value == 1 {
			// iface is up
			upIndexes = append(upIndexes, idx)
		}
	}

	var tabRes []TabularResults
	r.Debugf(2, ">> walk running meas %s", meas.Name)
	for _, metric := range meas.Metrics {
		var err error
		var res TabularResults
		r.Debugf(2, ">>> polling metric %s", metric.Name)
		switch {
		case metric.Oid == ifOperStatus.Oid:
			res = operStatus
		case metric.RunningIfaceOnly && !metric.ExportAsLabel:
			res, err = r.walkRunningMetric(ctx, metric, upIndexes)
		default:
			res, err = r.walkSingleMetric(ctx, metric)
		}
		if err != nil {
			r.Errorf("walk running: %s: %v", metric.Name, err)
			continue
		}
		tabRes = append(tabRes, res)
	}
	return MakeIndexed(r.UID, meas, tabRes), nil
}

// walkRunningMetric is similar to walkMetric but does a snmp get on all the indexed oids of running.
// It is more efficient when the proportion of running ports is small relatively to the total port number.
func (r *SnmpRequest) walkRunningMetric(ctx context.Context, metric model.Metric, indexes []string) (TabularResults, error) {
	walkRes := make(TabularResults)
	r.Debugf(2, "start get running metric %s for %d ifaces", metric.Name, len(indexes))
	var allOids []string // all indexed oids to get
	for _, idx := range indexes {
		allOids = append(allOids, string(metric.Oid)+"."+idx)
	}
	var oidBatches [][]string // batches of oids to query on a single get
	// ref: https://github.com/golang/go/wiki/SliceTricks#batching-with-minimal-allocation
	for len(allOids) > maxGetOids {
		allOids, oidBatches = allOids[maxGetOids:], append(oidBatches, allOids[:maxGetOids:maxGetOids])
	}
	oidBatches = append(oidBatches, allOids)
	for _, oids := range oidBatches {
		r.Debugf(2, ">> snmp get with %d oids", len(oids))
		pkt, err := r.snmpClis[0].GetWithCtx(ctx, oids)
		if err != nil {
			r.Errorf("get running %s: %v", metric.Name, err)
			if isTimeout(err) {
				// no need to continue polling...
				return walkRes, err
			}
			continue
		}
		for _, v := range pkt.Variables {
			res, err := MakeResult(v, metric)
			if err != nil {
				r.Errorf("make result %s: %v", metric.Name, err)
				continue
			}
			walkRes[res.index] = append(walkRes[res.index], res)
		}
	}
	r.Debugf(2, "done get running metric %s", metric.Name)
	return walkRes, nil
}
