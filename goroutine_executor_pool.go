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
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

var _ GoroutineExecutor = (*GoroutineExecutorPool)(nil)

// Deprecated: Worker is an implementation detail kept for backward compatibility.
type Worker struct {
	ctx      context.Context
	runnable Runnable
	callable Callable
}

func (w Worker) run() {
	w.runnable(normalizeContext(w.ctx))
}

func (w Worker) call() any {
	return w.callable(normalizeContext(w.ctx))
}

// Deprecated: Task is an implementation detail kept for backward compatibility.
type Task struct {
	worker Worker
	future *future
	out    chan any
}

// Option mutates pool configuration during construction.
type Option func(opts *GoroutineExecutorPool)

// Deprecated: Options is reserved for backward compatibility and not used directly by new code.
type Options struct {
	queueSize    int
	logger       Logger
	panicHandler PanicHandler
}

// GoroutineExecutorPool is the default fixed-size worker pool implementation.
type GoroutineExecutorPool struct {
	poolSize     int
	queueSize    int
	taskQueue    chan Task
	logger       Logger
	panicHandler PanicHandler
	state        int32
	shutdownCh   chan struct{}
	terminatedCh chan struct{}
	shutdownOnce sync.Once
	submitMu     sync.Mutex
	submitters   sync.WaitGroup
	workers      sync.WaitGroup
}

func (pool *GoroutineExecutorPool) Execute(task Runnable, ctx context.Context) error {
	if task == nil {
		return ErrNilRunnable
	}

	return pool.enqueue(NewTaskr(normalizeContext(ctx), task))
}

func (pool *GoroutineExecutorPool) Submit(task Callable, ctx context.Context) (Future, error) {
	if task == nil {
		return nil, ErrNilCallable
	}

	f := newFuture()
	taskc, err := newCallableTask(normalizeContext(ctx), task, f)
	if err != nil {
		return nil, err
	}
	if err := pool.enqueue(*taskc); err != nil {
		return nil, err
	}

	return f, nil
}

