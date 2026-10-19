package layoutkit

import (
	"image"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// testCtx 是“父级只给上限”的约束：Min=0，Max=窗口。
// 页面内部的行列大多拿到这种约束，交叉轴才按内容算。
func testCtx(w, h int) layout.Context {
	return layout.Context{
		Constraints: layout.Constraints{Max: image.Pt(w, h)},
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Now:         time.Now(),
		Ops:         new(op.Ops),
	}
}

// exactCtx 是“父级已经分好一块地”的约束：Min=Max。
// 根窗口、定高面板、Flexed 槽位给子项的就是这个。
func exactCtx(w, h int) layout.Context {
	ctx := testCtx(w, h)
	ctx.Constraints.Min = ctx.Constraints.Max
	return ctx
}

// probe 模拟“不守规矩”的内容：按自己的尺寸上报，不看父级给了多少。
//
// 真实代码里手绘面板、自定义控件就是这样报尺寸的，这也正是 Flex 里
// “兄弟上浮/撑破父级”的来源：Flex 用子项返回的尺寸给下一个兄弟定位。
type probe struct {
	report image.Point
	gotMin image.Point
	gotMax image.Point
}

func (p *probe) widget() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		p.gotMin = gtx.Constraints.Min
		p.gotMax = gtx.Constraints.Max
		return layout.Dimensions{Size: p.report}
	}
}

func TestRowPadsUnderreportedChild(t *testing.T) {
	ctx := testCtx(300, 50)
	a := &probe{report: image.Pt(40, 20)}
	b := &probe{report: image.Pt(0, 0)}

	dims := Row(ctx, 10, Fit(a.widget()), Grow(1, b.widget()))

	if dims.Size.X != 300 {
		t.Fatalf("Row 宽度 = %d，想要 300：子项报小了没被补齐", dims.Size.X)
	}
	if b.gotMax.X != 250 {
		t.Fatalf("Grow 拿到的主轴空间 = %d，想要 250", b.gotMax.X)
	}
	if dims.Size.Y != 20 {
		t.Fatalf("交叉轴 = %d，想要按内容取 20", dims.Size.Y)
	}
}

func TestGrowDoesNotInflateCrossAxis(t *testing.T) {
	dims := Row(testCtx(200, 80), 0, Grow(1, (&probe{}).widget()))

	if dims.Size.Y != 0 {
		t.Fatalf("交叉轴被 Grow 撑到了 %d：Grow 只该补主轴", dims.Size.Y)
	}
	if dims.Size.X != 200 {
		t.Fatalf("主轴 = %d，想要 200", dims.Size.X)
	}
}

func TestHeightInRowFixesCrossAxis(t *testing.T) {
	p := &probe{report: image.Pt(50, 10)}
	dims := Row(testCtx(200, 80), 0, Height(30, p.widget()))

	if p.gotMin.Y != 30 || p.gotMax.Y != 30 {
		t.Fatalf("子项拿到的纵向约束 = %d..%d，想要 30..30", p.gotMin.Y, p.gotMax.Y)
	}
	if dims.Size.Y != 30 {
		t.Fatalf("行高 = %d，想要 30", dims.Size.Y)
	}
}

func TestWidthInColFixesCrossAxis(t *testing.T) {
	p := &probe{report: image.Pt(10, 10)}
	dims := Col(testCtx(300, 200), 0, Width(120, p.widget()))

	if p.gotMin.X != 120 || p.gotMax.X != 120 {
		t.Fatalf("子项拿到的横向约束 = %d..%d，想要 120..120", p.gotMin.X, p.gotMax.X)
	}
	if dims.Size != (image.Pt(120, 10)) {
		t.Fatalf("列尺寸 = %v，想要 120×10", dims.Size)
	}
}

