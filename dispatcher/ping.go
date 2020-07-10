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
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/kosctelecom/horus/log"
	"github.com/kosctelecom/horus/model"
	"github.com/lib/pq"
)

// PingBatchCount is the number of hosts per ping request, set at startup.
var PingBatchCount int

var pingHostRepartition = make(map[int]int)

// PingHosts retrieves alls hosts to be pinged.
func PingHosts() ([]model.PingHost, error) {
	var hosts []model.PingHost
	log.Debug("retrieving available ping jobs")
	err := db.Select(&hosts, `SELECT d.hostname,
                                     d.id,
                                     COALESCE(d.ip_address, '') AS ip_address,
                                     p.category,
                                     p.model,
                                     p.vendor
                                FROM devices d,
                                     profiles p
                               WHERE d.active = TRUE
                                 AND (d.last_pinged_at IS NULL OR EXTRACT(EPOCH FROM CURRENT_TIMESTAMP - d.last_pinged_at) >= d.ping_frequency)
                                 AND d.ping_frequency > 0
                                 AND d.profile_id = p.id
                            ORDER BY d.last_pinged_at`)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	log.Debugf("got %d ping hosts", len(hosts))
	for i, host := range hosts {
		if host.IPAddr == "" {
			addrs, err := net.LookupHost(host.Name)
			if err != nil {
				log.Errorf("ping: lookup %s: %v", host.Name, err)
				continue
			}
			log.Debug2f("host %s resolved to %s", host.Name, addrs[0])
			hosts[i].IPAddr = addrs[0]
		}
	}
	return hosts, nil
}

// SendPingRequests sends current ping requests to agents
// with load-balancing and agent stickyness
func SendPingRequests(ctx context.Context) {
	var agents []Agent
	var agentHosts = make(map[int][]model.PingHost)
	var unaffectedHosts []model.PingHost
	for _, agent := range currentAgentsCopy() {
		if agent.Alive {
			agents = append(agents, *agent)
		}
	}
	if len(agents) == 0 {
		log.Debug("ping: no active agent, skipping this round...")
		return
	}
	hosts, err := PingHosts()
	if err != nil {
		log.Errorf("ping: unable to get hosts: %v", err)
		return
	}
	for _, agent := range agents {
		agentHosts[agent.ID] = nil
	}

	maxAgentHosts := int(math.Ceil(float64(len(hosts)) / float64(len(agents))))
	for _, host := range hosts {
		agentID, ok := pingHostRepartition[host.ID]
		if ok && agentFromID(agentID, agents).ID > 0 && len(agentHosts[agentID]) < maxAgentHosts {
			log.Debug2f("host %d affected to previous agent %d", host.ID, agentID)
			agentHosts[agentID] = append(agentHosts[agentID], host)
		} else {
			log.Debug2f("host %d unaffected", host.ID)
			unaffectedHosts = append(unaffectedHosts, host)
		}
	}
	for _, host := range unaffectedHosts {
		agentID, minCount := leastLoadedAgent(agentHosts)
		log.Debug2f("unaffected host %d affected to least loaded agent %d (host count: %d)", host.ID, agentID, minCount)
		agentHosts[agentID] = append(agentHosts[agentID], host)
		pingHostRepartition[host.ID] = agentID
	}

	for agentID, hosts := range agentHosts {
		agent := agentFromID(agentID, agents)
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
		for _, req := range reqs {
			if len(req.Hosts) == 0 {
				continue
			}
			err := postPingRequest(ctx, req, agent)
			if err != nil {
				log.Errorf("%s - post ping request: %v, skipped", req.UID, err)
				continue
			}
		}
	}
}

// postPingRequest posts a ping job to an agent. Returns an error if the post fails or
// if the agent returns a code other than 202.
func postPingRequest(ctx context.Context, req model.PingRequest, agent Agent) error {
	buf, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal: %v", err)
	}
	htReq, err := http.NewRequest("POST", agent.pingJobURL, bytes.NewBuffer(buf))
	if err != nil {
		return fmt.Errorf("new http request: %v", err)
	}
	htReq = htReq.WithContext(ctx)
	htReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Duration(HTTPTimeout) * time.Second}
	log.Debugf("%s - posting to agent #%d (%s)", req.UID, agent.ID, agent.name)
	log.Debug2f(">> %s@%s - pinged hosts: %s", req.UID, agent.name, strings.Join(req.Targets(), " "))
	resp, err := client.Do(htReq)
	if err != nil {
		return fmt.Errorf("http post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 202 {
		return fmt.Errorf("agent #%d (%s) rejected with code %d", agent.ID, agent.name, resp.StatusCode)
	}
	sqlExec(req.UID, "setDevLastPingedAt", setDevLastPingedAt, pq.Array(req.HostIDs()))
	return nil
}

// agentFromID return the agent in the agents list with the given ID
func agentFromID(agentID int, agents []Agent) Agent {
	for _, agent := range agents {
		if agent.ID == agentID {
			return agent
		}
	}
	return Agent{}
}

// leastLoadedAgent return the agent ID of the agentHosts map
// with the lowest number of hosts.
func leastLoadedAgent(agentHosts map[int][]model.PingHost) (int, int) {
	var minAgentID = -1
	var minCount = math.MaxInt32
	for agentID, hosts := range agentHosts {
		if len(hosts) < minCount {
			minCount = len(hosts)
			minAgentID = agentID
		}
	}
	return minAgentID, minCount
}
