package hystrix

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"testing/quick"
)

func TestSuccess(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testSuccess(t, "hystrix-success-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testSuccess(t, "hystrix-success-sync")
			synctest.Wait()
		})
	})
}

func testSuccess(t *testing.T, circuitName string) {
	resultChan := make(chan int)
	errChan := GoC(context.Background(), circuitName, func(ctx context.Context) error {
		resultChan <- 1
		return nil
	}, nil)

	// reading from that channel should provide the expected value
	if val := <-resultChan; val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := len(errChan); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}

	time.Sleep(time.Millisecond)
	cb, _, _ := GetCircuit(circuitName)
	// metrics are recorded
	if val := cb.metrics.DefaultCollector().Successes().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
}

func TestFallback(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testFallback(t, "hystrix-fallback-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testFallback(t, "hystrix-fallback-sync")
			synctest.Wait()
		})
	})
}

func testFallback(t *testing.T, circuitName string) {
	resultChan := make(chan int)
	errChan := GoC(context.Background(), circuitName, func(ctx context.Context) error {
		return fmt.Errorf("error")
	}, func(ctx context.Context, err error) error {
		if err.Error() == "error" {
			resultChan <- 1
		}
		return nil
	})

	// reading from that channel should provide the expected value
	if val := <-resultChan; val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}

	if val := len(errChan); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	time.Sleep(time.Millisecond)
	cb, _, _ := GetCircuit(circuitName)
	if val := cb.metrics.DefaultCollector().Successes().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Failures().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().FallbackSuccesses().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testTimeout(t, "hystrix-timeout-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testTimeout(t, "hystrix-timeout-sync")
			synctest.Wait()
		})
	})
}

func testTimeout(t *testing.T, circuitName string) {
	ctx := t.Context()
	ConfigureCommand(circuitName, CommandConfig{Timeout: 100})

	resultChan := make(chan int, 1)
	errChan := GoC(ctx, circuitName, func(ctx context.Context) error {
		interuptibleSleep(ctx, time.Second)
		resultChan <- 1
		return nil
	}, func(ctx context.Context, err error) error {
		if err == ErrTimeout {
			resultChan <- 2
		}
		return nil
	})

	if val := <-resultChan; val != 2 {
		t.Errorf("expected 2 but got %v", val)
	}
	if val := len(errChan); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
}

func TestTimeoutEmptyFallback(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testTimeoutEmptyFallback(t, "test-timeout-empty-fallback-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testTimeoutEmptyFallback(t, "test-timeout-empty-fallback-sync")
			synctest.Wait()
		})
	})
}

func testTimeoutEmptyFallback(t *testing.T, circuitName string) {
	ctx := t.Context()
	ConfigureCommand(circuitName, CommandConfig{Timeout: 100})

	errChan := GoC(ctx, circuitName, func(ctx context.Context) error {
		interuptibleSleep(ctx, 1*time.Second)
		return nil
	}, nil)

	if val := <-errChan; !errors.Is(val, ErrTimeout) {
		t.Errorf("expected ErrTimeout but got %v", val)
	}

	cb, _, _ := GetCircuit(circuitName)
	if val := cb.metrics.DefaultCollector().Successes().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().FallbackSuccesses().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().FallbackFailures().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
}

func TestMaxConcurrent(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testMaxConcurrent(t, "hystrix-max-concurrent-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testMaxConcurrent(t, "hystrix-max-concurrent-sync")
			synctest.Wait()
		})
	})
}

func testMaxConcurrent(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{MaxConcurrentRequests: 2})

	run := func(ctx context.Context) error {
		interuptibleSleep(ctx, time.Second)
		return nil
	}

	// and 3 of those commands try to execute at the same time
	var good, bad int

	for i := 0; i < 3; i++ {
		errChan := GoC(t.Context(), circuitName, run, nil)
		time.Sleep(10 * time.Millisecond)

		select {
		case err := <-errChan:
			if err == ErrMaxConcurrency {
				bad++
			}
		default:
			good++
		}
	}

	if bad != 1 {
		t.Errorf("expected 1 but got %v", bad)
	}
	if good != 2 {
		t.Errorf("expected 2 but got %v", good)
	}
}

func TestForceOpenCircuit(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testForceOpenCircuit(t, "hystrix-force-open-circuit-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testForceOpenCircuit(t, "hystrix-force-open-circuit-sync")
			synctest.Wait()
		})
	})
}

func testForceOpenCircuit(t *testing.T, circuitName string) {
	cb, _, err := GetCircuit(circuitName)
	if err != nil {
		t.Fatalf("unexpected error getting circuit: %v", err)
	}

	cb.toggleForceOpen(true)

	errChan := GoC(context.Background(), circuitName, func(ctx context.Context) error {
		return nil
	}, nil)

	// a 'circuit open' error is returned"
	if val := <-errChan; !errors.Is(val, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen but got %v", val)
	}

	time.Sleep(time.Millisecond)
	cb, _, _ = GetCircuit(circuitName)
	if val := cb.metrics.DefaultCollector().Successes().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ShortCircuits().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
}

