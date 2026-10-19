// Package layoutkit 是 Gio 之上的一层“布局算术”。
//
// 它只做两件事：把尺寸契约翻译成 Constraints，再把子项返回的尺寸改写成父级分配的尺寸。
// 它不持有状态、不包装控件、不隐藏 Gio —— 需要原生能力时用 Raw 直接放 layout.FlexChild，
// 两者可以混排。
//
// 所有尺寸参数都是设计 dp（int），内部只换算一次（gtx.Dp）。
// 要按 1920×1080 设计稿布局，在根上调用一次 Scale，之后页面里写的数字都按设计稿算。
//
// 为什么需要这一层：layout.Flex 是拿子项**返回**的尺寸给下一个兄弟定位的。
// 子项报小了，后面的兄弟就整体上浮、压到前一块上；报大了，就撑破父级。
// 每个页面都手写“先压 Constraints、再改返回尺寸”，既重复又容易漏。
// 这个包把记账收在 Slot.widget 一处，页面代码只描述“这一格占多大”。
package layoutkit

import (
	"gioui.org/layout"
	"gioui.org/unit"
)

// Slot 描述一个子项在主轴与交叉轴上的尺寸契约：占多大、要不要吃剩余空间。
//
// 它只是一次 Layout 里的一个值：没有状态、不跨帧存在、不持有控件。
// w/h 为 0 表示该轴不固定（按内容或按父级分配）。
type Slot struct {
	w, h    int
	weight  float32
	fillX   bool
	fillY   bool
	padOnly bool
	space   int
	raw     layout.FlexChild
	isRaw   bool
	content layout.Widget
}

// Fit 按内容定尺寸。
func Fit(content layout.Widget) Slot {
	return Slot{content: content}
}

// Width 固定宽度（设计 dp）；放在 Col 里表示“这一行多宽”。
func Width(n int, content layout.Widget) Slot {
	return Slot{w: n, content: content}
}

// Height 固定高度（设计 dp）；放在 Row 里表示“这一行多高”。
func Height(n int, content layout.Widget) Slot {
	return Slot{h: n, content: content}
}

// Size 两轴都固定（设计 dp）。
func Size(w, h int, content layout.Widget) Slot {
	return Slot{w: w, h: h, content: content}
}

// Grow 吃主轴剩余空间，交叉轴按内容。
func Grow(weight float32, content layout.Widget) Slot {
	return Slot{weight: weight, content: content}
}

// Fill 吃主轴剩余空间，并在交叉轴填满。
//
// 交叉轴必须由父级给出有界范围（页面根、定高面板、Flexed 槽位）；
// 在无界区域里它会按可用空间上报，把父级撑开。
func Fill(content layout.Widget) Slot {
	return Slot{weight: 1, fillX: true, fillY: true, content: content}
}

// FillWeight 吃主轴剩余空间，并把返回尺寸补到交叉轴可用高度；weight 用于按比例分配。
//
// 与 Fill 的差别：它**不改内容拿到的约束**，只在返回时补齐尺寸。内容自己按内容尺寸
// 居中/排版时用它（改约束会让居中、基线等出现 1px 偏移）。
func FillWeight(weight float32, content layout.Widget) Slot {
	return Slot{weight: weight, fillX: true, fillY: true, padOnly: true, content: content}
}

// Spacer 是一个吃剩余空间的空槽位：把后面的内容推到对侧（Row 里推右、Col 里推下）。
func Spacer(weight float32) Slot {
	return Slot{weight: weight, content: func(layout.Context) layout.Dimensions {
		return layout.Dimensions{}
	}}
}

// FixedSpace 是一个固定尺寸的空槽位（设计 dp）：Row 里占宽、Col 里占高。
//
// 交叉轴按父级 Min 上报，与原生 layout.Spacer 的行为一致。
func FixedSpace(n int) Slot {
	return Slot{space: n}
}

// Raw 逃生口：直接放一个原生 FlexChild，与其它 Slot 混排。
func Raw(child layout.FlexChild) Slot {
	return Slot{raw: child, isRaw: true}
}

// Row 横向排布，交叉轴居中 —— 顶部栏、表单行、按钮行都是这个语义。
func Row(gtx layout.Context, gap int, slots ...Slot) layout.Dimensions {
	return RowAt(gtx, layout.Middle, gap, slots...)
}

