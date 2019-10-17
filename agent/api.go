package agent

import (
	"encoding/json"
	"fmt"
	"horus/log"
	"horus/model"
	"io/ioutil"
	"net/http"

	"github.com/vma/glog"
)

// HandleSnmpRequest handles snmp polling job requests.
func HandleSnmpRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Debugf("rejecting request from %s with %s method", r.RemoteAddr, r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "%.4f", CurrentLoad())
		return
	}

	if GracefulQuitMode {
		log.Debug("in graceful quit mode, rejecting all new requests")
		w.WriteHeader(http.StatusLocked)
		fmt.Fprintf(w, "%.4f", CurrentLoad())
		return
	}

	log.Debug2f("got new poll request from %s", r.RemoteAddr)
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Debug2f("error reading body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%.4f", CurrentLoad())
		return
	}
	r.Body.Close()
	log.Debug3f("new request: %s", b)
	var req SnmpRequest
	if err := json.Unmarshal(b, &req); err != nil {
		log.Debugf("invalid json request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%.4f", CurrentLoad())
		return
	}
	if AddSnmpRequest(req) {
		log.Debugf("%s - request successfully queued", req.UID)
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "%.4f", CurrentLoad())
	} else {
		glog.Warningf("no more workers, rejecting request %s", req.UID)
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintf(w, "%.4f", CurrentLoad())
	}
}

// HandleCheck responds to keep-alive checks.
// Returns current worker count in body.
func HandleCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%.4f", CurrentLoad())
}

// HandleOngoing returns the list of ongoing snmp requests,
// their count, and the total workers count.
func HandleOngoing(w http.ResponseWriter, r *http.Request) {
	var ongoing model.OngoingPolls

	ongoingMu.RLock()
	for id := range ongoingReqs {
		ongoing.Requests = append(ongoing.Requests, id)
	}
	ongoingMu.RUnlock()
	ongoing.Load = CurrentLoad()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ongoing)
}

// HandlePingRequest handles ping job requests.
// Returns a status 202 when the job is accepted, a 4XX error status otherwise.
func HandlePingRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Debugf("rejecting request from %s with %s method", r.RemoteAddr, r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if GracefulQuitMode {
		log.Debug("in graceful quit mode, rejecting all new requests")
		w.WriteHeader(http.StatusLocked)
		return
	}
	log.Debug2f("got new ping request from %s", r.RemoteAddr)
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Debug2f("error reading body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	r.Body.Close()
	var pingReq model.PingRequest
	if err := json.Unmarshal(b, &pingReq); err != nil {
		log.Debugf("invalid ping request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(pingReq.Hosts) == 0 {
		log.Warningf("%s - ping job with no host, rejecting", pingReq.UID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Debug3f("ping request: %+v", pingReq)
	if AddPingRequest(pingReq) {
		log.Debugf("%s - ping job successfully queued (%d hosts)", pingReq.UID, len(pingReq.Hosts))
		w.WriteHeader(http.StatusAccepted)
	} else {
		glog.Warningf("%s - no more workers, rejecting ping request", pingReq.UID)
		w.WriteHeader(http.StatusTooManyRequests)
	}
}
