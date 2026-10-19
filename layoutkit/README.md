# layoutkit — Gio 的布局算术层

`layoutkit` 只做两件事：**把尺寸契约翻译成约束**，**把子项返回的尺寸改写成父级分配的尺寸**。
它不持有状态、不包装控件、不隐藏 Gio——需要原生能力时用 `Raw` 直接放一个 `layout.FlexChild`，两者可以混排。

组件（按钮、表格、对话框）请用 [`gio-components`](https://github.com/kuncheng-lab/gio-components)；
本包只负责组件之间的**排布**。

## 为什么需要这一层

Gio 的 `layout.Flex` 有三个反直觉的地方，每个页面都要为它们付一遍代价：

**1. Flex 拿子项“返回的尺寸”给下一个兄弟定位**

```go
layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
	layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return logo(gtx) // 报了 150×44，Flex 就按 150 往右排
	}),
	layout.Rigid(title),
)
```

子项报小了，后面的兄弟整体上浮、压到前一块上；报大了就撑破父级。所以“吃剩余空间”的槽位
必须把返回尺寸改写成分配到的尺寸——本包把它收在 `Slot.widget` 一处。

**2. `Constraints.Min` 同时是对齐基准**

骨架里得先 `gtx.Constraints.Min = gtx.Constraints.Max`，否则 `layout.Center` / `layout.E`
会静默退化成靠左/贴顶。`Row/Col/Fill/Center` 已经替你处理。

**3. `Label` 的返回尺寸会被当前约束夹住**

在 `Min == Max` 的定宽区域里直接读 Label 的返回值，拿到的是**容器**尺寸而不是文字尺寸。
要按内容定尺寸，用 `Measure`（它会放开约束量一次）。

## 快速开始

```go
import lay "github.com/kuncheng-lab/gio-toolkit/layoutkit"

func (p *Page) Layout(gtx layout.Context) layout.Dimensions {
	return lay.Col(gtx, 12,
		lay.Height(96, lay.FillX(p.header)), // 定高 96、横向铺满
		lay.Grow(1, lay.FillBoth(p.body)),   // 吃剩余高度、两轴填满
		lay.Fit(p.footer),                   // 按内容定高
	)
}
```

同一个页面的横向行：

```go
lay.Row(gtx, 8,
	lay.Width(76, label),      // 定宽 76
	lay.Grow(1, editor),       // 吃剩余
	lay.Fit(button),           // 内容驱动
)
```

## API

所有 `int` 参数都是**设计 dp**（`Scale` 之后就是设计稿像素），内部只换算一次。

### 槽位（`Slot`）：描述“这一格占多大”

| 构造器 | 主轴 | 交叉轴 |
| --- | --- | --- |
| `Fit(content)` | 内容 | 内容 |
| `Width(n, content)` | 定宽（Col 里是交叉轴） | — |
| `Height(n, content)` | 定高（Row 里是交叉轴） | — |
| `Size(w, h, content)` | 两轴固定 | 两轴固定 |
| `Grow(weight, content)` | 吃剩余 | 内容 |
| `Fill(content)` | 吃剩余 | 填满（先改约束再画） |
| `FillWeight(weight, content)` | 吃剩余 | 只补返回尺寸、不改约束 |
| `Spacer(weight)` | 吃剩余的空槽位 | — |
| `FixedSpace(n)` | 固定 n 的空槽位（Row 里占宽、Col 里占高） | — |
| `Raw(child)` | 原样交给 Gio | 原样交给 Gio |

- `0` 表示该轴不固定；
- **声明尺寸优先**：`Width/Height/Size` 写多少就是多少，不夹父级可用空间
  （与迁移前的手写 box 一致；`FixedSpace` 例外，它和原生 `layout.Spacer` 一样按约束夹取）；
- `Fill` 会把内容的交叉轴约束设成 `Min == Max`（内容按填满排版，适合表格/面板）；
  `FillWeight` 只把返回尺寸补齐（内容仍按自然尺寸排版，适合里面的居中、编辑器）——
  两者在“内容自己会居中”时差 1px，选错了会出现细微错位。

### 排布

| 函数 | 行为 |
| --- | --- |
| `Row(gtx, gap, slots...)` | 横向，交叉轴**居中** |
| `RowAt(gtx, cross, gap, ...)` | 横向，指定交叉轴对齐 |
| `Col(gtx, gap, slots...)` | 纵向，交叉轴**靠左** |
| `ColAt(gtx, cross, gap, ...)` | 纵向，指定交叉轴对齐 |

### 内容包装（`layout.Widget` 级）

| 函数 | 行为 |
| --- | --- |
| `FillBoth(content)` | 两轴填满并上报填满尺寸 |
| `FillX(content)` / `FillY(content)` | 单轴填满 |
| `Center(content)` | 在已分配区域里居中；内容保持自然尺寸 |

`Center` 是“居中并占满”：它返回的是可用区域的尺寸，所以直接当 `Grow` 的子项用，
不要再套一层补尺寸。

### 测量

```go
w := lay.Measure(gtx, chipLabel).X   // 返回设计 dp，可直接喂 Width
lay.Row(gtx, 8, lay.Width(w+16, chip), lay.Grow(1, editor))
```

`Measure` 会录制并丢弃这次绘制（不显示），只取尺寸；用于文字、图标这类轻量内容。

### 列宽：表头与数据行共用一组 `Track`

```go
tracks := []lay.Track{lay.Px(60), lay.Px(140), lay.Fr(1), lay.Px(170)}

lay.RowTracks(gtx, 8, tracks, []layout.Widget{idCell, seriesCell, msgCell, timeCell})
// 数据行用同一组 tracks，列就不会错位
```

- `Px(n)`：固定列宽；`Fr(w)`：按权重分剩余宽度；
- `Widths(total, gap, tracks)` 返回每列宽度（设计 dp），余数给最后一个 `Fr`；
- 固定列总宽超出预算时按比例压缩，保证“列宽之和 + 间隙”不超过总宽。

### 设计稿缩放

```go
func (ui *UI) Layout(gtx layout.Context) layout.Dimensions {
	gtx = lay.Scale(gtx, 1920, 1080) // 只在根上做一次
	area := lay.Space(gtx)           // 当前可用区域，换算回设计稿
	_ = area
	// 之后页面里所有数字都按 1920×1080 写
}
```

`Scale` 取两轴较小比例，窗口比例与设计稿不同时留边，而不是裁切或撑破。
`Space(gtx)` 是唯一把 `gtx.Constraints.Max` 当尺寸用的入口——直接把像素当设计稿用，
只在窗口正好等于设计稿时才不出错。

## 唯一的不变量

```text
Flex 给每个子项约束 ──► 子项按约束画内容 ──► 返回尺寸 ──► Flex 用它给下一个兄弟定位
                                              ▲
                                   这一层在这里改写：
                                   固定的轴按声明报，
                                   吃剩余的轴按分配报。
```

页面代码只描述“这一格占多大”，不需要记住“返回尺寸要改回去”。

## 边界（有意不做的）

- **没有 wrap / grid-auto / 二次测量**：语义上限就是 Gio 的 Flex，需要新排布就加原语；
- **不持状态、不缓存测量**：所有函数都是纯排布变换；
- **不包装控件**：`Clickable` / `Editor` / `List` 仍由调用方持有；
- **`Fill` 需要父级给出有界的交叉轴**（页面根、定高面板、`Flexed` 槽位），
  无界区域里它会按可用空间上报，把父级撑开；
- **`0` 表示“该轴不固定”**，所以表达不了“零高”；
- **随时可穿透**：任何一处想手写原生 `layout.Flex`，直接写，和 `Raw` 混排即可。

## 测试

`layoutkit_test.go` 覆盖：

- 子项少报尺寸时，行/列仍然按分配尺寸记账（Flex 兄弟上浮的根因）；
- `Grow` 只补主轴，不会把交叉轴撑到可用高度；
- 定宽/定高在 `Row`/`Col` 里的轴向解释，超过可用空间时被夹住；
- `Fill` 两轴填满、`Center` 保持内容尺寸、`Measure` 清 `Min` 且换算成设计 dp；
- `Overlay` 每层拿到整块区域；顶部栏形状（logo 定宽、标题居中、按钮槽位）的验收用例；
- 列宽：固定 + 权重、余数归属、固定列超预算时按比例压缩；
- `Raw` 与原生 `FlexChild` 混排。

## 迁移示例

`apps/rfid/ui/ui_work.go` 的 WorkPage 已从手写 `layout.Flex` 迁到本包。
迁移前后逐像素对比：除了表格里的时间戳（两次渲染的墙钟不同），**布局完全一致**。
