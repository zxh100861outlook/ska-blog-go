// accel_test.go —— 滚轮加速纯函数测试（连滚翻倍、封顶、超时重置）。
package tui

import (
	"testing"
	"time"
)

func TestAccelFirstEvent(t *testing.T) {
	t0 := time.Now()
	if got := accel(time.Time{}, t0); got != 1 {
		t.Errorf("首次事件应 1, got %d", got)
	}
	if got := accel(t0, t0); got != 1 {
		t.Errorf("同刻事件应 1, got %d", got)
	}
}

func TestAccelTimeoutReset(t *testing.T) {
	t0 := time.Now()
	if got := accel(t0, t0.Add(accelWindow)); got != 1 {
		t.Errorf("间隔=300ms 应重置 1, got %d", got)
	}
	if got := accel(t0, t0.Add(2*time.Second)); got != 1 {
		t.Errorf("长停顿后应重置 1, got %d", got)
	}
	if got := accel(t0, t0.Add(-time.Second)); got != 1 {
		t.Errorf("时间倒流应防御性重置 1, got %d", got)
	}
}

func TestAccelDoubling(t *testing.T) {
	// 连滚间隔打拍翻倍：0拍=1, 1拍=2, 2拍=4, 3拍=8, ≥4拍封顶 16
	cases := []struct {
		d    time.Duration
		want int
	}{
		{10 * time.Millisecond, 1},
		{accelBeat, 2},
		{2 * accelBeat, 4},
		{3 * accelBeat, 8},
		{4 * accelBeat, 16},
		{accelWindow - 1, 16}, // 260ms 仍在窗口内，倍率封顶 16
	}
	t0 := time.Now()
	for _, c := range cases {
		if got := accel(t0, t0.Add(c.d)); got != c.want {
			t.Errorf("accel(+%v) = %d, want %d", c.d, got, c.want)
		}
	}
}
