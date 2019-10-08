package model

import "testing"

func TestUnmarshalMetric(t *testing.T) {
	tests := []struct {
		in            string
		valid         bool
		indexRegexNil bool
	}{
		{`{"Name": "ifName", "Oid":".1.3.6.1.2.1.31.1.1.1.1", "IndexPattern":""}`, true, true},
		{`{"Name":"sdslDowntreamAttenuation", "Oid":".1.3.6.1.2.1.10.48.1.5.1.1", "IndexPattern":".1.3.6.1.2.1.10.48.1.5.1.1.(\\d+).1.2.\\d"}`, true, false},
		{`{"Name":"sdslUpstreamMargin", "Oid":".1.3.6.1.2.1.10.48.1.5.1.2", "IndexPattern":"\\.1\\.3\\.6\\.1.2.1.10.48.1.5.1.2.(\\d+).1.2.\\d"}`, true, false},
		{`{"Name":"sdslUpstreamAttenuation", "Oid":".1.3.6.1.2.1.10.48.1.5.1.1", "IndexPattern":".1.3.6.1.2.1.10.48.1.5.1.1.\\d+.2.1.\\d"}`, false, false},
	}
	for i, tt := range tests {
		var m Metric
		err := m.UnmarshalJSON([]byte(tt.in))
		valid := (err == nil)
		if tt.valid != valid {
			t.Fatalf("UnmarshalJSON metric[%d] (`%s`), valid? expected %v, got %v (err: %v)", i, tt.in, tt.valid, valid, err)
		}
		indexRegexNil := (m.IndexRegex == nil)
		if valid && tt.indexRegexNil != indexRegexNil {
			t.Fatalf("metric[%d]: is IndexRegex nil? expected: %v, got: %v (pattern=`%s`)", i, tt.indexRegexNil, indexRegexNil, m.IndexPattern)
		}
		t.Logf("metric[%d] (`%s`): IndexRegex=%v", i, tt.in, m.IndexRegex)
	}
}

func TestGroupByOid(t *testing.T) {
	jsonMetrics := []string{
		`{"name": "ifName", "oid":".1.3.6.1.2.1.31.1.1.1.1", "index_pattern":""}`,
		`{"name":"sdslDowntreamAttenuation", "oid":".1.3.6.1.2.1.10.48.1.5.1.1", "index_pattern":".1.3.6.1.2.1.10.48.1.5.1.1.(\\d+).1.2.\\d"}`,
		`{"name":"sdslUpstreamMargin", "oid":".1.3.6.1.2.1.10.48.1.5.1.2", "index_pattern":"\\.1\\.3\\.6\\.1.2.1.10.48.1.5.1.2.(\\d+).1.2.\\d"}`,
		`{"name":"sdslUpstreamAttenuation", "oid":".1.3.6.1.2.1.10.48.1.5.1.1", "index_pattern":".1.3.6.1.2.1.10.48.1.5.1.1.(\\d+).2.1.\\d"}`,
	}
	metrics := make([]Metric, len(jsonMetrics))
	for i, jm := range jsonMetrics {
		var m Metric
		if err := m.UnmarshalJSON([]byte(jm)); err != nil {
			t.Fatalf("ERR: UnmarshalJSON metric[%d]: %v", i, err)
		}
		metrics[i] = m
	}
	grouped := GroupByOid(metrics)
	t.Logf("grouped: %+v", grouped)
	if len(grouped) != 3 {
		t.Fatalf("GroupByOid: expected 3 entries, got %d", len(grouped))
	}
}
