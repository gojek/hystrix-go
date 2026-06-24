package metricCollector

import (
	"github.com/gojek/hystrix-go/hystrix/rolling"
)

// DefaultMetricCollector holds information about the circuit state.
// This implementation of MetricCollector is the canonical source of information about the circuit.
// It is used for for all internal hystrix operations
// including circuit health checks and metrics sent to the hystrix dashboard.
//
// Metric Collectors do not need Mutexes as they are updated by circuits within a locked context.
type DefaultMetricCollector struct {
	numRequests *rolling.Number
	errors      *rolling.Number

	successes               *rolling.Number
	failures                *rolling.Number
	rejects                 *rolling.Number
	shortCircuits           *rolling.Number
	timeouts                *rolling.Number
	contextCanceled         *rolling.Number
	contextDeadlineExceeded *rolling.Number

	fallbackSuccesses *rolling.Number
	fallbackFailures  *rolling.Number
	totalDuration     *rolling.Timing
	runDuration       *rolling.Timing
}

func newDefaultMetricCollector(_ string) MetricCollector {
	return &DefaultMetricCollector{
		numRequests:             rolling.NewNumber(),
		errors:                  rolling.NewNumber(),
		successes:               rolling.NewNumber(),
		failures:                rolling.NewNumber(),
		rejects:                 rolling.NewNumber(),
		shortCircuits:           rolling.NewNumber(),
		timeouts:                rolling.NewNumber(),
		contextCanceled:         rolling.NewNumber(),
		contextDeadlineExceeded: rolling.NewNumber(),
		fallbackSuccesses:       rolling.NewNumber(),
		fallbackFailures:        rolling.NewNumber(),
		totalDuration:           rolling.NewTiming(),
		runDuration:             rolling.NewTiming(),
	}
}

// NumRequests returns the rolling number of requests
func (d *DefaultMetricCollector) NumRequests() *rolling.Number {
	return d.numRequests
}

// Errors returns the rolling number of errors
func (d *DefaultMetricCollector) Errors() *rolling.Number {
	return d.errors
}

// Successes returns the rolling number of successes
func (d *DefaultMetricCollector) Successes() *rolling.Number {
	return d.successes
}

// Failures returns the rolling number of failures
func (d *DefaultMetricCollector) Failures() *rolling.Number {
	return d.failures
}

// Rejects returns the rolling number of rejects
func (d *DefaultMetricCollector) Rejects() *rolling.Number {
	return d.rejects
}

// ShortCircuits returns the rolling number of short circuits
func (d *DefaultMetricCollector) ShortCircuits() *rolling.Number {
	return d.shortCircuits
}

// Timeouts returns the rolling number of timeouts
func (d *DefaultMetricCollector) Timeouts() *rolling.Number {
	return d.timeouts
}

// FallbackSuccesses returns the rolling number of fallback successes
func (d *DefaultMetricCollector) FallbackSuccesses() *rolling.Number {
	return d.fallbackSuccesses
}

func (d *DefaultMetricCollector) ContextCanceled() *rolling.Number {
	return d.contextCanceled
}

func (d *DefaultMetricCollector) ContextDeadlineExceeded() *rolling.Number {
	return d.contextDeadlineExceeded
}

// FallbackFailures returns the rolling number of fallback failures
func (d *DefaultMetricCollector) FallbackFailures() *rolling.Number {
	return d.fallbackFailures
}

// TotalDuration returns the rolling total duration
func (d *DefaultMetricCollector) TotalDuration() *rolling.Timing {
	return d.totalDuration
}

// RunDuration returns the rolling run duration
func (d *DefaultMetricCollector) RunDuration() *rolling.Timing {
	return d.runDuration
}

func (d *DefaultMetricCollector) Update(r MetricResult) {
	d.numRequests.Increment(r.Attempts)
	d.errors.Increment(r.Errors)
	d.successes.Increment(r.Successes)
	d.failures.Increment(r.Failures)
	d.rejects.Increment(r.Rejects)
	d.shortCircuits.Increment(r.ShortCircuits)
	d.timeouts.Increment(r.Timeouts)
	d.fallbackSuccesses.Increment(r.FallbackSuccesses)
	d.fallbackFailures.Increment(r.FallbackFailures)
	d.contextCanceled.Increment(r.ContextCanceled)
	d.contextDeadlineExceeded.Increment(r.ContextDeadlineExceeded)

	d.totalDuration.Add(r.TotalDuration)
	d.runDuration.Add(r.RunDuration)
}

// Reset resets all metrics in this collector to 0.
func (d *DefaultMetricCollector) Reset() {
	d.numRequests.Reset()
	d.errors.Reset()
	d.successes.Reset()
	d.rejects.Reset()
	d.shortCircuits.Reset()
	d.failures.Reset()
	d.timeouts.Reset()
	d.fallbackSuccesses.Reset()
	d.fallbackFailures.Reset()
	d.contextCanceled.Reset()
	d.contextDeadlineExceeded.Reset()
	d.totalDuration.Reset()
	d.runDuration.Reset()
}
