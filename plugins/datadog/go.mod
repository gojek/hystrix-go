module github.com/gojek/hystrix-go/plugins/datadog

go 1.25

replace github.com/gojek/hystrix-go/hystrix => ../../hystrix

require (
	github.com/DataDog/datadog-go v4.8.3+incompatible
	github.com/gojek/hystrix-go/hystrix v0.0.0-00010101000000-000000000000
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/sys v0.10.0 // indirect
)
