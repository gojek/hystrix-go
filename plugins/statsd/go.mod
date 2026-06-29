module github.com/gojek/hystrix-go/plugins/statsd

go 1.25

replace github.com/gojek/hystrix-go/hystrix => ../../hystrix

require (
	github.com/cactus/go-statsd-client/v6 v6.0.0
	github.com/gojek/hystrix-go/hystrix v1.0.0
)
