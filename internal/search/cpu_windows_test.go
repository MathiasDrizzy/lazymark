//go:build windows

package search

import "time"

func cpuTime() (time.Duration, bool) { return 0, false }
