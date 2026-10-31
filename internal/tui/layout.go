// layout.go —— 根布局：单线边框根框（左上角标题 + 底部键位提示条） +
// 主列（列表/正文，占剩余宽度）与右侧 Sidebar（总宽 30% 且最小 24 列）。
//
// 注：lipgloss v1.1 的 Style 没有"边框标题"能力，因此顶部标题行与
// 底部提示行是手工拼装的单线边框行（┌─标题─┐ / └─提示─┘），
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

// hints 返回当前视图对应的底部键位提示条文案（只写真实实现了的键位）。
func hints(v viewState) string {
	switch v {
	case viewList:
		return "↑↓/j/k 移动  Enter 阅读  Ctrl+T 主题  Q/Ctrl+C 退出"
	default: // viewRead
		return "↑↓/PgUp/PgDn 滚动  g/End 行首尾  ←/h/Esc 返回  Ctrl+T 主题  Q/Ctrl+C 退出"
	}
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

// bannerBottom 渲染底边框行："└" + 居中提示条（暗淡文字）+ "┘"。
// 提示过长时按显示宽度截断，保证行宽恒等于 w。
func bannerBottom(w int, s string, p theme.Palette) string {
	hint := s
	maxW := w - 4 // 角 + 与两侧至少 1 列填充
	if render.DisplayWidth(hint) > maxW {
		runes := []rune(hint)
		acc, i := 0, 0
		for i < len(runes) {
			cw := render.RuneWidth(runes[i])
			if acc+cw > maxW-1 {
				break
			}
			acc += cw
			i++
		}
		hint = string(runes[:i]) + "…"
	}
	hw := render.DisplayWidth(hint)
	pad := w - 2 - hw
	left := pad / 2
	right := pad - left
	bar := lipgloss.NewStyle().Foreground(p.Border)
	text := lipgloss.NewStyle().Foreground(p.TextMuted).Render(hint)
	return bar.Render("└"+strings.Repeat("─", left)) + text +
		bar.Render(strings.Repeat("─", right)) + bar.Render("┘")
}

// rootRender 把中部内容行装进根框（顶部标题行 + 内容区 + 底部提示行）。
func rootRender(mid string, w, rows int, p theme.Palette, v viewState) string {
	body := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderRight(true).
		BorderForeground(p.Border).
		Width(w - 2).
		Height(rows).
		Render(mid)
	return bannerTop(w, p) + "\n" + body + "\n" + bannerBottom(w, hints(v), p)
}

// renderRoot 组装整屏：主列（列表/正文）+ Sidebar，再包根框。
func renderRoot(arts []content.Article, user string, l listModel, r readerModel,
	v viewState, w, rows int, p theme.Palette) string {

	inner := w - 2
	sbW := sidebarWidth(inner)
	mainW := inner - sbW
	if mainW < 1 {
		mainW = 1
	}

	var main string
	if v == viewRead && r.article != nil {
		main = l_pad(r.view(mainW, rows, p), mainW)
	} else {
		main = l_pad(l.view(arts, rows, p), mainW)
	}

	side := l_pad(sidebarView(arts, user, curArticle(arts, l, r, v), sbW, rows, p), sbW)
	mid := lipgloss.JoinHorizontal(lipgloss.Top, main, side)
	return rootRender(mid, w, rows, p, v)
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
