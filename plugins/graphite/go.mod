module github.com/gojek/hystrix-go/plugins/graphite

go 1.25

replace github.com/gojek/hystrix-go/hystrix => ../../hystrix

require (
	github.com/gojek/hystrix-go/hystrix v1.0.0
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9
)
