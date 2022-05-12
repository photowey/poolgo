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
	`context`
	`errors`
	`fmt`
)

var _ GoroutineExecutor = (*GoroutineExecutorPool)(nil)

type Task struct {
	runnable Runnable
	callable Callable
	resultCh chan any
	ctx      context.Context
}

func NewTaskr(task Runnable, ctx context.Context) Task {
	return Task{
		runnable: task, ctx: ctx,
	}
}

func NewTaskc(task Callable, ctx context.Context, ch chan any) (*Task, error) {
	if task == nil {
		return nil, errors.New("async runnable task queue is not enabled")
	}
	return &Task{
		callable: task, ctx: ctx, resultCh: ch,
	}, nil
}

type GoroutineExecutorPool struct {
	poolSize         int
	maxTaskQueueSize int
	taskQueue        chan Task
}

func NewGoroutineExecutorPool(poolSize, maxTaskQueueSize int) GoroutineExecutor {
	pool := &GoroutineExecutorPool{
		poolSize:         poolSize,
		maxTaskQueueSize: maxTaskQueueSize,
		taskQueue:        make(chan Task, maxTaskQueueSize),
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

func (pool *GoroutineExecutorPool) Execute(task Runnable, ctx context.Context) error {
	if pool.taskQueue != nil {
		pool.taskQueue <- NewTaskr(task, ctx)
	} else {
		return errors.New("async runnable task queue is not enabled")
	}

	return nil
}

func (pool *GoroutineExecutorPool) Submit(task Callable, ctx context.Context) (Future, error) {
	if pool.taskQueue != nil {
		ch := make(chan any)
		taskc, err := NewTaskc(task, ctx, ch)
		if err != nil {
			return nil, err
		}
		pool.taskQueue <- *taskc
		return NewFuture(ch), nil
	} else {
		return nil, errors.New("async callable task queue is not enabled")
	}
}