// RowAt 横向排布并指定交叉轴对齐。
func RowAt(gtx layout.Context, cross layout.Alignment, gap int, slots ...Slot) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: cross,
		Gap:       dp(gtx, gap),
	}.Layout(gtx, flexChildren(layout.Horizontal, slots)...)
}

// Col 纵向排布，交叉轴靠左 —— 左对齐是页面里最常见的列语义。
func Col(gtx layout.Context, gap int, slots ...Slot) layout.Dimensions {
	return ColAt(gtx, layout.Start, gap, slots...)
}

// ColAt 纵向排布并指定交叉轴对齐。
func ColAt(gtx layout.Context, cross layout.Alignment, gap int, slots ...Slot) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: cross,
		Gap:       dp(gtx, gap),
	}.Layout(gtx, flexChildren(layout.Vertical, slots)...)
}

func flexChildren(axis layout.Axis, slots []Slot) []layout.FlexChild {
	out := make([]layout.FlexChild, 0, len(slots))
	for _, s := range slots {
		if s.isRaw {
			out = append(out, s.raw)
			continue
		}
		if s.weight > 0 {
			out = append(out, layout.Flexed(s.weight, s.widget(axis)))
			continue
		}
		out = append(out, layout.Rigid(s.widget(axis)))
	}
	return out
}

// widget 把契约翻成两步：先按契约压 Constraints，再按契约改写返回尺寸。
//
// 这是本包唯一的不变量：子项报给 Flex 的尺寸，必须是父级分配给它的尺寸。
// 少了后一步，Flex 就会按内容量出来的尺寸给下一个兄弟定位，于是“兄弟上浮”。
func (s Slot) widget(axis layout.Axis) layout.Widget {
	horiz := axis == layout.Horizontal
	return func(gtx layout.Context) layout.Dimensions {
		if s.space > 0 {
			// 固定空槽位：与原生 layout.Spacer 完全一致（主轴 n，交叉轴按约束夹取）。
			if horiz {
				return layout.Spacer{Width: unit.Dp(float32(s.space))}.Layout(gtx)
			}
			return layout.Spacer{Height: unit.Dp(float32(s.space))}.Layout(gtx)
		}
		if s.content == nil {
			return layout.Dimensions{}
		}
		c := gtx.Constraints
		mainFix, crossFix := s.w, s.h
		if !horiz {
			mainFix, crossFix = s.h, s.w
		}

		// 主轴：声明了尺寸就把约束定死（声明值优先，即使超过父级可用空间）。
		if mainFix > 0 {
			setMain(&c, horiz, dp(gtx, mainFix))
		}
		// 交叉轴：声明了尺寸同样定死。
		if crossFix > 0 {
			setCross(&c, horiz, dp(gtx, crossFix))
		}
		// 交叉轴填满：主轴已被 Flexed 定成 Min==Max，这里只补交叉轴。
		// padOnly 的槽位只补返回尺寸，不改内容拿到的约束。
		if s.fillCross(horiz) && !s.padOnly {
			if horiz {
				c.Min.Y = c.Max.Y
			} else {
				c.Min.X = c.Max.X
			}
		}
		gtx.Constraints = c

		dims := s.content(gtx)
		// 记账：固定的轴按约束报，吃剩余的轴按分配报，交叉轴填满的按可用报。
		if mainFix > 0 || s.weight > 0 {
			if horiz {
				dims.Size.X = c.Max.X
			} else {
				dims.Size.Y = c.Max.Y
			}
		}
		if crossFix > 0 || s.fillCross(horiz) {
			if horiz {
				dims.Size.Y = c.Max.Y
			} else {
				dims.Size.X = c.Max.X
			}
		}
		return dims
	}
}

// fillCross 报告契约是否要求填满交叉轴：Row 的交叉轴是 Y，Col 的是 X。
func (s Slot) fillCross(horiz bool) bool {
	if horiz {
		return s.fillY
	}
	return s.fillX
}

func setMain(c *layout.Constraints, horiz bool, px int) {
	if horiz {
		c.Min.X, c.Max.X = px, px
		return
	}
	c.Min.Y, c.Max.Y = px, px
}

func setCross(c *layout.Constraints, horiz bool, px int) {
	if horiz {
		c.Min.Y, c.Max.Y = px, px
		return
	}
	c.Min.X, c.Max.X = px, px
}

// dp 设计 dp → 设备像素。全包只有这一处换算。
func dp(gtx layout.Context, n int) int {
	return gtx.Dp(unit.Dp(n))
}
