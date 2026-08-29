//go:build windows

package runner

import (
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
)

const (
	swMaximize = 3
	swMinimize = 6
	swRestore  = 9

	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpFrameChanged = 0x0020
	swpShowWindow   = 0x0040
)

var (
	user32DLL   = syscall.NewLazyDLL("user32.dll")
	kernel32DLL = syscall.NewLazyDLL("kernel32.dll")

	showWindowProc               = user32DLL.NewProc("ShowWindow")
	setForegroundWindowProc      = user32DLL.NewProc("SetForegroundWindow")
	setWindowPosProc             = user32DLL.NewProc("SetWindowPos")
	getForegroundWindowProc      = user32DLL.NewProc("GetForegroundWindow")
	getWindowThreadProcessIDProc = user32DLL.NewProc("GetWindowThreadProcessId")
	attachThreadInputProc        = user32DLL.NewProc("AttachThreadInput")
	bringWindowToTopProc         = user32DLL.NewProc("BringWindowToTop")
	setActiveWindowProc          = user32DLL.NewProc("SetActiveWindow")
	setFocusProc                 = user32DLL.NewProc("SetFocus")

	getCurrentThreadIDProc = kernel32DLL.NewProc("GetCurrentThreadId")
)

// WindowAttention 处理无装饰窗口的最小化/最大化/还原与窗口置前，
// 在 Windows 上通过 win32 API 保证置前成功。
type WindowAttention struct {
	hwnd       atomic.Uintptr
	refreshing atomic.Bool
}

func (a *WindowAttention) RememberNativeWindow(e app.ViewEvent) {
	ev, ok := e.(app.Win32ViewEvent)
	if !ok {
		return
	}
	if !ev.Valid() {
		a.hwnd.Store(0)
		return
	}
	a.hwnd.Store(ev.HWND)
}

func (a *WindowAttention) Request(w *app.Window) {
	w.Perform(system.ActionRaise)
	w.Invalidate()

	hwnd := a.hwnd.Load()
	if hwnd == 0 {
		return
	}

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		raiseWindow(w, hwnd)
	}()
}

// Perform 执行窗口标题栏动作（最小化/最大化/还原等），
// 优先走 win32 实现，失败时回退到 Gio 内建动作。
func (a *WindowAttention) Perform(w *app.Window, actions system.Action) {
	if actions&system.ActionMinimize != 0 {
		if a.showWindow(w, swMinimize) {
			actions &^= system.ActionMinimize
		}
	}
	if actions&system.ActionMaximize != 0 {
		if a.showWindow(w, swMaximize) {
			actions &^= system.ActionMaximize
		}
	}
	if actions&system.ActionUnmaximize != 0 {
		if a.showWindow(w, swRestore) {
			actions &^= system.ActionUnmaximize
		}
	}
	if actions != 0 {
		w.Perform(actions)
		w.Invalidate()
	}
}

func (a *WindowAttention) Refresh(w *app.Window) {
	hwnd := a.hwnd.Load()
	if hwnd == 0 {
		w.Invalidate()
		return
	}
	if !a.refreshing.CompareAndSwap(false, true) {
		w.Invalidate()
		return
	}
	w.Invalidate()
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer a.refreshing.Store(false)

		refreshWindow(hwnd)
		w.Invalidate()
		time.Sleep(50 * time.Millisecond)
		w.Invalidate()
	}()
}

func (a *WindowAttention) showWindow(w *app.Window, cmd uintptr) bool {
	hwnd := a.hwnd.Load()
	if hwnd == 0 {
		return false
	}
	w.Invalidate()
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		showWindowProc.Call(hwnd, cmd)
		if cmd != swMinimize {
			refreshWindow(hwnd)
		}
		w.Invalidate()
		time.Sleep(50 * time.Millisecond)
		w.Invalidate()
		time.Sleep(150 * time.Millisecond)
		w.Invalidate()
	}()
	return true
}

func refreshWindow(hwnd uintptr) {
	flags := uintptr(swpNoMove | swpNoSize | swpNoZOrder | swpFrameChanged | swpShowWindow)
	setWindowPosProc.Call(hwnd, 0, 0, 0, 0, 0, flags)
}

func raiseWindow(w *app.Window, hwnd uintptr) {
	w.Option(app.TopMost(true))
	showWindowProc.Call(hwnd, swRestore)
	forceForeground(hwnd)
	time.Sleep(120 * time.Millisecond)
	w.Option(app.TopMost(false))
	w.Invalidate()
}

func forceForeground(hwnd uintptr) {
	currentThread, _, _ := getCurrentThreadIDProc.Call()
	foreground, _, _ := getForegroundWindowProc.Call()
	foregroundThread := uintptr(0)
	if foreground != 0 {
		foregroundThread, _, _ = getWindowThreadProcessIDProc.Call(foreground, 0)
	}
	targetThread, _, _ := getWindowThreadProcessIDProc.Call(hwnd, 0)

	attachedForeground := foregroundThread != 0 && foregroundThread != currentThread
	attachedTarget := targetThread != 0 && targetThread != currentThread
	if attachedForeground {
		attachThreadInputProc.Call(currentThread, foregroundThread, 1)
	}
	if attachedTarget {
		attachThreadInputProc.Call(currentThread, targetThread, 1)
	}

	bringWindowToTopProc.Call(hwnd)
	setForegroundWindowProc.Call(hwnd)
	setActiveWindowProc.Call(hwnd)
	setFocusProc.Call(hwnd)

	if attachedTarget {
		attachThreadInputProc.Call(currentThread, targetThread, 0)
	}
	if attachedForeground {
		attachThreadInputProc.Call(currentThread, foregroundThread, 0)
	}
}
