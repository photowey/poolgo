package poolgo_test

import (
	"context"
	"sync"
	"testing"

	"github.com/photowey/poolgo"
)

func BenchmarkExecute(b *testing.B) {
	pool, err := poolgo.NewGoroutineExecutorPool(8, poolgo.WithQueueSize(256), poolgo.WithLogger(poolgo.NewDiscardLogger()))
	if err != nil {
		b.Fatalf("new pool: %v", err)
	}
	defer pool.Shutdown(context.Background())

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(1)
		if err := pool.Execute(func(ctx context.Context) {
			wg.Done()
		}, context.Background()); err != nil {
			b.Fatalf("execute task: %v", err)
		}
		wg.Wait()
	}
}

func BenchmarkSubmitTyped(b *testing.B) {
	pool, err := poolgo.NewGoroutineExecutorPool(8, poolgo.WithQueueSize(256), poolgo.WithLogger(poolgo.NewDiscardLogger()))
	if err != nil {
		b.Fatalf("new pool: %v", err)
	}
	defer pool.Shutdown(context.Background())

	for i := 0; i < b.N; i++ {
		future, err := poolgo.SubmitTyped[int](pool, func(ctx context.Context) (int, error) {
			return 42, nil
		}, context.Background())
		if err != nil {
			b.Fatalf("submit typed task: %v", err)
		}
		if _, err := future.Await(context.Background()); err != nil {
			b.Fatalf("await typed task: %v", err)
		}
	}
}
