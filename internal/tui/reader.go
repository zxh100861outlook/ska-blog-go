// Package tui —— TUI 核心：根 model 状态机 + 卡片列表 + 正文阅读 + 侧栏。
// 布局对齐 ska-site-tui 参考实现：根边框（左上角标题/底部键位提示条）、
// 左主列（列表/正文）+ 右侧 30% STATUS 侧栏。
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"skablog/internal/content"
	"skablog/internal/render"
	"skablog/internal/theme"
)

// readerModel 是阅读视图：顶部进度行 + viewport 滚动渲染后的正文。
type readerModel struct {
	article *content.Article // 当前阅读的文章；nil 表示未进入阅读
	vp      viewport.Model   // 正文滚动视口（bubbles，KeyMap 自带但不走它的键处理）
}

// open 进入阅读模式：渲染文章正文并装入 viewport（按主列宽折行）。
func (r *readerModel) open(art content.Article, w, h int) {
	r.article = &art
	r.resize(w, h)
}

// resize 按主列宽重设 viewport 尺寸（h 已扣除顶部标题行），
// 并用当前主列宽重渲染正文（长行折行，避免被视口直接截断）。
func (r *readerModel) resize(w, h int) {
	if h < 1 {
		h = 1
	}
	r.vp = viewport.New(w, h)
	if r.article != nil {
		r.vp.SetContent(render.Render("# "+r.article.Title+"\n\n"+r.article.Body, w))
	}
}

// progress 返回阅读进度百分比（0~100 取整）。
func (r readerModel) progress() int {
	return int(r.vp.ScrollPercent() * 100) //nolint:mnd // ScrollPercent 本身就是 0~1
}

// view 渲染：顶部 "标题 (N%)" 一行 + viewport 正文。
func (r readerModel) view(w, h int, p theme.Palette) string {
	if r.article == nil {
		return ""
	}
	title := lipgloss.NewStyle().Foreground(p.Primary).Bold(true).
		Width(w).Render(fmt.Sprintf("%s (%d%%)", r.article.Title, r.progress()))
	return title + "\n" + r.vp.View()
}

// keys 处理阅读模式键位，返回是否退出 TUI（q 键）。
// 滚动只保留方向键：↑/k 上滚，↓/j 下滚；PgUp/PgDn/Space/Home/End/g/G
// 等一律不响应（交互纠偏，见 SPEC 键位速查的裁剪）。
func (r *readerModel) keys(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "down", "j":
		r.vp.LineDown(1)
	case "up", "k":
		r.vp.LineUp(1)
	case "q":
		return true
	}
	return false
}

// firstLine 取正文首个非空非标题行做列表摘要。
func firstLine(body string) string {
	for _, l := range strings.Split(body, "\n") {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		return t
	}
	return ""
}
