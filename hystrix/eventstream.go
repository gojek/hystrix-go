package hystrix

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	metricCollector "github.com/gojek/hystrix-go/hystrix/metric_collector"
	"github.com/gojek/hystrix-go/hystrix/rolling"
)

const (
	streamEventBufferSize = 10
)

// NewStreamHandler returns a server capable of exposing dashboard metrics via HTTP.
func NewStreamHandler() *StreamHandler {
	return &StreamHandler{}
}

// StreamHandler publishes metrics for each command and each pool once a second to all connected HTTP client.
type StreamHandler struct {
	done     chan struct{}
	requests map[*http.Request]chan []byte
	mu       sync.RWMutex
}

// Start begins watching the in-memory circuit breakers for metrics
func (sh *StreamHandler) Start() {
	sh.requests = make(map[*http.Request]chan []byte)
	sh.done = make(chan struct{})
	go sh.loop()
}

// Stop shuts down the metric collection routine
func (sh *StreamHandler) Stop() {
	close(sh.done)
}

var _ http.Handler = (*StreamHandler)(nil)

func (sh *StreamHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Make sure that the writer supports flushing.
	f, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	// enable timing metric if not already enabled
	if metricCollector.DefaultMetricCollectorTimingEnabled.CompareAndSwap(false, true) {
		defer func() {
			_ = metricCollector.DefaultMetricCollectorTimingEnabled.CompareAndSwap(true, false)
		}()
	}
	events := sh.register(req)
	defer sh.unregister(req)

	rw.Header().Add("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	for {
		select {
		case <-req.Context().Done():
			// client is gone
			return
		case event := <-events:
			_, err := rw.Write(event)
			if err != nil {
				return
			}
			f.Flush()
		}
	}
}

func (sh *StreamHandler) loop() {
	tick := time.Tick(1 * time.Second)
	for {
		select {
		case <-tick:
			for _, cb := range *circuitBreakers.Load() {
				if err := sh.publishMetrics(cb); err != nil {
					log.Printf("hystrix-go: publishing metrics: %v", err)
				}
				if err := sh.publishThreadPools(cb.executorPool); err != nil {
					log.Printf("hystrix-go: publishing threads: %v", err)
				}
			}
		case <-sh.done:
			return
		}
	}
}

func (sh *StreamHandler) publishMetrics(cb *CircuitBreaker) error {
	now := time.Now()
	reqCount := cb.metrics.Requests().Sum(now)
	errCount := cb.metrics.DefaultCollector().Errors().Sum(now)
	errPct := cb.metrics.ErrorPercent(now)

	eventBytes, err := json.Marshal(&streamCmdMetric{
		Type:           "HystrixCommand",
		Name:           cb.Name,
		Group:          cb.Name,
		Time:           currentTime(),
		ReportingHosts: 1,

		RequestCount:       reqCount,
		ErrorCount:         errCount,
		ErrorPct:           errPct,
		CircuitBreakerOpen: cb.IsOpen(),

		RollingCountSuccess:            cb.metrics.DefaultCollector().Successes().Sum(now),
		RollingCountFailure:            cb.metrics.DefaultCollector().Failures().Sum(now),
		RollingCountThreadPoolRejected: cb.metrics.DefaultCollector().Rejects().Sum(now),
		RollingCountShortCircuited:     cb.metrics.DefaultCollector().ShortCircuits().Sum(now),
		RollingCountTimeout:            cb.metrics.DefaultCollector().Timeouts().Sum(now),
		RollingCountFallbackSuccess:    cb.metrics.DefaultCollector().FallbackSuccesses().Sum(now),
		RollingCountFallbackFailure:    cb.metrics.DefaultCollector().FallbackFailures().Sum(now),

		LatencyTotal:       generateLatencyTimings(cb.metrics.DefaultCollector().TotalDuration()),
		LatencyTotalMean:   cb.metrics.DefaultCollector().TotalDuration().Mean(),
		LatencyExecute:     generateLatencyTimings(cb.metrics.DefaultCollector().RunDuration()),
		LatencyExecuteMean: cb.metrics.DefaultCollector().RunDuration().Mean(),

		// TODO: all hard-coded values should become configurable settings, per circuit

		RollingStatsWindow:         10000,
		ExecutionIsolationStrategy: "THREAD",

		CircuitBreakerEnabled:                true,
		CircuitBreakerForceClosed:            false,
		CircuitBreakerForceOpen:              false,
		CircuitBreakerErrorThresholdPercent:  uint32(getSettings(cb.Name).ErrorPercentThreshold),
		CircuitBreakerSleepWindow:            uint32(getSettings(cb.Name).SleepWindow.Seconds() * 1000),
		CircuitBreakerRequestVolumeThreshold: uint32(getSettings(cb.Name).RequestVolumeThreshold),
	})
	if err != nil {
		return err
	}
	return sh.writeToRequests(eventBytes)
}