func TestColGrowKeepsRemainingHeightOnly(t *testing.T) {
	ctx := testCtx(200, 300)
	a := &probe{report: image.Pt(40, 100)}
	b := &probe{report: image.Pt(40, 0)}

	dims := Col(ctx, 10, Height(100, a.widget()), Grow(1, b.widget()))

	if b.gotMax.Y != 190 {
		t.Fatalf("Grow 拿到的纵向空间 = %d，想要 190（300-100-10）", b.gotMax.Y)
	}
	if dims.Size.Y != 300 {
		t.Fatalf("列高 = %d，想要 300", dims.Size.Y)
	}
	if dims.Size.X != 40 {
		t.Fatalf("列宽 = %d，想要按内容 40（Grow 不该撑宽）", dims.Size.X)
	}
}

func TestFillFillsBothAxes(t *testing.T) {
	p := &probe{}
	dims := Col(testCtx(200, 300), 0, Fill(p.widget()))

	if p.gotMin != (image.Pt(200, 300)) || p.gotMax != (image.Pt(200, 300)) {
		t.Fatalf("Fill 的约束 = %v..%v，想要 200×300", p.gotMin, p.gotMax)
	}
	if dims.Size != (image.Pt(200, 300)) {
		t.Fatalf("列尺寸 = %v，想要 200×300", dims.Size)
	}
}

func TestCompositePage(t *testing.T) {
	header := &probe{report: image.Pt(0, 10)}
	body := &probe{}

	dims := Col(testCtx(800, 600), 12,
		Height(96, header.widget()),
		Grow(1, FillBoth(body.widget())),
	)

	if body.gotMin != (image.Pt(800, 492)) {
		t.Fatalf("正文约束 = %v，想要 800×492（600-96-12）", body.gotMin)
	}
	if dims.Size != (image.Pt(800, 600)) {
		t.Fatalf("页面尺寸 = %v，想要 800×600", dims.Size)
	}
}

func TestSizeFixesBothAxes(t *testing.T) {
	p := &probe{report: image.Pt(1, 1)}
	dims := Row(testCtx(300, 100), 0, Size(120, 40, p.widget()))

	if p.gotMin != (image.Pt(120, 40)) {
		t.Fatalf("子项约束 = %v，想要 120×40", p.gotMin)
	}
	if dims.Size != (image.Pt(120, 40)) {
		t.Fatalf("尺寸 = %v，想要 120×40", dims.Size)
	}
}

func TestFixedSizeWinsOverAvailable(t *testing.T) {
	p := &probe{report: image.Pt(10, 10)}
	q := &probe{report: image.Pt(10, 10)}

	dims := Row(testCtx(100, 50), 0, Width(300, p.widget()), Fit(q.widget()))

	// 声明尺寸优先（与迁移前的手写 box 一致）：子项拿到 300，行本身被父级夹到 100。
	if p.gotMax.X != 300 {
		t.Fatalf("定宽 300 应原样交给内容，得到 %d", p.gotMax.X)
	}
	if dims.Size.X != 100 {
		t.Fatalf("行宽 = %d，想要 100", dims.Size.X)
	}
}

func TestCenterKeepsContentSize(t *testing.T) {
	p := &probe{report: image.Pt(40, 20)}
	dims := Center(p.widget())(exactCtx(200, 100))

	if dims.Size != (image.Pt(200, 100)) {
		t.Fatalf("居中容器尺寸 = %v，想要 200×100", dims.Size)
	}
	if p.gotMin != (image.Point{}) {
		t.Fatalf("内容尺寸被 Min 夹住了：Min = %v", p.gotMin)
	}
}

func TestMeasureClearsMin(t *testing.T) {
	ctx := testCtx(300, 50)
	ctx.Constraints.Min = image.Pt(200, 40)

	p := &probe{report: image.Pt(30, 10)}
	got := Measure(ctx, p.widget())

	if got != (image.Pt(30, 10)) {
		t.Fatalf("量到的尺寸 = %v，想要 30×10", got)
	}
	if p.gotMin != (image.Point{}) {
		t.Fatalf("Measure 没有清 Min：%v", p.gotMin)
	}
}

