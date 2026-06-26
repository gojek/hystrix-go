package hystrix

import (
	"testing"
	"time"

	"github.com/gojek/hystrix-go/hystrix/internal/test"
)

func metricFailingPercent(circuitName string, p int) *metricExchange {
	m := newMetricExchange(circuitName)
	for i := range 100 {
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
	test.SyncTest(t, func(t *testing.T) {
		const circuitName = "test-error-percent"
		m := metricFailingPercent(circuitName, 40)
		now := time.Now()

		if p := m.ErrorPercent(now); p != 40 {
			t.Fatalf("expected error percent to be 40, got %v", p)
		}

		ConfigureCommand(circuitName, CommandConfig{ErrorPercentThreshold: 39})

		if m.IsHealthy(now) {
			t.Fatal("expected metrics to be unhealthy, but they are healthy")
		}
	})
}
