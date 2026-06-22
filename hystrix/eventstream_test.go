package hystrix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

type eventStreamTestServer struct {
	*httptest.Server
	*StreamHandler
}

func (s *eventStreamTestServer) stopTestServer() error {
	s.Close()
	s.Stop()

	return nil
}

func startTestServer() *eventStreamTestServer {
	hystrixStreamHandler := NewStreamHandler()
	hystrixStreamHandler.Start()
	return &eventStreamTestServer{
		httptest.NewServer(hystrixStreamHandler),
		hystrixStreamHandler,
	}
}

func sleepingCommand(t *testing.T, name string, duration time.Duration) {
	done := make(chan bool)
	errChan := Go(name, func() error {
		time.Sleep(duration)
		done <- true
		return nil
	}, nil)

	select {
	case _ = <-done:
		// do nothing
	case err := <-errChan:
		t.Fatal(err)
	}
}

func failingCommand(t *testing.T, name string, duration time.Duration) {
	done := make(chan bool)
	errChan := Go(name, func() error {
		time.Sleep(duration)
		return fmt.Errorf("fail")
	}, nil)

	select {
	case _ = <-done:
		t.Fatal("should not have succeeded")
	case _ = <-errChan:
		// do nothing
	}
}

func grabFirstCommandFromStream(t *testing.T, url string, commandName string) streamCmdMetric {
	var event streamCmdMetric

	metrics, done := streamMetrics(t, url)
	for m := range metrics {
		if !strings.Contains(m, "HystrixCommand") {
			continue
		}
		if err := json.Unmarshal([]byte(m), &event); err != nil {
			t.Fatal(err)
		}
		if event.Name != commandName {
			continue
		}

		done <- true
		close(done)
		return event
	}

	return event
}

func grabFirstThreadPoolFromStream(t *testing.T, url string, name string) streamThreadPoolMetric {
	var event streamThreadPoolMetric

	metrics, done := streamMetrics(t, url)
	for m := range metrics {
		if !strings.Contains(m, "HystrixThreadPool") {
			continue
		}
		err := json.Unmarshal([]byte(m), &event)
		if err != nil {
			t.Fatal(err)
		}

		if event.Name != name {
			continue
		}

		done <- true
		close(done)
		return event
	}

	return event
}

func streamMetrics(t *testing.T, url string) (chan string, chan bool) {
	metrics := make(chan string, 1)
	done := make(chan bool, 1)

	go func() {
		res, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		buf := []byte{0}
		data := ""
		for {
			_, err := res.Body.Read(buf)
			if err != nil {
				t.Fatal(err)
			}

			data += string(buf)
			if strings.Contains(data, "\n\n") {
				data = strings.Replace(data, "data:{", "{", 1)
				metrics <- data
				data = ""
			}

			select {
			case _ = <-done:
				close(metrics)
				return
			default:
			}
		}
	}()

	return metrics, done
}

func TestEventStream(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testEventStream(t, "eventstream-parallel", "eventstream-errorpercent-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			t.Skip(`TODO: fix me`)

			testEventStream(t, "eventstream-sync", "eventstream-errorpercent-sync")
			synctest.Wait()
		})
	})
}

func testEventStream(t *testing.T, successCircuitName string, failingCircuitName string) {
	server := startTestServer()
	defer server.stopTestServer()

	//after 2 successful commands"
	sleepingCommand(t, successCircuitName, 1*time.Millisecond)
	sleepingCommand(t, successCircuitName, 1*time.Millisecond)

	event := grabFirstCommandFromStream(t, server.URL, successCircuitName)
	if event.Name != successCircuitName {
		t.Fatalf("expected event name to be %v, but was %v", successCircuitName, event.Name)
	}
	if event.RequestCount != 2 {
		t.Fatalf("expected event request count to be 2, but was %v", event.RequestCount)
	}

	// after 1 successful command and 2 unsuccessful commands
	sleepingCommand(t, failingCircuitName, 1*time.Millisecond)
	failingCommand(t, failingCircuitName, 1*time.Millisecond)
	failingCommand(t, failingCircuitName, 1*time.Millisecond)

	metric := grabFirstCommandFromStream(t, server.URL, failingCircuitName)
	if metric.ErrorPct != 67 {
		fmt.Println(metric)
		t.Fatalf("expected metric error percent to be 67, but was %v", metric.ErrorPct)
	}
}

func TestClientCancelEventStream(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testClientCancelEventStream(t, "clientcancel-eventstream-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			t.Skip(`TODO: fix me`)

			testClientCancelEventStream(t, "clientcancel-eventstream-sync")
			synctest.Wait()
		})
	})
}

func testClientCancelEventStream(t *testing.T, circuitName string) {
	server := startTestServer()
	defer server.stopTestServer()

	sleepingCommand(t, circuitName, 1*time.Millisecond)

	// after a client connects
	ctx, cnclFn := context.WithCancel(context.Background())
	defer cnclFn()

	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := new(http.Client)
	wait := make(chan struct{})
	afterFirstRead := &sync.WaitGroup{}
	afterFirstRead.Add(1)

	go func() {
		afr := afterFirstRead
		buf := []byte{0}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		for {
			select {
			case <-wait:
				//wait for master goroutine to break us out
				cnclFn()
				return
			default:
				//read something
				_, err = res.Body.Read(buf)
				if err != nil {
					t.Fatal(err)
				}
				if afr != nil {
					afr.Done()
					afr = nil
				}
			}
		}
	}()
	// need to make sure our request has round-tripped to the server
	afterFirstRead.Wait()

	// it should be registered
	server.StreamHandler.mu.RLock()
	if len(server.StreamHandler.requests) != 1 {
		t.Fatalf("expected 1 request, but got %d", len(server.StreamHandler.requests))
	}
	server.StreamHandler.mu.RUnlock()

	// after client disconnects
	// let the request be cancelled and the body closed
	close(wait)
	// wait for the server to clean up
	time.Sleep(2000 * time.Millisecond)
	// it should be detected as disconnected and de-registered
	// confirm we have 0 clients
	server.StreamHandler.mu.RLock()
	if len(server.StreamHandler.requests) != 0 {
		t.Fatalf("expected 0 request, but got %d", len(server.StreamHandler.requests))
	}
	server.StreamHandler.mu.RUnlock()
}

func TestThreadPoolStream(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testThreadPoolStream(t, "threadpoolstream-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			t.Skip(`TODO: fix me`)

			testThreadPoolStream(t, "threadpoolstream-sync")
			synctest.Wait()
		})
	})
}

func testThreadPoolStream(t *testing.T, circuitName string) {
	server := startTestServer()
	defer server.stopTestServer()

	// after a successful command
	sleepingCommand(t, circuitName, 1*time.Millisecond)
	metric := grabFirstThreadPoolFromStream(t, server.URL, circuitName)

	// the rolling count of executions should increment
	if metric.RollingCountThreadsExecuted != 1 {
		t.Fatalf("expected 1 request, but got %d", metric.RollingCountThreadsExecuted)
	}

	// the pool size should be 10
	if metric.CurrentPoolSize != 10 {
		t.Fatalf("expected 10 request, but got %d", metric.CurrentPoolSize)
	}
}
