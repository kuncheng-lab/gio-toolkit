package statekit

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"
)

type testState struct {
	Count int
	Logs  []string
}

type testHandler struct{}

func (testHandler) ApplyEvent(s *testState, event any) bool {
	switch e := event.(type) {
	case int:
		s.Count += e
		s.Logs = append(s.Logs, strconv.Itoa(s.Count))
		return true
	case noChangeEvent:
		return false
	}
	return false
}

func (testHandler) CloneState(s testState) testState {
	clone := s
	clone.Logs = append([]string(nil), s.Logs...)
	return clone
}

type noChangeEvent struct{}

func newTestCore() (*Core[testState], *countingNotifier) {
	notify := &countingNotifier{}
	core := New(testHandler{}, testState{})
	core.SetNotifier(notify.Notify)
	core.Start()
	return core, notify
}

type countingNotifier struct {
	mu    sync.Mutex
	calls int
}

func (n *countingNotifier) Notify() {
	n.mu.Lock()
	n.calls++
	n.mu.Unlock()
}

func (n *countingNotifier) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.calls
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func TestApplyPublishesSnapshotAndNotifies(t *testing.T) {
	core, notify := newTestCore()
	defer core.Close()

	core.Post(1)
	core.Post(2)

	waitUntil(t, func() bool { return core.Snapshot().Count == 3 })
	if got := notify.count(); got != 2 {
		t.Fatalf("notifier calls = %d, want 2", got)
	}
	if got := len(core.Snapshot().Logs); got != 2 {
		t.Fatalf("log entries = %d, want 2", got)
	}
}

func TestNoPublishWhenStateUnchanged(t *testing.T) {
	core, notify := newTestCore()
	defer core.Close()

	core.Post(noChangeEvent{})
	time.Sleep(100 * time.Millisecond)

	if got := notify.count(); got != 0 {
		t.Fatalf("notifier calls = %d, want 0 (未变化不应发布)", got)
	}
	if got := core.Snapshot().Count; got != 0 {
		t.Fatalf("count = %d, want 0", got)
	}
}

func TestSnapshotIsolation(t *testing.T) {
	core, _ := newTestCore()
	defer core.Close()

	core.Post(1)
	waitUntil(t, func() bool { return core.Snapshot().Count == 1 })
	first := core.Snapshot()
	if len(first.Logs) == 0 {
		t.Fatal("expected log entry in snapshot")
	}

	core.Post(2)
	core.Post(3)
	waitUntil(t, func() bool { return core.Snapshot().Count == 6 })

	if len(first.Logs) != 1 {
		t.Fatalf("published snapshot mutated: logs = %v", first.Logs)
	}
	if first.Count != 1 {
		t.Fatalf("published snapshot mutated: count = %d", first.Count)
	}
}

func TestPostAfterCloseDoesNotBlock(t *testing.T) {
	core, _ := newTestCore()
	core.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		core.Post(1)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Post blocked after Close")
	}
}

func TestConcurrentPosts(t *testing.T) {
	core, _ := newTestCore()
	defer core.Close()

	const goroutines, perGoroutine = 8, 50
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				core.Post(1)
			}
		}()
	}
	wg.Wait()

	waitUntil(t, func() bool { return core.Snapshot().Count == goroutines*perGoroutine })
}

func TestView(t *testing.T) {
	core, _ := newTestCore()
	defer core.Close()

	core.Post(7)
	waitUntil(t, func() bool { return core.Snapshot().Count == 7 })

	text := core.View(func(s testState) string { return "count=" + strconv.Itoa(s.Count) })
	if text != "count=7" {
		t.Fatalf("View result = %q, want %q", text, "count=7")
	}
}

func TestSetNotifierNil(t *testing.T) {
	core, _ := newTestCore()
	defer core.Close()

	core.SetNotifier(nil)
	core.Post(1)
	waitUntil(t, func() bool { return core.Snapshot().Count == 1 })
}

func TestSend(t *testing.T) {
	ch := make(chan int, 1)
	if !Send(context.Background(), ch, 1) {
		t.Fatal("Send to buffered channel should succeed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	full := make(chan int)
	if Send(ctx, full, 1) {
		t.Fatal("Send with cancelled ctx should return false")
	}
}
