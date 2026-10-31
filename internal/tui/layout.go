// layout.go —— 根布局（soft-serve 式，无整周包边）：
// 顶部一行 header（无边框）：左 "SKA·BLOG" 品牌字（Primary 粗体），
// 右对齐 "访客：<user> · 文章 N 篇"；header 下一条细分割线（Divider 色）；
// 主体为左右两栏（主列 + Sidebar），两栏间一条竖分割线（仅侧栏展开时画）。
// 取消原根框的整周包边与底部键位提示条（键位全部在侧栏 KEYS 小节）。
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"skablog/internal/content"
	"skablog/internal/render"
	"skablog/internal/theme"
)

// sideRatio / sideMin 是 Sidebar 宽度规则：总宽 30%，最小 24 列。
const (
	sideRatio = 3.0 / 10.0
	sideMin   = 24
)

// sidebarWidth 按总宽计算 Sidebar 宽度；窄终端保底给主列留空间。
func sidebarWidth(inner int) int {
	w := int(float64(inner) * sideRatio)
	if w < sideMin {
		w = sideMin
	}
	// 极窄终端兜底：Sidebar 最多吃掉 inner-7（1 列留给竖分割线），主列至少留 6 列
	if max := inner - 7; w > max {
		w = max
	}
	if w < sideMin {
		w = sideMin
	}
	return w
}

// headerLine 渲染顶部 header 行（无边框）：左品牌，右访客信息右对齐。
func headerLine(m Model, w int, p theme.Palette) string {
	brand := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render("SKA·BLOG")
	right := fmt.Sprintf("访客：%s · 文章 %d 篇", m.username, len(m.articles))
	rightW := w - render.DisplayWidth(brand)
	if rightW < 1 {
		rightW = 1
	}
	rightS := lipgloss.NewStyle().Foreground(p.TextMuted).
		Width(rightW).Align(lipgloss.Right).Render(right)
	return brand + rightS
}

// dividerLine 渲染 header 下的水平细分割线。
func dividerLine(w int, p theme.Palette) string {
	if w < 1 {
		w = 1
	}
	return lipgloss.NewStyle().Foreground(p.Divider).Render(strings.Repeat("─", w))
}

// vdivider 渲染两栏之间的竖分割线（高 rows 行）。
func vdivider(rows int, p theme.Palette) string {
	if rows < 1 {
		rows = 1
	}
	b := lipgloss.NewStyle().Foreground(p.Divider)
	return b.Render(strings.Repeat("│\n", rows-1)) + b.Render("│")
}

// renderRoot 组装整屏：header + 分割线 + 主体（主列 + 竖分割线 + Sidebar）。
func renderRoot(m Model, w, h int) string {
	p := theme.Get(m.theme)
	bodyRows := h - 2 // header(1) + 分割线(1)
	if bodyRows < 1 {
		bodyRows = 1
	}
	mainW := m.mainW()
	if mainW < 1 {
		mainW = 1
	}

	// 输入态（搜索/跳转）：主列顶部多占 1 行输入行
	listRowsN := bodyRows
	if m.mode != inputNone {
		listRowsN = bodyRows - 1
		if listRowsN < 1 {
			listRowsN = 1
		}
	}

	var main string
	if m.view == viewRead && m.reader.article != nil {
		main = l_pad(m.reader.view(mainW, bodyRows, p), mainW)
	} else {
		main = l_pad(m.list.view(m.visible, listRowsN, p), mainW)
	}
	if m.mode != inputNone {
		main = inputLine(m.mode, m.query, m.jumpErr, mainW, p) + "\n" + main
	}

	sbW := m.sbW()
	var body string
	if sbW > 0 {
		side := l_pad(sidebarView(m, sbW, bodyRows, p), sbW)
		body = lipgloss.JoinHorizontal(lipgloss.Top, main, vdivider(bodyRows, p), side)
	} else {
		body = main
	}
	return headerLine(m, w, p) + "\n" + dividerLine(w, p) + "\n" + body
}

// inputLine 渲染搜索/跳转输入行（显示在主列顶部）。
func inputLine(mode inputMode, query []rune, err bool, w int, p theme.Palette) string {
	prompt := "/"
	if mode == inputJump {
		prompt = ":"
	}
	text := prompt + " " + string(query) + "▌"
	if mode == inputJump && err {
		text = prompt + " " + string(query) + " 无此文▌"
	}
	text = truncateVisible(text, w)
	return lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Width(w).Render(text)
}

// curArticle 取当前"阅读中"的文章指针（列表态返回 nil）。
func curArticle(arts []content.Article, l listModel, r readerModel, v viewState) *content.Article {
	if v == viewRead && r.article != nil {
		return r.article
	}
	return nil
}

// l_pad 用空行使块高度等于 rows（JoinHorizontal 需要两列行数对齐）。
// 注：主列内容已按 rows 行生成；这里兜底防短内容错位。
func l_pad(s string, w int) string {
	if w < 1 {
		w = 1
	}
	return lipgloss.NewStyle().Width(w).Render(s)
}
