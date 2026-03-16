package poolgo

import "errors"

var (
	ErrInvalidPoolSize      = errors.New("poolgo: pool size must be greater than zero")
	ErrInvalidQueueSize     = errors.New("poolgo: queue size must be greater than or equal to zero")
	ErrPoolClosed           = errors.New("poolgo: goroutine executor pool is shutting down")
	ErrNilRunnable          = errors.New("poolgo: runnable task is nil")
	ErrNilCallable          = errors.New("poolgo: callable task is nil")
	ErrInvalidTask          = errors.New("poolgo: task has no runnable or callable")
	ErrTaskPanicked         = errors.New("poolgo: task panicked")
	ErrNilFutureChannel     = errors.New("poolgo: future result channel is nil")
	ErrFutureChannelClosed  = errors.New("poolgo: future result channel closed before resolution")
	ErrUnexpectedFutureType = errors.New("poolgo: unexpected future result type")
)
