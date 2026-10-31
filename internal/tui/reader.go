// Package tui —— TUI 核心：根 model 状态机 + 卡片列表 + 正文阅读 + 侧栏。
// 布局对齐 ska-site-tui 参考实现：根边框（左上角标题/底部键位提示条）、
// 左主列（列表/正文）+ 右侧 30% STATUS 侧栏。
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"skablog/internal/content"
	"skablog/internal/render"
	"skablog/internal/theme"
)

// 滚轮加速参数。语义对齐参考实现 src/lib/scroll-accel.ts 的 fastScroll：
// 连续滚轮事件合并加速，超时回退到 1 行/次。
const (
	// accelBeat 是加速序列的一"拍"：加速期内每经过一拍，增量翻倍。
	accelBeat = 50 * time.Millisecond
	// accelWindow 是"连续"判定窗口：间隔 <300ms 算连续滚轮，超过则超时重置。
	accelWindow = 300 * time.Millisecond
	// accelMax 是加速倍率上限。
	accelMax = 16
)

// accel 是纯函数：根据上一次滚轮事件时间 lastTick 与本次事件时间 now，
// 返回本次滚动的行数。
//
// 语义（1→2→4→8→16 封顶、超时重置 1）：
//   - 首次事件（lastTick 零值）或间隔 ≥ accelWindow：重置为 1；
//   - 连续区间内按 elapsed 打拍：每过一拍 accelBeat 增量翻倍
//     （拍数 k=区间/accelBeat，倍率=1<<k，超过 accelMax 封顶 16）。
//
// 注意：函数签名只有两个时间参数、装不下"已连滚几次"的历史，
// 因此用"间隔时长分拍"近似"每事件翻倍"——终端滚轮事件的到达节奏
// 基本恒定（约 50~60ms/次），效果与逐事件翻倍一致：
// 60ms 连滚一次序列为 1 → 2 → 4 → 8 → 16 → 16，停手 300ms 后回到 1。
func accel(lastTick time.Time, now time.Time) int {
	if lastTick.IsZero() {
		return 1
	}
	d := now.Sub(lastTick)
	if d < 0 || d >= accelWindow {
		return 1 // 超时（或时间倒流）重置
	}
	k := int(d / accelBeat)
	m := 1 << uint(k) //nolint:gosec // k 已由窗口上界限制，不会溢出
	if m > accelMax {
		m = accelMax
	}
	return m
}

// readerModel 是阅读视图：顶部进度行 + viewport 滚动渲染后的正文。
type readerModel struct {
	article  *content.Article // 当前阅读的文章；nil 表示未进入阅读
	vp       viewport.Model   // 正文滚动视口（bubbles，KeyMap 自带但不走它的键处理）
	lastTick time.Time        // 上一次滚轮事件时间，供 accel 计算
}

// open 进入阅读模式：渲染文章正文并装入 viewport。
func (r *readerModel) open(art content.Article, w, h int) {
	r.article = &art
	r.resize(w, h)
	// 标题按一级标题着色渲染，正文原样交给渲染器（与 main.go -render 一致）
	r.vp.SetContent(render.Render("# " + art.Title + "\n\n" + art.Body))
}

// resize 按主列宽重设 viewport 尺寸（h 已扣除顶部标题行）。
func (r *readerModel) resize(w, h int) {
	if h < 1 {
		h = 1
	}
	r.vp = viewport.New(w, h)
	if r.article != nil {
		r.vp.SetContent(render.Render("# " + r.article.Title + "\n\n" + r.article.Body))
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

// wheel 处理滚轮：按 accel 加速的行数上下滚动。
func (r *readerModel) wheel(now time.Time, up bool) {
	if r.article == nil {
		return
	}
	n := accel(r.lastTick, now)
	r.lastTick = now
	if up {
		r.vp.LineUp(n)
	} else {
		r.vp.LineDown(n)
	}
}

// keys 处理阅读模式键位，返回是否退出 TUI（q 键）。
func (r *readerModel) keys(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "pgup", "b":
		r.vp.PageUp()
	case "pgdown", " ":
		r.vp.PageDown()
	case "j":
		r.vp.LineDown(1)
	case "k":
		r.vp.LineUp(1)
	case "home", "g":
		r.vp.GotoTop()
	case "end", "G":
		r.vp.GotoBottom()
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
