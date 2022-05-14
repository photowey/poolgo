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
	"fmt"
)

var _ GoroutineExecutor = (*GoroutineExecutorPool)(nil)

type Task struct {
	ctx      context.Context
	runnable Runnable
	callable Callable
	resultCh chan any
}

type GoroutineExecutorPool struct {
	poolSize  int
	maxSize   int
	taskQueue chan Task
}

func (pool *GoroutineExecutorPool) Execute(task Runnable, ctx context.Context) error {
	if pool.taskQueue != nil {
		pool.taskQueue <- NewTaskr(ctx, task)
	} else {
		return errors.New("async runnable task queue is not enabled")
	}

	return nil
}

func (pool *GoroutineExecutorPool) Submit(task Callable, ctx context.Context) (Future, error) {
	if pool.taskQueue != nil {
		ch := make(chan any)
		taskc, err := NewTaskc(ctx, task, ch)
		if err != nil {
			return nil, err
		}
		pool.taskQueue <- *taskc
		return NewFuture(ch), nil
	} else {
		return nil, errors.New("async callable task queue is not enabled")
	}
}

func NewGoroutineExecutorPool(poolSize int, maxSizes ...int) GoroutineExecutor {
	max := -1
	switch len(maxSizes) {
	case 1:
		max = maxSizes[0]
	}

	pool := &GoroutineExecutorPool{
		poolSize: poolSize,
		maxSize:  max,
	}

	if max < 0 {
		pool.taskQueue = make(chan Task)
	} else {
		pool.taskQueue = make(chan Task, max)
	}

	for i := 0; i < pool.poolSize; i++ {
		go func() {
			for {
				task, notClosed := <-pool.taskQueue
				if !notClosed {
					fmt.Println("the taskQueue is closed")
					return
				} else {
					// pool.Execute()
					if task.runnable != nil {
						task.runnable(task.ctx)
					}

					// pool.Submit()
					if task.callable != nil {
						result := task.callable(task.ctx)
						if task.resultCh != nil { // usable here?
							select {
							case <-task.ctx.Done():
								fmt.Println("the resultCh is closed")
							default:
								task.resultCh <- result
							}
						}
					}
				}
			}
		}()
	}

	return pool
}

func NewTaskr(ctx context.Context, task Runnable) Task {
	return Task{
		ctx:      ctx,
		runnable: task,
	}
}

func NewTaskc(ctx context.Context, task Callable, ch chan any) (*Task, error) {
	if task == nil {
		return nil, errors.New("async runnable task queue is not enabled")
	}
	return &Task{
		ctx:      ctx,
		callable: task,
		resultCh: ch,
	}, nil
}
