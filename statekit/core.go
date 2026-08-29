// Package statekit 实现 Gio 桌面应用的单写者状态循环：
//
//	事件 → inbox → 单一状态循环 → applyEvent → clone → atomic 快照 → notifier
//
// UI 状态只由一个 goroutine 修改；对外发布的快照不可变，UI 通过 Snapshot 只读。
// notifier 通常绑定 Gio 的 w.Invalidate，由状态变化驱动窗口重绘。
package statekit

import (
	"context"
	"sync/atomic"
)

// Handler 定义状态类型 S 的事件转换规则。
type Handler[S any] interface {
	// ApplyEvent 将事件应用到可变工作状态上，返回状态是否发生了 UI 可见的变化。
	// 只有返回 true 时 Core 才会发布快照并调用 notifier。
	ApplyEvent(state *S, event any) bool

	// CloneState 深拷贝状态中所有可变字段（slice、map、嵌套指针），
	// 保证发布后的快照不会被后续事件修改。
	CloneState(state S) S
}

// HandlerFunc 用函数实现 Handler，适合状态机较小的应用。
type HandlerFunc[S any] struct {
	Apply func(state *S, event any) bool
	Clone func(state S) S
}

func (h HandlerFunc[S]) ApplyEvent(state *S, event any) bool { return h.Apply(state, event) }
func (h HandlerFunc[S]) CloneState(state S) S                { return h.Clone(state) }

type notifierRef struct{ fn func() }

// Core 是状态类型 S 的事件循环与快照发布器。
// 它不持有任何业务字段，应嵌入到应用自己的 Core 中：
//
//	type Core struct {
//	    *statekit.Core[UIState]
//	    // 业务字段：设备客户端、数据库、配置……
//	}
type Core[S any] struct {
	inbox chan any

	snapshot atomic.Pointer[S]
	notifier atomic.Pointer[notifierRef]

	handler Handler[S]

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// New 创建 Core 并发布初始快照。调用 Start 启动状态循环。
func New[S any](handler Handler[S], initial S) *Core[S] {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Core[S]{
		inbox:   make(chan any, 256),
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
	published := handler.CloneState(initial)
	c.snapshot.Store(&published)
	return c
}

// Start 启动状态循环 goroutine。每个 Core 只调用一次。
func (c *Core[S]) Start() {
	go c.loop()
}

func (c *Core[S]) loop() {
	defer close(c.done)
	// state 是循环私有的工作副本，永远不直接发布；
	// 每次发布都通过 CloneState 生成新快照。
	state := *c.snapshot.Load()
	for {
		select {
		case <-c.ctx.Done():
			return
		case event := <-c.inbox:
			if c.handler.ApplyEvent(&state, event) {
				published := c.handler.CloneState(state)
				c.snapshot.Store(&published)
				if n := c.notifier.Load(); n != nil && n.fn != nil {
					n.fn()
				}
			}
		}
	}
}

// Post 投递事件到状态循环。Core 关闭后调用会立即返回，事件被丢弃。
func (c *Core[S]) Post(event any) {
	select {
	case c.inbox <- event:
	case <-c.ctx.Done():
	}
}

// Snapshot 返回当前已发布的状态快照。快照不可变，调用方不得修改其中内容。
func (c *Core[S]) Snapshot() S {
	return *c.snapshot.Load()
}

// View 投影当前快照为调用方需要的值，适合页面只取部分字段的场景：
//
//	status := core.View(func(s UIState) string { return s.StatusText })
func (c *Core[S]) View[U any](project func(S) U) U {
	return project(*c.snapshot.Load())
}

// SetNotifier 设置状态发布后的回调，应用入口通常传 w.Invalidate。
// 传 nil 取消回调。
func (c *Core[S]) SetNotifier(notify func()) {
	if notify == nil {
		c.notifier.Store(nil)
	} else {
		c.notifier.Store(&notifierRef{fn: notify})
	}
}

// Close 停止状态循环。已排队的事件不再处理。
func (c *Core[S]) Close() {
	c.cancel()
	<-c.done
}

// Send 向 channel 发送值，ctx 取消时放弃发送。
// 适合 worker 向请求队列投递任务时避免永久阻塞。
func Send[T any](ctx context.Context, ch chan<- T, value T) bool {
	select {
	case ch <- value:
		return true
	case <-ctx.Done():
		return false
	}
}
