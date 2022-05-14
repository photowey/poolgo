package poolgo

import (
	"context"
)

type (
	Runnable func(ctx context.Context)
	Callable func(ctx context.Context) any
)
