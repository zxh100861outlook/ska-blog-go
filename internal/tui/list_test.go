// list_test.go —— 卡片列表：可见窗口滚动跟随 + 摘要/时长格式。
package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"skablog/internal/content"
	"skablog/internal/theme"
)

// mkArts 造 n 篇长正文文章。
func mkArts(n int) []content.Article {
	arts := make([]content.Article, n)
	for i := range arts {
		arts[i] = content.Article{
			ID:        string(rune('a' + i)),
			Title:     "标题" + string(rune('0'+i)),
			Body:      "这是第" + strings.Repeat("很", 60) + "篇文章\n\n第二段",
			WordCount: 401, // ceil(401/400)=2 分钟
		}
	}
	return arts
}

// TestListFollow 窗口 6 行（2 张卡片）：聚焦第 3 张时窗口应整体
// 下移 1 卡（含空行 3 行），聚焦滑回顶部也回滚。
func TestListFollow(t *testing.T) {
	arts := mkArts(6)
	l := listModel{width: 20}
	rows := 6

	l.move(arts, 1, rows) // 聚焦 a1，仍在窗口
	if l.offset != 0 {
		t.Fatalf("聚焦第2张不应滚动, offset=%d", l.offset)
	}
	l.move(arts, 1, rows) // 聚焦 a2（行 6..8），窗口 0..5 装不下 → offset=3
	if l.offset != cardBlock {
		t.Errorf("向下出窗应跟随滚动, offset=%d want %d", l.offset, cardBlock)
	}
	l.move(arts, -1, rows) // 回到 a1：仍完整可见 → offset 维持 3（最小滚动原则）
	if l.focused != 1 || l.offset != cardBlock {
		t.Errorf("聚焦 a1 时卡片应仍可见, focused=%d offset=%d", l.focused, l.offset)
	}
	l.move(arts, -1, rows) // 回到 a0：滑出窗口顶 → 回滚到 0
	if l.focused != 0 || l.offset != 0 {
		t.Errorf("向上出窗应回滚, focused=%d offset=%d", l.focused, l.offset)
	}
}

// TestListFollowClamp 边界：到底不再滚，回顶不再滚。
func TestListFollowClamp(t *testing.T) {
	arts := mkArts(4)
	l := listModel{width: 20}
	rows := 3
	l.move(arts, 100, rows)
	want := len(arts) - 1
	if l.focused != want {
		t.Fatalf("应聚焦最后一张, focused=%d", l.focused)
	}
	topBefore := l.offset
	l.move(arts, 1, rows)
	if l.focused != want || l.offset != topBefore {
		t.Errorf("底部不应再滚: focused=%d offset=%d", l.focused, l.offset)
	}
}

// TestBlockLines 选中卡：▸ 前缀、[id] 标题、摘要截断、时长格式。
func TestBlockLines(t *testing.T) {
	a := content.Article{
		ID: "wj2", Title: "github clone 加速",
		Body:      "很长的摘要内容" + strings.Repeat("字", 40),
		WordCount: 401,
	}
	lines := blockLines(a, true, 60, theme.Get("aura"))
	if len(lines) != cardBlock {
		t.Fatalf("卡片应 %d 行, got %d", cardBlock, len(lines))
	}
	top := stripANSI(lines[0])
	if !strings.HasPrefix(top, "▍ ") {
		t.Errorf("选中卡应有 ▍ 块状光标, got %q", top)
	}
	if !strings.Contains(top, "wj2") || !strings.Contains(top, "github clone 加速") {
		t.Errorf("标题行应含 id 与标题, got %q", top)
	}
	if !strings.Contains(lines[1], "1段·约2分钟") {
		t.Errorf("摘要应含统计“1段·约2分钟”, got %q", lines[1])
	}
	if !strings.Contains(lines[1], "…") {
		t.Errorf("长摘要应截断加省略号, got %q", lines[1])
	}
	// 未选中：无 ▍，无背景色块
	linesOff := blockLines(a, false, 60, theme.Get("aura"))
	if strings.Contains(stripANSI(linesOff[0]), "▍") {
		t.Errorf("未选中不应带 ▍ 光标, got %q", stripANSI(linesOff[0]))
	}
	if strings.Contains(linesOff[0], "\x1b[48;") {
		t.Errorf("未选中不应有背景色块, got %q", linesOff[0])
	}
}

// TestSummaryPlainText 摘要应为去标记纯文本（无 ANSI 码）。
func TestSummaryPlainText(t *testing.T) {
	a := content.Article{ID: "t", Title: "标题", Body: "- 列表项**粗体**`码`"}
	lines := blockLines(a, false, 40, theme.Get("冷青"))
	sum := stripANSI(lines[1])
	if strings.ContainsAny(sum, "*`") {
		t.Errorf("摘要应去标记(星号/反引号), got %q", sum)
	}
	if !strings.Contains(sum, "- 列表项") {
		t.Errorf("列表项摘要应保留项目符号与文字, got %q", sum)
	}
}

// TestMinutes 时长公式 ceil(w/400) 且 ≥1。
func TestMinutes(t *testing.T) {
	if got := minutes(0); got != 1 {
		t.Errorf("0 字应 1 分钟, got %d", got)
	}
	if got := minutes(400); got != 1 {
		t.Errorf("400 字应 1 分钟, got %d", got)
	}
	if got := minutes(401); got != 2 {
		t.Errorf("401 字应 2 分钟, got %d", got)
	}
}

// 编译期防呆：确认 lipgloss.Style 仍被使用（样式从 palette 现取）。
var _ = lipgloss.Color("#000000")
