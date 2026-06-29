module github.com/gojek/hystrix-go/plugins

go 1.25

replace (
	github.com/gojek/hystrix-go/hystrix => ../hystrix
	github.com/gojek/hystrix-go/plugins/datadog => ./datadog
	github.com/gojek/hystrix-go/plugins/graphite => ./graphite
	github.com/gojek/hystrix-go/plugins/statsd => ./statsd
)

require (
	github.com/gojek/hystrix-go/hystrix v1.0.0
	github.com/gojek/hystrix-go/plugins/datadog v1.0.0
	github.com/gojek/hystrix-go/plugins/graphite v1.0.0
	github.com/gojek/hystrix-go/plugins/statsd v1.0.0
)

require (
	github.com/DataDog/datadog-go v4.8.3+incompatible // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/cactus/go-statsd-client/v6 v6.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	golang.org/x/sys v0.10.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
