package poolgo

import (
	`context`
)

type GoroutineExecutor interface {
	Executor
	Submit(task Callable, ctx context.Context) (Future, error)
}
