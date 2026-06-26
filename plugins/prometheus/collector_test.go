package prometheus

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gojek/hystrix-go/hystrix"
	metricCollector "github.com/gojek/hystrix-go/hystrix/metric_collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRegisterCollector(t *testing.T) {
	t.Parallel()
	reg := prometheus.NewRegistry()

	RegisterCollector(reg)

	hystrix.ConfigureCommand("test_command", hystrix.CommandConfig{
		Timeout:                100,
		MaxConcurrentRequests:  1,
		RequestVolumeThreshold: 1,
		SleepWindow:            30,
		ErrorPercentThreshold:  0,
	})

	err := hystrix.Do("test_command", func() error {
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("hystrix.Do failed: %v", err)
	}

	assertMetrics(t, reg, map[string]string{
		"hystrix_attempts": `# HELP hystrix_attempts [counter] subsystem: hystrix, metric: attempts
# TYPE hystrix_attempts counter
hystrix_attempts{command_name="test_command"} 1
`,
		"hystrix_errors": `# HELP hystrix_errors [counter] subsystem: hystrix, metric: errors
# TYPE hystrix_errors counter
hystrix_errors{command_name="test_command"} 0
`,
		`hystrix_successes`: `# HELP hystrix_successes [counter] subsystem: hystrix, metric: successes
# TYPE hystrix_successes counter
hystrix_successes{command_name="test_command"} 1
`,
	})

	hystrixErr := errors.New("hystrix err")
	err = hystrix.Do("test_command", func() error {
		return hystrixErr
	}, nil)
	if !errors.Is(err, hystrixErr) {
		t.Fatalf("%v expected but got %v", hystrixErr, err)
	}

	assertMetrics(t, reg, map[string]string{
		"hystrix_attempts": `# HELP hystrix_attempts [counter] subsystem: hystrix, metric: attempts
# TYPE hystrix_attempts counter
hystrix_attempts{command_name="test_command"} 2
`,
		"hystrix_errors": `# HELP hystrix_errors [counter] subsystem: hystrix, metric: errors
# TYPE hystrix_errors counter
hystrix_errors{command_name="test_command"} 1
`,
		`hystrix_successes`: `# HELP hystrix_successes [counter] subsystem: hystrix, metric: successes
# TYPE hystrix_successes counter
hystrix_successes{command_name="test_command"} 1
`,
	})
}

