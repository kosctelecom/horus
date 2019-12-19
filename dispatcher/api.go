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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"horus/log"
	"horus/model"
	"io/ioutil"
	"net/http"
	"strconv"
)

const (
	// DeviceListURI is the api endpoint for device listing
	DeviceListURI = "/d/list"

	// DeviceCreateURI is the api endpoint for creating new device
	DeviceCreateURI = "/d/create"

	// DeviceUpdateURI is the api endpoint for updating a device
	DeviceUpdateURI = "/d/update"

	// DeviceUpsertURI is the api endpoint for `upserting` a device
	DeviceUpsertURI = "/d/upsert"

	// DeviceDeleteURI is the api endpoint for deleting a device
	DeviceDeleteURI = "/d/delete"
)

// HandleDeviceList implements the CRUD list handler. When `id` parameter is given
// to the GET request, returns a json body with the device with this id. Otherwise,
// returns a json array with all devices ordered by id.
func HandleDeviceList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, http.StatusMethodNotAllowed, errors.New("Method Not Allowed, use GET"))
		return
	}
	if id := r.FormValue("id"); id != "" {
		var dev model.Device
		err := db.Get(&dev, `SELECT d.active,
                                    d.hostname,
                                    d.id,
                                    d.ip_address,
                                    d.ping_frequency,
                                    d.polling_frequency,
                                    d.snmp_community,
                                    d.snmp_connection_count,
                                    d.snmp_disable_bulk,
                                    d.snmp_port,
                                    d.snmp_retries,
                                    d.snmp_timeout,
                                    d.snmp_version,
                                    d.snmpv3_auth_passwd,
                                    d.snmpv3_auth_proto,
                                    d.snmpv3_auth_user,
                                    d.snmpv3_privacy_passwd,
                                    d.snmpv3_privacy_proto,
                                    d.snmpv3_security_level,
                                    d.tags,
                                    p.category,
                                    p.model,
                                    p.vendor
                               FROM devices d,
                                    profiles p
                              WHERE d.id = $1
                                AND d.profile_id = p.id`, id)
		if err == sql.ErrNoRows {
			jsonError(w, http.StatusNotFound, errors.New("Device not found"))
			return
		}
		if err != nil {
			log.Warningf("HandleDeviceList: select device %s: %v", id, err)
			jsonBadRequest(w, err)
			return
		}
		buf, _ := json.MarshalIndent(dev, "", "  ")
		fmt.Fprintf(w, "%s", buf)
		return
	}

	var devs []model.Device
	err := db.Select(&devs, `SELECT d.active,
                                    d.hostname,
                                    d.id,
                                    d.ip_address,
                                    d.ping_frequency,
                                    d.polling_frequency,
                                    d.snmp_community,
                                    d.snmp_connection_count,
                                    d.snmp_disable_bulk,
                                    d.snmp_port,
                                    d.snmp_retries,
                                    d.snmp_timeout,
                                    d.snmp_version,
                                    d.snmpv3_auth_passwd,
                                    d.snmpv3_auth_proto,
                                    d.snmpv3_auth_user,
                                    d.snmpv3_privacy_passwd,
                                    d.snmpv3_privacy_proto,
                                    d.snmpv3_security_level,
                                    d.tags,
                                    p.category,
                                    p.model,
                                    p.vendor
                               FROM devices d,
                                    profiles p
                              WHERE d.profile_id = p.id
                           ORDER BY d.id`)
	if err != nil {
		log.Warning("HandleDeviceList: select all devices: ", err)
		jsonBadRequest(w, err)
		return
	}
	buf, _ := json.MarshalIndent(devs, "", "  ")
	fmt.Fprintf(w, "%s", buf)
}

