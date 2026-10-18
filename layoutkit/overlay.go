package layoutkit

import (
	"gioui.org/layout"
)

// Overlay 在同一块区域上叠放多层，先给的在下、后给的在上。
//
// 每一层都拿到“Min=Max”的那块区域，所以层内可以正常用 Center/Fill 对齐；
// 这正好替掉“靠左右 padding 配平来让标题居中”的写法。
//
// 和 Fill 一样，它要求父级给出有界范围：叠加区按可用空间上报。
func Overlay(gtx layout.Context, layers ...layout.Widget) layout.Dimensions {
	// Stack 的最终尺寸会被 Min 兜住，先把它设成 Max，叠加区才占满整块区域。
	gtx.Constraints.Min = gtx.Constraints.Max
	children := make([]layout.StackChild, 0, len(layers))
	for _, layer := range layers {
		layer := layer
		children = append(children, layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = gtx.Constraints.Max
			return layer(gtx)
		}))
	}
	return layout.Stack{}.Layout(gtx, children...)
}