func TestNewCollector(t *testing.T) {
	t.Parallel()
	reg := prometheus.NewRegistry()

	cf := NewCollector(reg, WithHistogramBuckets([]float64{.001, .0025, .005, .01}))
	result := metricCollector.MetricResult{
		Attempts:                1,
		Errors:                  4,
		Successes:               16,
		Failures:                64,
		Rejects:                 256,
		ShortCircuits:           1024,
		Timeouts:                4096,
		FallbackSuccesses:       16384,
		FallbackFailures:        65536,
		ContextCanceled:         262144,
		ContextDeadlineExceeded: 1048576,
		TotalDuration:           8 * time.Millisecond,
		RunDuration:             2 * time.Millisecond,
		ConcurrencyInUse:        0.25,
	}
	for i := range 3 {
		count := i + 1
		c := cf(`test` + strconv.Itoa(count))
		for range count {
			c.Update(result)
		}
	}

	assertMetrics(t, reg, map[string]string{
		`hystrix_attempts`: `# HELP hystrix_attempts [counter] subsystem: hystrix, metric: attempts
# TYPE hystrix_attempts counter
hystrix_attempts{command_name="test1"} 1
hystrix_attempts{command_name="test2"} 2
hystrix_attempts{command_name="test3"} 3
`,
		`hystrix_successes`: `# HELP hystrix_successes [counter] subsystem: hystrix, metric: successes
# TYPE hystrix_successes counter
hystrix_successes{command_name="test1"} 16
hystrix_successes{command_name="test2"} 32
hystrix_successes{command_name="test3"} 48
`,
		`hystrix_errors`: `# HELP hystrix_errors [counter] subsystem: hystrix, metric: errors
# TYPE hystrix_errors counter
hystrix_errors{command_name="test1"} 4
hystrix_errors{command_name="test2"} 8
hystrix_errors{command_name="test3"} 12
`,
		`hystrix_failures`: `# HELP hystrix_failures [counter] subsystem: hystrix, metric: failures
# TYPE hystrix_failures counter
hystrix_failures{command_name="test1"} 64
hystrix_failures{command_name="test2"} 128
hystrix_failures{command_name="test3"} 192
`,
		`hystrix_rejects`: `# HELP hystrix_rejects [counter] subsystem: hystrix, metric: rejects
# TYPE hystrix_rejects counter
hystrix_rejects{command_name="test1"} 256
hystrix_rejects{command_name="test2"} 512
hystrix_rejects{command_name="test3"} 768
`,
		`hystrix_short_circuits`: `# HELP hystrix_short_circuits [counter] subsystem: hystrix, metric: short_circuits
# TYPE hystrix_short_circuits counter
hystrix_short_circuits{command_name="test1"} 1024
hystrix_short_circuits{command_name="test2"} 2048
hystrix_short_circuits{command_name="test3"} 3072
`,
		`hystrix_timeouts`: `# HELP hystrix_timeouts [counter] subsystem: hystrix, metric: timeouts
# TYPE hystrix_timeouts counter
hystrix_timeouts{command_name="test1"} 4096
hystrix_timeouts{command_name="test2"} 8192
hystrix_timeouts{command_name="test3"} 12288
`,
		`hystrix_fallback_successes`: `# HELP hystrix_fallback_successes [counter] subsystem: hystrix, metric: fallback_successes
# TYPE hystrix_fallback_successes counter
hystrix_fallback_successes{command_name="test1"} 16384
hystrix_fallback_successes{command_name="test2"} 32768
hystrix_fallback_successes{command_name="test3"} 49152
`,
		`hystrix_fallback_failures`: `# HELP hystrix_fallback_failures [counter] subsystem: hystrix, metric: fallback_failures
# TYPE hystrix_fallback_failures counter
hystrix_fallback_failures{command_name="test1"} 65536
hystrix_fallback_failures{command_name="test2"} 131072
hystrix_fallback_failures{command_name="test3"} 196608
`,
		`hystrix_context_deadline_exceeded`: `# HELP hystrix_context_deadline_exceeded [counter] subsystem: hystrix, metric: context_deadline_exceeded
# TYPE hystrix_context_deadline_exceeded counter
hystrix_context_deadline_exceeded{command_name="test1"} 1.048576e+06
hystrix_context_deadline_exceeded{command_name="test2"} 2.097152e+06
hystrix_context_deadline_exceeded{command_name="test3"} 3.145728e+06
`,
		`hystrix_context_canceled`: `# HELP hystrix_context_canceled [counter] subsystem: hystrix, metric: context_canceled
# TYPE hystrix_context_canceled counter
hystrix_context_canceled{command_name="test1"} 262144
hystrix_context_canceled{command_name="test2"} 524288
hystrix_context_canceled{command_name="test3"} 786432
`,
		`hystrix_total_duration`: `# HELP hystrix_total_duration [histogram] subsystem: hystrix, metric: total_duration
# TYPE hystrix_total_duration histogram
hystrix_total_duration_bucket{command_name="test1",le="0.001"} 0
hystrix_total_duration_bucket{command_name="test1",le="0.0025"} 0
hystrix_total_duration_bucket{command_name="test1",le="0.005"} 0
hystrix_total_duration_bucket{command_name="test1",le="0.01"} 1
hystrix_total_duration_bucket{command_name="test1",le="+Inf"} 1
hystrix_total_duration_sum{command_name="test1"} 0.008
hystrix_total_duration_count{command_name="test1"} 1
hystrix_total_duration_bucket{command_name="test2",le="0.001"} 0
hystrix_total_duration_bucket{command_name="test2",le="0.0025"} 0
hystrix_total_duration_bucket{command_name="test2",le="0.005"} 0
hystrix_total_duration_bucket{command_name="test2",le="0.01"} 2
hystrix_total_duration_bucket{command_name="test2",le="+Inf"} 2
hystrix_total_duration_sum{command_name="test2"} 0.016
hystrix_total_duration_count{command_name="test2"} 2
hystrix_total_duration_bucket{command_name="test3",le="0.001"} 0
hystrix_total_duration_bucket{command_name="test3",le="0.0025"} 0
hystrix_total_duration_bucket{command_name="test3",le="0.005"} 0
hystrix_total_duration_bucket{command_name="test3",le="0.01"} 3
hystrix_total_duration_bucket{command_name="test3",le="+Inf"} 3
hystrix_total_duration_sum{command_name="test3"} 0.024
hystrix_total_duration_count{command_name="test3"} 3
`,
		`hystrix_run_duration`: `# HELP hystrix_run_duration [histogram] subsystem: hystrix, metric: run_duration
# TYPE hystrix_run_duration histogram
hystrix_run_duration_bucket{command_name="test1",le="0.001"} 0
hystrix_run_duration_bucket{command_name="test1",le="0.0025"} 1
hystrix_run_duration_bucket{command_name="test1",le="0.005"} 1
hystrix_run_duration_bucket{command_name="test1",le="0.01"} 1
hystrix_run_duration_bucket{command_name="test1",le="+Inf"} 1
hystrix_run_duration_sum{command_name="test1"} 0.002
hystrix_run_duration_count{command_name="test1"} 1
hystrix_run_duration_bucket{command_name="test2",le="0.001"} 0
hystrix_run_duration_bucket{command_name="test2",le="0.0025"} 2
hystrix_run_duration_bucket{command_name="test2",le="0.005"} 2
hystrix_run_duration_bucket{command_name="test2",le="0.01"} 2
hystrix_run_duration_bucket{command_name="test2",le="+Inf"} 2
hystrix_run_duration_sum{command_name="test2"} 0.004
hystrix_run_duration_count{command_name="test2"} 2
hystrix_run_duration_bucket{command_name="test3",le="0.001"} 0
hystrix_run_duration_bucket{command_name="test3",le="0.0025"} 3
hystrix_run_duration_bucket{command_name="test3",le="0.005"} 3
hystrix_run_duration_bucket{command_name="test3",le="0.01"} 3
hystrix_run_duration_bucket{command_name="test3",le="+Inf"} 3
hystrix_run_duration_sum{command_name="test3"} 0.006
hystrix_run_duration_count{command_name="test3"} 3
`,
		`hystrix_concurrency_in_use`: `# HELP hystrix_concurrency_in_use [gauge] subsystem: hystrix, metric: concurrency_in_use
# TYPE hystrix_concurrency_in_use gauge
hystrix_concurrency_in_use{command_name="test1"} 0.25
hystrix_concurrency_in_use{command_name="test2"} 0.25
hystrix_concurrency_in_use{command_name="test3"} 0.25
`,
	})
}

func assertMetrics(t *testing.T, g prometheus.Gatherer, metrics map[string]string) {
	t.Helper()
	for name, value := range metrics {
		err := testutil.GatherAndCompare(g, strings.NewReader(value), name)
		if err != nil {
			t.Errorf("GatherAndCompare failed for metric %s: %v", name, err)
		}
	}
}
