package poolgo

import (
	"context"
)

type (
	// Runnable executes a task without returning a value.
	Runnable func(ctx context.Context)
	// Callable executes a task and returns an untyped result.
	// Prefer SubmitTyped for new code when a typed result is available.
	Callable func(ctx context.Context) any
)
