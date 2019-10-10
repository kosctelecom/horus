package agent

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/vma/glog"
)

const Req = `{
	"uid": "001",
	"device": {
		"id": 1001,
		"hostname": "XXX",
		"polling_frequency": 300,
		"category": "DSLAM",
		"vendor": "HUAWEI",
		"model": "MA5600T",
		"to_kafka": true,
		"ip_address": "XXX",
		"snmp_version": "2c",
		"snmp_community":"XXX",
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
			{"ID":10, "Name":"ifOperStatus", "Oid":".1.3.6.1.2.1.2.2.1.8", "Active":true, "RunningIfaceOnly":false},
			{"ID":11, "Name":"ifAdminStatus", "Oid":".1.3.6.1.2.1.2.2.1.7", "Active":true, "RunningIfaceOnly":false},
			{"ID":18, "Name":"ifHCInOctets", "Oid":".1.3.6.1.2.1.31.1.1.1.6", "Active":true, "RunningIfaceOnly":true},
			{"ID":19, "Name":"ifHCOutOctets", "Oid":".1.3.6.1.2.1.31.1.1.1.7", "Active":true, "RunningIfaceOnly":true},
			{"ID":14, "Name":"ifInErrors", "Oid":".1.3.6.1.2.1.2.2.1.14", "Active":true, "RunningIfaceOnly":true},
			{"ID":17, "Name":"ifOutErrors", "Oid":".1.3.6.1.2.1.2.2.1.20", "Active":true, "RunningIfaceOnly":true},
			{"ID":13, "Name":"ifInDiscards", "Oid":".1.3.6.1.2.1.2.2.1.13", "Active":true, "RunningIfaceOnly":true},
			{"ID":16, "Name":"ifOutDiscards", "Oid":".1.3.6.1.2.1.2.2.1.19", "Active":true, "RunningIfaceOnly":true}],
		"IndexMetricID": 46}
	]
}`

var (
	ipAddress = os.Getenv("SNMP_IP")
	community = os.Getenv("SNMP_COMMUNITY")
)

func init() {
	glog.WithConf(glog.Conf{Verbosity: 3})
}

func TestDial(t *testing.T) {
	if ipAddress == "" || community == "" {
		t.Skip("SNMP_IP or SNMP_COMMUNITY env vars not defined, skipping")
	}
	var req SnmpRequest
	if err := json.Unmarshal([]byte(Req), &req); err != nil {
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
	if ipAddress == "" || community == "" {
		t.Skip("SNMP_IP or SNMP_COMMUNITY env vars not defined, skipping")
	}
	var req SnmpRequest
	if err := json.Unmarshal([]byte(Req), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	req.Device.IPAddress = ipAddress
	req.Device.Hostname = ipAddress
	req.Device.Community = community
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
	if ipAddress == "" || community == "" {
		t.Skip("SNMP_IP or SNMP_COMMUNITY env vars not defined, skipping")
	}
	var req SnmpRequest
	if err := json.Unmarshal([]byte(Req), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	req.Device.IPAddress = ipAddress
	req.Device.Hostname = ipAddress
	req.Device.Community = community
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
	if ipAddress == "" || community == "" {
		t.Skip("SNMP_IP or SNMP_COMMUNITY env vars not defined, skipping")
	}
	var req SnmpRequest
	if err := json.Unmarshal([]byte(Req), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	req.Device.IPAddress = ipAddress
	req.Device.Hostname = ipAddress
	req.Device.Community = community
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
	if ipAddress == "" || community == "" {
		t.Skip("SNMP_IP or SNMP_COMMUNITY env vars not defined, skipping")
	}
	var req SnmpRequest
	if err := json.Unmarshal([]byte(Req), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	req.Device.IPAddress = ipAddress
	req.Device.Hostname = ipAddress
	req.Device.Community = community
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

func TestWalkRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	if ipAddress == "" || community == "" {
		t.Skip("SNMP_IP or SNMP_COMMUNITY env vars not defined, skipping")
	}
	var req SnmpRequest
	if err := json.Unmarshal([]byte(Req), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	req.Device.IPAddress = ipAddress
	req.Device.Hostname = ipAddress
	req.Device.Community = community
	if err := req.Dial(context.Background()); err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer req.Close()
	res, err := req.Walk(context.Background())
	if err != nil {
		t.Errorf("WalkRunning: %v", err)
	}
	if testing.Verbose() {
		t.Logf("res = %+v", res)
		_ = res
	}
}