// HandleDeviceCreate implements the CRUD create handler
func HandleDeviceCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, http.StatusMethodNotAllowed, errors.New("Method Not Allowed, use POST"))
		return
	}
	log.Infof("HandleCreate: new request from %s", r.RemoteAddr)
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warningf("HandleCreate: error reading body: %v", err)
		jsonBadRequest(w, err)
		return
	}
	defer r.Body.Close()
	var dev model.Device
	if err = json.Unmarshal(b, &dev); err != nil {
		log.Warning("HandleCreate: bad request:", err)
		jsonBadRequest(w, err)
		return
	}
	var currDev model.Device
	err = db.Get(&currDev, `SELECT id
                              FROM devices
                             WHERE id = $1`, dev.ID)
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("HandleCreate: devices select: %v", err)
		jsonError(w, 500, err)
		return
	}
	if currDev.ID != 0 {
		jsonBadRequest(w, fmt.Errorf("device #%d already exists", dev.ID))
		return
	}

	err = db.Get(&dev, `SELECT id AS profile_id
                          FROM profiles
                         WHERE category = $1
                           AND model = $2
                           AND vendor = $3`, dev.Category, dev.Model, dev.Vendor)
	if err != nil {
		log.Warning("HandleCreate: profiles select:", err)
		jsonBadRequest(w, fmt.Errorf("profile: invalid profile (%s, %s, %s)", dev.Category, dev.Vendor, dev.Model))
		return
	}
	_, err = db.NamedExec(`INSERT INTO devices (active,
                                                hostname,
                                                id,
                                                ip_address,
                                                ping_frequency,
                                                polling_frequency,
                                                profile_id,
                                                snmp_connection_count,
                                                snmp_community,
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
                                                tags)
                                        VALUES (:active,
                                                :hostname,
                                                :id,
                                                :ip_address,
                                                :ping_frequency,
                                                :polling_frequency,
                                                :profile_id,
                                                :snmp_connection_count,
                                                :snmp_community,
                                                :snmp_disable_bulk,
                                                :snmp_port,
                                                :snmp_retries,
                                                :snmp_timeout,
                                                :snmp_version,
                                                :snmpv3_auth_passwd,
                                                :snmpv3_auth_proto,
                                                :snmpv3_auth_user,
                                                :snmpv3_privacy_passwd,
                                                :snmpv3_privacy_proto,
                                                :snmpv3_security_level,
                                                :tags)`, dev)
	if err != nil {
		log.Warningf("HandleCreate: device insert: %v", err)
		jsonBadRequest(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleDeviceUpdate implements the CRUD update handler. All device required fields
// must be defined in the json as for the insert request.
func HandleDeviceUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, http.StatusMethodNotAllowed, errors.New("Method Not Allowed, use POST"))
		return
	}
	log.Debugf("HandleUpdate: new request from %s", r.RemoteAddr)
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warningf("HandleUpdate: error reading body: %v", err)
		jsonBadRequest(w, err)
		return
	}
	defer r.Body.Close()
	var dev model.Device
	if err = json.Unmarshal(b, &dev); err != nil {
		log.Warningf("HandleUpdate: bad request: %v", err)
		jsonBadRequest(w, err)
		return
	}
	err = db.Get(&dev, `SELECT id AS profile_id
                          FROM profiles
                         WHERE category = $1
                           AND model = $2
                           AND vendor = $3`, dev.Category, dev.Model, dev.Vendor)
	if err != nil {
		log.Warningf("HandleUpdate: profiles select: %v", err)
		jsonBadRequest(w, fmt.Errorf("profile: invalid category(%s) or vendor(%s) or model(%s)", dev.Category, dev.Vendor, dev.Model))
		return
	}
	var currDev model.Device
	err = db.Get(&currDev, `SELECT id
                              FROM devices
                             WHERE id = $1`, dev.ID)
	if err != nil {
		log.Warningf("HandleUpdate: dev select: %v", err)
		jsonError(w, http.StatusNotFound, fmt.Errorf("Device with id %d Not Found", dev.ID))
		return
	}
	_, err = db.NamedExec(`UPDATE devices
                              SET active = :active,
                                  hostname = :hostname,
                                  ip_address = :ip_address,
                                  ping_frequency = :ping_frequency,
                                  polling_frequency = :polling_frequency,
                                  profile_id = :profile_id,
                                  snmp_community = :snmp_community,
                                  snmp_connection_count = :snmp_connection_count,
                                  snmp_disable_bulk = :snmp_disable_bulk,
                                  snmp_port = :snmp_port,
                                  snmp_retries = :snmp_retries,
                                  snmp_timeout = :snmp_timeout,
                                  snmp_version = :snmp_version,
                                  snmpv3_auth_passwd = :snmpv3_auth_passwd,
                                  snmpv3_auth_proto = :snmpv3_auth_proto,
                                  snmpv3_auth_user = :snmpv3_auth_user,
                                  snmpv3_privacy_passwd = :snmpv3_privacy_passwd,
                                  snmpv3_privacy_proto = :snmpv3_privacy_proto,
                                  snmpv3_security_level = :snmpv3_security_level,
                                  tags = :tags
                            WHERE id = :id`, dev)
	if err != nil {
		log.Warningf("HandleUpdate: devices update: %v", err)
		jsonBadRequest(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleDeviceUpsert implements the CRUD upsert handler. All device required fields
// must be defined in the json as for the insert request.
func HandleDeviceUpsert(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, http.StatusMethodNotAllowed, errors.New("Method Not Allowed, use POST"))
		return
	}
	log.Debugf("new upsert request from %s", r.RemoteAddr)
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warningf("upsert: error reading body: %v", err)
		jsonBadRequest(w, err)
		return
	}
	defer r.Body.Close()
	var dev model.Device
	if err = json.Unmarshal(b, &dev); err != nil {
		log.Errorf("ERR: upsert: invalid request `%s`: %v", b, err)
		jsonBadRequest(w, err)
		return
	}
	err = db.Get(&dev, `SELECT id AS profile_id
                          FROM profiles
                         WHERE category = $1
                           AND model = $2
                           AND vendor = $3`, dev.Category, dev.Model, dev.Vendor)
	if err != nil {
		log.Errorf("ERR: upsert: invalid profile (%q,%q,%q) for dev#%d: %v", dev.Category, dev.Vendor, dev.Model, dev.ID, err)
		jsonBadRequest(w, fmt.Errorf("invalid profile (%q,%q,%q)", dev.Category, dev.Vendor, dev.Model))
		return
	}
	_, err = db.NamedExec(`INSERT INTO devices (active,
                                                hostname,
                                                id,
                                                ip_address,
                                                ping_frequency,
                                                polling_frequency,
                                                profile_id,
                                                snmp_connection_count,
                                                snmp_community,
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
                                                tags)
                                        VALUES (:active,
                                                :hostname,
                                                :id,
                                                :ip_address,
                                                :ping_frequency,
                                                :polling_frequency,
                                                :profile_id,
                                                :snmp_connection_count,
                                                :snmp_community,
                                                :snmp_disable_bulk,
                                                :snmp_port,
                                                :snmp_retries,
                                                :snmp_timeout,
                                                :snmp_version,
                                                :snmpv3_auth_passwd,
                                                :snmpv3_auth_proto,
                                                :snmpv3_auth_user,
                                                :snmpv3_privacy_passwd,
                                                :snmpv3_privacy_proto,
                                                :snmpv3_security_level,
                                                :tags)
                               ON CONFLICT(id)
                                     DO UPDATE
                                           SET active = :active,
                                               hostname = :hostname,
                                               ip_address = :ip_address,
                                               ping_frequency = :ping_frequency,
                                               polling_frequency = :polling_frequency,
                                               profile_id = :profile_id,
                                               snmp_community = :snmp_community,
                                               snmp_connection_count = :snmp_connection_count,
                                               snmp_disable_bulk = :snmp_disable_bulk,
                                               snmp_port = :snmp_port,
                                               snmp_retries = :snmp_retries,
                                               snmp_timeout = :snmp_timeout,
                                               snmp_version = :snmp_version,
                                               snmpv3_auth_passwd = :snmpv3_auth_passwd,
                                               snmpv3_auth_proto = :snmpv3_auth_proto,
                                               snmpv3_auth_user = :snmpv3_auth_user,
                                               snmpv3_privacy_passwd = :snmpv3_privacy_passwd,
                                               snmpv3_privacy_proto = :snmpv3_privacy_proto,
                                               snmpv3_security_level = :snmpv3_security_level,
                                               tags = :tags`, dev)
	if err != nil {
		log.Errorf("ERR: upsert dev#%d: %v:", dev.ID, err)
		jsonBadRequest(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleDeviceDelete implements the CRUD delete handler. The id of the device
// to delete must be given in `id` param to the POST request.
func HandleDeviceDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, http.StatusMethodNotAllowed, errors.New("Method Not Allowed, use POST"))
		return
	}
	log.Infof("HandleDelete: new request, from %s", r.RemoteAddr)
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		log.Warningf("HandleDelete: invalid id: %v", err)
		jsonBadRequest(w, errors.New("`id` parameter missing or invalid"))
		return
	}
	log.Infof("HandleDelete: deleting device %d", id)
	res, err := db.Exec(`DELETE FROM devices
                               WHERE id = $1`, id)
	if err != nil {
		log.Error("HandleDelete: delete device:", err)
		jsonBadRequest(w, err)
		return
	}
	if count, _ := res.RowsAffected(); count == 0 {
		jsonError(w, http.StatusNotFound, errors.New("Device not found"))
		return
	}
	w.WriteHeader(http.StatusOK)
}

// jsonError returns an HTTP error with given status code to `w` and
// a json body with an `error` param containing the detailled error message.
func jsonError(w http.ResponseWriter, status int, err error) {
	if err == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, err)
}

// jsonBadRequest returns an HTTP bad request error  to `w` and json error body.
func jsonBadRequest(w http.ResponseWriter, err error) {
	jsonError(w, 400, err)
}
