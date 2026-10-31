// theme_test.go —— 配色注册表与循环切换测试（标准库断言）。
package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestCycle 内置 3 套主题循环：默认 → aura → 冷青 → 默认。
func TestCycle(t *testing.T) {
	want := []string{"默认", "aura", "冷青"}
	if got := Names(); len(got) != 3 {
		t.Fatalf("应注册 3 套主题, got %d: %v", len(got), want)
	}
	cur := "默认"
	for i := 0; i < 7; i++ {
		cur = Cycle(cur)
		if cur != want[(i+1)%3] {
			t.Fatalf("第 %d 次切换应为 %q, got %q", i, want[(i+1)%3], cur)
		}
	}
}

// TestCycleUnknown 未知名字应从默认开始，而不是 panic。
func TestCycleUnknown(t *testing.T) {
	if got := Cycle("不存在"); got != "默认" {
		t.Errorf("未知名字 Cycle 应回退默认, got %q", got)
	}
}

// TestGet 未知名字回退默认；已知名字原样返回。
func TestGet(t *testing.T) {
	if got := Get("无此主题").Name; got != "默认" {
		t.Errorf("Get 未知名字应回退默认, got %q", got)
	}
	if got := Get("aura"); got.Primary != lipgloss.Color("#5cb66b") {
		t.Errorf("aura 主色应为 #5cb66b, got %q", got.Primary)
	}
}

// TestPaletteComplete 每套 palette 的颜色字段都必须非空，防止漏配。
func TestPaletteComplete(t *testing.T) {
	for _, name := range Names() {
		p := Get(name)
		for field, v := range map[string]lipgloss.Color{
			"Primary":    p.Primary,
			"Background": p.Background,
			"Text":       p.Text,
			"TextMuted":  p.TextMuted,
			"Border":     p.Border,
			"SelBg":      p.SelBg,
			"ListTitle":  p.ListTitle,
			"Meta":       p.Meta,
			"Divider":    p.Divider,
		} {
			if v == "" {
				t.Errorf("主题 %q 的 %s 颜色为空", name, field)
			}
		}
	}
}
