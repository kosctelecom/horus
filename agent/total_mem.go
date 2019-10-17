// +build !linux

package agent

// sysTotalMemory returns 0 on non linux systems.
func sysTotalMemory() uint64 {
	return 0
}
