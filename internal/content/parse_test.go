package content

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// sample 覆盖了解析器的全部关键场景：
// 博客总标题、Tab 脏格式、多空格、无 ID 自动编号、三级标题归属正文、ID 后跟数字的标题。
const sample = "# 个人博客\n\n这里收录我写的东西。\n\n" +
	"## wj1\thello,world\n\n这是一个开始\n\n" +
	"## wj2  github clone 加速\n\n正文甲\n\n### 子标题\n\n正文乙\n\n" +
	"## 免费域名\n\n免费域名正文\n\n" +
	"## wuji 5 免费 AI API 平台\n\n平台正文\n"

// TestParseBasic 验证基本的文章切分、ID/标题切分与自动编号。
func TestParseBasic(t *testing.T) {
	arts, err := Parse(sample)
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	if len(arts) != 4 {
		t.Fatalf("文章数 = %d, 期望 4", len(arts))
	}
	want := []struct{ id, title string }{
		{"wj1", "hello,world"},
		{"wj2", "github clone 加速"},
		{"a01", "免费域名"},            // 无 ID 前缀 -> 自动编号
		{"wuji", "5 免费 AI API 平台"}, // 按第一个空格切分
	}
	for i, w := range want {
		if arts[i].ID != w.id || arts[i].Title != w.title {
			t.Errorf("第 %d 篇 = (%q, %q), 期望 (%q, %q)",
				i, arts[i].ID, arts[i].Title, w.id, w.title)
		}
	}
}

// TestParseTabDirty 专门验证 "## wj1\thello,world" 这类 Tab 分隔的脏格式。
func TestParseTabDirty(t *testing.T) {
	arts, err := Parse("## wj1\thello,world\n\n正文\n")
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	if arts[0].ID != "wj1" || arts[0].Title != "hello,world" {
		t.Errorf("ID/Title = (%q, %q), 期望 (wj1, hello,world)", arts[0].ID, arts[0].Title)
	}
}

// TestParseAutoIDSequence 验证多篇无 ID 文章按 a01、a02 顺序编号。
func TestParseAutoIDSequence(t *testing.T) {
	arts, err := Parse("## 第一篇\n\n甲\n\n## 第二篇\n\n乙\n")
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	if arts[0].ID != "a01" || arts[1].ID != "a02" {
		t.Errorf("自动编号 = (%q, %q), 期望 (a01, a02)", arts[0].ID, arts[1].ID)
	}
}

// TestParseSkipsBlogTitle 验证一级标题与第一篇文章之前的简介被跳过。
func TestParseSkipsBlogTitle(t *testing.T) {
	arts, err := Parse(sample)
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	for _, a := range arts {
		if strings.Contains(a.Body, "收录") {
			t.Errorf("文章 %q 的正文混入了博客简介", a.ID)
		}
	}
}

// TestParseBodyScope 验证 "### " 三级标题及其后内容都归属当前文章，直到下一个 "## "。
func TestParseBodyScope(t *testing.T) {
	arts, err := Parse(sample)
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	wantBody := "正文甲\n\n### 子标题\n\n正文乙"
	if arts[1].Body != wantBody {
		t.Errorf("wj2 Body = %q, 期望 %q", arts[1].Body, wantBody)
	}
}

// TestParseWordCountAndParagraphs 验证字符数（不含空白）与段落数统计。
func TestParseWordCountAndParagraphs(t *testing.T) {
	arts, err := Parse(sample)
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	// 正文甲(3) + "###子标题"(6) + 正文乙(3) = 12 个非空白字符；3 个段落块
	if got := arts[1].WordCount; got != 12 {
		t.Errorf("wj2 WordCount = %d, 期望 12", got)
	}
	if got := arts[1].ParagraphCount(); got != 3 {
		t.Errorf("wj2 ParagraphCount = %d, 期望 3", got)
	}
	// 空正文的文章统计为 0
	empty, err := Parse("## a1\n\n## a2\n\n有内容\n")
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	if empty[0].Body != "" || empty[0].WordCount != 0 || empty[0].ParagraphCount() != 0 {
		t.Errorf("空正文统计错误: Body=%q WordCount=%d ParagraphCount=%d",
			empty[0].Body, empty[0].WordCount, empty[0].ParagraphCount())
	}
}

