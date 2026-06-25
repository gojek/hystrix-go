package datadog

import (
	"github.com/DataDog/datadog-go/statsd"
	metricCollector "github.com/gojek/hystrix-go/hystrix/metric_collector"
)

// These metrics are constants because we're leveraging the Datadog tagging
// extension to statsd.
//
// They only apply to the Collector and are only useful if providing your
// own implemenation of Client
const (
	// DM = Datadog Metric
	CircuitOpen       = "hystrix.circuitOpen"
	Attempts          = "hystrix.attempts"
	Errors            = "hystrix.errors"
	Successes         = "hystrix.successes"
	Failures          = "hystrix.failures"
	Rejects           = "hystrix.rejects"
	ShortCircuits     = "hystrix.shortCircuits"
	Timeouts          = "hystrix.timeouts"
	FallbackSuccesses = "hystrix.fallbackSuccesses"
	FallbackFailures  = "hystrix.fallbackFailures"
	TotalDuration     = "hystrix.totalDuration"
	RunDuration       = "hystrix.runDuration"
)

type (
	// Client is the minimum interface needed by
	// NewCollectorWithClient
	Client interface {
		Count(name string, value int64, tags []string, rate float64) error
		Gauge(name string, value float64, tags []string, rate float64) error
		TimeInMilliseconds(name string, value float64, tags []string, rate float64) error
	}

	// Collector fulfills the metricCollector interface allowing users to
	// ship circuit stats to Datadog.
	//
	// This Collector, by default, uses github.com/DataDog/datadog-go/statsd for
	// transport. The main advantage of this over statsd is building graphs and
	// multi-alert monitors around single metrics (constantized above) and
	// adding tag dimensions. You can set up a single monitor to rule them all
	// across services and geographies. Graphs become much simpler to setup by
	// allowing you to create queries like the following
	//
	//   {
	//     "viz": "timeseries",
	//     "requests": [
	//       {
	//         "q": "max:hystrix.runDuration.95percentile{$region} by {hystrixcircuit}",
	//         "type": "line"
	//       }
	//     ]
	//   }
	//
	// As new circuits come online you get graphing and monitoring "for free".
	Collector struct {
		client Client
		tags   []string
	}
)

// NewCollector creates a collector for a specific circuit with a
// "github.com/DataDog/datadog-go/statsd".(*Client).
//
// addr is in the format "<host>:<port>" (e.g. "localhost:8125")
//
// prefix may be an empty string
//
// Example use
//
//	package main
//
//	import (
//		"github.com/gojek/hystrix-go/plugins"
//		"github.com/gojek/hystrix-go/hystrix/metric_collector"
//	)
//
//	func main() {
//		collector, err := plugins.NewDatadogCollector("localhost:8125", "")
//		if err != nil {
//			panic(err)
//		}
//		metricCollector.Registry.Register(collector)
//	}
func NewCollector(addr, prefix string) (func(string) metricCollector.MetricCollector, error) {

	c, err := statsd.NewBuffered(addr, 100)
	if err != nil {
		return nil, err
	}

	// Prefix every metric with the app name
	c.Namespace = prefix

	return NewCollectorWithClient(c), nil
}

// NewCollectorWithClient accepts an interface which allows you to
// provide your own implementation of a statsd client, alter configuration on
// "github.com/DataDog/datadog-go/statsd".(*Client), provide additional tags per
// circuit-metric tuple, and add logging if you need it.
func NewCollectorWithClient(client Client) func(string) metricCollector.MetricCollector {
	return func(name string) metricCollector.MetricCollector {
		return &Collector{
			client: client,
			tags:   []string{"hystrixcircuit:" + name},
		}
	}
}

func (dc *Collector) Update(r metricCollector.MetricResult) {
	if r.Attempts > 0 {
		dc.client.Count(Attempts, r.Attempts, dc.tags, 1.0)
	}
	if r.Errors > 0 {
		dc.client.Count(Errors, r.Errors, dc.tags, 1.0)
	}
	if r.Successes > 0 {
		dc.client.Gauge(CircuitOpen, 0, dc.tags, 1.0)
		dc.client.Count(Successes, r.Successes, dc.tags, 1.0)
	}
	if r.Failures > 0 {
		dc.client.Count(Failures, r.Failures, dc.tags, 1.0)
	}
	if r.Rejects > 0 {
		dc.client.Count(Rejects, r.Rejects, dc.tags, 1.0)
	}
	if r.ShortCircuits > 0 {
		dc.client.Gauge(CircuitOpen, 1, dc.tags, 1.0)
		dc.client.Count(ShortCircuits, r.ShortCircuits, dc.tags, 1.0)
	}
	if r.Timeouts > 0 {
		dc.client.Count(Timeouts, r.Timeouts, dc.tags, 1.0)
	}
	if r.FallbackSuccesses > 0 {
		dc.client.Count(FallbackSuccesses, r.FallbackSuccesses, dc.tags, 1.0)
	}
	if r.FallbackFailures > 0 {
		dc.client.Count(FallbackFailures, r.FallbackFailures, dc.tags, 1.0)
	}

	ms := float64(r.TotalDuration.Nanoseconds() / 1000000)
	dc.client.TimeInMilliseconds(TotalDuration, ms, dc.tags, 1.0)

	ms = float64(r.RunDuration.Nanoseconds() / 1000000)
	dc.client.TimeInMilliseconds(RunDuration, ms, dc.tags, 1.0)
}

// Reset is a noop operation in this collector.
func (dc *Collector) Reset() {}
