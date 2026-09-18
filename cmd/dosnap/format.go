package main

import (
	"fmt"
	"time"
)

func formatElapsed(d time.Duration) string {
	if d < time.Second {
		ms := d.Milliseconds()
		if ms < 0 {
			ms = 0
		}
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func formatLine(tag string, n int, bytes uint64, d time.Duration) string {
	return fmt.Sprintf("%s %d volumes %dB %s", tag, n, bytes, formatElapsed(d))
}
