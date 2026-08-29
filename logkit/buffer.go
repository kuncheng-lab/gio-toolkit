// Package logkit 提供应用内操作日志的通用实现：
// Buffer 是嵌入 UIState 的数据缓冲（在状态循环内使用），
// View 是只读日志视图（Revision 门闩，避免每帧重设编辑器）。
package logkit

import "fmt"

// Buffer 是固定容量的环形日志缓冲，应作为应用 UIState 的一个字段。
// 只允许状态循环（applyEvent）内调用 Add/Addf；发布快照时用 Clone 深拷贝。
type Buffer struct {
	// MaxLines 是保留的最大行数，超出后丢弃最旧的日志。
	MaxLines int

	Lines    []string
	Revision int
}

// NewBuffer 创建容量为 maxLines 的日志缓冲。
func NewBuffer(maxLines int) Buffer {
	return Buffer{MaxLines: maxLines}
}

// Add 追加一行日志并递增 Revision。
func (b *Buffer) Add(line string) {
	b.Lines = append(b.Lines, line)
	if b.MaxLines > 0 && len(b.Lines) > b.MaxLines {
		b.Lines = b.Lines[len(b.Lines)-b.MaxLines:]
	}
	b.Revision++
}

// Addf 追加一行格式化日志。
func (b *Buffer) Addf(format string, args ...any) {
	b.Add(fmt.Sprintf(format, args...))
}

// Clone 返回深拷贝，用于 Handler.CloneState：
//
//	func (Handler) CloneState(s State) State {
//	    clone := s
//	    clone.Logs = s.Logs.Clone()
//	    return clone
//	}
func (b Buffer) Clone() Buffer {
	clone := b
	clone.Lines = append([]string(nil), b.Lines...)
	return clone
}

// Tail 返回最后 maxLines 行（不足则全部）。
func (b Buffer) Tail(maxLines int) []string {
	if maxLines <= 0 || len(b.Lines) <= maxLines {
		return b.Lines
	}
	return b.Lines[len(b.Lines)-maxLines:]
}
