package poolgo_test

import (
	`context`
	`fmt`
	`reflect`
	`testing`
	`time`

	`github.com/photowey/poolgo`
)

func TestGoroutineExecutorPool_Execute(t *testing.T) {
	type fields struct {
		poolSize         int
		maxTaskQueueSize int
	}
	type args struct {
		task poolgo.Runnable
		ctx  context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test goroutine executor pool execute",
			fields: fields{
				poolSize:         10,
				maxTaskQueueSize: 32,
			},
			args: args{
				task: func(ctx context.Context) {
					fmt.Println("---- start ----")
					time.Sleep(3 * time.Second)
					fmt.Println("do something")
					fmt.Println("---- end ----")
				},
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := poolgo.NewGoroutineExecutorPool(tt.fields.poolSize, tt.fields.maxTaskQueueSize)
			if err := pool.Execute(tt.args.task, tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			time.Sleep(4 * time.Second)
		})
	}
}

func TestGoroutineExecutorPool_Submit(t *testing.T) {
	type fields struct {
		poolSize         int
		maxTaskQueueSize int
	}
	type args struct {
		task poolgo.Callable
		ctx  context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "Test goroutine executor pool submit",
			fields: fields{
				poolSize:         10,
				maxTaskQueueSize: 32,
			},
			args: args{
				task: func(ctx context.Context) any {
					return "ok"
				},
				ctx: context.Background(),
			},
			want:    "ok",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := poolgo.NewGoroutineExecutorPool(tt.fields.poolSize, tt.fields.maxTaskQueueSize)
			future, err := pool.Submit(tt.args.task, tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Submit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, err := future.Await()
			t.Logf("Acquire the async task result = %v", got)
			if err != nil {
				t.Errorf("Submit() Await result error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Submit() got = %v, want %v", got, tt.want)
			}
		})
	}
}
