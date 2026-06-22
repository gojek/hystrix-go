module github.com/gojek/hystrix-go/loadtest

go 1.25

replace (
	github.com/gojek/hystrix-go/hystrix => ../hystrix
	github.com/gojek/hystrix-go/plugins/statsd => ../plugins/statsd
)

require (
	github.com/cactus/go-statsd-client/v6 v6.0.0
	github.com/gojek/hystrix-go/hystrix v0.0.0-00010101000000-000000000000
	github.com/gojek/hystrix-go/plugins/statsd v0.0.0-00010101000000-000000000000
)
