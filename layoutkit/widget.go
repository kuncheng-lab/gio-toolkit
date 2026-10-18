package layoutkit

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
)

// FillBoth 让内容填满两轴，并按填满的尺寸上报。
//
// 它是 Slot 版 Fill 的“内容级”版本：已经用 Grow 拿到槽位、但槽位内部还要再铺满时用它。
// 只用在父级给出了有界范围的场合（页面根、定高面板、Flexed 槽位）：
// 约束的 Min 会被设成 Max，无界区域里等于把内容拉到无穷大。
func FillBoth(content layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min = gtx.Constraints.Max
		dims := content(gtx)
		dims.Size = gtx.Constraints.Max
		return dims
	}
}

// FillX 只在水平方向填满。
func FillX(content layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		dims := content(gtx)
		dims.Size.X = gtx.Constraints.Max.X
		return dims
	}
}

// FillY 只在竖直方向填满。
func FillY(content layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
		dims := content(gtx)
		dims.Size.Y = gtx.Constraints.Max.Y
		return dims
	}
}

// Center 在可用区域里居中：内容按自然尺寸，返回值是可用区域的尺寸。
//
// 注意它是“居中并占满”，不是“居中并收缩”：放在 Flex 里会按可用空间记账，
// 所以不要再套一层 Grow 去补尺寸。
func Center(content layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = image.Point{}
			return content(gtx)
		})
	}
}

// Measure 量一次内容的自然尺寸（换算成设计 dp），并丢掉这次录下的绘制。
//
// Gio 的 Label 会把返回尺寸夹进当前约束：在 Min==Max 的定宽区域里直接读返回值，
// 拿到的是容器尺寸而不是文字尺寸。需要“按内容定尺寸”时先过这一步。
//
// 返回值与本包其它尺寸参数同一套单位（设计 dp），可以直接喂给 Width/Height/Size；
// 在 Scale 过的上下文里不会出现“量到像素、又按设计稿再乘一次”的双重缩放。
//
// 它只该用于轻量内容（文字、图标）。录制本身不画，但内容会被执行一遍。
func Measure(gtx layout.Context, content layout.Widget) image.Point {
	m := gtx
	// 放开两轴，Min 清零：量到的才是内容本身。
	m.Constraints = layout.Constraints{Max: image.Pt(1<<15, 1<<15)}
	macro := op.Record(gtx.Ops)
	dims := content(m)
	macro.Stop() // 丢弃录制：只取尺寸，不画。
	s := gtx.Metric.PxPerDp
	if s <= 0 {
		s = 1
	}
	return image.Pt(
		int(float32(dims.Size.X)/s+0.5),
		int(float32(dims.Size.Y)/s+0.5),
	)
}
