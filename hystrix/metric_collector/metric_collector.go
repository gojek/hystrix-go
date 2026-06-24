package metricCollector

import (
	"sync"
	"time"
)

// Registry is the default metricCollectorRegistry that circuits will use to
// collect statistics about the health of the circuit.
var Registry = metricCollectorRegistry{
	lock: &sync.RWMutex{},
	registry: []func(name string) MetricCollector{
		newDefaultMetricCollector,
	},
}

type metricCollectorRegistry struct {
	lock     *sync.RWMutex
	registry []func(name string) MetricCollector
}

// InitializeMetricCollectors runs the registried MetricCollector Initializers to create an array of MetricCollectors.
func (m *metricCollectorRegistry) InitializeMetricCollectors(name string) []MetricCollector {
	m.lock.RLock()
	defer m.lock.RUnlock()

	metrics := make([]MetricCollector, len(m.registry))
	for i, metricCollectorInitializer := range m.registry {
		metric := metricCollectorInitializer(name)
		metric.Reset()
		metrics[i] = metric
	}
	return metrics
}

// Register places a MetricCollector Initializer in the registry maintained by this metricCollectorRegistry.
func (m *metricCollectorRegistry) Register(initMetricCollector func(string) MetricCollector) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.registry = append(m.registry, initMetricCollector)
}

type MetricResult struct {
	Attempts                int64
	Errors                  int64
	Successes               int64
	Failures                int64
	Rejects                 int64
	ShortCircuits           int64
	Timeouts                int64
	FallbackSuccesses       int64
	FallbackFailures        int64
	ContextCanceled         int64
	ContextDeadlineExceeded int64
	TotalDuration           time.Duration
	RunDuration             time.Duration
	ConcurrencyInUse        float64
}

// MetricCollector represents the contract that all collectors must fulfill to gather circuit statistics.
// Implementations of this interface do not have to maintain locking around thier data stores so long as
// they are not modified outside of the hystrix context.
type MetricCollector interface {
	// Update accepts a set of metrics from a command execution for remote instrumentation
	// Note: Update will be called synchronously by hystrix-go, so custom plugin needs to
	//make sure they can be called concurrently as well as do not block for long time.
	Update(MetricResult)
	// Reset resets the internal counters and timers.
	Reset()
}
