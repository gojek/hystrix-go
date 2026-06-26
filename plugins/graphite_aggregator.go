// Plugins allows users to operate on statistics recorded for each circuit operation.
// Plugins should be careful to be lightweight as they will be called frequently.
package plugins

import (
	metricCollector "github.com/gojek/hystrix-go/hystrix/metric_collector"
	"github.com/gojek/hystrix-go/plugins/graphite"
)

// GraphiteCollector fulfills the metricCollector interface allowing users to ship circuit
// stats to a graphite backend. To use users must call InitializeGraphiteCollector before
// circuits are started. Then register NewGraphiteCollector with metricCollector.Registry.Register(NewGraphiteCollector).
//
// This Collector uses github.com/rcrowley/go-metrics for aggregation. See that repo for more details
// on how metrics are aggregated and expressed in graphite.
//
// Deprecated: use graphite.Collector instead
//
//go:fix inline
type GraphiteCollector = graphite.Collector

// GraphiteCollectorConfig provides configuration that the graphite client will need.
//
// Deprecated: use graphite.CollectorConfig instead
//
//go:fix inline
type GraphiteCollectorConfig = graphite.CollectorConfig

// InitializeGraphiteCollector creates the connection to the graphite server
// and should be called before any metrics are recorded.
//
// Deprecated: use graphite.InitializeCollector instead
//
//go:fix inline
func InitializeGraphiteCollector(config *graphite.CollectorConfig) {
	graphite.InitializeCollector(config)
}

// NewGraphiteCollector creates a collector for a specific circuit. The
// prefix given to this circuit will be {config.Prefix}.{circuit_name}.{metric}.
// Circuits with "/" in their names will have them replaced with ".".
//
// Deprecated: use graphite.NewCollector instead
//
//go:fix inline
func NewGraphiteCollector(name string) metricCollector.MetricCollector {
	return graphite.NewCollector(name)
}
