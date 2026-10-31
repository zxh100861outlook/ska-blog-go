// theme_test.go —— 配色注册表与循环切换测试（标准库断言）。
package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestCycle 内置 4 套主题循环：默认 → aura → 冷青 → gruvbox → 默认。
func TestCycle(t *testing.T) {
	want := []string{"默认", "aura", "冷青", "gruvbox"}
	if got := Names(); len(got) != 4 {
		t.Fatalf("应注册 4 套主题, got %d: %v", len(got), want)
	}
	cur := "默认"
	for i := 0; i < 9; i++ {
		cur = Cycle(cur)
		if cur != want[(i+1)%4] {
			t.Fatalf("第 %d 次切换应为 %q, got %q", i, want[(i+1)%4], cur)
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
	g := Get("gruvbox")
	for field, want := range map[string]lipgloss.Color{
		"Primary":    "#fe8019",
		"Background": "#282828",
		"Text":       "#ebdbb2",
		"TextMuted":  "#a89984",
		"Border":     "#504945",
	} {
		got := map[string]lipgloss.Color{
			"Primary":    g.Primary,
			"Background": g.Background,
			"Text":       g.Text,
			"TextMuted":  g.TextMuted,
			"Border":     g.Border,
		}[field]
		if got != want {
			t.Errorf("gruvbox %s 应为 %q, got %q", field, want, got)
		}
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
		} {
			if v == "" {
				t.Errorf("主题 %q 的 %s 颜色为空", name, field)
			}
		}
	}
}