func (sh *StreamHandler) publishThreadPools(pool *executorPool) error {
	now := time.Now()

	eventBytes, err := json.Marshal(&streamThreadPoolMetric{
		Type:           "HystrixThreadPool",
		Name:           pool.Name,
		ReportingHosts: 1,

		CurrentActiveCount:        pool.ActiveCount(),
		CurrentTaskCount:          0,
		CurrentCompletedTaskCount: 0,

		RollingCountThreadsExecuted: pool.Executed.Sum(now),
		RollingMaxActiveThreads:     pool.MaxActiveRequests.Max(now),

		CurrentPoolSize:        pool.Max,
		CurrentCorePoolSize:    pool.Max,
		CurrentLargestPoolSize: pool.Max,
		CurrentMaximumPoolSize: pool.Max,

		RollingStatsWindow:          10000,
		QueueSizeRejectionThreshold: 0,
		CurrentQueueSize:            0,
	})
	if err != nil {
		return err
	}
	return sh.writeToRequests(eventBytes)
}

func (sh *StreamHandler) writeToRequests(eventBytes []byte) error {
	var b bytes.Buffer
	_, err := b.Write([]byte("data:"))
	if err != nil {
		return err
	}

	_, err = b.Write(eventBytes)
	if err != nil {
		return err
	}
	_, err = b.Write([]byte("\n\n"))
	if err != nil {
		return err
	}
	dataBytes := b.Bytes()
	sh.mu.RLock()

	for _, requestEvents := range sh.requests {
		select {
		case requestEvents <- dataBytes:
		default:
		}
	}
	sh.mu.RUnlock()

	return nil
}

func (sh *StreamHandler) register(req *http.Request) <-chan []byte {
	sh.mu.RLock()
	events, ok := sh.requests[req]
	sh.mu.RUnlock()
	if ok {
		return events
	}

	events = make(chan []byte, streamEventBufferSize)
	sh.mu.Lock()
	sh.requests[req] = events
	sh.mu.Unlock()
	return events
}

func (sh *StreamHandler) unregister(req *http.Request) {
	sh.mu.Lock()
	delete(sh.requests, req)
	sh.mu.Unlock()
}

func generateLatencyTimings(r *rolling.Timing) streamCmdLatency {
	return streamCmdLatency{
		Timing0:   r.Percentile(0),
		Timing25:  r.Percentile(25),
		Timing50:  r.Percentile(50),
		Timing75:  r.Percentile(75),
		Timing90:  r.Percentile(90),
		Timing95:  r.Percentile(95),
		Timing99:  r.Percentile(99),
		Timing995: r.Percentile(99.5),
		Timing100: r.Percentile(100),
	}
}

