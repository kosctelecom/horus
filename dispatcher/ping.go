package dispatcher

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"horus-core/log"
	"horus-core/model"
	"net/http"
	"strings"
	"time"

	"github.com/lib/pq"
)

// PingBatchCount is the number of hosts per ping request, set at startup.
var PingBatchCount int

// PingRequests returns current available ping jobs. Retrieves from db the list of hosts
// that where pinged past the ping frequency and merges them into
func PingRequests() ([]model.PingRequest, error) {
	var hosts []model.PingHost
	log.Debug("retrieving available ping jobs")
	err := db.Select(&hosts, `SELECT d.hostname, d.ip_address, d.to_prometheus, d.to_kafka, d.to_influx, p.category, p.vendor, p.model
                                FROM devices d, profiles p
                               WHERE d.active = true
                                 AND d.profile_id = p.id
                                 AND d.ping_frequency > 0
                                 AND (d.last_pinged_at IS NULL OR EXTRACT(EPOCH FROM CURRENT_TIMESTAMP - d.last_pinged_at) >= d.ping_frequency)
                            ORDER BY d.last_pinged_at`)
	if err == sql.ErrNoRows || len(hosts) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	log.Debugf("got %d ping hosts", len(hosts))
	var parts [][]model.PingHost
	for len(hosts) > PingBatchCount {
		hosts, parts = hosts[PingBatchCount:], append(parts, hosts[0:PingBatchCount])
	}
	parts = append(parts, hosts)
	reqs := make([]model.PingRequest, 0, len(parts))
	for _, part := range parts {
		uid, _ := sid.Generate()
		reqs = append(reqs, model.PingRequest{UID: uid, Hosts: part})
	}
	log.Debugf("got %d ping requests", len(reqs))
	log.Debug3f("ping reqs: %+v", reqs)
	return reqs, nil
}

// SendPingRequests retrieves current ping jobs and send each job to an agent in a round-robin mode.
// If the agent rejects the ping job, it is discarded. After a successful post, the device's db last
// ping time is updated.
func SendPingRequests(ctx context.Context) {
	var agents []Agent
	for _, agent := range currentAgentsCopy() {
		if agent.Alive {
			agents = append(agents, *agent)
		}
	}
	if len(agents) == 0 {
		log.Debug("no active agent, skipping this round...")
		return
	}
	log.Debug("poll: getting ping requests")
	reqs, err := PingRequests()
	if err != nil {
		log.Errorf("ping requests: %v", err)
		return
	}
	if reqs == nil {
		log.Debug("no new ping req available")
		return
	}

	for i, req := range reqs {
		if len(req.Hosts) == 0 {
			continue
		}
		err := postPingRequests(ctx, req, agents[i%len(agents)])
		if err != nil {
			log.Errorf("%s - post ping request: %v, skipping...", req.UID, err)
			continue
		}
		sqlExec(req.UID, "setDevLastPingedAt", setDevLastPingedAt, pq.Array(req.Targets()))
	}
}

// postPingRequests posts the request to the agent. Returns an error if the post fails or
// if the agent returns a code other than 202.
func postPingRequests(ctx context.Context, req model.PingRequest, agent Agent) (err error) {
	buf, err := json.Marshal(req)
	if err != nil {
		return
	}
	htReq, err := http.NewRequest("POST", agent.pingJobURL, bytes.NewBuffer(buf))
	if err != nil {
		return
	}
	htReq = htReq.WithContext(ctx)
	htReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 3 * time.Second}
	log.Debugf("%s - posting to agent #%d (%s)", req.UID, agent.ID, agent.name)
	log.Debug2f(">> %s@%s - pinged hosts: %s", req.UID, agent.name, strings.Join(req.Targets(), " "))
	resp, err := client.Do(htReq)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 202 {
		err = fmt.Errorf("agent #%d (%s) rejected with code %d", agent.ID, agent.name, resp.StatusCode)
	}
	return
}