func TestMeasureReturnsDesignUnits(t *testing.T) {
	ctx := testCtx(300, 50)
	ctx.Metric.PxPerDp = 0.5
	ctx.Metric.PxPerSp = 0.5

	// 内容按像素报 30×10；在 0.5 倍缩放里等于 60×20 设计 dp。
	p := &probe{report: image.Pt(30, 10)}
	if got := Measure(ctx, p.widget()); got != (image.Pt(60, 20)) {
		t.Fatalf("缩放上下文里 Measure = %v，想要 60×20 设计 dp", got)
	}
}

func TestScaleAndSpace(t *testing.T) {
	sc := Scale(testCtx(960, 540), 1920, 1080)

	if got := sc.Dp(unit.Dp(96)); got != 48 {
		t.Fatalf("缩放后 96 设计 dp = %d 像素，想要 48", got)
	}
	if got := Space(sc); got != (image.Pt(1920, 1080)) {
		t.Fatalf("Space = %v，想要 1920×1080", got)
	}
}

func TestWidths(t *testing.T) {
	got := Widths(600, 10, []Track{Px(60), Fr(1), Fr(2)})
	want := []int{60, 173, 347}

	if len(got) != len(want) {
		t.Fatalf("列数 = %d，想要 %d", len(got), len(want))
	}
	sum := 0
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("列宽 = %v，想要 %v（余数应给最后一个按权重分的列）", got, want)
		}
		sum += got[i]
	}
	if sum+10*(len(got)-1) != 600 {
		t.Fatalf("列宽之和 + 间隙 = %d，想要 600", sum+10*(len(got)-1))
	}
}

func TestWidthsShrinksWhenFixedOverflows(t *testing.T) {
	got := Widths(300, 0, []Track{Px(200), Px(200)})
	if got[0] != 150 || got[1] != 150 {
		t.Fatalf("固定列超预算时应按比例压到 150/150，得到 %v", got)
	}
}

func TestRowTracks(t *testing.T) {
	ctx := testCtx(600, 40)
	a := &probe{report: image.Pt(0, 10)}
	b := &probe{report: image.Pt(0, 10)}

	dims := RowTracks(ctx, 0, []Track{Px(100), Fr(1)}, []layout.Widget{a.widget(), b.widget()})

	if a.gotMin.X != 100 || b.gotMin.X != 500 {
		t.Fatalf("列宽 = %d / %d，想要 100 / 500", a.gotMin.X, b.gotMin.X)
	}
	if dims.Size.X != 600 {
		t.Fatalf("行宽 = %d，想要 600", dims.Size.X)
	}
}

func TestRawPassthrough(t *testing.T) {
	a := &probe{report: image.Pt(40, 20)}
	b := &probe{report: image.Pt(40, 20)}

	dims := Row(testCtx(200, 50), 10,
		Raw(layout.Rigid(a.widget())),
		Fit(b.widget()),
	)

	if dims.Size.X != 90 {
		t.Fatalf("行宽 = %d，想要 40+10+40=90", dims.Size.X)
	}
}

func TestSpacePushesSiblings(t *testing.T) {
	a := &probe{report: image.Pt(40, 10)}
	b := &probe{report: image.Pt(40, 10)}

	dims := Row(testCtx(300, 20), 0, Fit(a.widget()), Spacer(1), Fit(b.widget()))

	if dims.Size.X != 300 {
		t.Fatalf("行宽 = %d，想要 300（Spacer 应吃掉 220）", dims.Size.X)
	}
}

func TestFillWeightFillsCrossAxis(t *testing.T) {
	a := &probe{report: image.Pt(20, 10)}
	b := &probe{}
	c := &probe{}
	d := &probe{}

	dims := Row(testCtx(320, 80), 0,
		Fit(a.widget()),
		FillWeight(2, b.widget()),
		Fit(c.widget()),
		FillWeight(1, d.widget()),
	)

	if dims.Size != (image.Pt(320, 80)) {
		t.Fatalf("尺寸 = %v，想要 320×80（FillWeight 填满交叉轴）", dims.Size)
	}
}

