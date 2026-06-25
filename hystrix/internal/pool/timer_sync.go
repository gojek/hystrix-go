//go:build synctest

package pool

import (
	"time"
)

// avoid reusing timer across synctest bubble
func AcquireTimer(d time.Duration) *time.Timer {
	return time.NewTimer(d)
}

func ReleaseTimer(_ *time.Timer) {}
