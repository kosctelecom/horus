package dispatcher

import (
	"database/sql"
	"horus/log"
	"net/http"
	"strconv"
	"time"
)

// HandleReport saves the polling report to db and unlocks the device.
func HandleReport(w http.ResponseWriter, r *http.Request) {
	reqUID := r.FormValue("request_id")
	agentID := r.FormValue("agent_id")
	pollDur := r.FormValue("poll_duration_ms")
	pollErr := r.FormValue("poll_error")
	if pollDur == "" {
		pollDur = "0"
	}
	log.Debug2f(">>%s - new report received, poll duration=%s", reqUID, pollDur)
	if dur, _ := strconv.Atoi(pollDur); dur <= 500 {
		// sleep some time if the request was executed too quickly to avoid a race
		// where the request entry is deleted before it is inserted in poll.go:SendPollingJobs()
		time.Sleep(500 * time.Millisecond)
	}
	currLoad := r.FormValue("current_load")
	metricCount := r.FormValue("metric_count")
	log.Debugf("report: req_uid=%s agent_id=%s snmp_dur=%s snmp_err=`%s` metric_count=%s curr_load=%s",
		reqUID, agentID, pollDur, pollErr, metricCount, currLoad)
	if err := sqlExec(reqUID, "unlockDevFromReportStmt", unlockDevFromReportStmt, reqUID); err != nil {
		log.Errorf("%s - unlock dev from request: %v", reqUID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var err error
	if pollErr == "" {
		log.Debugf("%s - removing terminated report entry", reqUID)
		var rs sql.Result
		rs, err = db.Exec("DELETE FROM reports WHERE uuid = $1", reqUID)
		count, _ := rs.RowsAffected()
		log.Debugf("%s - %d row deleted", reqUID, count)
	} else {
		// keep only errors
		log.Debugf("%s - saving error report entry", reqUID)
		err = sqlExec(reqUID, "updReportStmt", updReportStmt, reqUID, pollDur, pollErr)
	}
	if err != nil {
		log.Errorf("%s - handle report: %v", reqUID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	currentAgentsMu.Lock()
	defer currentAgentsMu.Unlock()
	for i, agent := range currentAgents {
		if strconv.Itoa(agent.ID) == agentID {
			if load, err := strconv.ParseFloat(currLoad, 64); err == nil {
				currentAgents[i].setLoad(load)
			} else {
				log.Warningf("%s - unable to parse current_load: %v", reqUID, err)
			}
			break
		}
	}
	w.WriteHeader(http.StatusOK)
}
