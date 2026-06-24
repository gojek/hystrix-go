package hystrix

import (
	"testing"
	"time"

	"github.com/gojek/hystrix-go/hystrix/internal/test"
)

func TestReturn(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		pool := newExecutorPool("pool-return-sync")
		ticket := <-pool.Tickets
		pool.Return(ticket)

		if val := pool.Executed.Sum(time.Now()); val != 1 {
			t.Fatalf("expected 1 executed request, got %v", val)
		}
	})
}

func TestActiveCount(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		// when 3 tickets are pulled
		pool := newExecutorPool("pool-activecount-sync")
		<-pool.Tickets
		<-pool.Tickets
		ticket := <-pool.Tickets

		if val := pool.ActiveCount(); val != 3 {
			t.Errorf("expected 3 active requests, got %v", val)
		}

		pool.Return(ticket)
		if val := pool.MaxActiveRequests.Max(time.Now()); val != 3 {
			t.Errorf("expected 3 max requests, got %v", val)
		}
	})
}
