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

package dispatcher

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"horus/log"
	"horus/model"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/teris-io/shortid"
)

var (
	// LocalIP is the local IP address used for API and report web server.
	// Defaults to the first non localhost address.
	LocalIP = getLocalIP()

	// Port is the web server port
	Port = 8080

	sid = shortid.MustNew(0, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ$.", 1373)
)

// RequestFromDB returns the request with the given device id from db.
func RequestFromDB(devID int) (model.SnmpRequest, error) {
	var req model.SnmpRequest
	err := db.Get(&req.Device, `SELECT d.id,hostname,ip_address,snmp_port,active,polling_frequency,
                                       snmp_version,snmp_community,snmp_timeout,snmp_retries,snmp_disable_bulk,tags,
                                       snmp_connection_count,to_kafka,to_influx,to_prometheus,snmpv3_security_level,
                                       snmpv3_auth_user,snmpv3_auth_passwd,snmpv3_auth_proto,snmpv3_privacy_passwd,
                                       snmpv3_privacy_proto,p.category,p.vendor,p.model,p.honor_running_only
                                  FROM devices d, profiles p
                                 WHERE d.profile_id = p.id
                                   AND d.id = $1`, devID)
	if err != nil {
		return req, fmt.Errorf("request: %v", err)
	}

	err = db.Select(&req.ScalarMeasures, `SELECT m.id,m.name,m.description,m.polling_frequency,
                                                 (SELECT last_polled_at
                                                    FROM measure_poll_times t
                                                   WHERE t.device_id = d.id AND t.measure_id = m.id
                                                ORDER BY last_polled_at DESC LIMIT 1)
                                            FROM measures m, devices d, profile_measures pm
                                           WHERE m.is_indexed = false
                                             AND d.profile_id = pm.profile_id
                                             AND m.id = pm.measure_id
                                             AND m.polling_frequency >= 0
                                             AND d.id = $1
                                        ORDER BY m.id`, devID)
	if err != nil {
		return req, fmt.Errorf("select scalar measures: %v", err)
	}
	err = db.Select(&req.IndexedMeasures, `SELECT m.id,m.name,m.description,m.polling_frequency,m.index_metric_id,
                                                  m.filter_metric_id,m.filter_pattern,m.invert_filter_match,
                                                  (SELECT last_polled_at
                                                     FROM measure_poll_times t
                                                    WHERE t.device_id = d.id
                                                      AND t.measure_id = m.id
                                                 ORDER BY last_polled_at DESC LIMIT 1)
                                             FROM measures m, devices d, profile_measures pm
                                            WHERE m.is_indexed = true
                                              AND d.profile_id = pm.profile_id
                                              AND m.id = pm.measure_id
                                              AND m.polling_frequency >= 0
                                              AND d.id = $1
                                         ORDER BY m.id`, devID)
	if err != nil {
		return req, fmt.Errorf("select indexed measures: %v", err)
	}
	req.FilterMeasures()
	for i, scalar := range req.ScalarMeasures {
		err = db.Select(&scalar.Metrics, `SELECT m.id,m.name,m.oid,m.description,m.active,m.export_as_label
                                            FROM metrics m, measure_metrics mm
                                           WHERE m.active = true
                                             AND m.id = mm.metric_id
                                             AND mm.measure_id = $1
                                        ORDER BY m.id`, scalar.ID)
		if err != nil {
			return req, fmt.Errorf("select scalar metrics: %v", err)
		}
		req.ScalarMeasures[i] = scalar
	}
	for i, indexed := range req.IndexedMeasures {
		err = db.Select(&indexed.Metrics, `SELECT m.id,m.name,m.oid,m.description,m.index_pattern,
                                                  m.active,m.export_as_label,m.running_if_only
                                             FROM metrics m, measure_metrics mm
                                            WHERE m.active = true
                                              AND m.id = mm.metric_id
                                              AND mm.measure_id = $1
                                         ORDER BY m.id`, indexed.ID)
		if err != nil {
			return req, fmt.Errorf("select indexed metrics: %v", err)
		}
		req.IndexedMeasures[i] = indexed
	}
	if LocalIP != "" && Port != 0 {
		req.ReportURL = fmt.Sprintf("http://%s:%d%s", LocalIP, Port, model.ReportURI)
	}
	uid, err := sid.Generate()
	if err != nil {
		return req, fmt.Errorf("shortid: %v", err)
	}
	req.UID = fmt.Sprintf("%s@%d", uid, req.Device.ID)
	return req, nil
}

// SnmpJobs returns a list of pollable device ids. A device is pollable if there
// is no ongoing polling job and was last polled past its polling frequency.
func SnmpJobs() ([]int, error) {
	var devs []int
	log.Debug("retrieving available snmp jobs")
	err := db.Select(&devs, `SELECT id
                               FROM devices
                              WHERE active = true
                                AND polling_frequency > 0
                                AND is_polling = false
                                AND (last_polled_at IS NULL OR EXTRACT(EPOCH FROM CURRENT_TIMESTAMP - last_polled_at) >= polling_frequency)
                           ORDER BY last_polled_at,id`)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	log.Debugf("got %d snmp jobs", len(devs))
	return devs, nil
}

// RequestWithLock builds a model.Request from the given
// device id and locks the device if there is no error.
func RequestWithLock(id int) (model.SnmpRequest, error) {
	log.Debug2f("retrieving request for device #%d", id)
	req, err := RequestFromDB(id)
	if err != nil {
		return req, fmt.Errorf("request from db: %v", err)
	}
	log.Debug2f("%s - locking device #%d", req.UID, id)
	_, err = lockDevStmt.Exec(req.Device.ID)
	if err != nil {
		return req, fmt.Errorf("lock device: %v", err)
	}
	return req, nil
}

// updateLastPolledAt updates device and measure with last request time. For the measures,
// a new entry is added to measure_poll_times tables iff the measure is non-zero.
func updateLastPolledAt(req model.SnmpRequest) {
	sqlExec(req.UID, "setDevLastPolledAt", setDevLastPolledAt, req.Device.ID)
	for _, scalar := range req.ScalarMeasures {
		if scalar.PollingFrequency > 0 {
			sqlExec(req.UID, "insertMeasLastPolledAt", insertMeasLastPolledAt, req.Device.ID, scalar.ID)
		}
	}
	for _, indexed := range req.IndexedMeasures {
		if indexed.PollingFrequency > 0 {
			sqlExec(req.UID, "insertMeasLastPolledAt", insertMeasLastPolledAt, req.Device.ID, indexed.ID)
		}
	}
}

// SendRequest sends the given request to the given agent.
// Returns the http status code, the agent's current load
// and an error if unsuccessful.
func SendRequest(ctx context.Context, req model.SnmpRequest, agent Agent) (stCode int, load float64, err error) {
	log.Debug3f("%s - marshaling request", req.UID)
	req.AgentID = agent.ID
	buf, err := json.Marshal(req)
	if err != nil {
		return
	}
	log.Debug3f("%s - payload to send: %s", req.UID, buf)
	htReq, err := http.NewRequest("POST", agent.snmpJobURL, bytes.NewBuffer(buf))
	if err != nil {
		err = fmt.Errorf("http request: %v", err)
		return
	}
	htReq = htReq.WithContext(ctx)
	htReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 3 * time.Second}
	log.Debug2f("%s - posting request to agent #%d (%s:%d)", req.UID, agent.ID, agent.Host, agent.Port)
	resp, err := client.Do(htReq)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	stCode = resp.StatusCode
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	load, _ = strconv.ParseFloat(string(b), 64)
	return
}

// getLocalIP returns the system first non localhost IP address as string.
// Returns an empty string on error.
func getLocalIP() string {
	var localIP string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Warningf("getLocalIP: %v", err)
		return ""
	}
	for _, addr := range addrs {
		ip := addr.String()
		if strings.HasPrefix(ip, "127.0.0.1") {
			continue
		}
		slashIdx := strings.Index(ip, "/")
		localIP = ip
		if slashIdx > 0 {
			localIP = ip[:slashIdx]
		}
		break
	}
	log.Debugf("local ip: %s", localIP)
	return localIP
}

// FlushReports removes old report entries.
func FlushReports(maxDays int) {
	log.Debugf("flushing reports older than %d days", maxDays)
	rs, err := db.Exec(`DELETE FROM reports WHERE requested_at <= $1`, time.Now().Add(-24*time.Hour*time.Duration(maxDays)))
	if err != nil {
		log.Errorf("flush old reports: %v", err)
		return
	}
	count, _ := rs.RowsAffected()
	log.Debugf("%d old reports flushed", count)
}
