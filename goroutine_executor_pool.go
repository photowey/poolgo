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
							task.resultCh <- result // how to judge whether the channel has been closed?
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
