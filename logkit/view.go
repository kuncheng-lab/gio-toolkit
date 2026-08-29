package logkit

import (
	"strings"

	"github.com/kuncheng-lab/gio-components"

	"gioui.org/layout"
	"gioui.org/widget"
)

const defaultVisibleLines = 200

// View 是只读日志视图：内部维护 Revision 门闩，
// 只在缓冲变化时重设编辑器内容，避免每帧 SetText 造成的抖动。
type View struct {
	// MaxVisible 限制视图中渲染的最大行数，0 表示 defaultVisibleLines。
	MaxVisible int

	Editor       widget.Editor
	scroll       components.ScrollArea
	lastRevision int
}

// Layout 渲染日志视图。buffer 通常来自应用状态快照：
//
//	logView.Layout(gtx, state.Logs)
func (v *View) Layout(gtx layout.Context, buffer Buffer) layout.Dimensions {
	if buffer.Revision != v.lastRevision {
		v.lastRevision = buffer.Revision
		maxVisible := v.MaxVisible
		if maxVisible <= 0 {
			maxVisible = defaultVisibleLines
		}
		v.Editor.ReadOnly = true
		v.Editor.SetText(strings.Join(buffer.Tail(maxVisible), "\n"))
	}
	return v.scroll.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return components.EditorBox(gtx, &v.Editor, "")
	}, components.ScrollAreaOptions{})
}
