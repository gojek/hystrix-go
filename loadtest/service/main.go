// Package main implements an http server which executes a hystrix command each request and
// sends metrics to a statsd instance to aid performance testing.
package main

import (
	"flag"
	"log"
	"math/rand/v2"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	gostatsd "github.com/cactus/go-statsd-client/v6/statsd"
	"github.com/gojek/hystrix-go/hystrix"
	metricCollector "github.com/gojek/hystrix-go/hystrix/metric_collector"
	"github.com/gojek/hystrix-go/plugins/statsd"
)

const (
	deltaWindow = 10
	minDelay    = 35
	maxDelay    = 55
)

var delay int

const (
	up = iota
	down
)

func init() {
	delay = minDelay
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	statsdHost := flag.String("statsd", "", "Statsd host to record load test metrics")
	flag.Parse()

	stats, err := gostatsd.NewClientWithConfig(&gostatsd.ClientConfig{
		Address:     *statsdHost,
		Prefix:      "hystrix.loadtest.service",
		UseBuffered: false,
	})
	if err != nil {
		log.Fatalf("could not initialize statsd client: %v", err)
	}

	c, err := statsd.InitializeCollector(&statsd.CollectorConfig{
		StatsdAddr: *statsdHost,
		Prefix:     "hystrix.loadtest.circuits",
	})
	if err != nil {
		log.Fatalf("could not initialize statsd client: %v", err)
	}
	metricCollector.Registry.Register(c.NewStatsdCollector)

	hystrix.ConfigureCommand("test", hystrix.CommandConfig{
		Timeout: 50,
	})

	go rotateDelay()

	http.HandleFunc("/", timedHandler(handle, stats))
	log.Print("starting server")
	log.Fatal(http.ListenAndServe(":8888", nil))
}

func timedHandler(fn func(w http.ResponseWriter, r *http.Request), stats gostatsd.Statter) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		fn(w, r)
		_ = stats.TimingDuration("request", time.Since(start), 1)
	}
}

func handle(w http.ResponseWriter, _ *http.Request) {
	done := make(chan struct{}, 1)
	errChan := hystrix.Go("test", func() error {
		delta := rand.IntN(deltaWindow)
		time.Sleep(time.Duration(delay+delta) * time.Millisecond)
		done <- struct{}{}
		return nil
	}, func(_ error) error {
		done <- struct{}{}
		return nil
	})

	select {
	case err := <-errChan:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	case <-done:
		_, _ = w.Write([]byte("OK"))
	}
}

func rotateDelay() {
	direction := up
	for {
		if direction == up && delay == maxDelay {
			direction = down
		}
		if direction == down && delay == minDelay {
			direction = up
		}

		if direction == up {
			delay++
		} else {
			delay--
		}

		time.Sleep(5 * time.Second)
		log.Printf("setting delay to %v", delay)
	}
}