func (pool *GoroutineExecutorPool) Shutdown(ctx context.Context) error {
	ctx = normalizeContext(ctx)

	pool.shutdownOnce.Do(func() {
		pool.submitMu.Lock()
		atomic.StoreInt32(&pool.state, int32(PoolStateShuttingDown))
		close(pool.shutdownCh)
		pool.submitMu.Unlock()

		go func() {
			pool.submitters.Wait()
			close(pool.taskQueue)
			pool.workers.Wait()
			atomic.StoreInt32(&pool.state, int32(PoolStateStopped))
			pool.logger.Infof("goroutine pool shutdown %s", "completed")
			close(pool.terminatedCh)
		}()
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-pool.terminatedCh:
		return nil
	}
}

func (pool *GoroutineExecutorPool) State() PoolState {
	return PoolState(atomic.LoadInt32(&pool.state))
}

func (pool *GoroutineExecutorPool) enqueue(task Task) error {
	if err := pool.beginSubmit(); err != nil {
		return err
	}
	defer pool.submitters.Done()

	ctx := normalizeContext(task.worker.ctx)
	task.worker.ctx = ctx

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	select {
	case <-pool.shutdownCh:
		return ErrPoolClosed
	case <-ctx.Done():
		return ctx.Err()
	case pool.taskQueue <- task:
		return nil
	}
}

func (pool *GoroutineExecutorPool) beginSubmit() error {
	pool.submitMu.Lock()
	defer pool.submitMu.Unlock()

	if pool.State() != PoolStateRunning {
		return ErrPoolClosed
	}

	pool.submitters.Add(1)
	return nil
}

// start - start the goroutine pool
func (pool *GoroutineExecutorPool) start() {
	for i := 0; i < pool.poolSize; i++ {
		pool.fire()
	}
}

// fire - create a goroutine and run
func (pool *GoroutineExecutorPool) fire() {
	pool.workers.Add(1)
	go func() {
		defer pool.workers.Done()
		for task := range pool.taskQueue {
			pool.runTask(task)
		}
	}()
}

func (pool *GoroutineExecutorPool) runTask(task Task) {
	defer pool.handlePanic(task)

	ctx := normalizeContext(task.worker.ctx)
	task.worker.ctx = ctx
	if err := ctx.Err(); err != nil {
		pool.logger.Warnf("skip task because context is %v", err)
		pool.resolveTask(task, nil, err)
		return
	}

	switch {
	case task.worker.runnable != nil:
		task.worker.run()
	case task.worker.callable != nil:
		pool.resolveTask(task, task.worker.call(), nil)
	default:
		pool.logger.Errorf("reject invalid task: %v", ErrInvalidTask)
		pool.resolveTask(task, nil, ErrInvalidTask)
	}
}

func (pool *GoroutineExecutorPool) handlePanic(task Task) {
	if cause := recover(); cause != nil {
		info := PanicInfo{
			Context:   normalizeContext(task.worker.ctx),
			Recovered: cause,
			Stack:     debug.Stack(),
		}
		err := pool.panicHandler(info)
		if err == nil {
			err = DefaultPanicHandler(info)
		}
		pool.resolveTask(task, nil, err)
		pool.logger.Errorf("task panic recovered: %v\n%s", cause, info.Stack)
	}
}

func (pool *GoroutineExecutorPool) resolveTask(task Task, result any, err error) {
	if task.future != nil {
		task.future.resolve(result, err)
		return
	}

	if task.out == nil {
		return
	}

	defer close(task.out)
	if err != nil {
		return
	}

	select {
	case <-normalizeContext(task.worker.ctx).Done():
		return
	case task.out <- result:
	}
}

// NewGoroutineExecutorPool constructs a fixed-size goroutine pool.
func NewGoroutineExecutorPool(poolSize int, opts ...Option) (GoroutineExecutor, error) {
	if poolSize <= 0 {
		return nil, ErrInvalidPoolSize
	}

	pool := &GoroutineExecutorPool{
		poolSize: poolSize,
		state:    int32(PoolStateRunning),
	}

	for _, opt := range opts {
		opt(pool)
	}

	if pool.queueSize < 0 {
		return nil, ErrInvalidQueueSize
	}
	pool.taskQueue = make(chan Task, pool.queueSize)
	pool.shutdownCh = make(chan struct{})
	pool.terminatedCh = make(chan struct{})

	if pool.logger == nil {
		pool.logger = NewLogger(os.Stdout)
	}
	if pool.panicHandler == nil {
		pool.panicHandler = DefaultPanicHandler
	}

	pool.start()

	pool.logger.Infof("start the goroutine pool %s...", "successfully")

	return pool, nil
}

// MustNewGoroutineExecutorPool constructs a pool or panics if the configuration is invalid.
func MustNewGoroutineExecutorPool(poolSize int, opts ...Option) GoroutineExecutor {
	pool, err := NewGoroutineExecutorPool(poolSize, opts...)
	if err != nil {
		panic(err)
	}

	return pool
}

// NewTaskr creates a runnable task.
// Deprecated: prefer Execute.
func NewTaskr(ctx context.Context, fx Runnable) Task {
	return Task{
		worker: NewRunnableWorker(normalizeContext(ctx), fx),
	}
}

// NewTaskc creates a callable task backed by a result channel.
// Deprecated: prefer Submit or SubmitTyped.
func NewTaskc(ctx context.Context, fx Callable, ch chan any) (*Task, error) {
	if fx == nil {
		return nil, ErrNilCallable
	}
	return &Task{
		worker: NewCallableWorker(normalizeContext(ctx), fx),
		out:    ch,
	}, nil
}

func newCallableTask(ctx context.Context, fx Callable, f *future) (*Task, error) {
	if fx == nil {
		return nil, ErrNilCallable
	}

	return &Task{
		worker: NewCallableWorker(normalizeContext(ctx), fx),
		future: f,
	}, nil
}

// NewRunnableWorker creates a runnable worker wrapper.
// Deprecated: prefer Execute.
func NewRunnableWorker(ctx context.Context, runnable Runnable) Worker {
	return Worker{
		ctx:      normalizeContext(ctx),
		runnable: runnable,
	}
}

// NewCallableWorker creates a callable worker wrapper.
// Deprecated: prefer Submit or SubmitTyped.
func NewCallableWorker(ctx context.Context, callable Callable) Worker {
	return Worker{
		ctx:      normalizeContext(ctx),
		callable: callable,
	}
}

// WithQueueSize configures the internal task queue capacity.
func WithQueueSize(queueSize int) Option {
	return func(pool *GoroutineExecutorPool) {
		pool.queueSize = queueSize
	}
}

// WithMaxSize sets the internal queue capacity.
// Deprecated: use WithQueueSize.
func WithMaxSize(maxSize int) Option {
	return WithQueueSize(maxSize)
}

// WithLogger configures the pool logger.
func WithLogger(logger Logger) Option {
	return func(pool *GoroutineExecutorPool) {
		pool.logger = logger
	}
}

// WithPanicHandler configures how recovered panics are converted into errors.
func WithPanicHandler(handler PanicHandler) Option {
	return func(pool *GoroutineExecutorPool) {
		pool.panicHandler = handler
	}
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}
