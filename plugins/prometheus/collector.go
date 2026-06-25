package prometheus

import (
	metricCollector "github.com/gojek/hystrix-go/hystrix/metric_collector"
	"github.com/prometheus/client_golang/prometheus"
)

const lblCmdName = `command_name`

var labels = []string{lblCmdName}

type collectorVec struct {
	attempts                *prometheus.CounterVec
	successes               *prometheus.CounterVec
	errors                  *prometheus.CounterVec
	failures                *prometheus.CounterVec
	rejects                 *prometheus.CounterVec
	shortCircuits           *prometheus.CounterVec
	timeouts                *prometheus.CounterVec
	fallbackSuccesses       *prometheus.CounterVec
	fallbackFailures        *prometheus.CounterVec
	contextDeadlineExceeded *prometheus.CounterVec
	contextCanceled         *prometheus.CounterVec
	totalDuration           *prometheus.HistogramVec
	runDuration             *prometheus.HistogramVec
	concurrencyInUse        *prometheus.GaugeVec
}

type collector struct {
	attempts                prometheus.Counter
	successes               prometheus.Counter
	errors                  prometheus.Counter
	failures                prometheus.Counter
	rejects                 prometheus.Counter
	shortCircuits           prometheus.Counter
	timeouts                prometheus.Counter
	fallbackSuccesses       prometheus.Counter
	fallbackFailures        prometheus.Counter
	contextDeadlineExceeded prometheus.Counter
	contextCanceled         prometheus.Counter
	totalDuration           prometheus.Observer
	runDuration             prometheus.Observer
	concurrencyInUse        prometheus.Gauge
}

// RegisterCollector register metricCollector.MetricCollector to metricCollector.Registry. This metricCollector.MetricCollector is
// exposed to prometheus.Registerer r for given Option opts
func RegisterCollector(r prometheus.Registerer, opts ...Option) {
	metricCollector.Registry.Register(NewCollector(r, opts...))
}

// NewCollector creates metricCollector.MetricCollector per command name, This metricCollector.MetricCollector is
// exposed to prometheus.Registerer r for given Option opts
func NewCollector(r prometheus.Registerer, opts ...Option) func(string) metricCollector.MetricCollector {
	o := buildOptions(opts...)
	c := collectorVec{
		attempts:                newCounterVec(r, `attempts`, o),
		successes:               newCounterVec(r, `successes`, o),
		errors:                  newCounterVec(r, `errors`, o),
		failures:                newCounterVec(r, `failures`, o),
		rejects:                 newCounterVec(r, `rejects`, o),
		shortCircuits:           newCounterVec(r, `short_circuits`, o),
		timeouts:                newCounterVec(r, `timeouts`, o),
		fallbackSuccesses:       newCounterVec(r, `fallback_successes`, o),
		fallbackFailures:        newCounterVec(r, `fallback_failures`, o),
		contextDeadlineExceeded: newCounterVec(r, `context_deadline_exceeded`, o),
		contextCanceled:         newCounterVec(r, `context_canceled`, o),
		totalDuration:           newHistogramVec(r, `total_duration`, o),
		runDuration:             newHistogramVec(r, `run_duration`, o),
		concurrencyInUse:        newGaugeVec(r, `concurrency_in_use`, o),
	}
	return c.collector
}

func (c collectorVec) collector(cmdName string) metricCollector.MetricCollector {
	lbls := map[string]string{lblCmdName: cmdName}
	return collector{
		attempts:                c.attempts.With(lbls),
		successes:               c.successes.With(lbls),
		errors:                  c.errors.With(lbls),
		failures:                c.failures.With(lbls),
		rejects:                 c.rejects.With(lbls),
		shortCircuits:           c.shortCircuits.With(lbls),
		timeouts:                c.timeouts.With(lbls),
		fallbackSuccesses:       c.fallbackSuccesses.With(lbls),
		fallbackFailures:        c.fallbackFailures.With(lbls),
		contextDeadlineExceeded: c.contextDeadlineExceeded.With(lbls),
		contextCanceled:         c.contextCanceled.With(lbls),
		totalDuration:           c.totalDuration.With(lbls),
		runDuration:             c.runDuration.With(lbls),
		concurrencyInUse:        c.concurrencyInUse.With(lbls),
	}
}

func (c collector) Update(result metricCollector.MetricResult) {
	if result.Attempts > 0 {
		c.attempts.Add(float64(result.Attempts))
	}
	if result.Successes > 0 {
		c.successes.Add(float64(result.Successes))
	}
	if result.Errors > 0 {
		c.errors.Add(float64(result.Errors))
	}
	if result.Failures > 0 {
		c.failures.Add(float64(result.Failures))
	}
	if result.Rejects > 0 {
		c.rejects.Add(float64(result.Rejects))
	}
	if result.ShortCircuits > 0 {
		c.shortCircuits.Add(float64(result.ShortCircuits))
	}
	if result.Timeouts > 0 {
		c.timeouts.Add(float64(result.Timeouts))
	}
	if result.FallbackSuccesses > 0 {
		c.fallbackSuccesses.Add(float64(result.FallbackSuccesses))
	}
	if result.FallbackFailures > 0 {
		c.fallbackFailures.Add(float64(result.FallbackFailures))
	}
	if result.ContextDeadlineExceeded > 0 {
		c.contextDeadlineExceeded.Add(float64(result.ContextDeadlineExceeded))
	}
	if result.ContextCanceled > 0 {
		c.contextCanceled.Add(float64(result.ContextCanceled))
	}

	c.totalDuration.Observe(result.TotalDuration.Seconds())
	c.runDuration.Observe(result.RunDuration.Seconds())
	c.concurrencyInUse.Set(result.ConcurrencyInUse)
}

// Reset is a noop operation in this collector.
func (c collector) Reset() {}

func newCounterVec(r prometheus.Registerer, metric string, _ options) *prometheus.CounterVec {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "hystrix",
		Name:      metric,
		Help:      "[counter] subsystem: hystrix, metric: " + metric,
	}, labels)
	r.MustRegister(vec)
	return vec
}

func newGaugeVec(r prometheus.Registerer, metric string, _ options) *prometheus.GaugeVec {
	vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "hystrix",
		Name:      metric,
		Help:      "[gauge] subsystem: hystrix, metric: " + metric,
	}, labels)
	r.MustRegister(vec)
	return vec
}

func newHistogramVec(r prometheus.Registerer, metric string, o options) *prometheus.HistogramVec {
	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: "hystrix",
		Name:      metric,
		Help:      "[histogram] subsystem: hystrix, metric: " + metric,
		Buckets:   o.HistogramBuckets,
	}, labels)
	r.MustRegister(vec)
	return vec
}
