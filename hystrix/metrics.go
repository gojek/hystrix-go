package hystrix

import (
	"time"

	metricCollector "github.com/gojek/hystrix-go/hystrix/metric_collector"
	"github.com/gojek/hystrix-go/hystrix/rolling"
)

type commandExecution struct {
	PrimaryEvent     string
	SecondaryEvent   string
	Start            time.Time
	RunDuration      time.Duration
	ConcurrencyInUse float64
}

type metricExchange struct {
	Name string

	metricCollectors []metricCollector.MetricCollector
}

func newMetricExchange(name string) *metricExchange {
	return &metricExchange{
		Name:             name,
		metricCollectors: metricCollector.Registry.InitializeMetricCollectors(name),
	}
}

// The Default Collector function will panic if collectors are not setup to specification.
func (m *metricExchange) DefaultCollector() *metricCollector.DefaultMetricCollector {
	if len(m.metricCollectors) < 1 {
		panic("No Metric Collectors Registered.")
	}
	collection, ok := m.metricCollectors[0].(*metricCollector.DefaultMetricCollector)
	if !ok {
		panic("Default metric collector is not registered correctly. The default metric collector must be registered first.")
	}
	return collection
}

func (m *metricExchange) Update(e commandExecution) {
	result := e.buildResult()
	for _, collector := range m.metricCollectors {
		collector.Update(result)
	}
}

func (m *metricExchange) Reset() {
	for _, collector := range m.metricCollectors {
		collector.Reset()
	}
}

func (m *metricExchange) Requests() *rolling.Number {
	return m.requestsLocked()
}

func (m *metricExchange) requestsLocked() *rolling.Number {
	return m.DefaultCollector().NumRequests()
}

func (m *metricExchange) ErrorPercent(now time.Time) int {
	var errPct float64
	reqs := m.requestsLocked().Sum(now)
	errs := m.DefaultCollector().Errors().Sum(now)

	if reqs > 0 {
		errPct = (float64(errs) / float64(reqs)) * 100
	}

	return int(errPct + 0.5)
}

func (m *metricExchange) IsHealthy(now time.Time) bool {
	return m.ErrorPercent(now) < getSettings(m.Name).ErrorPercentThreshold
}

func (e commandExecution) buildResult() metricCollector.MetricResult {
	r := metricCollector.MetricResult{
		Attempts:         1,
		TotalDuration:    time.Since(e.Start),
		RunDuration:      e.RunDuration,
		ConcurrencyInUse: e.ConcurrencyInUse,
	}

	switch e.PrimaryEvent {
	case "success":
		r.Successes = 1
	case "failure":
		r.Failures = 1
		r.Errors = 1
	case "rejected":
		r.Rejects = 1
		r.Errors = 1
	case "short-circuit":
		r.ShortCircuits = 1
		r.Errors = 1
	case "timeout":
		r.Timeouts = 1
		r.Errors = 1
	case "context_canceled":
		r.ContextCanceled = 1
	case "context_deadline_exceeded":
		r.ContextDeadlineExceeded = 1
	}

	switch e.SecondaryEvent {
	case "fallback-success":
		r.FallbackSuccesses = 1
	case "fallback-failure":
		r.FallbackFailures = 1
	}

	return r
}
