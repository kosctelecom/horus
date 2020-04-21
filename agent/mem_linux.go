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
	"horus/log"
	"runtime"
	"syscall"
)

// sysTotalMemory returns the total system memory on linux.
func sysTotalMemory() uint64 {
	in := &syscall.Sysinfo_t{}
	if err := syscall.Sysinfo(in); err != nil {
		log.Errorf("sysTotalMemory: %v", err)
		return 0
	}
	return uint64(in.Totalram) * uint64(in.Unit)
}

// CurrentLoad returns the current relative memory usage of the agent.
func CurrentLoad() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	used, total := float64(m.HeapSys-m.HeapReleased), float64(sysTotalMemory())
	if total == 0 {
		return 0
	}
	return used / total
}