func TestNilFallbackRunError(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testNilFallbackRunError(t, "hystrix-nil-fallback-run-error-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testNilFallbackRunError(t, "hystrix-nil-fallback-run-error-sync")
			synctest.Wait()
		})
	})
}

func testNilFallbackRunError(t *testing.T, circuitName string) {
	errChan := GoC(context.Background(), circuitName, func(ctx context.Context) error {
		return fmt.Errorf("run_error")
	}, nil)

	if err := <-errChan; err.Error() != "run_error" {
		t.Errorf("expected run_error but got %v", err.Error())
	}
}

func TestFailedFallback(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testFailedFallback(t, "hystrix-failed-fallback-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testFailedFallback(t, "hystrix-failed-fallback-sync")
			synctest.Wait()
		})
	})
}

func testFailedFallback(t *testing.T, circuitName string) {
	errChan := GoC(context.Background(), circuitName, func(ctx context.Context) error {
		return fmt.Errorf("run_error")
	}, func(ctx context.Context, err error) error {
		return fmt.Errorf("fallback_error")
	})

	if err := <-errChan; err.Error() != "fallback failed with 'fallback_error'. run error was 'run_error'" {
		t.Errorf(`expected "fallback failed with 'fallback_error'. run error was 'run_error'" but got %v`, err.Error())
	}
}

func TestCloseCircuitAfterSuccess(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testCloseCircuitAfterSuccess(t, "hystrix-close-circuit-after-success-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testCloseCircuitAfterSuccess(t, "hystrix-close-circuit-after-success-sync")
			synctest.Wait()
		})
	})
}

func testCloseCircuitAfterSuccess(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{SleepWindow: 500})
	cb, _, err := GetCircuit(circuitName)
	if err != nil {
		t.Fatalf("unexpected error getting circuit: %v", err)
	}
	cb.setOpen()

	// commands immediately following should short-circuit
	errChan := GoC(context.Background(), circuitName, func(ctx context.Context) error {
		return nil
	}, nil)
	if val := <-errChan; !errors.Is(val, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen but got %v", val)
	}

	// and a successful command is run after the sleep window
	time.Sleep(600 * time.Millisecond)

	done := make(chan bool, 1)
	GoC(context.Background(), circuitName, func(ctx context.Context) error {
		done <- true
		return nil
	}, nil)

	// the circuit should be closed
	if val := <-done; !val {
		t.Errorf("expected done to be true")
	}
	time.Sleep(100 * time.Millisecond)
	if cb.IsOpen() {
		t.Errorf("circuit should not be open")
	}
}

func TestFailAfterTimeout(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testFailAfterTimeout(t, "hystrix-fail-after-timeout-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testFailAfterTimeout(t, "hystrix-fail-after-timeout-sync")
			synctest.Wait()
		})
	})
}

func testFailAfterTimeout(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{Timeout: 10})

	out := make(chan struct{}, 2)
	errChan := GoC(context.Background(), circuitName, func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return fmt.Errorf("foo")
	}, func(ctx context.Context, err error) error {
		out <- struct{}{}
		return err
	})

	// wait for command to fail, should not panic
	if val := <-errChan; !strings.Contains(val.Error(), ErrTimeout.Error()) {
		t.Errorf("expected ErrTimeout but got %v", val)
	}
	time.Sleep(100 * time.Millisecond)

	// we do not call the fallback twice
	time.Sleep(100 * time.Millisecond)
	if val := len(out); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
}

func TestSlowFallbackOpenCircuit(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testSlowFallbackOpenCircuit(t, "hystrix-slow-fallback-open-circuit-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testSlowFallbackOpenCircuit(t, "hystrix-slow-fallback-open-circuit-sync")
			synctest.Wait()
		})
	})
}

func testSlowFallbackOpenCircuit(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{Timeout: 10})

	cb, _, err := GetCircuit(circuitName)
	if err != nil {
		t.Fatalf("unexpected error getting circuit: %v", err)
	}
	cb.setOpen()

	out := make(chan struct{}, 2)

	// when the command short circuits
	GoC(t.Context(), circuitName, func(ctx context.Context) error {
		return nil
	}, func(ctx context.Context, err error) error {
		time.Sleep(100 * time.Millisecond)
		out <- struct{}{}
		return nil
	})

	// the fallback only fires for the short-circuit, not both
	time.Sleep(250 * time.Millisecond)
	if val := len(out); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}

	if val := cb.metrics.DefaultCollector().ShortCircuits().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
}

