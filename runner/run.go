// Package runner 提供 Gio 窗口事件循环的统一实现：
//
//	runner.Run(options, setup)
//
// setup 在窗口 goroutine 内调用，应用在其中绑定 notifier（w.Invalidate）、
// 注意请求回调并构造 UI；之后 ViewEvent/ConfigEvent/FrameEvent/DestroyEvent
// 的处理由本包负责，main.go 不再重复窗口循环样板。
package runner

import (
	"os"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// UI 是窗口事件循环对应用界面的最小要求。
type UI interface {
	Layout(gtx layout.Context) layout.Dimensions
	SetMaximized(maximized bool)
}

// Options 描述窗口初始属性。
type Options struct {
	Title     string
	Width     int
	Height    int
	Decorated bool
}

// Run 创建窗口、调用 setup 并驱动事件循环，直到窗口销毁。
// 内部启动 app.Main，因此会阻塞当前 goroutine。
func Run(opts Options, setup func(w *app.Window, attn *WindowAttention) UI) {
	go func() {
		w := new(app.Window)
		attn := new(WindowAttention)
		w.Option(app.Title(opts.Title), app.Size(unit.Dp(opts.Width), unit.Dp(opts.Height)),
			app.Decorated(opts.Decorated))

		var ui UI
		if setup != nil {
			ui = setup(w, attn)
		}
		loop(w, attn, ui)
	}()
	app.Main()
}

func loop(w *app.Window, attn *WindowAttention, ui UI) {
	var ops op.Ops
	focused := false
	for {
		switch e := w.Event().(type) {
		case app.ViewEvent:
			attn.RememberNativeWindow(e)
		case app.ConfigEvent:
			if ui != nil {
				ui.SetMaximized(e.Config.Mode == app.Maximized)
			}
			if e.Config.Focused && !focused {
				attn.Refresh(w)
			}
			focused = e.Config.Focused
			w.Invalidate()
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			if ui != nil {
				ui.Layout(gtx)
			}
			e.Frame(gtx.Ops)
		case app.DestroyEvent:
			os.Exit(0)
		}
	}
}
