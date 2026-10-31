// Package render 把 Markdown 渲染成带 ANSI 颜色的终端字符串。
// 出于 SSH 环境兼容性考虑（见 PLAN.md），这里不引入 glamour 等第三方库，
// 而是自研一个极简渲染器，只处理博客文章中实际出现的语法：
//
//   - 标题："# " ~ "###### "，按级别着色加粗；
//   - 围栏代码块：``` / ~~~ 包裹，整块统一背景色，内部不做行内格式化，
//     并按 CJK 宽度（中文按 2 列）补齐空格，保证背景右边缘对齐；
//   - 粗体：**x** / __x__ 去掉星号（下划线）后用 ANSI 粗体显示；
//   - 链接：[文字](URL) 只保留文字；
//   - 图片：![说明](URL) 渲染为 "[图片: 说明]"（终端无法显示图片）;
//   - 列表：无序列表统一用 "-" 作项目符号，有序列表保留编号，标记着色；
//   - 其他：表格按行铺平（表头加粗、分隔行丢弃），引用加 "> " 前缀，
//     水平线渲染为一串减号，行内代码 `x` 着色显示。
//
// 所有宽度计算均以 rune 为单位，CJK/全角/emoji 字符按 2 列计，避免中文错位。
package render

import (
	"regexp"
	"strings"
)

// ANSI 转义序列常量（仅使用标准 SGR 控制序列，不依赖第三方库）。
const (
	reset = "\x1b[0m" // 复位所有属性
	bold  = "\x1b[1m" // 粗体
	dim   = "\x1b[2m" // 暗淡

	colorH1 = "\x1b[1;95m" // 一级标题：加粗亮洋红
	colorH2 = "\x1b[1;96m" // 二级标题：加粗亮青
	colorH3 = "\x1b[1;93m" // 三级标题：加粗亮黄
	colorH4 = "\x1b[1;90m" // 四级及以下：加粗亮灰

	colorList  = "\x1b[36m" // 列表标记：青色
	colorCode  = "\x1b[33m" // 行内代码：黄色
	colorQuote = "\x1b[2m"  // 引用前缀：暗淡
	// 代码块整行样式：前景亮白 + 背景 256 色深灰
	styleCode = "\x1b[97;48;5;236m"
)

// 各类 Markdown 语法的正则。注意顺序敏感的行内替换规则见 inline() 的注释。
var (
	reFence      = regexp.MustCompile("^\\s*(```|~~~)")                         // 围栏代码块的开始/结束
	reHeading    = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)                      // 标题
	reListItem   = regexp.MustCompile(`^(\s*)([-*+]|\d{1,3}[.)])\s*(.*)$`)      // 列表项，容忍 "1.我的" 这类无空格脏格式
	reQuote      = regexp.MustCompile(`^\s*>\s?(.*)$`)                          // 引用
	reLinkImg    = regexp.MustCompile(`\[!\[([^\]]*)\]\([^)]*\)\s*\]\([^)]*\)`) // [![alt](图片)](链接) 徽章写法
	reImage      = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)                 // 图片
	reLink       = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)                  // 链接
	reBold       = regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)            // 粗体
	reItalic     = regexp.MustCompile(`\*([^*\s][^*]*)\*`)                      // 斜体（仅处理星号，避免误伤 snake_case）
	reInlineCode = regexp.MustCompile("`([^`]+)`")                              // 行内代码
)

// Render 把一段 Markdown 文本渲染为带 ANSI 颜色的字符串。
// 返回值不带末尾换行符，由调用方决定如何输出。
func Render(src string) string {
	lines := splitLines(src)
	var (
		out       []string // 输出行
		code      []string // 当前代码块缓存的内容行
		fenceMark string   // 未闭合围栏的标记（"" 表示不在代码块中）
	)

	// flushCode 把缓存的代码块渲染成带背景色的行并追加到输出
	flushCode := func() {
		out = append(out, renderCodeBlock(code)...)
		code = code[:0]
		fenceMark = ""
	}

	for i := 0; i < len(lines); {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 代码块内部：遇到配对围栏则结束，否则原样缓存（保留缩进，不做行内格式化）
		if fenceMark != "" {
			if strings.HasPrefix(trimmed, fenceMark) {
				flushCode()
			} else {
				code = append(code, line)
			}
			i++
			continue
		}

		// 围栏开始（``` 或 ~~~，可带语言标注）
		if m := reFence.FindStringSubmatch(line); m != nil {
			fenceMark = m[1]
			i++
			continue
		}

		// 标题
		if m := reHeading.FindStringSubmatch(line); m != nil {
			out = append(out, renderHeading(len(m[1]), strings.TrimSpace(m[2])))
			i++
			continue
		}

		// 表格：连续以 "|" 开头的行整体收集处理
		if strings.HasPrefix(trimmed, "|") {
			j := i
			for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "|") {
				j++
			}
			out = append(out, renderTable(lines[i:j])...)
			i = j
			continue
		}

		// 水平线（要先于列表判断，否则 "---" 会被当成列表项）
		if isHorizontalRule(line) {
			out = append(out, dim+strings.Repeat("-", 32)+reset)
			i++
			continue
		}

		// 以 "**" 开头的行是粗体段落而不是列表，先排除再匹配列表
		if strings.HasPrefix(trimmed, "**") {
			out = append(out, inline(line))
			i++
			continue
		}

		// 列表项
		if m := reListItem.FindStringSubmatch(line); m != nil {
			out = append(out, renderListItem(m[1], m[2], m[3]))
			i++
			continue
		}

		// 引用
		if m := reQuote.FindStringSubmatch(line); m != nil {
			out = append(out, colorQuote+"> "+reset+inline(strings.TrimSpace(m[1])))
			i++
			continue
		}

		// 普通段落行：只做行内格式化，不做重排（保证 CJK 不被错误换行）
		out = append(out, inline(line))
		i++
	}

	// 未闭合的围栏兜底输出，避免内容丢失
	if fenceMark != "" {
		flushCode()
	}

	// 去掉末尾多余的空行
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// splitLines 把文本按行切分，统一处理 CRLF 与孤立的 CR。
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

