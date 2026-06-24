package pool

import (
	"errors"
	"testing"

	"github.com/gojek/hystrix-go/hystrix/internal/test"
)

func TestAcquiredChannelIsBufferedWithCapacityOne(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		ch := AcquireSingleErrorChan()
		defer ReleaseSingleErrorChan(ch)

		if cap(ch) != 1 {
			t.Fatalf("expected channel capacity 1, got %d", cap(ch))
		}
	})
}

func TestAcquiredChannelIsInitiallyEmpty(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		ch := AcquireSingleErrorChan()
		defer ReleaseSingleErrorChan(ch)

		if len(ch) != 0 {
			t.Fatalf("expected empty channel, got length %d", len(ch))
		}
	})
}

func TestAcquiredChannelCanSendAndReceiveError(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		ch := AcquireSingleErrorChan()
		defer ReleaseSingleErrorChan(ch)

		sentinel := errors.New("sentinel error")
		ch <- sentinel

		got := <-ch
		if got != sentinel {
			t.Fatalf("expected %v, got %v", sentinel, got)
		}
	})
}

func TestReleasedChannelIsReturnedBySubsequentAcquire(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		first := AcquireSingleErrorChan()
		ReleaseSingleErrorChan(first)

		second := AcquireSingleErrorChan()
		defer ReleaseSingleErrorChan(second)

		if cap(second) != 1 {
			t.Fatalf("expected reused channel to have capacity 1, got %d", cap(second))
		}
	})
}

func TestReleasedChannelWithUnreadErrorIsNotReused(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		ch := AcquireSingleErrorChan()
		ch <- errors.New("leftover error")
		<-ch // drain before release
		ReleaseSingleErrorChan(ch)

		fresh := AcquireSingleErrorChan()
		defer ReleaseSingleErrorChan(fresh)

		if len(fresh) != 0 {
			t.Fatalf("expected fresh empty channel after release with unread error, got length %d", len(fresh))
		}
	})
}

func TestConcurrentAcquireAndReleaseDoNotRace(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		const goroutines = 50
		done := make(chan struct{})

		for range goroutines {
			go func() {
				ch := AcquireSingleErrorChan()
				ReleaseSingleErrorChan(ch)
				done <- struct{}{}
			}()
		}

		for range goroutines {
			<-done
		}
	})
}

func TestMultipleAcquiresReturnIndependentChannels(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		ch1 := AcquireSingleErrorChan()
		ch2 := AcquireSingleErrorChan()
		defer ReleaseSingleErrorChan(ch1)
		defer ReleaseSingleErrorChan(ch2)

		sentinel := errors.New("only for ch1")
		ch1 <- sentinel

		if len(ch2) != 0 {
			t.Fatal("expected ch2 to be unaffected by send to ch1")
		}
		<-ch1
	})
}
