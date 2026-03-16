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
	"fmt"
	"log"

	"github.com/photowey/poolgo"
)

func ExampleNewGoroutineExecutorPool() {
	pool, err := poolgo.NewGoroutineExecutorPool(2, poolgo.WithQueueSize(4), poolgo.WithLogger(poolgo.NewDiscardLogger()))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Shutdown(context.Background())

	future, err := poolgo.SubmitTyped[string](pool, func(ctx context.Context) (string, error) {
		return "ok", nil
	}, context.Background())
	if err != nil {
		log.Fatal(err)
	}

	result, err := future.Await(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
	// Output:
	// ok
}

func ExampleWithPanicHandler() {
	pool, err := poolgo.NewGoroutineExecutorPool(
		1,
		poolgo.WithLogger(poolgo.NewDiscardLogger()),
		poolgo.WithPanicHandler(func(info poolgo.PanicInfo) error {
			return errors.New("task failed")
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Shutdown(context.Background())

	future, err := pool.Submit(func(ctx context.Context) any {
		panic("boom")
	}, context.Background())
	if err != nil {
		log.Fatal(err)
	}

	_, err = future.Await(context.Background())
	fmt.Println(err)
	// Output:
	// task failed
}
