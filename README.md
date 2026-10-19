# gio-toolkit

Gio 桌面应用的**两层基础库**，从真实页面里沉淀出来：

- **layoutkit**：布局算术。把“约束 → 内容 → 改写返回尺寸”的记账收在一处，页面只描述每一格占多大；
- **refreshkit**：刷新策略。把“下一帧什么时候来”从时间点变成状态判断。

组件（按钮、表格、对话框、图表）请用 [`gio-components`](https://github.com/kuncheng-lab/gio-components)；
本库只负责组件之间的排布与重绘节奏。

要求 Go 1.27+。

> 本库曾经包含 `statekit` / `runner` / `logkit` / `configkit` / `workerkit` 等应用骨架。
> 这些包没有被任何应用采用，接入需要重写 app 的 core 与 main，反而增加改造面，已移除；
> 需要时从 git 历史里取。

## 包

| 包 | 依赖 | 内容 |
| --- | --- | --- |
| `layoutkit` | gioui.org | 布局算术：`Row/Col`、`Width/Height/Size/Grow/Fit/Fill`、`Overlay`、`Measure`、设计稿缩放、`Px/Fr` 列宽 |
| `refreshkit` | gioui.org | 刷新策略：`Policy`/`Cadence`/`Delay`/`Follow`（帧内预约 `InvalidateCmd`） |

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

- `Row/Col` + `Width/Height/Size/Grow/Fit/Fill/FillWeight/Spacer/FixedSpace` 描述主轴与交叉轴；
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

`apps/rfid` 是两层都用上的示范：

- `ui/ui_work.go`：用 `layoutkit` 排 WorkPage（固定顶栏、吃剩余的表格、内容驱动的状态栏）；
- `ui/ui.go`：在根布局末尾用 `refreshkit` 决定下一帧；
- `core/state.go`：`UIState.Revision` 每次发布自增，是刷新握手的版本号。

## 本地开发

主项目 go.mod 用 replace 指向平级目录，改本库立即生效（已启用）：

```go
require github.com/kuncheng-lab/gio-toolkit v0.0.0

replace github.com/kuncheng-lab/gio-toolkit => ../gio-toolkit
```

拉取本库默认走模块代理（goproxy.cn 已验证可用）。若遇到新版本
刚发布后代理回源短暂 500，可等待代理预热，或一次性将团队域切为
直连 GitHub：

```sh
go env -w GOPRIVATE=github.com/kuncheng-lab/*
```