func TestFallbackAfterRejected(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testFallbackAfterRejected(t, "hystrix-fallback-after-rejected-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testFallbackAfterRejected(t, "hystrix-fallback-after-rejected-sync")
			synctest.Wait()
		})
	})
}

func testFallbackAfterRejected(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{MaxConcurrentRequests: 1})
	cb, _, err := GetCircuit(circuitName)
	if err != nil {
		t.Fatal(err)
	}
	<-cb.executorPool.Tickets

	// executing a successful fallback function due to rejection
	runChan := make(chan bool, 1)
	fallbackChan := make(chan bool, 1)
	GoC(context.Background(), circuitName, func(ctx context.Context) error {
		// if run executes after fallback, this will panic due to sending to a closed channel
		runChan <- true
		close(fallbackChan)
		return nil
	}, func(ctx context.Context, err error) error {
		fallbackChan <- true
		close(runChan)
		return nil
	})

	if val := <-fallbackChan; !val {
		t.Errorf("expected fallback to be true")
	}
	if val := <-runChan; val {
		t.Errorf("expected run to be false")
	}
}

func TestReturnTicket_QuickCheck(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testReturnTicket_QuickCheck(t, "hystrix-return-ticket-quick-check-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testReturnTicket_QuickCheck(t, "hystrix-return-ticket-quick-check-sync")
			synctest.Wait()
		})
	})
}

func testReturnTicket_QuickCheck(t *testing.T, circuitName string) {
	compareTicket := func() bool {
		ConfigureCommand(circuitName, CommandConfig{Timeout: 2})
		errChan := GoC(t.Context(), circuitName, func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		}, nil)

		if err := <-errChan; err == nil {
			t.Error("expected error but got nil")
		}
		cb, _, err := GetCircuit(circuitName)
		if err != nil {
			t.Fatalf("unexpected error getting circuit: %v", err)
		}
		return cb.executorPool.ActiveCount() == 0
	}

	// with a run command that doesn't return
	// checking many times that after GoC(context.Background(), ), the ticket returns to the pool after the timeout
	err := quick.Check(compareTicket, nil)
	if err != nil {
		t.Fatalf("unexpected error from quick.Check: %v", err)
	}
}

func TestReturnTicket(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testReturnTicket(t, "hystrix-return-ticket-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testReturnTicket(t, "hystrix-return-ticket-sync")
			synctest.Wait()
		})
	})
}

func testReturnTicket(t *testing.T, circuitName string) {
	ctx := t.Context()
	ConfigureCommand(circuitName, CommandConfig{Timeout: 10})

	errChan := GoC(ctx, circuitName, func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}, nil)

	// after GoC(context.Background(), ), the ticket returns to the pool after the timeout
	if err := <-errChan; err == nil {
		t.Error("expected error but got nil")
	}

	cb, _, err := GetCircuit(circuitName)
	if err != nil {
		t.Fatalf("unexpected error getting circuit: %v", err)
	}
	if val := cb.executorPool.ActiveCount(); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
}

func TestContextHandling(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testContextHandling(t, "hystrix-context-handling-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testContextHandling(t, "hystrix-context-handling-sync")
			synctest.Wait()
		})
	})
}

func testContextHandling(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{Timeout: 15})
	cb, _, err := GetCircuit(circuitName)
	if err != nil {
		t.Fatal(err)
	}

	run := func(ctx context.Context) error {
		interuptibleSleep(ctx, 200*time.Millisecond)
		return nil
	}

	fallback := func(ctx context.Context, e error) error {
		return nil
	}

	//with a valid context
	errChan := GoC(t.Context(), circuitName, run, nil)
	time.Sleep(time.Millisecond)
	if err := <-errChan; err != ErrTimeout {
		t.Errorf("expected ErrTimeout but got %v", err)
	}
	if val := cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Failures().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	cb.metrics.DefaultCollector().Reset()

	//with a valid context and a fallback
	errChan = GoC(t.Context(), circuitName, run, fallback)
	time.Sleep(25 * time.Millisecond)
	if val := len(errChan); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Failures().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().FallbackSuccesses().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	cb.metrics.DefaultCollector().Reset()

	//with a context timeout
	testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	errChan = GoC(testCtx, circuitName, run, nil)
	time.Sleep(time.Millisecond)
	if err := <-errChan; err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded but got %v", err)
	}

	if val := cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Failures().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	cancel()
	cb.metrics.DefaultCollector().Reset()

	//with a context timeout and a fallback
	testCtx, cancel = context.WithTimeout(context.Background(), 5*time.Millisecond)
	errChan = GoC(testCtx, circuitName, run, fallback)
	time.Sleep(25 * time.Millisecond)
	if val := len(errChan); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Failures().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().FallbackSuccesses().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	cancel()
	cb.metrics.DefaultCollector().Reset()

	//with a canceled context
	testCtx, cancel = context.WithCancel(context.Background())
	errChan = GoC(testCtx, circuitName, run, nil)
	time.Sleep(5 * time.Millisecond)
	cancel()
	time.Sleep(time.Millisecond)
	if err := <-errChan; err != context.Canceled {
		t.Errorf("expected context.Canceled but got %v", err)
	}
	if val := cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Failures().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	cb.metrics.DefaultCollector().Reset()

	//with a canceled context and a fallback
	testCtx, cancel = context.WithCancel(context.Background())
	errChan = GoC(testCtx, circuitName, run, fallback)
	time.Sleep(5 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
	if val := len(errChan); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().NumRequests().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Failures().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().Timeouts().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextCanceled().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().ContextDeadlineExceeded().Sum(time.Now()); val != 0 {
		t.Errorf("expected 0 but got %v", val)
	}
	if val := cb.metrics.DefaultCollector().FallbackSuccesses().Sum(time.Now()); val != 1 {
		t.Errorf("expected 1 but got %v", val)
	}
}

