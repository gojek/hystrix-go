package hystrix

import (
	"testing"
	"testing/synctest"
	"time"
)

func metricFailingPercent(circuitName string, p int) *metricExchange {
	m := newMetricExchange(circuitName)
	for i := 0; i < 100; i++ {
		t := "success"
		if i < p {
			t = "failure"
		}
		m.Update(commandExecution{PrimaryEvent: t})
	}

	return m
}

func TestErrorPercent(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()

		testErrorPercent(t, "test-error-percent-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testErrorPercent(t, "test-error-percent-sync")
			synctest.Wait()
		})
	})
}

func testErrorPercent(t *testing.T, circuitName string) {
	m := metricFailingPercent(circuitName, 40)
	now := time.Now()

	if p := m.ErrorPercent(now); p != 40 {
		t.Fatalf("expected error percent to be 40, got %v", p)
	}

	ConfigureCommand(circuitName, CommandConfig{ErrorPercentThreshold: 39})

	if m.IsHealthy(now) {
		t.Fatal("expected metrics to be unhealthy, but they are healthy")
	}
}
