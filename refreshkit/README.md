# refreshkit — 下一帧由谁决定

Gio 只在**有事件**或**有人预约**时画帧。把“再排一帧”散写在各个页面里，
等于每个页面各自猜一个延时；漏掉一条分支，界面就停在旧值上。
本包把这件事收成一处状态判断：一个 `Policy` → 一个延时 → 一次帧内预约。

## 原理

`w.Invalidate()` 是**一次性放行**：成功一次就把门关上，直到事件队列再次被取空；
它不排队、不重试。异步结果（网络回执、落库完成）常落在“本帧 layout 跑完、还在提交”
的缝里，那一下可能被丢掉——表现是“点了要卡一会儿才出来”。

`gtx.Execute(op.InvalidateCmd{At: t})` 走另一条路：router 逐帧收单（同一帧取最早的 `At`），
不受那扇门影响；帧末由窗口取走并挂定时器，到点补一帧。

所以：**“什么时候刷”应该是状态问题，不是时间点问题。**

## API

```go
type Cadence struct{ Animate, Wait time.Duration } // 零值 = 16ms / 100ms

type Policy struct {
	Animating bool // 动画在跑
	Waiting   bool // 有人在等后端结果
	Stale     bool // 画完发现快照又变了
}

func (c Cadence) Delay(p Policy) time.Duration
func Follow(gtx layout.Context, d time.Duration) bool
```

| Policy | Delay | 场景 |
| --- | --- | --- |
| `Stale` | `Animate`（16ms） | 帧尾发现快照已换版：尽快补 |
| `Animating` | `Animate`（16ms） | 进度条、过渡 |
| `Waiting` | `Wait`（100ms） | 上传/落库/登录/导出：有人盯着结果 |
| 都没有 | `0` | 不排，交给下一个事件 |

优先级 `Stale > Animating > Waiting`：刚画完就发现数据又变了，不该再等 100ms。

## 用法：根布局末尾调用一次

```go
func (ui *UI) Layout(gtx layout.Context) layout.Dimensions {
	state := ui.core.Snapshot()

	dims := ui.draw(gtx, state) // 照常画

	latest := ui.core.Snapshot() // 再读一版：画完才知道有没有换版
	refreshkit.Follow(gtx, refreshkit.Cadence{}.Delay(refreshkit.Policy{
		Waiting: state.Exporting || state.Processing,
		Stale:   latest.Revision != state.Revision,
	}))
	return dims
}
```

`Stale` 需要一个**每发布一版就递增**的版本号（`UIState.Revision`）：

```go
if c.applyEvent(&uiState, event) {
	uiState.Revision++ // 发布边界
	c.snapshot.Store(cloneUIState(uiState))
}
```

> 帧首读到的 `state` 就是这一帧画的内容；帧尾再读一次，不相等就说明结果落在
> “layout 之后、提交之前”的缝里——这时 notifier 的一次性 Invalidate 可能已经丢了，
> 由这里补一帧。

## 实测

真实窗口，5 秒，分阶段统计（两次运行结果一致）：

| 阶段 | 帧数 | 帧间隔 min/avg/max |
| --- | --- | --- |
| 空闲（只排边界帧） | 启动后 0 帧 | — |
| `Waiting`：每帧 `At: now+100ms` | 10 / 秒 | 100 / 100 / 101 ms |
| `Animating`：每帧 `At: now+16ms` | 31 / 0.5 秒 | 16 / 16 / 17 ms |
| `At` 在过去 | 立即出帧 | 6 / 7 / 9 ms |

另有两个结论：

- **不空转**：`Delay` 返回 0 时确实不画（空闲段 0 帧）；
- **不是每次状态变化都对应一帧**：两次变化可能共用一帧（画面仍是最新快照），
  所以“10 次变更 9 帧”是合并，不是丢更新。

## 边界

- `Wait` 是轮询间隔，也就是**最坏延迟**；要更跟手就调小，代价是成比例的帧数；
- `Stale` 在帧尾读快照，**最后一次读之后、提交之前**仍有极窄的窗口；
  彻底的“零竞态”需要发布/提交两侧加锁握手，代价不值得；
- 不要把 `w.Invalidate()` 当动画驱动：它每轮只放行一次，而且不排队；
- 窗口最小化/完全遮挡时平台可能不产生帧，这与本包无关。

## 测试

`refreshkit_test.go` 覆盖：6 种 `Policy` 组合、自定义 `Cadence`、
`Follow` 预约的时刻正好是 `gtx.Now + d`（用 `input.Router` 断言）、`d = 0` 不预约。
