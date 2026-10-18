// Package refreshkit 把“下一帧什么时候来”收成一处状态判断。
//
// Gio 只在“有事件”或“有人预约”时画帧。把预约散写在各个页面里，等于每个页面
// 各自猜一个延时；漏掉一条分支，界面就停在旧值上。这里只做两件事：
// 把 Policy 翻译成延时，以及把这个延时变成帧内预约（op.InvalidateCmd）。
//
// 用法（在根 layout 末尾调用一次）：
//
//	d := refreshkit.Cadence{}.Delay(refreshkit.Policy{
//		Waiting: state.Waiting,
//		Stale:   latest.Revision != drawn.Revision,
//	})
//	refreshkit.Follow(gtx, d)
package refreshkit

import (
	"time"

	"gioui.org/layout"
	"gioui.org/op"
)

// Cadence 是两类续帧的间隔。
type Cadence struct {
	// Animate 是动画 / 补帧的间隔；零值用 16ms（≈60fps）。
	Animate time.Duration
	// Wait 是“有人在等后端结果”时的轮询间隔；零值用 100ms。
	Wait time.Duration
}

// Policy 描述这一帧之后还要不要再画。
type Policy struct {
	// Animating：动画正在跑，按帧率续。
	Animating bool
	// Waiting：有人在等后端结果（上传 / 落库 / 登录 / 导出…），低频续。
	Waiting bool
	// Stale：画的过程中快照又变了（Revision 不一致），尽快补一帧。
	Stale bool
}

// Delay 返回下一帧的延时；0 表示不排。
//
// 优先级：Stale > Animating > Waiting。Stale 意味着“屏幕上这一帧已经旧了”，
// 所以按动画间隔补，而不是等 100ms。
func (c Cadence) Delay(p Policy) time.Duration {
	switch {
	case p.Stale, p.Animating:
		return c.animate()
	case p.Waiting:
		return c.wait()
	default:
		return 0
	}
}

// Follow 在需要时预约下一帧，返回是否排了。
//
// 延时从 gtx.Now 起算，不用 time.Now：离屏渲染和测试里两者不同，
// 用帧时间才能确定性地验证与驱动。
func Follow(gtx layout.Context, d time.Duration) bool {
	if d <= 0 {
		return false
	}
	gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(d)})
	return true
}

func (c Cadence) animate() time.Duration {
	if c.Animate > 0 {
		return c.Animate
	}
	return 16 * time.Millisecond
}

func (c Cadence) wait() time.Duration {
	if c.Wait > 0 {
		return c.Wait
	}
	return 100 * time.Millisecond
}
