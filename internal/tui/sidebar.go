// sidebar.go —— 右侧边栏：STATUS 框 + 动态"提示"键位面板。
// STATUS 内容随视图状态变化：
//
//	列表态：当前位置：首页 / 当前用户：<user> / 文章：N 篇
//	阅读态：当前文章：<标题> / 篇幅：N段 · M字
//
// "提示"面板按当前视图（列表/阅读/搜索/跳转）列出可用键（每键一行，
// 从实际 case 生成），并附"主题：<名字>"与"侧栏：开/收"两行状态。
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"skablog/internal/content"
	"skablog/internal/render"
	"skablog/internal/theme"
)

// sidebarView 渲染整个右栏（宽 w，即含边框的总宽），高度由调用方容器补足。
func sidebarView(m Model, w, cols int, p theme.Palette) string {
	innerW := w - 2 // 圆角边框占 2 列
	if innerW < 4 {
		innerW = 4
	}
	status := statusBox(m.articles, m.username, curArticle(m.articles, m.list, m.reader, m.view), innerW, p)
	tips := keyPanel(m.view, m.mode, m.sidebarOpen, p.Name, innerW, p)
	body := lipgloss.JoinVertical(lipgloss.Left, status, "", tips)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(p.Border).
		Width(w - 2).
		MaxHeight(cols).
		Height(cols).
		Render(body)
}

// statusBox 渲染 STATUS 框（含标题行 " STATUS "）。
func statusBox(arts []content.Article, user string, reading *content.Article,
	innerW int, p theme.Palette) string {

	labelS := lipgloss.NewStyle().Foreground(p.Primary).Bold(true)
	infoS := lipgloss.NewStyle().Foreground(p.Text)
	mutedS := lipgloss.NewStyle().Foreground(p.TextMuted)
	line := func(label, val string, muted bool) string {
		style := infoS
		if muted {
			style = mutedS
		}
		return style.Width(innerW - 1).Render(labelS.Render(label) + style.Render(val))
	}

	var lines []string
	if reading == nil {
		// 首页态
		lines = append(lines,
			line("当前位置：", "首页", false),
			line("当前用户：", user, false),
			line("文章：", fmt.Sprintf("%d 篇", len(arts)), false),
		)
	} else {
		// 阅读态
		lines = append(lines,
			line("当前文章：", reading.Title, false),
			line("篇幅：", fmt.Sprintf("%d段 · %d字", reading.ParagraphCount(), reading.WordCount), false),
		)
	}

	head := lipgloss.NewStyle().Foreground(p.TextMuted).Bold(true).
		Render(" STATUS ")
	sep := lipgloss.NewStyle().Foreground(p.Border).
		Render(strings.Repeat("─", maxInt(1, innerW-render.DisplayWidth(head)-1)))
	titleRow := head + " " + sep

	return lipgloss.JoinVertical(lipgloss.Left, append([]string{titleRow}, lines...)...)
}

// keyPanel 渲染动态"提示"面板：按当前视图/输入态列出可用键（每键一行），
// 附"主题：<名字>"与"侧栏：开/收"两行状态。
func keyPanel(v viewState, mode inputMode, sidebarOpen bool, themeName string,
	innerW int, p theme.Palette) string {

	head := lipgloss.NewStyle().Foreground(p.TextMuted).Bold(true).
		Render(" 提示 ")
	sep := lipgloss.NewStyle().Foreground(p.Border).
		Render(strings.Repeat("─", maxInt(1, innerW-render.DisplayWidth(head)-1)))
	titleRow := head + " " + sep

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
	lines := []string{titleRow}
	for _, kv := range keys {
		lines = append(lines, muted.Width(innerW-1).Render("· "+kv[0]+" "+kv[1]))
	}
	// 状态行：主题名（纯文字，无色终端下 Ctrl+T 也能感知切换）+ 侧栏开/收
	sc := "开"
	if !sidebarOpen {
		sc = "收"
	}
	lines = append(lines, muted.Width(innerW-1).Render("· 主题："+themeName))
	lines = append(lines, muted.Width(innerW-1).Render("· 侧栏："+sc))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// maxInt 取较大值（tips 里挡住极窄宽度）。
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
