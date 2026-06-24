package hystrix

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestConfigureConcurrency(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testConfigureConcurrency(t, "configure-concurrency-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testConfigureConcurrency(t, "configure-concurrency-sync")
			synctest.Wait()
		})
	})
}

func testConfigureConcurrency(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{MaxConcurrentRequests: 100})

	if val := getSettings(circuitName).MaxConcurrentRequests; val != 100 {
		t.Fatalf("expected MaxConcurrentRequests to be 100, got %d", val)
	}
}

func TestConfigureTimeout(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testConfigureTimeout(t, "configure-timeout-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testConfigureTimeout(t, "configure-timeout-sync")
			synctest.Wait()
		})
	})
}

func testConfigureTimeout(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{Timeout: 10000})

	if val := getSettings(circuitName).Timeout; val != 10*time.Second {
		t.Fatalf("expected MaxConcurrentRequests to be 10s, got %s", val)
	}
}

func TestConfigureRVT(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testConfigureRVT(t, "configure-RVT-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testConfigureRVT(t, "configure-RVT-sync")
			synctest.Wait()
		})
	})
}

func testConfigureRVT(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{RequestVolumeThreshold: 30})

	if val := getSettings(circuitName).RequestVolumeThreshold; val != 30 {
		t.Fatalf("expected MaxConcurrentRequests to be 30, got %d", val)
	}
}

func TestSleepWindowDefault(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testSleepWindowDefault(t, "sleepwindow-default-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testSleepWindowDefault(t, "sleepwindow-default-sync")
			synctest.Wait()
		})
	})
}

func testSleepWindowDefault(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{})

	if val := getSettings(circuitName).SleepWindow; val != 5*time.Second {
		t.Fatalf("expected MaxConcurrentRequests to be 5s, got %s", val)
	}
}

func TestGetCircuitSettings(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testGetCircuitSettings(t, "GetCircuitSettings-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testGetCircuitSettings(t, "GetCircuitSettings-sync")
			synctest.Wait()
		})
	})
}

func testGetCircuitSettings(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{Timeout: 30000})

	settings := getSettings(circuitName)
	if val := settings.Timeout; val != 30*time.Second {
		t.Fatalf("expected MaxConcurrentRequests to be 30s, got %s", val)
	}

	settings2 := GetCircuitSettings()[circuitName]
	if settings2 != settings {
		t.Fatalf("expected settings to be %v, got %v", settings, settings2)
	}
}
