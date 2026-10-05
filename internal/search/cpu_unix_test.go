//go:build !windows

package search

import (
	"syscall"
	"time"
)

// cpuTime es el tiempo de CPU (usuario + sistema) que ha gastado este proceso.
func cpuTime() (time.Duration, bool) {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0, false
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano()), true
}
