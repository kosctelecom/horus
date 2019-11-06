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
	"fmt"
	"horus/log"
	"horus/model"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Agent represents an snmp agent
type Agent struct {
	// ID is the agent id
	ID int `db:"id"`

	// Host is the agent web server IP address
	Host string `db:"ip_address"`

	// Port is the agent web server listen port
	Port int `db:"port"`

	// Alive indicates wether this agent responds to keep-alives
	Alive bool `db:"is_alive"`

	// name is the agent's unique name (ip:port)
	name string

	// snmpJobURL is the full url for posting agent's snmp jobs
	snmpJobURL string

	// checkURL is the full url for pinging this agent
	checkURL string

	// pingJobURL is the full url for posting agent's ping jobs
	pingJobURL string

	// last agent loads, used for load avg calculation. The key is the timestamp returned by time.UnixNano().
	// On each job post or keepalive, a new value is added and the entries older than LoadAvgWindow are purged.
	lastLoads map[int64]float64

	lastLoadsMu sync.Mutex

	// loadAvg is the load average taken over LoadAvgWindow.
	loadAvg float64
}

// Agents is a map of Agent pointers with agent name as key.
type Agents map[string]*Agent

var (
	// MaxLoadDelta is the maximum average load difference allowed between agents
	// before moving a device to the least loaded agent.
	MaxLoadDelta = 0.1

	// LoadAvgWindow is the window for agent load average calculation.
	LoadAvgWindow = time.Minute

	// currentAgents is the list of currently active agents in memory.
	currentAgents   = make(Agents)
	currentAgentsMu sync.RWMutex

	// jobDistrib maps a device to an agent (dev id => agent name)
	jobDistrib   = make(map[int]string)
	jobDistribMu sync.RWMutex
)

// ByLoad is an Agent slice implementing Sort interface
// for sorting by average load.
type ByLoad []*Agent

func (a ByLoad) Len() int           { return len(a) }
func (a ByLoad) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLoad) Less(i, j int) bool { return a[i].loadAvg < a[j].loadAvg }

// AgentsForDevice return a list of agents to which send a polling request by order
// of priority. We try to be sticky as much as possible but with balanced load:
// - the current list of active agents is sorted by load
// - if the device is not in jobDistrib map, this list is returned as is.
// - if the device is in jobDistrib map and its associated agent is active,
//   - if the load difference between the associated agent and the least
//     loaded agent is under the MaxLoadDelta, we stick to this agent: a
//     modified load-sorted list is returned where this agent is moved at
//     the first position.
//   - if the load difference exceeds MaxLoadDelta, we rebalance the
//   load: the load sorted list is returned.
func AgentsForDevice(devID int) []*Agent {
	var workingAgents []*Agent
	currAgents := currentAgentsCopy()
	for k, a := range currAgents {
		if a.Alive {
			workingAgents = append(workingAgents, currAgents[k])
		}
	}
	if len(workingAgents) == 0 {
		return nil
	}

	log.Debug3f(">> dev#%d: working agents: %+v", devID, workingAgents)
	sort.Sort(ByLoad(workingAgents))
	jobDistribMu.RLock()
	index := getAgentIndex(jobDistrib[devID], workingAgents)
	jobDistribMu.RUnlock()
	if index == -1 {
		// previously used agent is not in list: send load sorted list
		log.Debug2f(">> dev#%d: not in job list, req sent to load sorted agents [%s,...] ", devID, workingAgents[0])
		return workingAgents
	}
	// previously used agent is in list
	agent := workingAgents[index]
	loadDelta := agent.loadAvg - workingAgents[0].loadAvg
	if loadDelta <= MaxLoadDelta {
		// acceptable load delta, use same agent first
		workingAgents = append(workingAgents[:index], workingAgents[index+1:]...) // remove
		workingAgents = append([]*Agent{agent}, workingAgents...)                 // unshift
		log.Debug2f(">> dev#%d: stick to prev (%s), delta=%.2f", devID, agent, loadDelta)
	} else {
		log.Debug2f(">> dev#%d: req sent to load sorted agents [%s,...], delta=%.2f", devID, workingAgents[0], loadDelta)
	}
	return workingAgents
}

// CheckAgents sends a keepalive to each agent
// and updates its status & current load.
func CheckAgents() error {
	log.Debug2("start checking agents")

	// make a local copy as check reply can be long
	agents := currentAgentsCopy()
	deadAgents := make(Agents)
	for _, agent := range agents {
		isAlive, load := agent.Check()
		agent.Alive = isAlive
		agent.setLoad(load)
		log.Debugf("agent #%d (%s:%d): alive=%v load=%.2f loadAvg=%.2f", agent.ID, agent.Host, agent.Port, isAlive, load, agent.loadAvg)
		sqlExec("agent #"+strconv.Itoa(agent.ID), "checkAgentStmt", checkAgentStmt, agent.ID, isAlive, agent.loadAvg)
		if !isAlive {
			// unlock all devices locked on a failed agent
			sqlExec("agent #"+strconv.Itoa(agent.ID), "unlockFromAgent", unlockFromAgentStmt, agent.ID)
			deadAgents[agent.name] = agent
		}
	}
	jobDistribMu.Lock()
	defer jobDistribMu.Unlock()
	for devID, agentName := range jobDistrib {
		// remove all mappings to dead agents
		if _, ok := deadAgents[agentName]; ok {
			delete(jobDistrib, devID)
		}
	}
	log.Debug2("done checking agents")
	return nil
}

