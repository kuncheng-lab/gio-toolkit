package refreshkit

import (
	"image"
	"testing"
	"time"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
)

func TestDelay(t *testing.T) {
	c := Cadence{}
	cases := []struct {
		name string
		p    Policy
		want time.Duration
	}{
		{"空闲", Policy{}, 0},
		{"有人在等", Policy{Waiting: true}, 100 * time.Millisecond},
		{"动画", Policy{Animating: true}, 16 * time.Millisecond},
		{"画完发现又变了", Policy{Stale: true}, 16 * time.Millisecond},
		{"又变又等：尽快", Policy{Stale: true, Waiting: true}, 16 * time.Millisecond},
		{"又变又在动画：按动画", Policy{Stale: true, Animating: true}, 16 * time.Millisecond},
	}
	for _, tc := range cases {
		if got := c.Delay(tc.p); got != tc.want {
			t.Errorf("%s：Delay = %v，想要 %v", tc.name, got, tc.want)
		}
	}
}

func TestDelayCustomCadence(t *testing.T) {
	c := Cadence{Animate: 8 * time.Millisecond, Wait: 50 * time.Millisecond}
	if got := c.Delay(Policy{Animating: true}); got != 8*time.Millisecond {
		t.Errorf("动画间隔 = %v，想要 8ms", got)
	}
	if got := c.Delay(Policy{Waiting: true}); got != 50*time.Millisecond {
		t.Errorf("等待间隔 = %v，想要 50ms", got)
	}
}

func TestFollowSchedulesAt(t *testing.T) {
	var r input.Router
	var ops op.Ops
	now := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	gtx := layout.Context{
		Constraints: layout.Constraints{Max: image.Pt(100, 100)},
		Now:         now,
		Source:      r.Source(),
		Ops:         &ops,
	}

	if !Follow(gtx, 100*time.Millisecond) {
		t.Fatal("Follow 应报告已预约")
	}
	r.Frame(&ops)
	got, ok := r.WakeupTime()
	if !ok {
		t.Fatal("预约之后应该有 wakeup")
	}
	if want := now.Add(100 * time.Millisecond); !got.Equal(want) {
		t.Fatalf("预约时刻 = %v，想要 %v", got, want)
	}
}

func TestFollowZeroDoesNotSchedule(t *testing.T) {
	var r input.Router
	var ops op.Ops
	gtx := layout.Context{Ops: &ops, Source: r.Source(), Now: time.Now()}

	if Follow(gtx, 0) {
		t.Fatal("延时为 0 时 Follow 不该预约")
	}
	r.Frame(&ops)
	if got, ok := r.WakeupTime(); ok {
		t.Fatalf("不该有 wakeup，得到 %v", got)
	}
}
