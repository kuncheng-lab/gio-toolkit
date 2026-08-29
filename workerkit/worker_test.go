package workerkit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestStartProcessesRequestsSerially(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var processed []int
	done := make(chan struct{})

	worker := Start(ctx, 16, func(req int) {
		mu.Lock()
		processed = append(processed, req)
		mu.Unlock()
		if len(processed) == 3 {
			close(done)
		}
	})

	worker.Post(1)
	worker.Post(2)
	worker.Post(3)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not process requests in time")
	}

	mu.Lock()
	defer mu.Unlock()
	for i, want := range []int{1, 2, 3} {
		if processed[i] != want {
			t.Fatalf("processed[%d] = %d, want %d (顺序应保持 FIFO)", i, processed[i], want)
		}
	}
}

func TestPostAfterCancelReturnsFalse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	release := make(chan struct{})
	started := make(chan struct{})
	worker := Start(ctx, 1, func(req int) {
		if req == 1 {
			close(started)
			<-release
		}
	})

	// worker 正在处理 req=1 且缓冲被 req=2 占满时，Post(3) 只能等待。
	if !worker.Post(1) || !worker.Post(2) {
		t.Fatal("Post(1)/Post(2) should succeed")
	}
	<-started

	cancel()
	if worker.Post(3) {
		t.Fatal("Post after cancel should return false")
	}
	close(release)

	select {
	case <-worker.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not exit after cancel")
	}
}

func TestPostBlocksUntilBufferAvailable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	release := make(chan struct{})
	started := make(chan struct{})
	worker := Start(ctx, 1, func(req int) {
		if req == 1 {
			close(started)
			<-release
		}
	})

	if !worker.Post(1) {
		t.Fatal("Post(1) should succeed")
	}
	<-started

	if !worker.Post(2) {
		t.Fatal("Post(2) should succeed into the free buffer slot")
	}

	// handle 忙、缓冲满：Post(3) 必须阻塞。
	posted := make(chan bool, 1)
	go func() { posted <- worker.Post(3) }()
	select {
	case <-posted:
		t.Fatal("Post(3) should block while handle is busy and buffer is full")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	select {
	case ok := <-posted:
		if !ok {
			t.Fatal("Post(3) should succeed after space frees up")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Post(3) did not complete in time")
	}
}

func TestConcurrentPosts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	count := 0
	done := make(chan struct{})
	const goroutines, perGoroutine = 8, 25

	worker := Start(ctx, 64, func(req int) {
		mu.Lock()
		count++
		if count == goroutines*perGoroutine {
			close(done)
		}
		mu.Unlock()
	})

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				worker.Post(n*perGoroutine + j)
			}
		}(i)
	}
	wg.Wait()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("count = %d, want %d", count, goroutines*perGoroutine)
	}
}

func TestHandleSeesRequestsInOrderUnderLoad(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var last int
	ordered := true
	done := make(chan struct{})
	const total = 200

	worker := Start(ctx, total, func(req int) {
		mu.Lock()
		if req <= last {
			ordered = false
		}
		last = req
		if req == total {
			close(done)
		}
		mu.Unlock()
	})

	for i := 1; i <= total; i++ {
		worker.Post(i)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not finish in time")
	}

	mu.Lock()
	defer mu.Unlock()
	if !ordered {
		t.Fatal("handle observed out-of-order requests")
	}
}