// Check pings an agent and returns its active status and ongoing polls count.
// The check is a http query to the agents checkURL which returns a status 200 OK and
// the current load in body when it is healthy.
func (a Agent) Check() (bool, float64) {
	log.Debug2f("checking agent #%d", a.ID)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(a.checkURL)
	if err != nil {
		log.Debug2f("check agent %s: %v", a.name, err)
		return false, 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Warningf("agent #%d responded to check with %s", a.ID, resp.Status)
		return false, 0
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("agent #%d: read check reply read: %v", a.ID, err)
		return false, 0
	}
	load, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		log.Errorf("agent #%d: reply parse: %v", a.ID, err)
	}
	return true, load
}

// String implements the stringer interface for the Agent type.
func (a Agent) String() string {
	return fmt.Sprintf("Agent<id:%d name:%s:%d load:%.4f>", a.ID, a.Host, a.Port, a.loadAvg)
}

// ActiveAgentCount returns the number of current active agents.
func ActiveAgentCount() int {
	count := 0
	currentAgentsMu.RLock()
	defer currentAgentsMu.RUnlock()
	for _, agent := range currentAgents {
		if agent.Alive {
			count++
		}
	}
	return count
}

// LoadAgents retrieves agent list from db and updates the current agent list
// (removes deleted, adds new).
// Note: key for comparision is agent name (host:port).
func LoadAgents() error {
	var agents []Agent
	err := db.Select(&agents, `SELECT id,ip_address,port,is_alive
                                 FROM agents
                                WHERE active = true
                             ORDER BY load`)
	if err != nil {
		return fmt.Errorf("load agents: %v", err)
	}
	log.Debugf("got %d agents from db", len(agents))
	newAgents := make(Agents)
	for _, a := range agents {
		a := a // !!shadowing needed for last assignment
		a.snmpJobURL = fmt.Sprintf("http://%s:%d%s", a.Host, a.Port, model.SnmpJobURI)
		a.checkURL = fmt.Sprintf("http://%s:%d%s", a.Host, a.Port, model.CheckURI)
		a.pingJobURL = fmt.Sprintf("http://%s:%d%s", a.Host, a.Port, model.PingJobURI)
		a.name = fmt.Sprintf("%s:%d", a.Host, a.Port)
		newAgents[a.name] = &a
	}
	log.Debug2f(">> LoadAgents: new agents = %+v", newAgents)

	agentsCopy := currentAgentsCopy() // copy holds a rlock, must be called outside of next line lock
	currentAgentsMu.Lock()
	defer currentAgentsMu.Unlock()
	for k := range agentsCopy {
		if _, ok := newAgents[k]; !ok {
			delete(currentAgents, k)
		} else {
			delete(newAgents, k)
		}
	}
	for k, a := range newAgents {
		currentAgents[k] = a
	}
	log.Debug2f(">> LoadAgents: curr agents = %+v", currentAgents)
	return nil
}

// setLoad saves agents last instataneous load, updates its
// load average and removes old load samples.
func (a *Agent) setLoad(load float64) {
	a.lastLoadsMu.Lock()
	defer a.lastLoadsMu.Unlock()

	if !a.Alive {
		a.lastLoads = nil
		a.loadAvg = 0
		return
	}

	var acc float64
	if a.lastLoads == nil {
		a.lastLoads = make(map[int64]float64)
	}
	now := time.Now().UnixNano()
	a.lastLoads[now] = load
	for ts, load := range a.lastLoads {
		if ts < now-int64(LoadAvgWindow) {
			delete(a.lastLoads, ts)
		} else {
			acc += load
		}
	}
	a.loadAvg = acc / float64(len(a.lastLoads))
}

// currentAgentsCopy makes a locked copy of current agents map.
func currentAgentsCopy() Agents {
	cpy := make(Agents)
	currentAgentsMu.RLock()
	defer currentAgentsMu.RUnlock()
	for k, a := range currentAgents {
		cpy[k] = a
	}
	return cpy
}

// getAgentIndex returns the index at which the agent with `name`
// is in the `agents` array. Returns -1 if not found.
func getAgentIndex(name string, agents []*Agent) int {
	for i, a := range agents {
		if a.name == name {
			return i
		}
	}
	return -1
}