func TestDoC_Success(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testDoC_Success(t, "hystrix-doc-success-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testDoC_Success(t, "hystrix-doc-success-sync")
			synctest.Wait()
		})
	})
}

func testDoC_Success(t *testing.T, circuitName string) {
	out := make(chan bool, 1)
	err := DoC(context.Background(), circuitName, func(ctx context.Context) error {
		out <- true
		return nil
	}, nil)
	if err != nil {
		t.Errorf("expected success but got %v", err)
	}
	if val := <-out; val != true {
		t.Errorf("expected true but got %v", val)
	}
}

func TestDoC_Fails(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testDoC_Fails(t, "hystrix-doc-fails-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testDoC_Fails(t, "hystrix-doc-fails-sync")
			synctest.Wait()
		})
	})
}

func testDoC_Fails(t *testing.T, circuitName string) {
	run := func(ctx context.Context) error {
		return fmt.Errorf("i failed")
	}

	err := DoC(context.Background(), circuitName, run, nil)
	if err.Error() != "i failed" {
		t.Errorf("expected 'i failed' but got %v", err)
	}

	// with a succeeding fallback"
	out := make(chan bool, 1)
	err = DoC(context.Background(), circuitName, run, func(ctx context.Context, err error) error {
		out <- true
		return nil
	})
	if err != nil {
		t.Errorf("expected success but got %v", err)
	}
	if val := <-out; val != true {
		t.Errorf("expected true but got %v", val)
	}

	// with a failing fallback"
	err = DoC(context.Background(), circuitName, run, func(ctx context.Context, err error) error {
		return fmt.Errorf("fallback failed")
	})
	if err.Error() != "fallback failed with 'fallback failed'. run error was 'i failed'" {
		t.Errorf(`expected "fallback failed with 'fallback failed'. run error was 'i failed'" but got %v`, err)
	}
}

func TestDoC_Timesout(t *testing.T) {
	t.Parallel()
	t.Run(`parallel`, func(t *testing.T) {
		t.Parallel()
		testDoC_Timesout(t, "hystrix-doc-timesout-parallel")
	})
	t.Run(`sync`, func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testDoC_Timesout(t, "hystrix-doc-timesout-sync")
			synctest.Wait()
		})
	})
	testDoC_Timesout(t, "hystrix-doc-timesout-parallel")
}

func testDoC_Timesout(t *testing.T, circuitName string) {
	ConfigureCommand(circuitName, CommandConfig{Timeout: 10})

	err := DoC(t.Context(), circuitName, func(ctx context.Context) error {
		interuptibleSleep(ctx, 100*time.Millisecond)
		return nil
	}, nil)

	if err != ErrTimeout {
		t.Errorf("expected ErrTimeout but got %v", err)
	}
}

func interuptibleSleep(ctx context.Context, duration time.Duration) {
	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}
}

// go test -bench="Benchmark.*" -run ^$ -benchtime=3s -count=6 ./
// go test -bench="Benchmark.*" -run ^$ -memprofile mem.out ./
// go test -bench="Benchmark.*" -run ^$ -cpuprofile cpu.out ./
func BenchmarkDoC(b *testing.B) {
	//b.Skip()
	const name = "bench"
	ConfigureCommand(name, CommandConfig{Timeout: 50, MaxConcurrentRequests: 200})
	_, _, _ = GetCircuit(name)

	b.SetParallelism(1)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := DoC(b.Context(), name, func(ctx context.Context) error {
				return nil
			}, nil)
			if err != nil {
				b.Fatalf("expected success but got %v", err)
			}
		}
	})
}
