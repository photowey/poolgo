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

package poolgo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/photowey/poolgo"
)

func TestNewGoroutineExecutorPoolValidateConfig(t *testing.T) {
	t.Parallel()

	_, err := poolgo.NewGoroutineExecutorPool(0)
	if !errors.Is(err, poolgo.ErrInvalidPoolSize) {
		t.Fatalf("expected ErrInvalidPoolSize, got %v", err)
	}

	_, err = poolgo.NewGoroutineExecutorPool(1, poolgo.WithQueueSize(-1))
	if !errors.Is(err, poolgo.ErrInvalidQueueSize) {
		t.Fatalf("expected ErrInvalidQueueSize, got %v", err)
	}
}

func TestExecuteRejectsNilRunnable(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t, 1)
	if err := pool.Execute(nil, context.Background()); !errors.Is(err, poolgo.ErrNilRunnable) {
		t.Fatalf("expected ErrNilRunnable, got %v", err)
	}
}

func TestSubmitRejectsNilCallable(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t, 1)
	if _, err := pool.Submit(nil, context.Background()); !errors.Is(err, poolgo.ErrNilCallable) {
		t.Fatalf("expected ErrNilCallable, got %v", err)
	}
}

func TestSubmitAwaitCanBeCalledMultipleTimes(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t, 1)
	future, err := pool.Submit(func(ctx context.Context) any {
		return "ok"
	}, context.Background())
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}

	for i := 0; i < 2; i++ {
		got, err := future.Await()
		if err != nil {
			t.Fatalf("await #%d: %v", i+1, err)
		}
		if got != "ok" {
			t.Fatalf("await #%d result mismatch: %v", i+1, got)
		}
	}
}

func TestSubmitTypedReturnsTypedValueAndTaskError(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t, 1)
	taskErr := errors.New("typed failure")
	typedFuture, err := poolgo.SubmitTyped[string](pool, func(ctx context.Context) (string, error) {
		return "", taskErr
	}, context.Background())
	if err != nil {
		t.Fatalf("submit typed task: %v", err)
	}

	value, err := typedFuture.Await()
	if value != "" {
		t.Fatalf("expected zero string, got %q", value)
	}
	if !errors.Is(err, taskErr) {
		t.Fatalf("expected typed failure, got %v", err)
	}
}

func TestSubmitResultDoesNotBlockWorkerProgress(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t, 1)
	future, err := pool.Submit(func(ctx context.Context) any {
		return "ready"
	}, context.Background())
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}

	done := make(chan struct{})
	if err := pool.Execute(func(ctx context.Context) {
		close(done)
	}, context.Background()); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker stalled while previous future result was not awaited")
	}

	got, err := future.Await()
	if err != nil {
		t.Fatalf("await future: %v", err)
	}
	if got != "ready" {
		t.Fatalf("unexpected future result: %v", got)
	}
}

func TestSubmitRespectsContextWhileQueueIsBlocked(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t, 1)
	release := make(chan struct{})
	slowFuture, err := pool.Submit(func(ctx context.Context) any {
		<-release
		return "slow"
	}, context.Background())
	if err != nil {
		t.Fatalf("submit slow task: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = pool.Submit(func(ctx context.Context) any {
		return "blocked"
	}, ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}

	close(release)
	if _, err := slowFuture.Await(); err != nil {
		t.Fatalf("await slow future: %v", err)
	}
}

func TestShutdownRejectsNewTasksAndWaitsForRunningWork(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})

	if err := pool.Execute(func(ctx context.Context) {
		close(started)
		<-release
		close(finished)
	}, context.Background()); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("task did not start")
	}

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- pool.Shutdown(context.Background())
	}()

	select {
	case err := <-shutdownDone:
		t.Fatalf("shutdown returned before running task finished: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	if err := pool.Execute(func(context.Context) {}, context.Background()); !errors.Is(err, poolgo.ErrPoolClosed) {
		t.Fatalf("expected ErrPoolClosed after shutdown starts, got %v", err)
	}

	close(release)

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("shutdown failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not complete")
	}

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("running task was not allowed to finish")
	}

	if state := pool.State(); state != poolgo.PoolStateStopped {
		t.Fatalf("expected stopped state, got %s", state)
	}
}

func TestTaskPanicIsRecoveredAndWorkerRemainsAvailable(t *testing.T) {
	t.Parallel()

	pool := newTestPool(t, 1)
	future, err := pool.Submit(func(ctx context.Context) any {
		panic("boom")
	}, context.Background())
	if err != nil {
		t.Fatalf("submit panic task: %v", err)
	}

	if _, err := future.Await(); !errors.Is(err, poolgo.ErrTaskPanicked) {
		t.Fatalf("expected ErrTaskPanicked, got %v", err)
	}

	done := make(chan struct{})
	if err := pool.Execute(func(ctx context.Context) {
		close(done)
	}, context.Background()); err != nil {
		t.Fatalf("execute follow-up task: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not recover after panic")
	}
}

func TestCustomPanicHandlerOverridesReturnedError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("panic converted")
	handlerErr := make(chan error, 1)
	pool := newTestPool(t, 1, poolgo.WithPanicHandler(func(info poolgo.PanicInfo) error {
		if info.Context == nil {
			handlerErr <- errors.New("panic info context should not be nil")
			return sentinel
		}
		if len(info.Stack) == 0 {
			handlerErr <- errors.New("panic info stack should not be empty")
			return sentinel
		}
		return sentinel
	}))

	future, err := pool.Submit(func(ctx context.Context) any {
		panic("boom")
	}, context.Background())
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}

	if _, err := future.Await(); !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}

	select {
	case err := <-handlerErr:
		t.Fatal(err)
	default:
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	t.Parallel()

	pool, err := poolgo.NewGoroutineExecutorPool(1, poolgo.WithLogger(poolgo.NewDiscardLogger()))
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := pool.Shutdown(ctx); err != nil {
		t.Fatalf("first shutdown: %v", err)
	}
	if err := pool.Shutdown(ctx); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}
}

func newTestPool(t *testing.T, poolSize int, opts ...poolgo.Option) poolgo.GoroutineExecutor {
	t.Helper()

	opts = append(opts, poolgo.WithLogger(poolgo.NewDiscardLogger()))
	pool, err := poolgo.NewGoroutineExecutorPool(poolSize, opts...)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := pool.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("shutdown pool: %v", err)
		}
	})

	return pool
}
