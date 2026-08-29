# gio-toolkit

Gio 桌面应用**框架层**工具库，沉淀自 `apps/rfid` 验证过的"事件驱动 + 单写者状态循环"模式。

组件（按钮、表格、对话框等）请使用 [`gio-components`](https://github.com/kuncheng-lab/gio-components)；
本库只负责组件之上的应用骨架：状态循环、窗口事件循环和通用辅助。

要求 Go 1.27+（使用了泛型方法）。

## 包

| 包 | 依赖 | 内容 |
| --- | --- | --- |
| `statekit` | 无 gio 依赖 | `Core[S]` 泛型状态循环、`Handler[S]`、`Send[T]` |
| `runner` | gioui.org | `Run` 窗口事件循环、`WindowAttention` |
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

## 完整示范

`apps/example` 是本库的示范应用：
`core/core.go` 演示 `statekit.Core` 的完整用法，`ui/ui_state.go` 演示快照渲染
与 `LogRevision` 局部刷新门闩，`main.go` 演示 `runner.Run` 接入。

## 本地开发

主项目 go.mod 使用 replace 指向平级目录，改本库立即生效：

```go
require github.com/kuncheng-lab/gio-toolkit v0.x.y

replace github.com/kuncheng-lab/gio-toolkit => ../gio-toolkit
```
