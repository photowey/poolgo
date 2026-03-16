/*
 * Copyright © 2022 photowey (photowey@gmail.com)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package poolgo

import (
	"context"
	"sync"
)

const (
	single = 1
)

var _ Future = (*future)(nil)

// AwaitFunc awaits an asynchronously produced value.
type AwaitFunc func(ctx context.Context) (any, error)

// AwaitFuncFactory builds an AwaitFunc from a result channel.
type AwaitFuncFactory func(ch chan any) AwaitFunc

// Future represents an untyped asynchronous result.
// Prefer TypedFuture with SubmitTyped for new code.
type Future interface {
	Await(ctxs ...context.Context) (any, error)
}

type future struct {
	done   chan struct{}
	once   sync.Once
	result any
	err    error
}

// Await blocks until the future resolves or the provided context is canceled.
func (f *future) Await(ctxs ...context.Context) (any, error) {
	ctx := context.Background()
	switch len(ctxs) {
	case single:
		if ctxs[0] != nil {
			ctx = ctxs[0]
		}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-f.done:
		return f.result, f.err
	}
}

// CreateAwaitFunc creates a channel-backed AwaitFunc.
// Deprecated: prefer Submit or SubmitTyped instead of building futures from channels directly.
func CreateAwaitFunc(ch chan any) AwaitFunc {
	f := NewFuture(ch)
	return func(ctx context.Context) (any, error) {
		return f.Await(ctx)
	}
}

// NewFuture adapts a legacy result channel into a Future.
// Deprecated: prefer Submit or SubmitTyped instead of channels.
func NewFuture(ch chan any) Future {
	f := newFuture()
	if ch == nil {
		f.resolve(nil, ErrNilFutureChannel)
		return f
	}

	go func() {
		result, ok := <-ch
		if !ok {
			f.resolve(nil, ErrFutureChannelClosed)
			return
		}
		f.resolve(result, nil)
	}()

	return f
}

func newFuture() *future {
	return &future{
		done: make(chan struct{}),
	}
}

func (f *future) resolve(result any, err error) {
	f.once.Do(func() {
		f.result = result
		f.err = err
		close(f.done)
	})
}