func TestFillWeightKeepsContentConstraints(t *testing.T) {
	p := &probe{report: image.Pt(10, 10)}

	dims := Row(testCtx(320, 80), 0, FillWeight(1, p.widget()))

	// 旧 grow 的语义：只补返回尺寸，不改内容拿到的交叉轴约束。
	// （主轴由 Gio 的 Flexed 定成 Min==Max，那是 Flex 自己的行为。）
	if p.gotMin.Y != 0 {
		t.Fatalf("FillWeight 不该改交叉轴约束：Min = %v", p.gotMin)
	}
	if dims.Size != (image.Pt(320, 80)) {
		t.Fatalf("尺寸 = %v，想要 320×80", dims.Size)
	}
}

func TestFixedSpaceUsesDesignUnits(t *testing.T) {
	a := &probe{report: image.Pt(40, 10)}
	b := &probe{report: image.Pt(40, 10)}

	// 30 设计 dp 的固定空槽位；在 0.5 倍缩放下等于 15 像素。
	ctx := testCtx(300, 40)
	ctx.Metric.PxPerDp = 0.5
	ctx.Metric.PxPerSp = 0.5

	dims := Row(ctx, 0, Fit(a.widget()), FixedSpace(30), Fit(b.widget()))

	if dims.Size.X != 95 {
		t.Fatalf("行宽 = %d，想要 95（40 + 15 + 40）", dims.Size.X)
	}
}

func TestOverlayLayersFillArea(t *testing.T) {
	a := &probe{report: image.Pt(10, 10)}
	b := &probe{report: image.Pt(10, 10)}

	dims := Overlay(testCtx(300, 120), a.widget(), b.widget())

	if a.gotMin != (image.Pt(300, 120)) || b.gotMin != (image.Pt(300, 120)) {
		t.Fatalf("图层约束 = %v / %v，想要整块 300×120", a.gotMin, b.gotMin)
	}
	if dims.Size != (image.Pt(300, 120)) {
		t.Fatalf("叠加区尺寸 = %v，想要 300×120", dims.Size)
	}
}

// TestHeaderShape 是真实顶部栏的验收用例：
// 左留白 90、logo 150×44、标题相对整行居中、右侧状态块与窗口按钮 140。
// 现在的写法要反算 leftPad = side - contentX - logoBoxW 来配平标题；
// 用 Overlay 之后，标题层直接拿整行宽度，不需要知道左右控件多宽。
func TestHeaderShape(t *testing.T) {
	logo := &probe{report: image.Pt(150, 44)}
	chips := &probe{report: image.Pt(320, 24)}
	win := &probe{report: image.Pt(140, 32)}
	title := &probe{report: image.Pt(400, 36)}

	empty := func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }

	dims := Overlay(exactCtx(1920, 96),
		Center(title.widget()),
		func(gtx layout.Context) layout.Dimensions {
			return Row(gtx, 24,
				Width(90, empty),
				Size(150, 44, logo.widget()),
				Grow(1, empty),
				Fit(chips.widget()),
				Width(140, win.widget()),
			)
		},
	)

	if dims.Size != (image.Pt(1920, 96)) {
		t.Fatalf("顶部栏尺寸 = %v，想要 1920×96", dims.Size)
	}
	if logo.gotMin != (image.Pt(150, 44)) {
		t.Fatalf("logo 约束 = %v，想要 150×44", logo.gotMin)
	}
	if win.gotMax.X != 140 {
		t.Fatalf("窗口按钮槽位 = %d，想要 140", win.gotMax.X)
	}
	if title.gotMax != (image.Pt(1920, 96)) {
		t.Fatalf("标题层拿到的区域 = %v，想要整行 1920×96", title.gotMax)
	}
}
