// sidebar.go —— 右侧边栏：STATUS 框 + 提示框。
// STATUS 内容随视图状态变化：
//
//	列表态：当前位置：首页 / 当前用户：<user> / 文章：N 篇
//	阅读态：当前文章：<标题> / 篇幅：N段 · M字
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
func sidebarView(arts []content.Article, user string, reading *content.Article,
	w, cols int, p theme.Palette) string {

	innerW := w - 2 // 圆角边框占 2 列
	if innerW < 4 {
		innerW = 4
	}
	status := statusBox(arts, user, reading, innerW, p)
	tips := tipsBox(innerW, p)
	body := lipgloss.JoinVertical(lipgloss.Left, status, "", tips)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(p.Border).
		Width(w - 2).
		MaxHeight(cols).
		Height((cols)).
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

// tipsBox 渲染 " 提示 " 框：3 行小贴士。
func tipsBox(innerW int, p theme.Palette) string {
	head := lipgloss.NewStyle().Foreground(p.TextMuted).Bold(true).
		Render(" 提示 ")
	sep := lipgloss.NewStyle().Foreground(p.Border).
		Render(strings.Repeat("─", maxInt(1, innerW-render.DisplayWidth(head)-1)))
	titleRow := head + " " + sep

	muted := lipgloss.NewStyle().Foreground(p.TextMuted)
	lines := []string{titleRow}
	for _, tip := range []string{
		"· 滚轮连滚会加速",
		"· Ctrl+T 循环三套配色",
		"· h/Esc 从正文返回列表",
	} {
		lines = append(lines, muted.Width(innerW-1).Render(tip))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// maxInt 取较大值（tips 里挡住极窄宽度）。
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
