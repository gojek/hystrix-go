package hystrix

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"math/rand/v2"
	"testing/quick"

	"github.com/gojek/hystrix-go/hystrix/internal/test"
)

func TestGetCircuit(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		const circuitName = `foo`
		_, created, err := GetCircuit(circuitName)
		if err != nil {
			t.Fatalf("GetCircuit() failed: %v", err)
		}
		if !created {
			t.Errorf("expected circuit to be created, but it was not")
		}

		_, created, err = GetCircuit(circuitName)
		if err != nil {
			t.Fatalf("GetCircuit() failed: %v", err)
		}
		if created {
			t.Errorf("expected new circuit to be not created, but it was created")
		}
	})
}

func TestMultithreadedGetCircuit(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		numThreads := 100
		var numCreates int32
		var numRunningRoutines int32
		var startingLine sync.WaitGroup
		var finishLine sync.WaitGroup
		startingLine.Add(1)
		finishLine.Add(numThreads)

		for i := 0; i < numThreads; i++ {
			go func() {
				if atomic.AddInt32(&numRunningRoutines, 1) == int32(numThreads) {
					startingLine.Done()
				} else {
					startingLine.Wait()
				}

				_, created, _ := GetCircuit(`foo-multi-threaded-sync`)

				if created {
					atomic.AddInt32(&numCreates, 1)
				}

				finishLine.Done()
			}()
		}

		finishLine.Wait()

		if numCreates != 1 {
			t.Errorf("expected exactly 1 circuit to be created, but %d were created", numCreates)
		}
	})
}

func TestReportEventOpenThenClose(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		const circuitName = "foo-report"
		ConfigureCommand(circuitName, CommandConfig{ErrorPercentThreshold: 50})

		cb, _, err := GetCircuit(circuitName)
		if err != nil {
			t.Fatalf("GetCircuit() failed: %v", err)
		}
		if cb.IsOpen() {
			t.Fatalf("circuit should not be open")
		}
		cb.mutex.RLock()
		openedTime := cb.openedOrLastTestedTime
		cb.mutex.RUnlock()

		// unhealthy metrics
		cb.metrics = metricFailingPercent(circuitName, 100)
		if cb.metrics.IsHealthy(time.Now()) {
			t.Fatalf("circuit should not be healthy")
		}

		err = cb.ReportEvent([]string{"success"}, time.Now(), 0)
		if err != nil {
			t.Fatalf("ReportEvent() failed: %v", err)
		}

		cb.mutex.RLock()
		recentOpenedTime := cb.openedOrLastTestedTime
		cb.mutex.RUnlock()
		if recentOpenedTime != openedTime {
			t.Errorf("expected openedOrLastTestedTime to remain unchanged, but it changed from %v to %v", openedTime, cb.openedOrLastTestedTime)
		}
	})
}

func TestReportEventMultiThreaded(t *testing.T) {
	t.Parallel()
	test.SyncTest(t, func(t *testing.T) {
		const circuitName = `foo-report-multi-threaded`
		run := func() bool {
			// Make the circuit easily open and close intermittently.
			ConfigureCommand(circuitName, CommandConfig{
				MaxConcurrentRequests:  1,
				ErrorPercentThreshold:  1,
				RequestVolumeThreshold: 1,
				SleepWindow:            10,
			})
			cb, _, _ := GetCircuit(circuitName)
			count := 5
			wg := &sync.WaitGroup{}
			wg.Add(count)
			c := make(chan bool, count)
			for i := 0; i < count; i++ {
				go func() {
					defer func() {
						if r := recover(); r != nil {
							t.Error(r)
							c <- false
						} else {
							wg.Done()
						}
					}()
					// randomized eventType to open/close circuit
					eventType := "rejected"
					if rand.IntN(3) == 1 {
						eventType = "success"
					}
					err := cb.ReportEvent([]string{eventType}, time.Now(), time.Second)
					if err != nil {
						t.Error(err)
					}
					time.Sleep(time.Millisecond)
					// cb.IsOpen() internally calls cb.setOpen()
					cb.IsOpen()
				}()
			}
			go func() {
				wg.Wait()
				c <- true
			}()
			return <-c
		}
		if err := quick.Check(run, nil); err != nil {
			t.Error(err)
		}
	})
}
