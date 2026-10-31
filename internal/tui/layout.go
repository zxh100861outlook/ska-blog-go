// layout.go —— 根布局：单线边框根框（左上角标题，底部无键位提示条——
// 键位已全部搬进右侧 Sidebar 的动态"提示"面板）+ 主列（列表/正文，占剩余宽度）
// 与右侧 Sidebar（总宽 30% 且最小 24 列；Tab 可收起为 0，主列全宽）。
//
// 注：lipgloss v1.1 的 Style 没有"边框标题"能力，因此顶部标题行与
// 底部边框行是手工拼装的单线边框行（┌─标题─┐ / └─────┘），
// 中部内容区用 BorderLeft/Right 拼左右立柱，三段 JoinVertical 成根框。
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"skablog/internal/content"
	"skablog/internal/render"
	"skablog/internal/theme"
)

// rootTitle 是根框左上角标题（两侧空格保留呼吸感）。
const rootTitle = " SKA-BLOG://GO "

// sideRatio / sideMin 是 Sidebar 宽度规则：总宽 30%，最小 24 列。
const (
	sideRatio = 3.0 / 10.0
	sideMin   = 24
)

// sidebarWidth 按总内宽计算 Sidebar 宽度；窄终端保底给主列留空间。
func sidebarWidth(inner int) int {
	w := int(float64(inner) * sideRatio)
	if w < sideMin {
		w = sideMin
	}
	// 极窄终端兜底：Sidebar 最多吃掉 inner-6，主列至少留 6 列
	if max := inner - 6; w > max {
		w = max
	}
	if w < sideMin {
		w = sideMin
	}
	return w
}

// bannerTop 渲染顶边框行："┌" + " SKA-BLOG://GO "（主色加粗）+ "─" 填充 + "┐"。
func bannerTop(w int, p theme.Palette) string {
	bar := lipgloss.NewStyle().Foreground(p.Border)
	cornerL := bar.Render("┌")
	cornerR := bar.Render("┐")
	titleW := render.DisplayWidth(rootTitle)
	fill := w - 2 - titleW
	if fill < 1 {
		fill = 1
	}
	title := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).Render(rootTitle)
	return cornerL + title + bar.Render(strings.Repeat("─", fill)) + cornerR
}

// bannerBottom 渲染底边框行（无文字）："└" + "─" 填充 + "┘"。
// 键位提示已搬进侧栏"提示"面板，这里只保留纯边框。
func bannerBottom(w int, p theme.Palette) string {
	bar := lipgloss.NewStyle().Foreground(p.Border)
	return bar.Render("└"+strings.Repeat("─", w-2)) + bar.Render("┘")
}

// rootRender 把中部内容行装进根框（顶部标题行 + 内容区 + 底部纯边框行）。
func rootRender(mid string, w, rows int, p theme.Palette) string {
	body := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderRight(true).
		BorderForeground(p.Border).
		Width(w - 2).
		Height(rows).
		Render(mid)
	return bannerTop(w, p) + "\n" + body + "\n" + bannerBottom(w, p)
}

// renderRoot 组装整屏：主列（列表/正文，可带搜索/跳转输入行）+ Sidebar，再包根框。
func renderRoot(m Model, w, rows int) string {
	p := theme.Get(m.theme)
	mainW := m.mainW()
	if mainW < 1 {
		mainW = 1
	}

	// 输入态（搜索/跳转）：主列顶部多占 1 行输入行
	listRowsN := rows
	if m.mode != inputNone {
		listRowsN = rows - 1
		if listRowsN < 1 {
			listRowsN = 1
		}
	}

	var main string
	if m.view == viewRead && m.reader.article != nil {
		main = l_pad(m.reader.view(mainW, rows, p), mainW)
	} else {
		main = l_pad(m.list.view(m.visible, listRowsN, p), mainW)
	}
	if m.mode != inputNone {
		main = inputLine(m.mode, m.query, m.jumpErr, mainW, p) + "\n" + main
	}

	sbW := m.sbW()
	var mid string
	if sbW > 0 {
		side := l_pad(sidebarView(m, sbW, rows, p), sbW)
		mid = lipgloss.JoinHorizontal(lipgloss.Top, main, side)
	} else {
		mid = main
	}
	return rootRender(mid, w, rows, p)
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
