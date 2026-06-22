package plugins

import (
	"github.com/gojek/hystrix-go/plugins/statsd"
)

// StatsdCollector fulfills the metricCollector interface allowing users to ship circuit
// stats to a Statsd backend. To use users must call InitializeStatsdCollector before
// circuits are started. Then register NewStatsdCollector with metricCollector.Registry.Register(NewStatsdCollector).
//
// This Collector uses https://github.com/cactus/go-statsd-client/ for transport.
// Deprecated: use statsd.Collector instead
//
//go:fix inline
type StatsdCollector = statsd.Collector

// Deprecated: use statsd.CollectorClient instead
//
//go:fix inline
type StatsdCollectorClient = statsd.CollectorClient

// https://github.com/etsy/statsd/blob/master/docs/metric_types.md#multi-metric-packets
const (
	// Deprecated: use statsd.WANStatsdFlushBytes instead
	//
	//go:fix inline
	WANStatsdFlushBytes = statsd.WANStatsdFlushBytes
	// Deprecated: use statsd.LANStatsdFlushBytes instead
	//
	//go:fix inline
	LANStatsdFlushBytes = statsd.LANStatsdFlushBytes
	// Deprecated: use statsd.GigabitStatsdFlushBytes instead
	//
	//go:fix inline
	GigabitStatsdFlushBytes = statsd.GigabitStatsdFlushBytes
)

// StatsdCollectorConfig provides configuration that the Statsd client will need.
// Deprecated: use statsd.CollectorConfig instead
//
//go:fix inline
type StatsdCollectorConfig = statsd.CollectorConfig

// InitializeStatsdCollector creates the connection to the Statsd server
// and should be called before any metrics are recorded.
//
// Users should ensure to call Close() on the client.
// Deprecated: use stats.InitializeCollector instead
//
//go:fix inline
func InitializeStatsdCollector(config *statsd.CollectorConfig) (*statsd.CollectorClient, error) {
	return statsd.InitializeCollector(config)
}
