package hystrix

import (
	"errors"
	"maps"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreaker is created for each ExecutorPool to track whether requests
// should be attempted, or rejected if the Health of the circuit is too low.
type CircuitBreaker struct {
	Name                   string
	open                   atomic.Bool
	openedOrLastTestedTime atomic.Int64

	executorPool *executorPool
	metrics      *metricExchange
}

var (
	// circuitBreakersMutex only used for adding, removing(flush) circuit breakers to limit concurrent writes. Each
	// writes involve copying existing circuitBreakers, modifying it and then doing atomic operation. This setup
	// allows us to skip sync.Mutex/sync.RWMutex operations for happy path reads, which is the most common case
	circuitBreakersMutex sync.Mutex
	circuitBreakers      atomic.Pointer[map[string]*CircuitBreaker]
)

func init() {
	circuitBreakers.Store(&map[string]*CircuitBreaker{})
}

// GetCircuit returns the circuit for the given command and whether this call created it.
func GetCircuit(name string) (*CircuitBreaker, bool, error) {
	if cb, ok := (*circuitBreakers.Load())[name]; ok {
		return cb, false, nil
	}
	circuitBreakersMutex.Lock()
	defer circuitBreakersMutex.Unlock()

	cbs := *circuitBreakers.Load()
	if cb, ok := cbs[name]; ok {
		return cb, false, nil
	}
	cbs = maps.Clone(cbs) // clone to ensure all concurrent access are limited to read
	cb := newCircuitBreaker(name)
	cbs[name] = cb
	circuitBreakers.Store(&cbs)
	return cb, true, nil
}

// Flush purges all circuit and metric information from memory.
func Flush() {
	circuitBreakersMutex.Lock()
	defer circuitBreakersMutex.Unlock()

	cbs := *circuitBreakers.Load()
	for _, cb := range cbs {
		cb.metrics.Reset()
		cb.executorPool.ResetMetrics()
	}
	circuitBreakers.Store(&map[string]*CircuitBreaker{})
}

// newCircuitBreaker creates a CircuitBreaker with associated Health
func newCircuitBreaker(name string) *CircuitBreaker {
	return &CircuitBreaker{
		Name:         name,
		executorPool: newExecutorPool(name),
		metrics:      newMetricExchange(name),
	}
}

// IsOpen is called before any Command execution to check whether or
// not it should be attempted. An "open" circuit means it is disabled.
func (circuit *CircuitBreaker) IsOpen() bool {
	if circuit.open.Load() {
		return true
	}

	now := time.Now()
	if uint64(circuit.metrics.Requests().Sum(now)) < getSettings(circuit.Name).RequestVolumeThreshold {
		return false
	}

	if !circuit.metrics.IsHealthy(now) {
		// too many failures, open the circuit
		circuit.setOpen()
		return true
	}

	return false
}

// AllowRequest is checked before a command executes, ensuring that circuit state and metric health allow it.
// When the circuit is open, this call will occasionally return true to measure whether the external service
// has recovered.
func (circuit *CircuitBreaker) AllowRequest() bool {
	return !circuit.IsOpen() || circuit.allowSingleTest()
}

func (circuit *CircuitBreaker) allowSingleTest() bool {
	now := time.Now().UnixNano()
	openedOrLastTestedTime := circuit.openedOrLastTestedTime.Load()
	if now > openedOrLastTestedTime+getSettings(circuit.Name).SleepWindow.Nanoseconds() {
		swapped := circuit.openedOrLastTestedTime.CompareAndSwap(openedOrLastTestedTime, now)
		if swapped {
			log.Printf("hystrix-go: allowing single test to possibly close circuit %v", circuit.Name)
		}
		return swapped
	}

	return false
}

func (circuit *CircuitBreaker) setOpen() {
	if circuit.open.CompareAndSwap(false, true) {
		log.Printf("hystrix-go: opening circuit %v", circuit.Name)
		circuit.openedOrLastTestedTime.Store(time.Now().UnixNano())
	}
}

func (circuit *CircuitBreaker) setClose() {
	if circuit.open.CompareAndSwap(true, false) {
		log.Printf("hystrix-go: closing circuit %v", circuit.Name)
		circuit.metrics.Reset()
	}
}

// ReportEvent records command metrics for tracking recent error rates and exposing data to the dashboard.
func (circuit *CircuitBreaker) ReportEvent(eventTypes []string, start time.Time, runDuration time.Duration) error {
	if len(eventTypes) > 1 {
		return circuit.reportEvent(eventTypes[0], eventTypes[1], start, runDuration)
	}
	if len(eventTypes) > 0 {
		return circuit.reportEvent(eventTypes[0], ``, start, runDuration)
	}
	return nil
}

func (circuit *CircuitBreaker) reportEvent(primaryEvent, secondaryEvent string, start time.Time, runDuration time.Duration) error {
	if primaryEvent == "" {
		return errors.New("no event types sent for metrics")
	}

	if primaryEvent == "success" && circuit.open.Load() {
		circuit.setClose()
	}

	var concurrencyInUse float64
	if circuit.executorPool.Max > 0 {
		concurrencyInUse = float64(circuit.executorPool.ActiveCount()) / float64(circuit.executorPool.Max)
	}

	circuit.metrics.Update(commandExecution{
		PrimaryEvent:     primaryEvent,
		SecondaryEvent:   secondaryEvent,
		Start:            start,
		RunDuration:      runDuration,
		ConcurrencyInUse: concurrencyInUse,
	})

	return nil
}
