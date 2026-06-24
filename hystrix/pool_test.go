package hystrix

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestReturn(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testReturn(t, "pool-return-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			t.Skip(`TODO: fix me`)

			testReturn(t, "pool-return-sync")
			synctest.Wait()
		})
	})
}

func testReturn(t *testing.T, circuitName string) {
	pool := newExecutorPool(circuitName)
	ticket := <-pool.Tickets
	pool.Return(ticket)
	time.Sleep(1 * time.Millisecond)
	// total executed requests should increment
	if val := pool.Metrics.Executed.Sum(time.Now()); val != 1 {
		t.Fatalf("expected 1 executed request, got %v", val)
	}
}

func TestActiveCount(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testActiveCount(t, "pool-activecount-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			t.Skip(`TODO: fix me`)

			testActiveCount(t, "pool-activecount-sync")
			synctest.Wait()
		})
	})
}

func testActiveCount(t *testing.T, circuitName string) {
	// when 3 tickets are pulled
	pool := newExecutorPool(circuitName)
	<-pool.Tickets
	<-pool.Tickets
	ticket := <-pool.Tickets

	if val := pool.ActiveCount(); val != 3 {
		t.Errorf("expected 3 active requests, got %v", val)
	}

	pool.Return(ticket)
	time.Sleep(1 * time.Millisecond) // allow poolMetrics to process channel
	if val := pool.Metrics.MaxActiveRequests.Max(time.Now()); val != 3 {
		t.Errorf("expected 3 max requests, got %v", val)
	}
}