// renderHeading 按标题级别选择颜色渲染标题文字。
func renderHeading(level int, text string) string {
	var color string
	switch level {
	case 1:
		color = colorH1
	case 2:
		color = colorH2
	case 3:
		color = colorH3
	default:
		color = colorH4
	}
	return color + inline(text) + reset
}

// renderListItem 渲染一个列表项：标记着色，无序列表统一为 "-" 项目符号
// （避免使用 East Asian Ambiguous 宽度的 "•" 等 Unicode 符号导致中文终端错位）。
// indent 保留原始缩进以支持简单的嵌套层级。
func renderListItem(indent, marker, text string) string {
	m := marker
	if marker == "*" || marker == "+" {
		m = "-"
	}
	sep := " "
	if text == "" {
		sep = ""
	}
	return indent + colorList + m + sep + reset + inline(text)
}

// renderTable 渲染表格：丢弃 |---| 分隔行，每行铺平为 "单元格 | 单元格"，
// 第一行（表头）加粗。单元格内的行内语法照常处理。
func renderTable(rows []string) []string {
	out := make([]string, 0, len(rows))
	header := true
	for _, row := range rows {
		if isTableSeparator(row) {
			continue // |---|---| 对齐行直接丢弃
		}
		cells := splitTableRow(row)
		for i := range cells {
			cells[i] = inline(cells[i])
		}
		line := strings.Join(cells, " | ")
		if header {
			line = bold + line + reset
			header = false
		}
		out = append(out, line)
	}
	return out
}

