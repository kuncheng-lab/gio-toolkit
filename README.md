# gio-toolkit

Gio 桌面应用**框架层**工具库，沉淀自 `apps/rfid` 等应用验证过的
"事件驱动 + 单写者状态循环"与"布局/刷新分层"模式。

组件（按钮、表格、对话框等）请使用 [`gio-components`](https://github.com/kuncheng-lab/gio-components)；
本库只负责组件之上的应用骨架：状态循环、窗口事件循环、布局算术与刷新策略。

要求 Go 1.27+（使用了泛型方法）。

## 包

| 包 | 依赖 | 内容 |
| --- | --- | --- |
| `statekit` | 无 gio 依赖 | `Core[S]` 泛型状态循环、`Handler[S]`、`Send[T]` |
| `runner` | gioui.org | `Run` 窗口事件循环、`WindowAttention` |
| `logkit` | gio-components | `Buffer` 状态内日志缓冲、`View` 日志视图（Revision 门闩） |
| `configkit` | 无 gio 依赖 | YAML 配置：defaults 基线 + 文件覆盖 + 首次运行落盘 |
| `workerkit` | 无 gio 依赖 | `Worker[T]` 单属主请求循环（store/任务 worker 公共骨架） |
| `layoutkit` | gioui.org | 布局算术：`Row/Col`、`Width/Height/Size/Grow/Fit/Fill`、`Overlay`、`Measure`、设计稿缩放、`Px/Fr` 列宽 |
| `refreshkit` | gioui.org | 刷新策略：`Policy`/`Cadence`/`Delay`/`Follow`（帧内预约 `InvalidateCmd`） |
| `apptool` | 无 | `DefaultConfigPath` 等小工具 |
| `testkit` | 无 | `WaitFor` 轮询断言 |

## statekit：单写者状态循环

```text
事件 → inbox → 单一状态循环 → applyEvent → clone → atomic 快照 → notifier → w.Invalidate
```

```go
type State struct {
	Count    int
	LogLines []string
}

type Handler struct{}

func (Handler) ApplyEvent(s *State, event any) bool {
	switch e := event.(type) {
	case int:
		s.Count += e
		s.LogLines = append(s.LogLines, strconv.Itoa(s.Count))
		return true // 状态变化，发布快照并通知窗口
	}
	return false // 未变化，不刷新
}

func (Handler) CloneState(s State) State {
	clone := s
	clone.LogLines = append([]string(nil), s.LogLines...) // slice 深拷贝
	return clone
}

type Core struct {
	*statekit.Core[State] // 嵌入状态循环
	// 业务字段：设备客户端、数据库、配置……
}

func NewCore() *Core {
	c := &Core{Core: statekit.New(Handler{}, State{})}
	c.Start()
	return c
}
```

应用入口绑定窗口刷新：

```go
core := NewCore()
defer core.Close()

runner.Run(runner.Options{
	Title: "My App", Width: 960, Height: 680, Decorated: false,
}, func(w *app.Window, attn *runner.WindowAttention) runner.UI {
	core.SetNotifier(w.Invalidate)
	return appui.NewUI(core, func(a system.Action) { attn.Perform(w, a) })
})
```

## runner：窗口事件循环

`runner.Run` 内部处理 `ViewEvent` / `ConfigEvent` / `FrameEvent` / `DestroyEvent`
和 `WindowAttention`（Windows 下 win32 置前/最小化/最大化/还原）。
应用 UI 只需满足：

```go
type UI interface {
	Layout(gtx layout.Context) layout.Dimensions
	SetMaximized(maximized bool)
}
```

## apptool 与 testkit

```go
cfgPath := apptool.DefaultConfigPath("myapp") // 优先 cwd/config.yaml

testkit.WaitFor(t, 3*time.Second, func() bool {
	return core.Snapshot().Ready
})
```

## logkit 与 configkit

日志：`logkit.Buffer` 作为 UIState 字段在 `applyEvent` 中追加，`logkit.View` 在页面上渲染
（内部用 Revision 门闩，只在变化时重设编辑器）：

```go
// UIState 中：
type State struct {
	Logs logkit.Buffer
}
// applyEvent 中：
s.Logs.Addf("处理完成 %d", id)
// CloneState 中：
clone.Logs = s.Logs.Clone()
// 页面中：
logView.Layout(gtx, state.Logs)
```

配置：代码内 defaults 为基线，YAML 只覆盖出现的字段，文件缺失自动落盘默认值：

```go
cfg := DefaultConfig()
if err := configkit.Load(apptool.DefaultConfigPath("myapp"), &cfg); err != nil {
	log.Fatal(err)
}
```

## workerkit：单属主请求循环

store worker、设备任务 worker 的公共骨架——一个 channel、一个 goroutine、串行处理，
结果以事件发回状态循环，不在 worker 内改 UI 状态：

```go
worker := workerkit.Start(c.ctx, 64, func(req StoreRequest) {
    rows := queryDB(req)          // 无锁访问自己的数据源
    core.Post(recordsLoaded(rows)) // 结果走事件回流
})
worker.Post(loadRequest{page: 1}) // ctx 取消时放弃发送，不会永久阻塞
```

## layoutkit：布局算术

Gio 的 `Flex` 用子项**返回的尺寸**给下一个兄弟定位，`Constraints.Min` 又是对齐基准，
`Label` 的返回尺寸还会被当前约束夹住——每个页面都要为这三件事对账。
`layoutkit` 把账收在一处，页面只描述“这一格占多大”：

```go
import lay "github.com/kuncheng-lab/gio-toolkit/layoutkit"

lay.Col(gtx, 12,
	lay.Height(96, lay.FillX(p.header)), // 定高、横向铺满
	lay.Grow(1, lay.FillBoth(p.body)),   // 吃剩余、两轴填满
	lay.Fit(p.footer),                   // 内容驱动
)
```

- `Row/Col` + `Width/Height/Size/Grow/Fit/Fill/Spacer` 描述主轴与交叉轴；
- `Overlay` 叠放（替代“靠左右 padding 配平让标题居中”）；
- `Measure` 量文字（返回设计 dp）、`Scale/Space` 支持 1920×1080 设计稿；
- `Px/Fr/Widths/RowTracks` 让表头与数据行共用一组列宽。

不持状态、不包装控件，`Raw` 可随时回原生 `layout.FlexChild`。
完整说明（动机、API、边界、测试）见 [`layoutkit/README.md`](layoutkit/README.md)。

## refreshkit：下一帧由谁决定

```go
refreshkit.Follow(gtx, refreshkit.Cadence{}.Delay(refreshkit.Policy{
	Waiting: state.Exporting || state.Processing,
	Stale:   latest.Revision != state.Revision,
}))
```

三级节奏：`Stale`/`Animating` → 16ms，`Waiting` → 100ms，否则不排。
配合 `UIState.Revision`（每发布一版自增）解决“结果落在画帧的缝里、notifier 被丢”的问题。
实测 100ms 轮询稳定在 10 帧/秒、空闲 0 帧。完整说明见
[`refreshkit/README.md`](refreshkit/README.md)。

## 完整示范

`apps/example` 是本库的示范应用：
`core/core.go` 演示 `statekit.Core` 的完整用法，`ui/ui_state.go` 演示快照渲染
与 `LogRevision` 局部刷新门闩，`main.go` 演示 `runner.Run` 接入。

## 本地开发

主项目 go.mod 用 replace 指向平级目录，改本库立即生效（已启用）：

```go
require github.com/kuncheng-lab/gio-toolkit v0.0.0

replace github.com/kuncheng-lab/gio-toolkit => ../gio-toolkit
```

> 本库 `go 1.27`（用到泛型方法），接入方的主模块 `go` 指令需要不低于 1.27。

拉取本库默认走模块代理（goproxy.cn 已验证可用）。若遇到新版本
刚发布后代理回源短暂 500，可等待代理预热，或一次性将团队域切为
直连 GitHub：

```sh
go env -w GOPRIVATE=github.com/kuncheng-lab/*
```
