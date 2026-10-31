// list.go —— 列表模式：卡片式文章列表 + 自实现可见窗口滚动跟随。
//
// 卡片固定 3 个逻辑行（2 行内容 + 1 行空行分隔）：
//
//	第 1 行：▸ [id] 标题   （选中 ▸ + 加粗 + 主色侧边条；未选中灰色侧边条）
//	第 2 行：摘要… · N段·约M分钟
//
// 移动 focusedIndex 时调整 offset 保证选中卡片整块可见。
package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"skablog/internal/content"
	"skablog/internal/render"
	"skablog/internal/theme"
)

// cardBlocK 高度（物理行数）：标题行 + 摘要行 + 空行。
const cardBlock = 3

// reANSI 用于把渲染后的行还原为纯文本（摘要不做着色，只取"去标记"文本）。
var reANSI = regexp.MustCompile("\x1b\\[[0-9;]*m")

// stripANSI 去掉字符串中的 SGR 颜色序列。
func stripANSI(s string) string { return reANSI.ReplaceAllString(s, "") }

// listModel 是列表模式的数据与视口状态。
type listModel struct {
	focused int // 当前聚焦的文章下标
	offset  int // 可见窗口顶部对应的行偏移（物理行）
	width   int // 主列宽度
}

// resize 记录主列宽度（高度直接影响窗口行数，由渲染时计算）。
func (l *listModel) resize(w int) { l.width = w }

// move 移动聚焦并执行滚动跟随。
func (l *listModel) move(arts []content.Article, d int, rows int) {
	if len(arts) == 0 {
		return
	}
	l.focused += d
	if l.focused < 0 {
		l.focused = 0
	}
	if l.focused >= len(arts) {
		l.focused = len(arts) - 1
	}
	l.follow(len(arts), rows)
}

// follow 调整 offset：保证 focused 卡片整块在窗口内
// （向上一格滑出窗口顶 → 窗口上移；向下滑出窗口底 → 窗口下移）。
func (l *listModel) follow(n, rows int) {
	if rows < 1 {
		rows = 1
	}
	start := l.focused * cardBlock
	if start < l.offset {
		l.offset = start
	}
	if end := start + cardBlock; end > l.offset+rows {
		l.offset = end - rows
	}
	// 兜底：文章减少时窗口不超过末尾
	if max := n*cardBlock - rows; l.offset > max {
		if max < 0 {
			l.offset = 0
		} else {
			l.offset = max
		}
	}
	if l.offset < 0 {
		l.offset = 0
	}
}

// view 渲染可见窗口内的卡片行，窗口填不满时用空行补足。
func (l listModel) view(arts []content.Article, rows int, p theme.Palette) string {
	if len(arts) == 0 {
		return ""
	}
	var all []string
	for i, a := range arts {
		all = append(all, blockLines(a, i == l.focused, l.width, p)...)
	}
	if l.offset >= len(all) {
		l.offset = 0 // 不变量兜底
	}
	end := l.offset + rows
	if end > len(all) {
		end = len(all)
	}
	lines := all[l.offset:end]
	// 补足空行使高度恒为 rows（插入在窗口头部，卡片自然沉到底部）
	for len(lines) < rows {
		lines = append([]string{strings.Repeat(" ", l.width)}, lines...)
	}
	return strings.Join(lines[:rows], "\n")
}

// blockLines 渲染一个卡片的 3 行（含尾部空行）。
func blockLines(a content.Article, focused bool, w int, p theme.Palette) (out []string) {
	var (
		titleS     = lipgloss.NewStyle().Width(w)
		summaryS   = lipgloss.NewStyle().Width(w)
		idS        = lipgloss.NewStyle().Foreground(p.Primary)
		titleTextS = lipgloss.NewStyle()
	)
	if focused {
		// 选中：主色边框 + ▸ 前缀 + 标题加粗
		titleS = titleS.BorderForeground(p.Primary).BorderLeft(true).Bold(true)
		summaryS = summaryS.BorderForeground(p.Primary).BorderLeft(true)
	} else {
		// 未选中：灰色边框
		titleS = titleS.BorderForeground(p.TextMuted).BorderLeft(true)
		summaryS = summaryS.BorderForeground(p.TextMuted).BorderLeft(true)
	}

	prefix := "  "
	if focused {
		prefix = "▸ "
	}
	id := idS.Render("[" + a.ID + "]")
	title := titleTextS.Render(a.Title)
	titleLine := titleS.Render(prefix + id + " " + title)

	// 摘要：渲染去标记 → 纯文本 → 按可用宽度截断加 …
	avail := w - 1 // 侧边条占 1 列
	meta := fmt.Sprintf(" · %d段·约%d分钟", a.ParagraphCount(), minutes(a.WordCount))
	summary := plain(firstLine(a.Body))
	budget := avail - render.DisplayWidth(meta)
	summary = truncateVisible(summary, budget-1) + metaRender(meta)
	summaryLine := summaryS.Render(summary)

	out = append(out,
		strings.TrimRight(titleLine, " "),
		strings.TrimRight(summaryLine, " "),
		strings.Repeat(" ", w), // 空行分隔
	)
	return out
}

// metaRender 摘要尾部统计信息，暗淡色。
func metaRender(meta string) string {
	return lipgloss.NewStyle().Faint(true).Render(meta)
}

// plain 把一行 Markdown 渲染后去掉颜色码，得到"去标记"纯文本。
func plain(line string) string { return stripANSI(render.Render(line)) }

// minutes 估读时长：ceil(WordCount/400)，至少 1 分钟。
func minutes(wordCount int) int {
	m := (wordCount + 399) / 400 //nolint:mnd // 400 字/分钟，向上取整
	if m < 1 {
		m = 1
	}
	return m
}

// truncateVisible 按显示宽度截断（CJK 记 2 列），截断处补 "…"；
// netWidth 是内容可用列数（不含省略号本身时传 netWidth-1）。
func truncateVisible(s string, netWidth int) string {
	if netWidth < 0 {
		netWidth = 0
	}
	if render.DisplayWidth(s) <= netWidth {
		return s
	}
	w, i := 0, 0
	runes := []rune(s)
	for i < len(runes) {
		cw := render.RuneWidth(runes[i])
		if w+cw > netWidth-1 { // 给 "…" 留 1 列
			break
		}
		w += cw
		i++
	}
	return string(runes[:i]) + "…"
}
