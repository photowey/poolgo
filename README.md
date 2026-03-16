# poolgo

`poolgo` is a fixed-size goroutine pool for internal reuse.

It is not intended to replace a mature general-purpose scheduling framework. The goal is to provide internal Go services with a unified, maintainable, and explicit concurrency primitive. The current version focuses on the following guarantees:

- fixed-size workers
- two execution models: `Execute` and `Submit`
- preferred typed APIs: `SubmitTyped` and `TypedFuture`
- backward-compatible `Future` support for legacy callers
- graceful shutdown through `Shutdown(ctx)`
- context-aware task submission
- panic recovery without silently losing worker capacity
- customizable panic-to-error conversion

## Good Fits

- controlled asynchronous work inside a service
- IO-heavy work that needs bounded concurrency
- internal components that need consistent panic, cancellation, and shutdown semantics
- a shared entry point for team-wide Go concurrency conventions

## Not a Good Fit

- distributed job scheduling
- persistent task queues
- complex priority scheduling, retry orchestration, or delayed queues
- preemptive cancellation of already running work

## Quick Start

```go
package main

import (
	"context"
	"fmt"

	"github.com/photowey/poolgo"
)

func main() {
	pool, err := poolgo.NewGoroutineExecutorPool(4, poolgo.WithQueueSize(16))
	if err != nil {
		panic(err)
	}
	defer pool.Shutdown(context.Background())

	future, err := poolgo.SubmitTyped[string](pool, func(ctx context.Context) (string, error) {
		return "ok", nil
	}, context.Background())
	if err != nil {
		panic(err)
	}

	result, err := future.Await(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println(result)
}
```

## API Overview

### Construction and Shutdown

- `NewGoroutineExecutorPool(poolSize, opts...) (GoroutineExecutor, error)`
- `MustNewGoroutineExecutorPool(poolSize, opts...) GoroutineExecutor`
- `NewSinglePool(opts...) (GoroutineExecutor, error)`
- `MustNewSinglePool(opts...) GoroutineExecutor`
- `Shutdown(ctx) error`
- `State() PoolState`

### Execution Models

- `Execute(Runnable, ctx)`: fire-and-forget execution without a return value
- `Submit(Callable, ctx)`: legacy-compatible API that returns `Future`
- `SubmitTyped[T](executor, task, ctx)`: preferred API that returns `TypedFuture[T]`

### Options

- `WithQueueSize(n)`: sets the internal queue capacity
- `WithLogger(logger)`: configures pool logging
- `WithPanicHandler(handler)`: customizes panic-to-error conversion

## Recommended Usage

- prefer `SubmitTyped` for all new code
- create the pool during service initialization and call `Shutdown(ctx)` during teardown
- pass bounded `context.Context` values when submitting work
- for fire-and-forget tasks, let the caller own idempotency and compensation
- for tasks that can fail in business logic, return the business error from the typed task itself

## Lifecycle Semantics

- `PoolStateRunning`: accepts new work
- `PoolStateShuttingDown`: rejects new work and drains accepted tasks
- `PoolStateStopped`: all workers have exited

`Shutdown(ctx)` behaves as follows:

- the first call transitions the pool into `shutting_down`
- any submission after that returns `ErrPoolClosed`
- already accepted tasks continue to run
- workers exit only after all accepted tasks are finished
- repeated `Shutdown` calls are safe

## Queue and Context Semantics

- when the queue is full, `Execute` and `Submit` block until the task can be enqueued
- the submission path watches the caller's `context.Context` while blocked
- if the context times out or is canceled first, submission returns immediately with that error
- the task context is checked again before execution; if already canceled, execution is skipped and the error is propagated to the waiter

## Error and Panic Semantics

- `Submit` remains supported for compatibility, but new code should use `SubmitTyped`
- business errors are returned from the typed task itself
- scheduling errors include `ErrInvalidPoolSize`, `ErrInvalidQueueSize`, `ErrPoolClosed`, `ErrNilRunnable`, and `ErrNilCallable`
- task panics are recovered and converted into `ErrTaskPanicked`-derived errors
- `WithPanicHandler` can translate panics into errors that better fit internal conventions

## Backward Compatibility

- `Submit` and `Future` remain available to support incremental migration
- `WithMaxSize` remains available as a compatibility alias for `WithQueueSize`
- older exported symbols such as `Task`, `Worker`, `NewFuture`, and `NewTaskc` still exist, but new code should avoid them

## Migration Guidance

Legacy code:

```go
future, err := pool.Submit(func(ctx context.Context) any {
	return "ok"
}, ctx)
```

Recommended replacement:

```go
future, err := poolgo.SubmitTyped[string](pool, func(ctx context.Context) (string, error) {
	return "ok", nil
}, ctx)
```

This removes `any` assertions and keeps business errors separate from scheduling errors.

## Development and Verification

Useful validation commands:

```bash
go test ./...
go test -race ./...
go vet ./...
go test -run '^$' -bench . ./...
```
