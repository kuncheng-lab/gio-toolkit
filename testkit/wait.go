// Package testkit 提供各应用 core 测试共用的断言辅助。
package testkit

import (
	"testing"
	"time"
)

// WaitFor 在 timeout 内轮询 cond，条件满足即返回；
// 超时仍未满足则使当前测试失败。
// 适合断言异步事件循环最终收敛到期望状态：
//
//	testkit.WaitFor(t, 3*time.Second, func() bool {
//	    return core.Snapshot().Ready
//	})
func WaitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}
