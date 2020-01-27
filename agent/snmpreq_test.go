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
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/vma/glog"
)

const Req = `{
	"uid": "001",
	"device": {
		"id": 1001,
		"hostname": "SNMP_IP",
		"polling_frequency": 300,
		"category": "DSLAM",
		"vendor": "HUAWEI",
		"model": "MA5600T",
		"to_kafka": true,
		"ip_address": "SNMP_IP",
		"snmp_version": "2c",
		"snmp_community":"SNMP_COMMUNITY",
		"snmp_timeout": 10,
		"snmp_connection_count": 1
	},
	"ScalarMeasures": [{
		"Name":"sysUsage",
		"Metrics":[
			{"Name":"sysName", "Oid":".1.3.6.1.2.1.1.5.0", "Active":true},
			{"Name":"sysUpTime", "Oid":".1.3.6.1.2.1.1.3.0", "Active":true},
			{"Name":"ifNumber", "Oid":".1.3.6.1.2.1.2.1.0", "Active":true},
			{"Name":"hwEntityMemUsage","Oid":".1.3.6.1.4.1.2011.6.3.17.2.0", "Active":true},
			{"Name":"hwEntityCpuUsage","Oid":".1.3.6.1.4.1.2011.6.3.17.3.0", "Active":true}]
		}
	],
	"IndexedMeasures": [
		{"Name":"ifStatus",
		"Metrics":[
			{"ID":8, "Name":"ifIndex", "Oid":".1.3.6.1.2.1.2.2.1.1", "Active":false},
			{"ID":46, "Name":"ifName", "Oid":".1.3.6.1.2.1.31.1.1.1.1", "Active":true, "AsLabel":true},
			{"ID":10, "Name":"ifOperStatus", "Oid":".1.3.6.1.2.1.2.2.1.8", "Active":true},
			{"ID":11, "Name":"ifAdminStatus", "Oid":".1.3.6.1.2.1.2.2.1.7", "Active":true},
			{"ID":18, "Name":"ifHCInOctets", "Oid":".1.3.6.1.2.1.31.1.1.1.6", "Active":true},
			{"ID":19, "Name":"ifHCOutOctets", "Oid":".1.3.6.1.2.1.31.1.1.1.7", "Active":true},
			{"ID":14, "Name":"ifInErrors", "Oid":".1.3.6.1.2.1.2.2.1.14", "Active":true},
			{"ID":17, "Name":"ifOutErrors", "Oid":".1.3.6.1.2.1.2.2.1.20", "Active":true},
			{"ID":13, "Name":"ifInDiscards", "Oid":".1.3.6.1.2.1.2.2.1.13", "Active":true},
			{"ID":16, "Name":"ifOutDiscards", "Oid":".1.3.6.1.2.1.2.2.1.19", "Active":true}],
		"IndexMetricID": 46}
	]
}`

var (
	snmpIPAddr    = os.Getenv("SNMP_IP")
	snmpCommunity = os.Getenv("SNMP_COMMUNITY")
)

func init() {
	glog.WithConf(glog.Conf{Verbosity: 3})
}

func TestDial(t *testing.T) {
	if snmpIPAddr == "" || snmpCommunity == "" {
		t.Skip("SNMP_IP or SNMP_COMMUNITY env vars not defined, skipping")
	}
	sreq := strings.ReplaceAll(strings.ReplaceAll(Req, "SNMP_IP", snmpIPAddr), "SNMP_COMMUNITY", snmpCommunity)
	var req *SnmpRequest
	if err := json.Unmarshal([]byte(sreq), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := req.Dial(context.Background()); err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer req.Close()
	if testing.Verbose() {
		t.Logf("snmpclis: %+v, first: %+v", req.snmpClis, *req.snmpClis[0])
	}
}

func TestGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	if snmpIPAddr == "" || snmpCommunity == "" {
		t.Skip("SNMP_IP or SNMP_COMMUNITY env vars not defined, skipping")
	}
	sreq := strings.ReplaceAll(strings.ReplaceAll(Req, "SNMP_IP", snmpIPAddr), "SNMP_COMMUNITY", snmpCommunity)
	var req SnmpRequest
	if err := json.Unmarshal([]byte(sreq), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := req.Dial(context.Background()); err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer req.Close()
	res, err := req.Get(context.Background())
	if err != nil {
		t.Fatalf("%v", err)
	}
	if testing.Verbose() {
		t.Logf("req: %+v\n", req)
		t.Logf(">scalar results: %+v", res)
	}
}

func TestWalkMetric(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	if snmpIPAddr == "" || snmpCommunity == "" {
		t.Skip("SNMP_IP or SNMP_COMMUNITY env vars not defined, skipping")
	}
	sreq := strings.ReplaceAll(strings.ReplaceAll(Req, "SNMP_IP", snmpIPAddr), "SNMP_COMMUNITY", snmpCommunity)
	var req SnmpRequest
	if err := json.Unmarshal([]byte(sreq), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := req.Dial(context.Background()); err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer req.Close()
	metr := req.IndexedMeasures[0].Metrics[0]
	res, err := req.walkSingleMetric(context.Background(), metr)
	if err != nil {
		t.Fatalf("walk single metric %s: %v", metr.Name, err)
	}
	if testing.Verbose() {
		t.Logf("req: %+v\n", req)
		t.Logf("results: %+v", res)
	}
}

func TestWalkIndexedMono(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	if snmpIPAddr == "" || snmpCommunity == "" {
		t.Skip("SNMP_IP or SNMP_COMMUNITY env vars not defined, skipping")
	}
	sreq := strings.ReplaceAll(strings.ReplaceAll(Req, "SNMP_IP", snmpIPAddr), "SNMP_COMMUNITY", snmpCommunity)
	var req SnmpRequest
	if err := json.Unmarshal([]byte(sreq), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := req.Dial(context.Background()); err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer req.Close()
	res, err := req.walkMeasure(context.Background(), req.IndexedMeasures[0])
	if err != nil {
		t.Errorf("walkIndexed: %v", err)
	}
	if testing.Verbose() {
		t.Logf("single connection mode: res = %+v", res)
	}
}

func TestWalkIndexedMulti(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	if snmpIPAddr == "" || snmpCommunity == "" {
		t.Skip("SNMP_IP or SNMP_COMMUNITY env vars not defined, skipping")
	}
	sreq := strings.ReplaceAll(strings.ReplaceAll(Req, "SNMP_IP", snmpIPAddr), "SNMP_COMMUNITY", snmpCommunity)
	var req SnmpRequest
	if err := json.Unmarshal([]byte(sreq), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	req.Device.ConnectionCount = 2
	if err := req.Dial(context.Background()); err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer req.Close()
	res, err := req.walkMeasure(context.Background(), req.IndexedMeasures[0])
	if err != nil {
		t.Errorf("walkIndexed: %v", err)
	}
	if testing.Verbose() {
		t.Logf("double connection mode: res = %+v", res)
	}
}
