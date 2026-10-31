// Package tui —— TUI 核心：根 model 状态机 + glow 式行列表 + 正文阅读 + 侧栏。
// 布局对齐 soft-serve/glow 的 Charm house style：无整周包边，header+分割线+
// 两栏（主列 + 侧栏），列表为无边框行列表，阅读正文用 glamour 渲染。
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"skablog/internal/content"
	"skablog/internal/render"
	"skablog/internal/theme"
)

// readerModel 是阅读视图：顶部进度行 + viewport 滚动渲染后的正文。
// 正文用 glamour 渲染（标题层级/代码块背景），失败时回退 internal/render。
type readerModel struct {
	article  *content.Article // 当前阅读的文章；nil 表示未进入阅读
	vp       viewport.Model   // 正文滚动视口（bubbles，KeyMap 自带但不走它的键处理）
	fallback bool             // 当前帧是否走 internal/render 兜底（调试/测试断言）
}

// open 进入阅读模式：按主列宽渲染文章正文并装入 viewport。
func (r *readerModel) open(art content.Article, w, h int, p theme.Palette) {
	r.article = &art
	r.resize(w, h, p)
}

// resize 按主列宽重设 viewport 尺寸（h 已扣除顶部进度行），
// 并用当前主列宽+主题重渲染正文（glamour 折行，失败回退 internal/render）。
func (r *readerModel) resize(w, h int, p theme.Palette) {
	if h < 1 {
		h = 1
	}
	if w < 1 {
		w = 1
	}
	r.vp = viewport.New(w, h)
	if r.article != nil {
		md := "# " + r.article.Title + "\n\n" + r.article.Body
		r.vp.SetContent(r.renderMarkdown(md, w, p))
	}
}

// renderMarkdown 渲染整篇 markdown：优先 glamour，失败回退 internal/render。
func (r *readerModel) renderMarkdown(md string, w int, p theme.Palette) string {
	if out, err := glamourRender(md, w, p); err == nil {
		r.fallback = false
		return out
	}
	r.fallback = true
	return render.Render(md, w)
}

// glamourRender 是 glamour 渲染入口，独立成包级变量以便单测兜底路径。
//
// 说明：glamour 底层用 goldmark，对非法 markdown 输入几乎不报错（容错），
// 错误面主要来自选项构造（如非法样式名）。因此测试通过注入一个必然失败的
// glamourRender 来触发 internal/render 兜底，并断言 fallback 生效。
var glamourRender = func(md string, w int, p theme.Palette) (string, error) {
	tr, err := glamour.NewTermRenderer(
		glamour.WithStyles(glamourStyle(p)),         // 按当前 Palette 映射 dark 风格
		glamour.WithWordWrap(w),                     // 按主列宽折行
		glamour.WithColorProfile(termenv.TrueColor), // 与 lipgloss 真彩一致
	)
	if err != nil {
		return "", err
	}
	return tr.Render(md)
}

// glamourStyle 把当前 Palette 映射成 glamour 的 ansi.StyleConfig：
// 以内置深色样式（styles.DarkStyleConfig）为基，标题/链接用主题主色，
// 正文用主题正文色，代码块背景沿用 dark 默认（有背景）。
func glamourStyle(p theme.Palette) ansi.StyleConfig {
	cfg := styles.DarkStyleConfig
	str := func(c lipgloss.Color) *string { s := string(c); return &s }
	cfg.Document.StylePrimitive.Color = str(p.Text)
	for _, h := range []*ansi.StyleBlock{&cfg.H1, &cfg.H2, &cfg.H3, &cfg.H4, &cfg.H5, &cfg.H6} {
		h.StylePrimitive.Color = str(p.Primary)
	}
	cfg.Link.Color = str(p.Primary)
	cfg.LinkText.Color = str(p.Primary)
	return cfg
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
