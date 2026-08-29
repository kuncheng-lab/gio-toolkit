// Package workerkit 提供单属主请求循环：一个 channel、一个 goroutine、串行处理。
//
// 这是各应用 core 中 store worker、任务 worker 的公共骨架：
//
//	worker := workerkit.Start(c.ctx, 64, func(req StoreRequest) {
//	    // 执行 IO，把结果以事件发回状态循环，不直接修改 UI 状态
//	    core.Post(resultEvent{...})
//	})
//	worker.Post(loadRequest{page: 1})
//
// 同一时刻只有一个 goroutine 调用 handle，worker 内部可以无锁访问自己的
// 数据源（数据库连接、设备客户端等）。
package workerkit

import (
	"context"
	"sync"
)

// Worker 是单属主请求循环。
type Worker[T any] struct {
	requests chan T
	ctx      context.Context
	done     chan struct{}
	runOnce  sync.Once
}

// Start 启动 worker goroutine，串行调用 handle 处理每个请求。
// ctx 取消后 worker 停止接收并退出，已排队但未处理的请求被丢弃。
// handle 不应阻塞在无关的长期操作上；长期等待请自行 select ctx.Done。
func Start[T any](ctx context.Context, buffer int, handle func(T)) *Worker[T] {
	if buffer <= 0 {
		buffer = 1
	}
	w := &Worker[T]{
		requests: make(chan T, buffer),
		ctx:      ctx,
		done:     make(chan struct{}),
	}
	w.runOnce.Do(func() {
		go w.run(handle)
	})
	return w
}

func (w *Worker[T]) run(handle func(T)) {
	defer close(w.done)
	for {
		select {
		case <-w.ctx.Done():
			return
		case req := <-w.requests:
			handle(req)
		}
	}
}

// Post 投递请求；缓冲已满时阻塞，直到入队、ctx 取消或 worker 退出。
// 返回 false 表示请求未被接收。
func (w *Worker[T]) Post(req T) bool {
	select {
	case w.requests <- req:
		return true
	case <-w.ctx.Done():
		return false
	}
}

// Done 返回 worker 退出完成的信号 channel。
func (w *Worker[T]) Done() <-chan struct{} {
	return w.done
}
