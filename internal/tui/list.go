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

// blockLines 渲染一篇文章的两行列表内容（glow 式，无边框盒）+ 尾部空行。
//
//	第 1 行（标题行）：{光标}{空格}{小字灰 id} · {标题}
//	  选中：▍ 块状光标 + 整行淡背景色块 + 标题主色加粗
//	  未选：空格占位 + 前景 TextMuted
//	第 2 行（元信息行）：{两格缩进}{id}·{N}段·约{M}分钟 · {摘要}（更暗一档）
//	第 3 行：空行分隔
func blockLines(a content.Article, focused bool, w int, p theme.Palette) []string {
	var (
		bg      lipgloss.Color
		titleFg = p.TextMuted // 未选中标题：前景 TextMuted
		cursor  = " "         // 未选中：空格占位（与选中对齐）
		idFg    = p.Meta      // 小字灰 id
	)
	if focused {
		bg = p.SelBg
		titleFg = p.Primary
		cursor = "▍"
		idFg = p.TextMuted
	}

	// 标题行：先按可用宽度截断标题（纯文本截断后再着色）
	idW := render.DisplayWidth(a.ID)
	prefixW := 1 /*cursor*/ + 1 /*空格*/ + idW + 3 /*" · "*/
	avail := w - prefixW
	if avail < 1 {
		avail = 1
	}
	title := lipgloss.NewStyle().
		Foreground(titleFg).Bold(focused).
		Render(truncateVisible(a.Title, avail))
	line1Text := cursor + " " +
		lipgloss.NewStyle().Foreground(idFg).Render(a.ID) +
		" · " + title

	// 元信息行：id·N段·约M分钟 · 摘要，更暗一档，同缩进两格
	meta := fmt.Sprintf("%s·%d段·约%d分钟 · %s",
		a.ID, a.ParagraphCount(), minutes(a.WordCount), plain(firstLine(a.Body)))
	metaText := truncateVisible(meta, w-2)

	line1 := lipgloss.NewStyle().Width(w).Render(line1Text)
	line2 := lipgloss.NewStyle().Width(w).Render("  " + metaText)
	if focused {
		// 选中：整行淡背景色块（含右侧补齐空格）
		line1 = lipgloss.NewStyle().Background(bg).Width(w).Render(line1Text)
		line2 = lipgloss.NewStyle().Background(bg).Width(w).Render("  " + metaText)
	}

	return []string{
		line1,
		line2,
		strings.Repeat(" ", w), // 空行分隔
	}
}

// plain 把一行 Markdown 渲染后去掉颜色码，得到"去标记"纯文本。
// width=0 表示不折行：摘要后续由 truncateVisible 负责截断。
func plain(line string) string { return stripANSI(render.Render(line, 0)) }

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
