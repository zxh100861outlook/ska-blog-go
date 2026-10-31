// sidebar.go —— 右侧边栏（无边框小节风格，对齐 soft-serve pane）：
// 两个小节 "STATUS" 与 "KEYS"，小节标题为大写小字灰，内容行缩进两格。
//
// STATUS 内容随视图状态变化：
//
//	列表态：当前位置：首页 / 当前用户：<user> / 文章：N 篇
//	阅读态：当前文章：<标题> / 篇幅：N段 · M字
//
// KEYS 小节按当前视图（列表/阅读/搜索/跳转）列出可用键（每键一行，
// 从实际 case 生成），并附"主题：<名字>"与"侧栏：开/收"两行状态。
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"skablog/internal/content"
	"skablog/internal/theme"
)

// sidebarView 渲染整个右栏（宽 w），高度 cols 由调用方容器补足（填满空白行）。
func sidebarView(m Model, w, cols int, p theme.Palette) string {
	if w < 4 {
		w = 4
	}
	var lines []string
	// STATUS 小节
	lines = append(lines, sectionTitle("STATUS", p))
	reading := curArticle(m.articles, m.list, m.reader, m.view)
	for _, l := range statusLines(m.articles, m.username, reading, p) {
		lines = append(lines, "  "+l)
	}
	lines = append(lines, "") // 小节间空一行
	// KEYS 小节
	lines = append(lines, sectionTitle("KEYS", p))
	for _, l := range keyLines(m.view, m.mode, m.sidebarOpen, p.Name, p) {
		lines = append(lines, "  "+l)
	}
	// 补足高度并统一宽度
	if len(lines) > cols {
		lines = lines[:cols]
	}
	for len(lines) < cols {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = lipgloss.NewStyle().Width(w).Render(lines[i])
	}
	return strings.Join(lines, "\n")
}

// sectionTitle 渲染无边框小节标题：大写小字灰（TextMuted）。
func sectionTitle(name string, p theme.Palette) string {
	return lipgloss.NewStyle().Foreground(p.TextMuted).Bold(true).
		Render(strings.ToUpper(name))
}

// statusLines 返回 STATUS 小节的内容行（不含缩进）。
func statusLines(arts []content.Article, user string, reading *content.Article,
	p theme.Palette) []string {

	labelS := lipgloss.NewStyle().Foreground(p.Primary).Bold(true)
	valS := lipgloss.NewStyle().Foreground(p.Text)
	line := func(label, val string) string {
		return labelS.Render(label) + valS.Render(val)
	}
	if reading == nil {
		// 首页态
		return []string{
			line("当前位置：", "首页"),
			line("当前用户：", user),
			line("文章：", fmt.Sprintf("%d 篇", len(arts))),
		}
	}
	// 阅读态
	return []string{
		line("当前文章：", reading.Title),
		line("篇幅：", fmt.Sprintf("%d段 · %d字", reading.ParagraphCount(), reading.WordCount)),
	}
}

// keyLines 返回 KEYS 小节的内容行（不含缩进）。
func keyLines(v viewState, mode inputMode, sidebarOpen bool, themeName string,
	p theme.Palette) []string {

	var keys [][2]string // {键, 说明}——逐条对应 internal/tui 里的 case 分支
	switch {
	case mode == inputSearch:
		keys = [][2]string{
			{"字符/退格", "输入关键词"},
			{"Enter", "打开命中"},
			{"Esc", "取消搜索"},
		}
	case mode == inputJump:
		keys = [][2]string{
			{"字符/退格", "输入文章ID"},
			{"Enter", "跳转"},
			{"Esc", "取消"},
		}
	case v == viewList:
		keys = [][2]string{
			{"↑↓/j/k", "移动"},
			{"Enter", "阅读"},
			{"/", "搜索"},
			{":", "跳转"},
			{"Tab", "侧栏"},
			{"Ctrl+T", "主题"},
			{"Q/Ctrl+C", "退出"},
		}
	default: // viewRead
		keys = [][2]string{
			{"↑↓/j/k", "滚动"},
			{"←/h/Esc", "返回列表"},
			{"Tab", "侧栏"},
			{"Ctrl+T", "主题"},
			{"Q/Ctrl+C", "退出"},
		}
	}

	muted := lipgloss.NewStyle().Foreground(p.TextMuted)
	lines := make([]string, 0, len(keys)+2)
	for _, kv := range keys {
		lines = append(lines, muted.Render("· "+kv[0]+" "+kv[1]))
	}
	// 状态行：主题名（纯文字，无色终端下 Ctrl+T 也能感知切换）+ 侧栏开/收
	sc := "开"
	if !sidebarOpen {
		sc = "收"
	}
	lines = append(lines, muted.Render("· 主题："+themeName))
	lines = append(lines, muted.Render("· 侧栏："+sc))
	return lines
}
