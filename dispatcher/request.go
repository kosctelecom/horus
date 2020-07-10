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

package dispatcher

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kosctelecom/horus/log"
	"github.com/kosctelecom/horus/model"
	"github.com/teris-io/shortid"
)

var (
	// LocalIP is the local IP address used for API and report web server.
	// Defaults to the first non localhost address.
	LocalIP = getLocalIP()

	// Port is the web server port
	Port = 8080

	// HTTPTimeout is the timeout in seconds for posting poll requests
	HTTPTimeout = 3

	sid = shortid.MustNew(0, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ$.", 1373)
)

// RequestFromDB returns the request with the given device id from db.
func RequestFromDB(devID int) (model.SnmpRequest, error) {
	var req model.SnmpRequest
	err := db.Get(&req.Device, `SELECT active,
                                       d.id,
                                       hostname,
                                       COALESCE(ip_address, '') AS ip_address,
                                       p.category,
                                       p.vendor,
                                       p.model,
                                       polling_frequency,
                                       snmp_alternate_community,
                                       snmp_community,
                                       snmp_connection_count,
                                       snmp_disable_bulk,
                                       snmp_port,
                                       snmp_retries,
                                       snmp_timeout,
                                       snmp_version,
                                       snmpv3_auth_passwd,
                                       snmpv3_auth_proto,
                                       snmpv3_auth_user,
                                       snmpv3_privacy_passwd,
                                       snmpv3_privacy_proto,
                                       snmpv3_security_level,
                                       tags
                                  FROM devices d,
                                       profiles p
                                 WHERE d.profile_id = p.id
                                   AND d.id = $1`, devID)
	if err != nil {
		return req, fmt.Errorf("request: %v", err)
	}

	if req.Device.SnmpParams.IPAddress == "" {
		addrs, err := net.LookupHost(req.Device.Hostname)
		if err != nil {
			return req, fmt.Errorf("snmp request: lookup %s: %v", req.Device.Hostname, err)
		}
		log.Debug2f("host %s resolved to %s", req.Device.Hostname, addrs[0])
		req.Device.SnmpParams.IPAddress = addrs[0]
	}

	var scalarMeasures []model.ScalarMeasure
	err = db.Select(&scalarMeasures, `SELECT m.description,
                                             m.id,
                                             m.name,
                                             m.use_alternate_community,
                                             m.to_influx,
                                             m.to_kafka,
                                             m.to_prometheus
                                        FROM devices d,
                                             measures m,
                                             profile_measures pm
                                       WHERE d.id = $1
                                         AND d.profile_id = pm.profile_id
                                         AND m.id = pm.measure_id
                                         AND m.is_indexed = FALSE
                                    ORDER BY m.id`, devID)
	if err != nil {
		return req, fmt.Errorf("select scalar measures: %v", err)
	}
	var indexedMeasures []model.IndexedMeasure
	err = db.Select(&indexedMeasures, `SELECT m.description,
                                              m.filter_metric_id,
                                              m.filter_pattern,
                                              m.id,
                                              m.index_metric_id,
                                              m.invert_filter_match,
                                              m.name,
                                              m.use_alternate_community,
                                              m.to_influx,
                                              m.to_kafka,
                                              m.to_prometheus
                                         FROM devices d,
                                              measures m,
                                              profile_measures pm
                                        WHERE d.id = $1
                                          AND d.profile_id = pm.profile_id
                                          AND m.id = pm.measure_id
                                          AND m.is_indexed = TRUE
                                     ORDER BY m.id`, devID)
	if err != nil {
		return req, fmt.Errorf("select indexed measures: %v", err)
	}
	for _, scalar := range scalarMeasures {
		err = db.Select(&scalar.Metrics, `SELECT m.active,
                                                 m.description,
                                                 m.export_as_label,
                                                 COALESCE(m.exported_name, m.name) AS exported_name,
                                                 m.id,
                                                 m.name,
                                                 m.oid,
                                                 m.polling_frequency,
                                                 m.post_processors,
                                                 t.last_polled_at
                                            FROM measure_metrics mm,
                                                 metrics m
                                       LEFT JOIN metric_poll_times t ON (t.metric_id = m.id AND t.device_id = $1)
                                           WHERE m.active = TRUE
                                             AND m.id = mm.metric_id
                                             AND mm.measure_id = $2
                                             AND (t.last_polled_at IS NULL OR EXTRACT(EPOCH FROM CURRENT_TIMESTAMP - t.last_polled_at) >= m.polling_frequency)
                                        ORDER BY m.id`, devID, scalar.ID)
		if err != nil {
			return req, fmt.Errorf("select scalar metrics: %v", err)
		}
		if len(scalar.Metrics) > 0 {
			req.ScalarMeasures = append(req.ScalarMeasures, scalar)
		}
	}
	for _, indexed := range indexedMeasures {
		err = db.Select(&indexed.Metrics, `SELECT m.active,
                                                  m.description,
                                                  m.export_as_label,
                                                  COALESCE(m.exported_name, m.name) AS exported_name,
                                                  m.id,
                                                  m.index_pattern,
                                                  m.name,
                                                  m.oid,
                                                  m.polling_frequency,
                                                  m.post_processors,
                                                  t.last_polled_at
                                             FROM measure_metrics mm,
                                                  metrics m
                                        LEFT JOIN metric_poll_times t ON (t.metric_id = m.id AND t.device_id = $1)
                                            WHERE m.active = TRUE
                                              AND m.id = mm.metric_id
                                              AND mm.measure_id = $2
                                              AND (t.last_polled_at IS NULL OR EXTRACT(EPOCH FROM CURRENT_TIMESTAMP - t.last_polled_at) >= m.polling_frequency)
                                         ORDER BY m.id`, devID, indexed.ID)
		if err != nil {
			return req, fmt.Errorf("select indexed metrics: %v", err)
		}

		var labelCount int
		for _, m := range indexed.Metrics {
			if m.ExportAsLabel {
				labelCount++
			}
		}
		indexed.LabelsOnly = (labelCount == len(indexed.Metrics))

		if len(indexed.Metrics) > 0 {
			req.IndexedMeasures = append(req.IndexedMeasures, indexed)
		}
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
// a new entry is added to measure_poll_times tables if the measure's polling frequency is non-zero.
func updateLastPolledAt(req model.SnmpRequest) {
	sqlExec(req.UID, "setDevLastPolledAt", setDevLastPolledAt, req.Device.ID)
	for _, scalar := range req.ScalarMeasures {
		for _, m := range scalar.Metrics {
			if m.PollingFrequency > 0 {
				sqlExec(req.UID, "insertMetricLastPolledAt", insertMetricLastPolledAt, req.Device.ID, m.ID)
			}
		}
	}
	for _, indexed := range req.IndexedMeasures {
		for _, m := range indexed.Metrics {
			if m.PollingFrequency > 0 {
				sqlExec(req.UID, "insertMetricLastPolledAt", insertMetricLastPolledAt, req.Device.ID, m.ID)
			}
		}
	}
}

// SendRequest sends the given request to the given agent. Returns the http status code, the agent's current load
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
	client := &http.Client{Timeout: time.Duration(HTTPTimeout) * time.Second}
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
func FlushReports(maxErrDays, maxEmptyHours int) {
	log.Debugf("flushing error reports older than %d days", maxErrDays)
	rs, err := db.Exec(`DELETE FROM reports WHERE requested_at <= $1`, time.Now().Add(-24*time.Hour*time.Duration(maxErrDays)))
	if err != nil {
		log.Errorf("flush error reports: %v", err)
		return
	}
	count, _ := rs.RowsAffected()
	log.Debugf("%d error reports flushed", count)
	log.Debugf("flushing unreceived reports older than %d hours", maxEmptyHours)
	rs, err = db.Exec(`DELETE FROM reports WHERE requested_at <= $1 AND report_received_at IS NULL`, time.Now().Add(-time.Hour*time.Duration(maxEmptyHours)))
	if err != nil {
		log.Errorf("flush empty reports: %v", err)
		return
	}
	count, _ = rs.RowsAffected()
	log.Debugf("%d empty reports flushed", count)
}
