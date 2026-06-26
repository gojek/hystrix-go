package plugins

import (
	metricCollector "github.com/gojek/hystrix-go/hystrix/metric_collector"
	"github.com/gojek/hystrix-go/plugins/datadog"
)

const (
	// Deprecated: use datadog.CircuitOpen instead
	//
	//go:fix inline
	DM_CircuitOpen = datadog.CircuitOpen
	// Deprecated: use datadog.Attempts instead
	//
	//go:fix inline
	DM_Attempts = datadog.Attempts
	// Deprecated: use datadog.Errors instead
	//
	//go:fix inline
	DM_Errors = datadog.Errors
	// Deprecated: use datadog.Successes instead
	//
	//go:fix inline
	DM_Successes = datadog.Successes
	// Deprecated: use datadog.Failures instead
	//
	//go:fix inline
	DM_Failures = datadog.Failures
	// Deprecated: use datadog.Rejects instead
	//
	//go:fix inline
	DM_Rejects = datadog.Rejects
	// Deprecated: use datadog.ShortCircuits instead
	//
	//go:fix inline
	DM_ShortCircuits = datadog.ShortCircuits
	// Deprecated: use datadog.Timeouts instead
	//
	//go:fix inline
	DM_Timeouts = datadog.Timeouts
	// Deprecated: use datadog.FallbackSuccesses instead
	//
	//go:fix inline
	DM_FallbackSuccesses = datadog.FallbackSuccesses
	// Deprecated: use datadog.FallbackFailures instead
	//
	//go:fix inline
	DM_FallbackFailures = datadog.FallbackFailures
	// Deprecated: use datadog.TotalDuration instead
	//
	//go:fix inline
	DM_TotalDuration = datadog.TotalDuration
	// Deprecated: use datadog.RunDuration instead
	//
	//go:fix inline
	DM_RunDuration = datadog.RunDuration
)

type (
	// DatadogClient is the minimum interface needed by
	// NewDatadogCollectorWithClient
	//
	// Deprecated: use datadog.Client instead
	//
	//go:fix inline
	DatadogClient = datadog.Client

	// DatadogCollector fulfills the metricCollector interface allowing users to
	// ship circuit stats to Datadog.
	//
	// Deprecated: use datadog.Collector instead
	//
	//go:fix inline
	DatadogCollector = datadog.Collector
)

// NewDatadogCollector creates a collector for a specific circuit with a
// "github.com/DataDog/datadog-go/statsd".(*Client).
//
// addr is in the format "<host>:<port>" (e.g. "localhost:8125")
//
// Deprecated: use datadog.NewCollector instead
//
//go:fix inline
func NewDatadogCollector(addr, prefix string) (func(string) metricCollector.MetricCollector, error) {
	return datadog.NewCollector(addr, prefix)
}

// NewDatadogCollectorWithClient accepts an interface which allows you to
// provide your own implementation of a statsd client, alter configuration on
// "github.com/DataDog/datadog-go/statsd".(*Client), provide additional tags per
// circuit-metric tuple, and add logging if you need it.
//
// Deprecated: use datadog.NewCollectorWithClient instead
//
//go:fix inline
func NewDatadogCollectorWithClient(client datadog.Client) func(string) metricCollector.MetricCollector {
	return datadog.NewCollectorWithClient(client)
}
