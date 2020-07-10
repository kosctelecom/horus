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

// +build linux

package agent

import (
	"runtime"
	"syscall"
	"time"

	"github.com/kosctelecom/horus/log"
)

var (
	totalMem             float64
	totalMemSamplingFreq = 30 * time.Minute

	usedMem             float64
	usedMemSamplingFreq = 10 * time.Second
)

func updateTotalMem() {
	ticker := time.NewTicker(totalMemSamplingFreq)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		log.Debug2(">> querying sys total mem stats")
		in := &syscall.Sysinfo_t{}
		if err := syscall.Sysinfo(in); err != nil {
			log.Errorf("sysinfo: %v", err)
		}
		totalMem = float64(in.Totalram) * float64(in.Unit)
	}
}

func updateUsedMem() {
	ticker := time.NewTicker(usedMemSamplingFreq)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		log.Debug2(">> querying heap mem stats")
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		usedMem = float64(m.HeapAlloc)
	}
}

// CurrentMemLoad returns the current relative memory usage of the agent.
func CurrentMemLoad() float64 {
	if totalMem == 0 {
		return -1
	}
	return usedMem / totalMem
}