// TestParseNoArticles 验证无文章时返回 ErrNoArticles。
func TestParseNoArticles(t *testing.T) {
	for name, input := range map[string]string{
		"空输入":   "",
		"只有总标题": "# 个人博客\n\n这里收录我写的东西。\n",
		"只有正文":  "随便一段文字\n",
	} {
		if _, err := Parse(input); !errors.Is(err, ErrNoArticles) {
			t.Errorf("%s: err = %v, 期望 ErrNoArticles", name, err)
		}
	}
}

// TestParseCRLF 验证 CRLF 换行不会在标题里留下 \r。
func TestParseCRLF(t *testing.T) {
	arts, err := Parse("## t1\t标题一\r\n\r\n第一段\r\n\r\n第二段\r\n")
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	if arts[0].Title != "标题一" {
		t.Errorf("Title = %q, 期望 %q（不应包含 \\r）", arts[0].Title, "标题一")
	}
	if arts[0].Body != "第一段\n\n第二段" {
		t.Errorf("Body = %q, 期望 %q", arts[0].Body, "第一段\n\n第二段")
	}
}

// TestParseIDHeuristic 锁定 ID 判定启发式的行为：纯 ASCII 数字开头也算 ID。
// 这样 "## 2026 年度回顾" 会得到 ID=2026，与 PLAN 的"第一个空格切分"规则一致。
func TestParseIDHeuristic(t *testing.T) {
	arts, err := Parse("## 2026 年度回顾\n\n正文\n")
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	if arts[0].ID != "2026" || arts[0].Title != "年度回顾" {
		t.Errorf("ID/Title = (%q, %q), 期望 (2026, 年度回顾)", arts[0].ID, arts[0].Title)
	}
}

// TestParseHeaderTabAfterHash 验证 "##\t标题"（## 后直接跟 Tab）的脏格式也能识别。
func TestParseHeaderTabAfterHash(t *testing.T) {
	arts, err := Parse("##\tdirty1 标题\n\n正文\n")
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	if arts[0].ID != "dirty1" || arts[0].Title != "标题" {
		t.Errorf("ID/Title = (%q, %q), 期望 (dirty1, 标题)", arts[0].ID, arts[0].Title)
	}
}

// TestFind 验证按 ID 查找文章。
func TestFind(t *testing.T) {
	arts, err := Parse(sample)
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	if a := Find(arts, "wj2"); a == nil || a.Title != "github clone 加速" {
		t.Errorf("Find(wj2) = %+v, 期望找到 wj2", a)
	}
	if a := Find(arts, "不存在"); a != nil {
		t.Errorf("Find(不存在) = %+v, 期望 nil", a)
	}
}

// TestParseRealArticlesFile 用仓库自带的 articles/articles.md 做一次集成验证。
func TestParseRealArticlesFile(t *testing.T) {
	data, err := os.ReadFile("../../articles/articles.md")
	if err != nil {
		t.Skipf("跳过：找不到仓库自带的文章文件: %v", err)
	}
	arts, err := Parse(string(data))
	if err != nil {
		t.Fatalf("Parse 报错: %v", err)
	}
	// 示例数据至少应包含 wj1~wj4 这四篇带 ID 的文章
	for _, id := range []string{"wj1", "wj2", "wj3", "wj4"} {
		a := Find(arts, id)
		if a == nil {
			t.Errorf("示例数据中找不到文章 %q", id)
			continue
		}
		if a.WordCount <= 0 || a.ParagraphCount() <= 0 {
			t.Errorf("文章 %q 统计异常: WordCount=%d ParagraphCount=%d",
				id, a.WordCount, a.ParagraphCount())
		}
	}
	// Tab 脏格式的 wj1 标题应正确切分
	if a := Find(arts, "wj1"); a != nil && a.Title != "hello,world" {
		t.Errorf("wj1 Title = %q, 期望 %q", a.Title, "hello,world")
	}
}
