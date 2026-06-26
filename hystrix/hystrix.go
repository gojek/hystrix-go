package hystrix

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gojek/hystrix-go/hystrix/internal/pool"
)

type (
	runFunc       func() error
	fallbackFunc  func(error) error
	runFuncC      func(context.Context) error
	fallbackFuncC func(context.Context, error) error
)

// A CircuitError is an error which models various failure states of execution,
// such as the circuit being open or a timeout.
type CircuitError struct {
	Message string
}

func (e CircuitError) Error() string {
	return "hystrix: " + e.Message
}

// command models the state used for a single execution on a circuit. "hystrix command" is commonly
// used to describe the pairing of your run/fallback functions with a circuit.
type command struct {
	ticket      *struct{}
	start       time.Time
	circuit     *CircuitBreaker
	fallback    fallbackFuncC
	runDuration time.Duration
}

var (
	// ErrMaxConcurrency occurs when too many of the same named command are executed at the same time.
	ErrMaxConcurrency = CircuitError{Message: "max concurrency"}
	// ErrCircuitOpen returns when an execution attempt "short circuits". This happens due to the circuit being measured as unhealthy.
	ErrCircuitOpen = CircuitError{Message: "circuit open"}
	// ErrTimeout occurs when the provided function takes too long to execute.
	ErrTimeout = CircuitError{Message: "timeout"}
)

// Go runs your function while tracking the health of previous calls to it.
// If your function begins slowing down or failing repeatedly, we will block
// new calls to it for you to give the dependent service time to repair.
//
// Define a fallback function if you want to define some code to execute during outages.
func Go(name string, run runFunc, fallback fallbackFunc) chan error {
	runC := func(_ context.Context) error {
		return run()
	}
	var fallbackC fallbackFuncC
	if fallback != nil {
		fallbackC = func(_ context.Context, err error) error {
			return fallback(err)
		}
	}
	return GoC(context.Background(), name, runC, fallbackC)
}

// GoC runs your function while tracking the health of previous calls to it.
// If your function begins slowing down or failing repeatedly, we will block
// new calls to it for you to give the dependent service time to repair.
//
// Define a fallback function if you want to define some code to execute during outages.
func GoC(ctx context.Context, name string, run runFuncC, fallback fallbackFuncC) chan error {
	errChan := make(chan error, 1)
	go func() {
		err := DoC(ctx, name, run, fallback)
		if err != nil {
			errChan <- err // send error to channel only if there is an error
		}
	}()

	return errChan
}

// Do runs your function in a synchronous manner, blocking until either your function succeeds
// or an error is returned, including hystrix circuit errors
func Do(name string, run runFunc, fallback fallbackFunc) error {
	runC := func(_ context.Context) error {
		return run()
	}
	var fallbackC fallbackFuncC
	if fallback != nil {
		fallbackC = func(_ context.Context, err error) error {
			return fallback(err)
		}
	}
	return DoC(context.Background(), name, runC, fallbackC)
}

// DoC runs your function in a synchronous manner, blocking until either your function succeeds
// or an error is returned, including hystrix circuit errors
func DoC(ctx context.Context, name string, run runFuncC, fallback fallbackFuncC) error {
	circuit, _, err := GetCircuit(name)
	if err != nil {
		return err
	}

	cmd := &command{
		start:    time.Now(),
		circuit:  circuit,
		fallback: fallback,
	}

	// Circuits get opened when recent executions have shown to have a high error rate.
	// Rejecting new executions allows backends to recover, and the circuit will allow
	// new traffic when it feels a healthly state has returned.
	if !cmd.circuit.AllowRequest() {
		return cmd.errorWithFallback(ctx, ErrCircuitOpen)
	}

	// As backends falter, requests take longer but don't always fail.
	//
	// When requests slow down but the incoming rate of requests stays the same, you have to
	// run more at a time to keep up. By controlling concurrency during these situations, you can
	// shed load which accumulates due to the increasing ratio of active commands to incoming requests.
	select {
	case cmd.ticket = <-circuit.executorPool.Tickets:
		// when we introduce request queuing calculate ticket elapsed time here,
		// so it can be used to adjust timeout and pass it to metric collector.
	default:
		return cmd.errorWithFallback(ctx, ErrMaxConcurrency)
	}

	runChan := pool.AcquireSingleErrorChan()
	runStart := time.Now()
	go func() {
		runChan <- run(ctx)
	}()

	timer := pool.AcquireTimer(getSettings(name).Timeout)
	defer pool.ReleaseTimer(timer)

	select {
	case runErr := <-runChan:
		cmd.runDuration = time.Since(runStart)
		cmd.circuit.executorPool.Return(cmd.ticket)
		pool.ReleaseSingleErrorChan(runChan) // safe to release as runChan is drained in this select-case block

		if runErr != nil {
			return cmd.errorWithFallback(ctx, runErr)
		}

		cmd.reportEvent(`success`, ``)
		return nil
	case <-ctx.Done():
		cmd.circuit.executorPool.Return(cmd.ticket)

		return cmd.errorWithFallback(ctx, ctx.Err())
	case <-timer.C:
		cmd.circuit.executorPool.Return(cmd.ticket)

		return cmd.errorWithFallback(ctx, ErrTimeout)
	}
}

func (c *command) reportEvent(primaryEvent, secondaryEvent string) {
	err := c.circuit.reportEvent(primaryEvent, secondaryEvent, c.start, c.runDuration)
	if err != nil {
		log.Printf(err.Error())
	}
}

func (c *command) errorWithFallback(ctx context.Context, err error) error {
	primaryEvent := "failure"
	ctxErr := ctx.Err()
	switch {
	case errors.Is(err, ErrCircuitOpen):
		primaryEvent = "short-circuit"
	case errors.Is(err, ErrMaxConcurrency):
		primaryEvent = "rejected"
	case errors.Is(err, ErrTimeout):
		primaryEvent = "timeout"
	case errors.Is(ctxErr, context.Canceled):
		primaryEvent = "context_canceled"
	case errors.Is(ctxErr, context.DeadlineExceeded):
		primaryEvent = "context_deadline_exceeded"
	}

	secondaryEvent, err := c.tryFallback(ctx, err)
	c.reportEvent(primaryEvent, secondaryEvent)
	return err
}

func (c *command) tryFallback(ctx context.Context, err error) (string, error) {
	if c.fallback == nil {
		// If we don't have a fallback return the original error.
		return "", err
	}

	fallbackErr := c.fallback(ctx, err)
	if fallbackErr != nil {
		return "fallback-failure", fmt.Errorf("fallback failed with '%w'. run error was '%w'", fallbackErr, err)
	}

	return "fallback-success", nil
}
