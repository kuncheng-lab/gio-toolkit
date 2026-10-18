package layoutkit

import (
	"image"

	"gioui.org/layout"
)

// Scale 把设计稿等比映射到当前可用空间：页面里写 1920×1080 的数字，窗口多大都成立。
//
// 只在根 layout 调一次。之后本包所有 int 参数、以及 gtx.Dp/unit.Dp 都按设计稿算。
// 取两轴的较小比例，所以窗口比例与设计稿不同时会留边，而不是裁切或撑破。
func Scale(gtx layout.Context, designW, designH int) layout.Context {
	avail := gtx.Constraints.Max
	if avail.X <= 0 || avail.Y <= 0 || designW <= 0 || designH <= 0 {
		return gtx
	}
	s := float32(avail.X) / float32(designW)
	if sy := float32(avail.Y) / float32(designH); sy < s {
		s = sy
	}
	if s <= 0 {
		return gtx
	}
	gtx.Metric.PxPerDp = s
	gtx.Metric.PxPerSp = s
	return gtx
}

// Space 把当前可用空间折算回设计稿尺寸。
//
// 凡是要把 gtx.Constraints.Max 当尺寸用的地方，都先过这一步；
// 直接拿 Max 当设计稿用，只在窗口正好等于设计稿时才不出错。
func Space(gtx layout.Context) image.Point {
	s := gtx.Metric.PxPerDp
	if s <= 0 {
		s = 1
	}
	return image.Pt(
		int(float32(gtx.Constraints.Max.X)/s),
		int(float32(gtx.Constraints.Max.Y)/s),
	)
}
