// Package content 负责 articles.md 的解析：把原始 Markdown 文本
// 切分成一篇篇结构化文章，供列表与渲染层使用。
//
// 解析规则（见 PLAN.md）：
//   - 一级标题 "# " 是博客总标题，直接跳过；
//   - 每个二级标题 "## " 开启一篇文章，标题按第一个空格/Tab 切分出 ID 与标题，
//     例如 "## wj1\thello,world"（Tab 脏格式）与 "## wj2  github clone 加速"（多空格）
//     都能正确得到 ID=wj1/wj2；
//   - 若标题开头不是 ID 形式的片段（纯 ASCII 字母/数字/-/_ 组成），
//     则整行视为标题，ID 按 a01/a02... 顺序自动生成；
//   - 三级标题 "### " 及其后的所有内容都属于当前文章的 Body，直到下一个 "## "；
//   - 文件中没有任何文章时返回 ErrNoArticles。
package content

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// ErrNoArticles 表示输入内容中没有解析出任何文章。
var ErrNoArticles = errors.New(`content: 未找到任何文章（没有以 "## " 开头的文章标题）`)

// Article 表示一篇文章。
type Article struct {
	ID        string // 文章唯一标识，如 "wj2"；标题无 ID 前缀时自动生成 "a01" 等序号
	Title     string // 文章标题（不含 ID 部分，已去除首尾空白）
	Body      string // 正文原始 Markdown（不含 "## " 标题行，已去掉首尾空行）
	WordCount int    // 正文字符数（不含空白，按 rune 统计），可用于估算阅读时长
}

// Parse 把 articles.md 的原始内容解析为文章列表。
// 输入中解析不出任何文章时返回 ErrNoArticles。
func Parse(data string) ([]Article, error) {
	var (
		arts      []Article
		cur       *Article // 当前正在收集正文的文章
		bodyLines []string // 当前文章的正文行缓存
		autoSeq   = 0      // 自动生成 ID 的计数器（a01、a02...）
	)

	// flush 把当前文章落盘：拼接正文、统计字数并追加到结果里
	flush := func() {
		if cur == nil {
			return
		}
		cur.Body = strings.Trim(strings.Join(bodyLines, "\n"), "\n")
		cur.WordCount = countChars(cur.Body)
		arts = append(arts, *cur)
		cur = nil
		bodyLines = bodyLines[:0]
	}

	for _, raw := range strings.Split(data, "\n") {
		line := strings.TrimRight(raw, "\r") // 兼容 CRLF 换行
		switch {
		case isArticleHeader(line):
			// 遇到 "## "：先结束上一篇文章，再开启新文章
			flush()
			id, title := splitIDTitle(strings.TrimSpace(line[2:]))
			if id == "" {
				// 标题没有 ID 前缀，按 a01/a02... 顺序自动生成
				autoSeq++
				id = autoID(autoSeq)
			}
			cur = &Article{ID: id, Title: title}
		case isBlogTitle(line):
			// 一级标题是博客总标题，跳过
			continue
		default:
			if cur != nil {
				bodyLines = append(bodyLines, line)
			}
			// 第一篇文章之前的内容（博客简介等）直接丢弃
		}
	}
	flush() // 收尾最后一篇文章

	if len(arts) == 0 {
		return nil, ErrNoArticles
	}
	return arts, nil
}

// Find 按 ID 查找文章，找不到时返回 nil。
func Find(articles []Article, id string) *Article {
	for i := range articles {
		if articles[i].ID == id {
			return &articles[i]
		}
	}
	return nil
}

// ParagraphCount 返回正文段落数：按"连续非空行"为一个段落分块统计。
// 列表的多个连续条目算一个段落，代码块整体也算一个段落。
func (a Article) ParagraphCount() int {
	count, inBlock := 0, false
	for _, raw := range strings.Split(a.Body, "\n") {
		empty := strings.TrimSpace(raw) == ""
		switch {
		case !empty && !inBlock:
			count++
			inBlock = true
		case empty:
			inBlock = false
		}
	}
	return count
}

// isArticleHeader 判断一行是否为文章标题（二级标题）。
// 除标准的 "## " 外，也容忍 "##\t标题" 这类 Tab 脏格式；
// 注意 "### " 以 "#" 开头，不会被误判。
func isArticleHeader(line string) bool {
	if !strings.HasPrefix(line, "##") || len(line) == 2 {
		return false
	}
	return line[2] == ' ' || line[2] == '\t'
}

// isBlogTitle 判断一行是否为博客总标题（一级标题 "# "）。
func isBlogTitle(line string) bool {
	return strings.HasPrefix(line, "# ") ||
		strings.HasPrefix(line, "#\t") ||
		strings.TrimSpace(line) == "#"
}

// splitIDTitle 按第一个空格/Tab 把标题行切成 ID 与标题两部分。
// 若开头的片段不像 ID（由 ASCII 字母/数字/-/_ 组成且后面还有内容），
// 则认为整行都是标题，返回空 ID，由调用方自动生成序号。
func splitIDTitle(s string) (id, title string) {
	sep := strings.IndexAny(s, " \t")
	if sep < 0 {
		// 整行没有分隔符：整行就是标题，ID 由调用方自动生成
		return "", s
	}
	head, tail := s[:sep], strings.TrimLeft(s[sep+1:], " \t")
	if isIDLike(head) && tail != "" {
		return head, tail
	}
	return "", s
}

// autoID 生成自动序号 ID，如 a01、a02。
func autoID(n int) string { return fmt.Sprintf("a%02d", n) }

// isIDLike 判断一个片段是否像文章 ID：仅由 ASCII 字母、数字、'-'、'_' 组成。
// 中文标题（如 "免费域名"）不会命中，从而走自动编号分支。
func isIDLike(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

// countChars 统计字符串中非空白的 rune 个数，作为正文字符数。
func countChars(s string) int {
	n := 0
	for _, r := range s {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}
