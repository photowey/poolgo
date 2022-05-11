package poolgo

import (
	`context`
)

type Runnable func(ctx context.Context)
type Callable func(ctx context.Context) any
