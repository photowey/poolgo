package poolgo

import (
	"context"
)

const (
	single = 1
)

var _ Future = (*future)(nil)

// AwaitFunc a func of await
type AwaitFunc func(ctx context.Context) (any, error)

// AwaitFuncFactory a factory of AwaitFunc
type AwaitFuncFactory func(ch chan any) AwaitFunc

// Future async/await programming model
type Future interface {
	Await(ctxs ...context.Context) (any, error)
}

type future struct {
	resultCh chan any
	await    AwaitFunc
}

// Await sync, await
func (f future) Await(ctxs ...context.Context) (any, error) {
	ctx := context.Background() // default: ctx
	switch len(ctxs) {
	case single:
		ctx = ctxs[0]
	}

	return f.await(ctx)
}

// CreateAwaitFunc a func of AwaitFuncFactory
func CreateAwaitFunc(ch chan any) AwaitFunc {
	return func(ctx context.Context) (any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-ch:
			defer func() {
				close(ch)
				// ch <- Task{} // panic: send on closed channel
			}()
			return result, nil
		}
	}
}

func NewFuture(ch chan any) Future {
	return &future{
		resultCh: ch,
		await:    CreateAwaitFunc(ch),
	}
}
