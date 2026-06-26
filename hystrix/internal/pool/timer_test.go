package pool

import (
	"sync"
	"testing"
	"time"

	"github.com/gojek/hystrix-go/hystrix/internal/test"
)

func TestAcquiredTimerFiresAfterGivenDuration(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		d := 50 * time.Millisecond
		timer := AcquireTimer(d)
		defer ReleaseTimer(timer)

		select {
		case <-timer.C:
		case <-time.After(2 * d):
			t.Fatal("timer did not fire within expected duration")
		}
	})
}

func TestAcquiredTimerDoesNotFireBeforeDuration(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		timer := AcquireTimer(200 * time.Millisecond)
		defer ReleaseTimer(timer)

		select {
		case <-timer.C:
			t.Fatal("timer fired too early")
		case <-time.After(10 * time.Millisecond):
		}
	})
}

func TestReusedTimerAfterReleaseFiresAfterNewDuration(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		first := AcquireTimer(10 * time.Millisecond)
		<-first.C
		ReleaseTimer(first)

		d := 50 * time.Millisecond
		start := time.Now()
		second := AcquireTimer(d)
		defer ReleaseTimer(second)

		select {
		case <-second.C:
			elapsed := time.Since(start)
			if elapsed < d {
				t.Fatalf("timer triggered in %s instead of %s", elapsed, d)
			}
		case <-time.After(2 * d):
			t.Fatal("reused timer did not fire within expected duration")
		}
	})
}

func TestReusedActiveTimerIsResetToNewDuration(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		first := AcquireTimer(10 * time.Second)
		ReleaseTimer(first)

		d := 50 * time.Millisecond
		second := AcquireTimer(d)
		defer ReleaseTimer(second)

		select {
		case <-second.C:
		case <-time.After(2 * d):
			t.Fatal("reused active timer was not reset to new duration")
		}
	})
}

func TestReleasingExpiredTimerChannelHasNoStaleValue(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		timer := AcquireTimer(1 * time.Millisecond)
		<-timer.C
		ReleaseTimer(timer)

		reacquired := AcquireTimer(10 * time.Second)
		defer ReleaseTimer(reacquired)

		select {
		case <-reacquired.C:
			t.Fatal("reacquired timer fired immediately; stale value in channel")
		case <-time.After(20 * time.Millisecond):
		}
	})
}

func TestConcurrentAcquireAndReleaseTimersDoNotRace(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(_ *testing.T) {
		var wg sync.WaitGroup
		for range 100 {
			wg.Go(func() {
				timer := AcquireTimer(10 * time.Millisecond)
				<-timer.C
				ReleaseTimer(timer)
			})
			wg.Go(func() {
				ReleaseTimer(AcquireTimer(10 * time.Millisecond))
			})
		}
		wg.Wait()
	})
}
