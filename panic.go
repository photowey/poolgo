package poolgo

import (
	"context"
	"fmt"
)

// PanicInfo describes a recovered panic from a task execution.
type PanicInfo struct {
	Context   context.Context
	Recovered any
	Stack     []byte
}

// PanicHandler converts a recovered panic into an error returned to callers.
type PanicHandler func(info PanicInfo) error

// PanicError carries recovered panic metadata.
type PanicError struct {
	Recovered any
	Stack     []byte
}

func (err *PanicError) Error() string {
	return fmt.Sprintf("%v: %v", ErrTaskPanicked, err.Recovered)
}

func (err *PanicError) Unwrap() error {
	return ErrTaskPanicked
}

// DefaultPanicHandler converts a recovered panic into a PanicError.
func DefaultPanicHandler(info PanicInfo) error {
	return &PanicError{
		Recovered: info.Recovered,
		Stack:     append([]byte(nil), info.Stack...),
	}
}
