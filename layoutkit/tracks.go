package layoutkit

import (
	"gioui.org/layout"
)

// Track 是一列的宽度契约：固定宽度，或按权重分剩余宽度。
type Track struct {
	px     int
	weight float32
}

// Px 固定列宽（设计 dp）。
func Px(n int) Track {
	return Track{px: n}
}

// Fr 按权重分剩余宽度。
func Fr(weight float32) Track {
	return Track{weight: weight}
}

// Widths 把列宽算成设计 dp：先扣间隙与固定列，剩余按权重分，余数给最后一个按权重分的列。
//
// 固定列总宽超出预算时按比例压缩，保证“各列之和 + 间隙”不超过 total。
// 表头与数据行共用同一组 Track 和同一个 total，列就不会错位。
func Widths(total, gap int, tracks []Track) []int {
	out := make([]int, len(tracks))
	if len(tracks) == 0 {
		return out
	}
	budget := total - gap*(len(tracks)-1)
	if budget < 0 {
		budget = 0
	}
	var fixed int
	var weight float32
	for _, t := range tracks {
		if t.weight > 0 {
			weight += t.weight
			continue
		}
		fixed += maxInt(t.px, 0)
	}
	if fixed > budget {
		for i, t := range tracks {
			if t.weight > 0 {
				continue
			}
			out[i] = maxInt(t.px, 0) * budget / fixed
		}
		return out
	}

	remaining := budget - fixed
	lastFr := -1
	var used int
	for i, t := range tracks {
		if t.weight > 0 {
			lastFr = i
			w := int(float32(remaining) * t.weight / weight)
			out[i] = w
			used += w
			continue
		}
		out[i] = maxInt(t.px, 0)
	}
	if lastFr >= 0 {
		out[lastFr] += remaining - used
	}
	return out
}

// RowTracks 按列契约横排一行：表头、数据行、多列表单共用同一组 Track。
//
// 列宽按当前可用宽度算（先过 Space），所以窗口变宽变窄都不用改代码。
func RowTracks(gtx layout.Context, gap int, tracks []Track, cells []layout.Widget) layout.Dimensions {
	widths := Widths(Space(gtx).X, gap, tracks)
	slots := make([]Slot, 0, len(widths))
	for i, w := range widths {
		var content layout.Widget
		if i < len(cells) {
			content = cells[i]
		}
		if content == nil {
			content = func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{}
			}
		}
		slots = append(slots, Width(w, content))
	}
	return Row(gtx, gap, slots...)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