// splitTableRow 把 "| a | b |" 切成 ["a", "b"]。
func splitTableRow(row string) []string {
	t := strings.TrimSpace(row)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	parts := strings.Split(t, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// isTableSeparator 判断是否为表格对齐行（只含 |、-、: 和空格且至少有一个 -）。
func isTableSeparator(line string) bool {
	if !strings.Contains(line, "-") {
		return false
	}
	rest := strings.Map(func(r rune) rune {
		switch r {
		case '|', '-', ':', ' ':
			return -1 // 删除这些字符
		}
		return r
	}, line)
	return rest == ""
}

// isHorizontalRule 判断水平线：整行（去掉空白）只由同一种符号（-、*、_）
// 组成且至少出现 3 次，如 "---"、"* * *"、"_____"。
// 不用正则实现，因为 Go 的 RE2 不支持反向引用。
func isHorizontalRule(line string) bool {
	count := 0
	var mark rune
	for _, r := range line {
		if r == ' ' || r == '\t' {
			continue
		}
		if r != '-' && r != '*' && r != '_' {
			return false
		}
		if mark == 0 {
			mark = r
		} else if r != mark {
			return false
		}
		count++
	}
	return count >= 3
}

// renderCodeBlock 渲染围栏代码块：每行套上统一的背景色，
// 并按显示宽度（CJK 按 2 列）补齐空格，使整块背景是一个对齐的矩形。
// 首尾各加一行空白背景行，视觉上与正文隔开。
func renderCodeBlock(lines []string) []string {
	// 内容最大显示宽度；至少为 1，保证空代码块也能渲染出背景条
	maxw := 1
	for _, l := range lines {
		if w := DisplayWidth(l); w > maxw {
			maxw = w
		}
	}
	pad := strings.Repeat(" ", maxw+2)
	out := make([]string, 0, len(lines)+2)
	out = append(out, styleCode+pad+reset)
	for _, l := range lines {
		// 行内容是原始文本（无 ANSI 码），宽度计算才准确
		fill := strings.Repeat(" ", maxw-DisplayWidth(l)+1)
		out = append(out, styleCode+" "+l+fill+reset)
	}
	out = append(out, styleCode+pad+reset)
	return out
}

// inline 处理一行文本中的行内 Markdown 语法。替换顺序经过精心安排：
//  1. 行内代码最先处理（内容保持原样，仅着色）；
//  2. "链接包裹图片"的徽章写法要先于普通链接/图片，否则嵌套方括号会残留；
//  3. 图片要先于链接，否则 "![a](u)" 中的 "[a](u)" 会被当成普通链接；
//  4. 粗体先于斜体，保证 "***x***" 也能退化为粗体；
//  5. 最后清理未配对残留的 "**"（示例数据 wj4 存在这种脏格式）。
func inline(s string) string {
	s = reInlineCode.ReplaceAllString(s, colorCode+"$1"+reset)
	s = reLinkImg.ReplaceAllString(s, "[图片: $1]")
	s = reImage.ReplaceAllString(s, "[图片: $1]")
	s = reLink.ReplaceAllString(s, "$1") // 链接只保留文字
	s = replaceBold(s)
	s = reItalic.ReplaceAllString(s, "$1") // 斜体去星号，不加样式
	s = strings.ReplaceAll(s, "**", "")    // 脏格式兜底：清除未配对的粗体星号
	return s
}

// replaceBold 把 **x** / __x__ 替换为 ANSI 粗体（星号已去除）。
func replaceBold(s string) string {
	return reBold.ReplaceAllStringFunc(s, func(m string) string {
		if t := reBold.FindStringSubmatch(m); t != nil {
			inner := t[1]
			if inner == "" {
				inner = t[2]
			}
			return bold + inner + reset
		}
		return m
	})
}

// RuneWidth 返回单个字符在等宽终端中占用的列数。
// 规则：控制字符与组合字符记 0 列；CJK/全角/emoji 等"宽字符"记 2 列；其余记 1 列。
// 本函数逐 rune 判断，不依赖任何第三方宽度库。
func RuneWidth(r rune) int {
	switch {
	case r == 0,
		r < 32,                // C0 控制字符
		r >= 0x7f && r < 0xa0: // C1 控制字符与 DEL
		return 0
	case isCombining(r):
		return 0
	case isWide(r):
		return 2
	default:
		return 1
	}
}

// isCombining 判断是否为零宽度字符（组合附加符号、零宽字符、变体选择符等）。
func isCombining(r rune) bool {
	switch {
	case r >= 0x0300 && r <= 0x036f, // 组合附加符号
		r >= 0x200b && r <= 0x200f, // 零宽空格等
		r >= 0xfe00 && r <= 0xfe0f, // 变体选择符
		r >= 0x20d0 && r <= 0x20ff: // 组合符号附加
		return true
	}
	return false
}

// isWide 判断是否为东亚宽字符（East Asian Wide/Fullwidth），按 2 列显示。
func isWide(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115f, // 谚文字母
		r >= 0x2e80 && r <= 0x303e,   // CJK 部首、符号、标点
		r >= 0x3041 && r <= 0x33ff,   // 假名、注音、CJK 兼容字符
		r >= 0x3400 && r <= 0x4dbf,   // CJK 扩展 A
		r >= 0x4e00 && r <= 0x9fff,   // CJK 基本区（常用汉字）
		r >= 0xa000 && r <= 0xa4cf,   // 彝文
		r >= 0xa960 && r <= 0xa97f,   // 谚文扩展 A
		r >= 0xac00 && r <= 0xd7a3,   // 谚文音节
		r >= 0xf900 && r <= 0xfaff,   // CJK 兼容表意文字
		r >= 0xfe10 && r <= 0xfe19,   // 竖排形式
		r >= 0xfe30 && r <= 0xfe6f,   // CJK 兼容形式
		r >= 0xff00 && r <= 0xff60,   // 全角 ASCII
		r >= 0xffe0 && r <= 0xffe6,   // 全角符号（￥ 等）
		r >= 0x1f300 && r <= 0x1f64f, // 常用 emoji（如 🔗🍺）
		r >= 0x1f900 && r <= 0x1f9ff, // emoji 补充符号
		r >= 0x20000 && r <= 0x3fffd: // CJK 扩展 B 及以后
		return true
	}
	return false
}

// DisplayWidth 计算字符串在终端中占用的列数：逐 rune 累加宽度，
// 并跳过 ANSI 转义序列（\x1b[...m），因此可以直接测量已着色的文本。
func DisplayWidth(s string) int {
	width, inEscape := 0, false
	for _, r := range s {
		switch {
		case inEscape:
			if r == 'm' { // SGR 序列以 m 结束
				inEscape = false
			}
		case r == 0x1b:
			inEscape = true
		default:
			width += RuneWidth(r)
		}
	}
	return width
}