type streamCmdMetric struct {
	Type                                             string           `json:"type"`
	Name                                             string           `json:"name"`
	Group                                            string           `json:"group"`
	ExecutionIsolationThreadPoolKeyOverride          string           `json:"propertyValue_executionIsolationThreadPoolKeyOverride"`
	ExecutionIsolationStrategy                       string           `json:"propertyValue_executionIsolationStrategy"`
	RollingCountTimeout                              int64            `json:"rollingCountTimeout"`
	RollingCountFallbackSuccess                      int64            `json:"rollingCountFallbackSuccess"`
	ErrorPct                                         int              `json:"errorPercentage"`
	Time                                             int64            `json:"currentTime"`
	RollingCountCollapsedRequests                    int64            `json:"rollingCountCollapsedRequests"`
	RollingCountExceptionsThrown                     int64            `json:"rollingCountExceptionsThrown"`
	RollingCountFailure                              int64            `json:"rollingCountFailure"`
	RollingCountFallbackFailure                      int64            `json:"rollingCountFallbackFailure"`
	RollingCountFallbackRejection                    int64            `json:"rollingCountFallbackRejection"`
	ErrorCount                                       int64            `json:"errorCount"`
	RollingCountResponsesFromCache                   int64            `json:"rollingCountResponsesFromCache"`
	RollingCountSemaphoreRejected                    int64            `json:"rollingCountSemaphoreRejected"`
	RollingCountShortCircuited                       int64            `json:"rollingCountShortCircuited"`
	RollingCountSuccess                              int64            `json:"rollingCountSuccess"`
	RollingCountThreadPoolRejected                   int64            `json:"rollingCountThreadPoolRejected"`
	RequestCount                                     int64            `json:"requestCount"`
	CurrentConcurrentExecutionCount                  int64            `json:"currentConcurrentExecutionCount"`
	LatencyTotal                                     streamCmdLatency `json:"latencyTotal"`
	LatencyExecute                                   streamCmdLatency `json:"latencyExecute"`
	LatencyExecuteMean                               uint32           `json:"latencyExecute_mean"`
	FallbackIsolationSemaphoreMaxConcurrentRequests  uint32           `json:"propertyValue_fallbackIsolationSemaphoreMaxConcurrentRequests"`
	CircuitBreakerRequestVolumeThreshold             uint32           `json:"propertyValue_circuitBreakerRequestVolumeThreshold"`
	CircuitBreakerSleepWindow                        uint32           `json:"propertyValue_circuitBreakerSleepWindowInMilliseconds"`
	CircuitBreakerErrorThresholdPercent              uint32           `json:"propertyValue_circuitBreakerErrorThresholdPercentage"`
	RollingStatsWindow                               uint32           `json:"propertyValue_metricsRollingStatisticalWindowInMilliseconds"`
	LatencyTotalMean                                 uint32           `json:"latencyTotal_mean"`
	ExecutionIsolationSemaphoreMaxConcurrentRequests uint32           `json:"propertyValue_executionIsolationSemaphoreMaxConcurrentRequests"`
	ReportingHosts                                   uint32           `json:"reportingHosts"`
	ExecutionIsolationThreadTimeout                  uint32           `json:"propertyValue_executionIsolationThreadTimeoutInMilliseconds"`
	ExecutionIsolationThreadInterruptOnTimeout       bool             `json:"propertyValue_executionIsolationThreadInterruptOnTimeout"`
	CircuitBreakerOpen                               bool             `json:"isCircuitBreakerOpen"`
	CircuitBreakerEnabled                            bool             `json:"propertyValue_circuitBreakerEnabled"`
	CircuitBreakerForceClosed                        bool             `json:"propertyValue_circuitBreakerForceClosed"`
	CircuitBreakerForceOpen                          bool             `json:"propertyValue_circuitBreakerForceOpen"`
	RequestCacheEnabled                              bool             `json:"propertyValue_requestCacheEnabled"`
	RequestLogEnabled                                bool             `json:"propertyValue_requestLogEnabled"`
}

type streamCmdLatency struct {
	Timing0   uint32 `json:"0"`
	Timing25  uint32 `json:"25"`
	Timing50  uint32 `json:"50"`
	Timing75  uint32 `json:"75"`
	Timing90  uint32 `json:"90"`
	Timing95  uint32 `json:"95"`
	Timing99  uint32 `json:"99"`
	Timing995 uint32 `json:"99.5"`
	Timing100 uint32 `json:"100"`
}

type streamThreadPoolMetric struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	ReportingHosts int    `json:"reportingHosts"`

	CurrentActiveCount        int `json:"currentActiveCount"`
	CurrentCompletedTaskCount int `json:"currentCompletedTaskCount"`
	CurrentCorePoolSize       int `json:"currentCorePoolSize"`
	CurrentLargestPoolSize    int `json:"currentLargestPoolSize"`
	CurrentMaximumPoolSize    int `json:"currentMaximumPoolSize"`
	CurrentPoolSize           int `json:"currentPoolSize"`
	CurrentQueueSize          int `json:"currentQueueSize"`
	CurrentTaskCount          int `json:"currentTaskCount"`

	RollingMaxActiveThreads     int64 `json:"rollingMaxActiveThreads"`
	RollingCountThreadsExecuted int64 `json:"rollingCountThreadsExecuted"`

	RollingStatsWindow          int `json:"propertyValue_metricsRollingStatisticalWindowInMilliseconds"`
	QueueSizeRejectionThreshold int `json:"propertyValue_queueSizeRejectionThreshold"`
}

func currentTime() int64 {
	return time.Now().UnixNano() / int64(1000000)
}
