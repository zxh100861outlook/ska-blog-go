package render

import (
	"strings"
	"testing"
)

// TestRenderHeadings 验证各级标题着色且去掉 # 记号。
func TestRenderHeadings(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"# 一级标题", colorH1 + "一级标题" + reset},
		{"## 二级标题", colorH2 + "二级标题" + reset},
		{"### 三级标题", colorH3 + "三级标题" + reset},
		{"#### 四级标题", colorH4 + "四级标题" + reset},
	}
	for _, c := range cases {
		if got := Render(c.in); got != c.want {
			t.Errorf("Render(%q) = %q, 期望 %q", c.in, got, c.want)
		}
	}
}

// TestRenderCodeBlock 验证代码块背景色与按 CJK 宽度补齐空格。
// "ab" 占 2 列，"中文字" 占 6 列，最大宽度 6，因此每行内容连同前后空格共 8 列。
func TestRenderCodeBlock(t *testing.T) {
	in := "```bash\nab\n中文字\n```"
	got := Render(in)
	want := strings.Join([]string{
		styleCode + strings.Repeat(" ", 8) + reset, // 顶部留白行
		styleCode + " ab" + strings.Repeat(" ", 5) + reset,
		styleCode + " 中文字 " + reset,                // 中文按 2 列计，只补 1 个空格
		styleCode + strings.Repeat(" ", 8) + reset, // 底部留白行
	}, "\n")
	if got != want {
		t.Errorf("Render 代码块 =\n%q\n期望\n%q", got, want)
	}
}

// TestRenderCodeBlockKeepsRawText 验证代码块内部不做行内格式化。
func TestRenderCodeBlockKeepsRawText(t *testing.T) {
	got := Render("```\n**x** [a](u)\n```")
	if !strings.Contains(got, "**x** [a](u)") {
		t.Errorf("代码块内容应保持原样, 得到 %q", got)
	}
	if strings.Contains(got, "\x1b[1m") {
		t.Errorf("代码块内部不应出现粗体样式: %q", got)
	}
}

// TestRenderUnclosedFence 验证未闭合围栏也能兜底输出内容。
func TestRenderUnclosedFence(t *testing.T) {
	got := Render("```\ncode line")
	if !strings.Contains(got, "code line") || !strings.Contains(got, styleCode) {
		t.Errorf("未闭合代码块应带背景输出, 得到 %q", got)
	}
}

// TestRenderBold 验证粗体去星号并加粗。
func TestRenderBold(t *testing.T) {
	for _, in := range []string{"这是 **加粗** 文字", "这是 __加粗__ 文字"} {
		got := Render(in)
		want := "这是 " + bold + "加粗" + reset + " 文字"
		if got != want {
			t.Errorf("Render(%q) = %q, 期望 %q", in, got, want)
		}
	}
	// 脏格式：未配对的 ** 也应被清除
	if got := Render("结尾有多余**"); strings.Contains(got, "**") {
		t.Errorf("未配对的 ** 应被清除, 得到 %q", got)
	}
}

// TestRenderItalic 验证单星号斜体去星号（不加样式）。
func TestRenderItalic(t *testing.T) {
	if got := Render("a *斜体* b"); got != "a 斜体 b" {
		t.Errorf("Render = %q, 期望 %q", got, "a 斜体 b")
	}
	// snake_case 中的下划线不应被当作斜体删除
	if got := Render("变量 some_var_name 保持原样"); !strings.Contains(got, "some_var_name") {
		t.Errorf("下划线标识符应保持原样, 得到 %q", got)
	}
}

// TestRenderLink 验证链接只保留文字。
func TestRenderLink(t *testing.T) {
	got := Render("去 [谷歌](https://google.com) 搜索")
	if got != "去 谷歌 搜索" {
		t.Errorf("Render = %q, 期望 %q", got, "去 谷歌 搜索")
	}
	if strings.Contains(got, "google.com") || strings.Contains(got, "](") {
		t.Errorf("链接 URL 不应保留: %q", got)
	}
}

// TestRenderImage 验证图片渲染为 "[图片: 说明]"。
func TestRenderImage(t *testing.T) {
	got := Render("看这个 ![截图](./images/a.png)")
	if got != "看这个 [图片: 截图]" {
		t.Errorf("Render = %q, 期望 %q", got, "看这个 [图片: 截图]")
	}
}

// TestRenderBadge 验证 [![alt](图片)](链接) 徽章写法正确退化。
func TestRenderBadge(t *testing.T) {
	got := Render("[![alt 文本](https://img.shields.io/x.svg)](https://example.com)")
	if got != "[图片: alt 文本]" {
		t.Errorf("Render = %q, 期望 %q", got, "[图片: alt 文本]")
	}
}

