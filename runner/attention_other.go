//go:build !windows

package runner

import (
	"gioui.org/app"
	"gioui.org/io/system"
)

// WindowAttention 处理无装饰窗口的最小化/最大化/还原与窗口置前。
// 非 Windows 平台直接使用 Gio 内建动作。
type WindowAttention struct{}

func (a *WindowAttention) RememberNativeWindow(e app.ViewEvent) {}

func (a *WindowAttention) Request(w *app.Window) {
	w.Perform(system.ActionRaise)
	w.Invalidate()
}

func (a *WindowAttention) Perform(w *app.Window, actions system.Action) {
	w.Perform(actions)
	w.Invalidate()
}

func (a *WindowAttention) Refresh(w *app.Window) {
	w.Invalidate()
}
