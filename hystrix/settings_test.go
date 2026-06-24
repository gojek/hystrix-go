package hystrix

import (
	"testing"
	"time"

	"github.com/gojek/hystrix-go/hystrix/internal/test"
)

func TestConfigureConcurrency(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		const circuitName = "configure-concurrency"
		ConfigureCommand(circuitName, CommandConfig{MaxConcurrentRequests: 100})

		if val := getSettings(circuitName).MaxConcurrentRequests; val != 100 {
			t.Fatalf("expected MaxConcurrentRequests to be 100, got %d", val)
		}
	})
}

func TestConfigureTimeout(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		const circuitName = "configure-timeout"
		ConfigureCommand(circuitName, CommandConfig{Timeout: 10000})

		if val := getSettings(circuitName).Timeout; val != 10*time.Second {
			t.Fatalf("expected MaxConcurrentRequests to be 10s, got %s", val)
		}
	})
}

func TestConfigureRVT(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		const circuitName = "configure-RVT"
		ConfigureCommand(circuitName, CommandConfig{RequestVolumeThreshold: 30})

		if val := getSettings(circuitName).RequestVolumeThreshold; val != 30 {
			t.Fatalf("expected MaxConcurrentRequests to be 30, got %d", val)
		}
	})
}

func TestSleepWindowDefault(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		const circuitName = "sleepwindow-default"
		ConfigureCommand(circuitName, CommandConfig{})

		if val := getSettings(circuitName).SleepWindow; val != 5*time.Second {
			t.Fatalf("expected MaxConcurrentRequests to be 5s, got %s", val)
		}
	})
}

func TestGetCircuitSettings(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		const circuitName = "GetCircuitSettings"
		ConfigureCommand(circuitName, CommandConfig{Timeout: 30000})

		settings := getSettings(circuitName)
		if val := settings.Timeout; val != 30*time.Second {
			t.Fatalf("expected MaxConcurrentRequests to be 30s, got %s", val)
		}

		settings2 := GetCircuitSettings()[circuitName]
		if settings2 != settings {
			t.Fatalf("expected settings to be %v, got %v", settings, settings2)
		}
	})
}