// TestRenderInlineCode 验证行内代码着色。
func TestRenderInlineCode(t *testing.T) {
	got := Render("运行 `git config` 即可")
	want := "运行 " + colorCode + "git config" + reset + " 即可"
	if got != want {
		t.Errorf("Render = %q, 期望 %q", got, want)
	}
}

// TestRenderLists 验证无序/有序列表渲染，含 "1.我的" 无空格脏格式。
func TestRenderLists(t *testing.T) {
	got := Render("* 甲\n- 乙\n+ 丙")
	want := strings.Join([]string{
		colorList + "- " + reset + "甲",
		colorList + "- " + reset + "乙",
		colorList + "- " + reset + "丙",
	}, "\n")
	if got != want {
		t.Errorf("无序列表 = %q, 期望 %q", got, want)
	}

	got = Render("1. 有编号\n2)第二项")
	want = strings.Join([]string{
		colorList + "1. " + reset + "有编号",
		colorList + "2) " + reset + "第二项",
	}, "\n")
	if got != want {
		t.Errorf("有序列表 = %q, 期望 %q", got, want)
	}

	// 列表项内的行内语法仍然生效
	got = Render("- **粗体** 项")
	if got != colorList+"- "+reset+bold+"粗体"+reset+" 项" {
		t.Errorf("列表项内粗体 = %q", got)
	}
}

// TestRenderTable 验证表格铺平：表头加粗、分隔行丢弃。
func TestRenderTable(t *testing.T) {
	in := "| 平台 | 地址 |\n|------|------|\n| A | https://a.com |"
	got := Render(in)
	want := bold + "平台 | 地址" + reset + "\nA | https://a.com"
	if got != want {
		t.Errorf("表格 = %q, 期望 %q", got, want)
	}
}

// TestRenderQuoteAndHR 验证引用前缀与水平线。
func TestRenderQuoteAndHR(t *testing.T) {
	if got := Render("> 引用内容"); got != colorQuote+"> "+reset+"引用内容" {
		t.Errorf("引用 = %q", got)
	}
	if got := Render("---"); got != dim+strings.Repeat("-", 32)+reset {
		t.Errorf("水平线 = %q", got)
	}
}

// TestRenderParagraphPassthrough 验证普通段落原样保留（不做重排，CJK 安全）。
func TestRenderParagraphPassthrough(t *testing.T) {
	in := "在平日里在github上看到好玩的项目想clone下来却发现clone速度慢到离谱！"
	if got := Render(in); got != in {
		t.Errorf("普通段落被改动: %q", got)
	}
}

// TestRenderMixedDoc 综合冒烟测试：各类语法混合时不残留 Markdown 记号。
func TestRenderMixedDoc(t *testing.T) {
	in := "# 总标题\n" +
		"\n" +
		"段落 **粗体** 与 [链接](https://x) 与 ![图](y.png)。\n" +
		"\n" +
		"- 甲\n" +
		"- 乙\n" +
		"\n" +
		"> 引用\n"
	got := Render(in)
	for _, banned := range []string{"**", "](", "![", "#"} {
		if strings.Contains(got, banned) {
			t.Errorf("输出不应残留记号 %q: %q", banned, got)
		}
	}
	for _, want := range []string{colorH1, "粗体", "链接", "[图片: 图]", "甲", "乙", "引用"} {
		if !strings.Contains(got, want) {
			t.Errorf("输出缺少 %q: %q", want, got)
		}
	}
}

// TestRuneWidth 验证单字符宽度判断。
func TestRuneWidth(t *testing.T) {
	cases := []struct {
		r    rune
		want int
	}{
		{'a', 1},
		{' ', 1},
		{'中', 2},      // CJK 基本区
		{'，', 2},      // 全角标点
		{'Ａ', 2},      // 全角字母
		{'가', 2},      // 谚文
		{'🍺', 2},      // emoji
		{'\x1b', 0},   // 控制字符
		{'\u0301', 0}, // 组合附加符号
		{'\u200b', 0}, // 零宽空格
	}
	for _, c := range cases {
		if got := RuneWidth(c.r); got != c.want {
			t.Errorf("RuneWidth(%q) = %d, 期望 %d", c.r, got, c.want)
		}
	}
}

// TestDisplayWidth 验证混合文本宽度与 ANSI 序列跳过。
func TestDisplayWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"中文", 4},
		{"中文ab", 6},            // 2+2+1+1
		{"a中b文", 6},            // 1+2+1+2
		{"\x1b[1m粗\x1b[0m", 2}, // ANSI 序列不计入宽度
		{bold + "中" + reset + "ab", 4},
	}
	for _, c := range cases {
		if got := DisplayWidth(c.in); got != c.want {
			t.Errorf("DisplayWidth(%q) = %d, 期望 %d", c.in, got, c.want)
		}
	}
}
