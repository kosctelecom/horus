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
	"context"
	"horus/log"
	"horus/model"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// SendPollingJobs retrieves all available snmp polling jobs and sends each of them sequentially
// to the next available agent. If the request cannot be sent to an agent, we try the next one
// until the last. If no agent accepts the job, it is discarded.
func SendPollingJobs(ctx context.Context) {
	if ActiveAgentCount() == 0 {
		log.Debug("no active agent, skipping this round...")
		return
	}
	log.Debug("poll: getting snmp jobs")
	jobs, err := SnmpJobs()
	if err != nil {
		log.Errorf("snmp jobs: %v", err)
		return
	}
	if len(jobs) == 0 {
		log.Debug("no new snmp jobs available")
		return
	}
	var accepted, discarded int64
	var wg sync.WaitGroup
	for j, id := range jobs {
		req, err := RequestWithLock(id)
		if err != nil {
			log.Errorf("dev #%d: request with lock: %v", id, err)
			continue
		}
		log.Debugf("%s - new req for job #%d, device #%d", req.UID, j+1, id)
		wg.Add(1)
		go func(ctx context.Context, req model.SnmpRequest) {
			defer wg.Done()
			var statusCode int
			log.Debug2f("%s - start sending request", req.UID)
			defer log.Debug2f("%s - request treated", req.UID)
			if req.ScalarMeasures == nil && req.IndexedMeasures == nil {
				log.Debugf("%s - no measure defined for device, skipping", req.UID)
				sqlExec(req.UID, "unlockDevStmt", unlockDevStmt, req.Device.ID)
				updateLastPolledAt(req)
				return
			}
			agents := AgentsForDevice(req.Device.ID)
			for i, agent := range agents {
				log.Debug2f("%s - try #%d: sending req to agent #%d (%s)", req.UID, i, agent.ID, agent.name)
				code, load, err := SendRequest(ctx, req, *agent)
				log.Debug2f("%s - try #%d: agent #%d replied %d", req.UID, i, agent.ID, code)
				statusCode = code
				status := http.StatusText(code)
				if err != nil {
					log.Errorf("%s - try #%d: send request: %v", req.UID, i, err)
					continue
				}
				switch code {
				case http.StatusAccepted:
					log.Debug2f(">>%s - inserting req entry", req.UID)
					sqlExec(req.UID, "insertReportStmt", insertReportStmt, req.UID, req.Device.ID, agent.ID, status)
					log.Debug2f(">>%s - updating dev last poll time", req.UID)
					updateLastPolledAt(req)
					log.Debug2f(">>%s - lock-updating agent load", req.UID)
					currentAgentsMu.Lock()
					log.Debug2f(">>>%s - before updating load: agent %s, load: avg=%.4f (%d entries)", req.UID, agent.name, agent.loadAvg, len(agent.lastLoads))
					agent.setLoad(load)
					log.Debug2f(">>>%s - after setting load: agent %s, load: last=%.2f avg=%.4f (%d entries)", req.UID, agent.name, load, agent.loadAvg, len(agent.lastLoads))
					currentAgentsMu.Unlock()
					log.Debug2f(">>%s - lock-updating job distrib map", req.UID)
					jobDistribMu.Lock()
					jobDistrib[req.Device.ID] = agent.name
					jobDistribMu.Unlock()
					log.Debug2f(">>%s - atomic increasing accepted count", req.UID)
					atomic.AddInt64(&accepted, 1)
					log.Debug2f("%s - request sent to agent #%d (load: %.4f)", req.UID, agent.ID, load)
					return
				case http.StatusTooManyRequests:
					log.Debugf("%s - agent #%d is full", req.UID, agent.ID)
					continue // try next
				case http.StatusLocked:
					log.Debugf("%s - agent #%d is terminating", req.UID, agent.ID)
					continue
				default:
					log.Warningf("%s - agent #%d replied `%s`", req.UID, agent.ID, status)
				}
			}
			if statusCode != http.StatusAccepted {
				log.Warningf("%s - polling job discarded (no worker found)", req.UID)
				atomic.AddInt64(&discarded, 1)
				sqlExec(req.UID, "unlockDevStmt", unlockDevStmt, req.Device.ID)
			}
		}(ctx, req)
		select {
		case <-ctx.Done():
			log.Debugf("cancelling all unposted jobs after #%d...", j+1)
			for _, devID := range jobs[j+1:] {
				log.Debug2f("unlocking dev #%d", devID)
				sqlExec("dev#"+strconv.Itoa(devID), "unlockDevStmt", unlockDevStmt, devID)
			}
			return
		case <-time.After(50 * time.Millisecond):
			// wait a few moments to avoid flooding the agents
		}
	}
	wg.Wait()
	log.Debugf("processed %d job(s): accepted=%d discarded=%d", len(jobs), accepted, discarded)
	if discarded > 0 {
		log.Warningf("not enough snmp workers available for %d jobs", discarded)
	}
}
