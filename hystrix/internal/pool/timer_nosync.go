//go:build !synctest

package pool

import (
	"sync"
	"time"
)

var timerPool sync.Pool

func AcquireTimer(d time.Duration) *time.Timer {
	t := timerPool.Get()
	if timer, ok := t.(*time.Timer); ok {
		timer.Reset(d)

		return timer
	}

	return time.NewTimer(d)
}

func ReleaseTimer(t *time.Timer) {
	_ = t.Stop() // from go 1.23 onwards t.C no longer has stale data once execution of Stop() is finished.
	timerPool.Put(t)
}
