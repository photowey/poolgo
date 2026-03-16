package poolgo

import (
	"context"
	"fmt"
)

// TypedCallable executes a task and returns a typed result plus an execution error.
type TypedCallable[T any] func(ctx context.Context) (T, error)

// TypedFuture awaits a typed asynchronous result.
type TypedFuture[T any] interface {
	Await(ctxs ...context.Context) (T, error)
}

type typedFuture[T any] struct {
	base Future
}

type typedResult[T any] struct {
	value T
	err   error
}

// SubmitTyped submits a typed task while keeping the legacy untyped API compatible.
func SubmitTyped[T any](executor GoroutineExecutor, task TypedCallable[T], ctx context.Context) (TypedFuture[T], error) {
	if task == nil {
		return nil, ErrNilCallable
	}

	future, err := executor.Submit(func(ctx context.Context) any {
		value, err := task(ctx)
		return typedResult[T]{
			value: value,
			err:   err,
		}
	}, ctx)
	if err != nil {
		return nil, err
	}

	return typedFuture[T]{
		base: future,
	}, nil
}

func (f typedFuture[T]) Await(ctxs ...context.Context) (T, error) {
	var zero T

	result, err := f.base.Await(ctxs...)
	if err != nil {
		return zero, err
	}

	resolved, ok := result.(typedResult[T])
	if !ok {
		return zero, fmt.Errorf("%w: got %T", ErrUnexpectedFutureType, result)
	}

	return resolved.value, resolved.err
}
