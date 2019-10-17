// +build linux

package agent

import (
	"horus/log"
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
