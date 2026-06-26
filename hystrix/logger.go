package hystrix

type logger interface {
	Printf(format string, items ...any)
}

// NoopLogger does not log anything.
type NoopLogger struct{}

// Printf does nothing.
func (l NoopLogger) Printf(_ string, _ ...any) {}
