package prometheus

type options struct {
	HistogramBuckets []float64
}

func buildOptions(opts ...Option) options {
	opt := options{}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

type Option func(*options)

// WithHistogramBuckets allows you to set the bucket for request latency histogram.
func WithHistogramBuckets(buckets []float64) Option {
	return func(c *options) {
		c.HistogramBuckets = buckets
	}
}
