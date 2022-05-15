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
	"errors"
	"os"
)

var _ GoroutineExecutor = (*GoroutineExecutorPool)(nil)

type Worker struct {
	ctx      context.Context
	runnable Runnable
	callable Callable
}

func (w Worker) run() {
	w.runnable(w.ctx)
}

func (w Worker) call() any {
	return w.callable(w.ctx)
}

type Task struct {
	ctx    context.Context
	worker Worker
	out    chan any
}

type Option func(opts *GoroutineExecutorPool)

type Options struct {
	maxSize int
	logger  Logger
}

type GoroutineExecutorPool struct {
	poolSize  int
	maxSize   int
	taskQueue chan Task
	logger    Logger
}

func (pool *GoroutineExecutorPool) Execute(task Runnable, ctx context.Context) error {
	if pool.taskQueue == nil {
		return errors.New("async runnable task queue is not enabled")
	}
	pool.taskQueue <- NewTaskr(ctx, task)

	return nil
}

func (pool *GoroutineExecutorPool) Submit(task Callable, ctx context.Context) (Future, error) {
	if pool.taskQueue == nil {
		return nil, errors.New("async callable task queue is not enabled")
	}
	ch := make(chan any)
	taskc, err := NewTaskc(ctx, task, ch)
	if err != nil {
		return nil, err
	}
	pool.taskQueue <- *taskc

	return NewFuture(ch), nil
}

// start - start the goroutine pool
func (pool GoroutineExecutorPool) start() {
	for i := 0; i < pool.poolSize; i++ {
		pool.fire()
	}
}

// fire - create a goroutine and run
func (pool GoroutineExecutorPool) fire() {
	go func() {
		for {
			task, notClosed := <-pool.taskQueue
			if !notClosed {
				pool.logger.Warnf("the taskQueue is closed")
				return
			} else {
				// pool.Execute()
				if task.worker.runnable != nil {
					task.worker.run()
				}

				// pool.Submit()
				if task.worker.callable != nil {
					result := task.worker.call()
					if task.out != nil { // usable here?
						select {
						case <-task.ctx.Done():
							pool.logger.Warnf("the task ctx is closed")
						default:
							task.out <- result
						}
					}
				}
			}
		}
	}()
}

func NewGoroutineExecutorPool(poolSize int, opts ...Option) GoroutineExecutor {
	pool := &GoroutineExecutorPool{
		poolSize: poolSize,
	}

	for _, opt := range opts {
		opt(pool)
	}

	if pool.maxSize > 0 {
		pool.taskQueue = make(chan Task, pool.maxSize)
	} else {
		pool.taskQueue = make(chan Task)
	}

	if pool.logger == nil {
		pool.logger = NewLogger(os.Stdout)
	}

	pool.start()

	pool.logger.Infof("start the goroutine pool successfully...")

	return pool
}

func NewTaskr(ctx context.Context, fx Runnable) Task {
	return Task{
		worker: NewRunnableWorker(ctx, fx),
	}
}

func NewTaskc(ctx context.Context, fx Callable, ch chan any) (*Task, error) {
	if fx == nil {
		return nil, errors.New("async runnable task queue is not enabled")
	}
	return &Task{
		worker: NewCallableWorker(ctx, fx),
		out:    ch,
	}, nil
}

func NewRunnableWorker(ctx context.Context, runnable Runnable) Worker {
	return Worker{
		ctx:      ctx,
		runnable: runnable,
	}
}

func NewCallableWorker(ctx context.Context, callable Callable) Worker {
	return Worker{
		ctx:      ctx,
		callable: callable,
	}
}

func WithMaxSize(maxSize int) Option {
	return func(pool *GoroutineExecutorPool) {
		pool.maxSize = maxSize
	}
}

func WithLogger(logger Logger) Option {
	return func(pool *GoroutineExecutorPool) {
		pool.logger = logger
	}
}
