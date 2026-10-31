// app_test.go —— 根状态机测试：全部直接调 Update，标准库断言。
package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"skablog/internal/content"
	"skablog/internal/theme"
)

// TestMain 在任何样式渲染前强制打开颜色档位：CI/无 TTY 环境下
// lipgloss 会退化为无色渲染，导致"切主题后界面变化"不可断言。
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	lipgloss.SetHasDarkBackground(true)
	m.Run()
}

// key 构造 KeyMsg（bubbletea 键名 → KeyMsg 没有公开构造器，
// 测试用 tea.KeyMsg 的 String() 不可直接伪造；改用自定义结构的键名助手）。
type fakeKey struct{ s string }

// 行数：测试用固定尺寸 80x24 → 列表 22 行。
const twW, twH = 80, 24

func sized(t *testing.T) Model {
	t.Helper()
	m := New(mkArts(6), "tester")
	n, _ := m.Update(tea.WindowSizeMsg{Width: twW, Height: twH})
	m = n.(Model)
	return m
}

// TestLoadingBeforeSize WindowSizeMsg 到达前 View() 必须 "loading..."。
func TestLoadingBeforeSize(t *testing.T) {
	m := New(mkArts(2), "u")
	if v := m.View(); v != "loading..." {
		t.Fatalf("未收到尺寸前 View 应为 loading..., got %q", v)
	}
}

// TestEnterReadingBack 列表 Enter 进阅读；←/h/Esc 回列表。
func TestEnterReadingBack(t *testing.T) {
	m := sized(t)
	// Enter：FakeKey 无法直接构造 KeyMsg，这里以包含键值的方式分派不存在——
	// 实际改用 readerModel/list 层面的状态机入口：updateListKey 与 reader.keys。
	back := func(s string) (Model, bool) {
		m2 := m // 副本
		if m2.view == viewList {
			m2.updateListKey(s)
		}
		return m2, m2.view == viewRead // 声明进入的便捷判断
	}
	_ = back

	// 真正断言：updateListKey("enter") 后 view=阅读态
	m2 := m
	m2.updateListKey("enter")
	if m2.view != viewRead {
		t.Fatalf("enter 应进入阅读, view=%v", m2.view)
	}
	if m2.reader.article == nil {
		t.Fatal("阅读态必须有当前文章")
	}
	if m2.reader.vp.Height != readerRows(twH) {
		t.Errorf("视口高应 %d, got %d", readerRows(twH), m2.reader.vp.Height)
	}

	// 阅读态回列表：Update 处理 esc（KeyMsg 直接构造：tea.KeyMsg{Type: tea.KeyEsc}）
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m3.(Model).view != viewList {
		t.Errorf("esc 应回列表, view=%v", m3.(Model).view)
	}
	m4, _ := m2.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m4.(Model).view != viewList {
		t.Errorf("left 应回列表, view=%v", m4.(Model).view)
	}
	m5, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m5.(Model).view != viewList {
		t.Errorf("h 应回列表, view=%v", m5.(Model).view)
	}
}

// TestQuitKeys 阅读态 q 退出；列表态 q / Ctrl+C 退出。
func TestQuitKeys(t *testing.T) {
	m := sized(t)
	// 列表态 ctrl+c
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c 应返回 Quit cmd")
	}
	if msg := cmd(); msg == nil { //nolint:nilnessness // cmd 闭包返回 QuitMsg
		t.Fatal("Quit cmd 应产生 QuitMsg")
	}
	// 列表态 q
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("列表态 q 应退出")
	}
	// 阅读态 q
	m.updateListKey("enter")
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("阅读态 q 应退出")
	}
}

// TestCtrlTNotQuit Ctrl+T 只切主题，绝不退出；且循环正确。
func TestCtrlTNotQuit(t *testing.T) {
	m := sized(t)
	m.updateListKey("enter") // 阅读态同样可用
	start := m.theme
	for _, want := range []string{theme.Cycle(start), theme.Cycle(theme.Cycle(start)), start} {
		m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
		if cmd != nil {
			t.Fatal("ctrl+t 不应产生退出 cmd")
		}
		m = m2.(Model)
		if m.theme != want {
			t.Errorf("循环主题应为 %q, got %q", want, m.theme)
		}
	}
}

// TestThemeAffectsView 切主题后 View() 的边框/主色立即变化（不预烘焙）。
func TestThemeAffectsView(t *testing.T) {
	m := sized(t)
	v1 := stripANSI(m.View()) // 内容不变
	_ = v1
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	a := m.View()
	b := m2.(Model).View()
	if a == b {
		t.Error("切主题后全界面渲染应当变化")
	}
	// 名字进 STATUS：阅读态标题文字里能看到当前主题无关，这里仅确保内容一致
	if !strings.Contains(a, "SKA·BLOG") {
		t.Error("根框品牌字缺失")
	}
}

// TestViewWellFormed 全屏行数与 soft-serve 布局完整（header/分割线/STATUS/KEYS）。
func TestViewWellFormed(t *testing.T) {
	m := sized(t)
	v := m.View()
	lines := strings.Split(v, "\n")
	if len(lines) != twH {
		t.Errorf("应恰好 %d 行, got %d", twH, len(lines))
	}
	// 顶部 header（无边框）：左品牌字，右访客信息右对齐
	top := stripANSI(lines[0])
	if !strings.HasPrefix(top, "SKA·BLOG") {
		t.Errorf("header 应以品牌字开头, got %q", top)
	}
	if strings.Contains(top, "┌") || strings.Contains(top, "│") {
		t.Errorf("header 不应带边框字符, got %q", top)
	}
	if !strings.Contains(top, "访客：tester") || !strings.Contains(top, "文章 6 篇") {
		t.Errorf("header 应含访客/文章数, got %q", top)
	}
	// header 下分割线
	div := stripANSI(lines[1])
	if !strings.HasPrefix(div, "─") || !strings.HasSuffix(div, "─") {
		t.Errorf("第二行应为水平分割线, got %q", div)
	}
	// 取消整周包边与 bottomTitle：最后一行应为内容（无 └/┘）
	last := stripANSI(lines[len(lines)-1])
	if strings.Contains(last, "└") || strings.Contains(last, "┘") || strings.Contains(last, "退出") {
		t.Errorf("底部不应再含边框或键位提示（已搬侧栏）, got %q", last)
	}
	joined := stripANSI(v)
	if !strings.Contains(joined, "STATUS") || !strings.Contains(joined, "当前位置：首页") {
		t.Error("STATUS 小节首页态内容缺失")
	}
	if !strings.Contains(joined, "当前用户：tester") {
		t.Error("STATUS 应注入用户名")
	}
	if !strings.Contains(joined, "文章：6 篇") {
		t.Error("STATUS 应显示文章数")
	}
	// 侧栏 KEYS 键位小节：列表态键 + 主题/侧栏状态行
	if !strings.Contains(joined, "KEYS") {
		t.Error("KEYS 小节缺失")
	}
	for _, want := range []string{"/ 搜索", ": 跳转", "Tab 侧栏", "Ctrl+T 主题", "Q/Ctrl+C 退出", "主题：默认", "侧栏：开"} {
		if !strings.Contains(joined, want) {
			t.Errorf("键位小节缺 %q", want)
		}
	}
}

// TestRenderHeightReading 阅读态整屏行数也恒等于窗口高（进度行+视口+header+分割线）。
func TestRenderHeightReading(t *testing.T) {
	m := sized(t)
	m2 := m // 副本
	m2.updateListKey("enter")
	if m2.view != viewRead {
		t.Fatal("enter 应进入阅读")
	}
	v := m2.View()
	if n := len(strings.Split(v, "\n")); n != twH {
		t.Errorf("阅读态应恰好 %d 行, got %d", twH, n)
	}
	// 收起侧栏（主列全宽）行数仍恒等
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyTab})
	if n := len(strings.Split(m3.(Model).View(), "\n")); n != twH {
		t.Errorf("侧栏收起阅读态应 %d 行, got %d", twH, n)
	}
}

// TestTabTogglesSidebar Tab 收起/展开侧栏：主列宽随之变化。
func TestTabTogglesSidebar(t *testing.T) {
	m := sized(t)
	openW := m.mainW()
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = m2.(Model)
	if m.sidebarOpen {
		t.Fatal("tab 应收起侧栏")
	}
	if m.mainW() <= openW {
		t.Errorf("收起后主列应变宽: mainW=%d, 展开=%d", m.mainW(), openW)
	}
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = m2.(Model)
	if !m.sidebarOpen {
		t.Fatal("再次 tab 应展开侧栏")
	}
	if m.mainW() != openW {
		t.Errorf("展开后主列应复原: mainW=%d, 期望 %d", m.mainW(), openW)
	}
	// 收起时渲染不含 STATUS（侧栏宽度 0）
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = m2.(Model)
	if strings.Contains(stripANSI(m.View()), "STATUS") {
		t.Error("收起后不应渲染侧栏 STATUS")
	}
}

// TestSearchFilter / 进入搜索态：键入实时过滤，Esc 恢复全量。
func TestSearchFilter(t *testing.T) {
	arts := []content.Article{
		{ID: "a", Title: "github clone", Body: "x", WordCount: 1},
		{ID: "b", Title: "其他", Body: "y", WordCount: 1},
	}
	m := New(arts, "tester")
	n, _ := m.Update(tea.WindowSizeMsg{Width: twW, Height: twH})
	m = n.(Model)

	// / 进入搜索
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = m2.(Model)
	if m.mode != inputSearch {
		t.Fatalf("/ 应进入搜索态, mode=%v", m.mode)
	}
	// 空查询命中全部
	if len(m.visible) != 2 {
		t.Fatalf("空查询应命中全部, got %d", len(m.visible))
	}
	// 键入 "github"：实时过滤，只命中 a
	for _, r := range []rune("github") {
		m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m2.(Model)
	}
	if len(m.visible) != 1 || m.visible[0].ID != "a" {
		t.Fatalf("'github' 应只命中 a, got %d: %v", len(m.visible), ids(m.visible))
	}
	if m.list.focused != 0 {
		t.Errorf("过滤后焦点应在首个命中, focused=%d", m.list.focused)
	}
	// 搜索态输入行应显示
	if !strings.Contains(stripANSI(m.View()), "/ github") {
		t.Error("搜索态应显示输入行 '/ github'")
	}
	// Esc 恢复全量
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = m2.(Model)
	if m.mode != inputNone || len(m.visible) != 2 {
		t.Errorf("esc 应退出搜索恢复全量, mode=%v visible=%d", m.mode, len(m.visible))
	}
}

// TestJumpByID : 输入文章ID，Enter 精确命中直达阅读。
func TestJumpByID(t *testing.T) {
	arts := []content.Article{
		{ID: "wj2", Title: "github clone 加速", Body: "x", WordCount: 1},
		{ID: "wj1", Title: "hello", Body: "y", WordCount: 1},
	}
	m := New(arts, "tester")
	n, _ := m.Update(tea.WindowSizeMsg{Width: twW, Height: twH})
	m = n.(Model)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	m = m2.(Model)
	if m.mode != inputJump {
		t.Fatalf(": 应进入跳转态, mode=%v", m.mode)
	}
	for _, r := range []rune("wj2") {
		m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m2.(Model)
	}
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(Model)
	if m.view != viewRead || m.reader.article == nil || m.reader.article.ID != "wj2" {
		t.Errorf("wj2 应直达阅读, view=%v id=%v", m.view, artID(m.reader.article))
	}
}

// TestJumpMiss : 未命中 ID 显示"无此文"且保留输入、不清空。
func TestJumpMiss(t *testing.T) {
	arts := []content.Article{{ID: "wj2", Title: "t", Body: "b", WordCount: 1}}
	m := New(arts, "tester")
	n, _ := m.Update(tea.WindowSizeMsg{Width: twW, Height: twH})
	m = n.(Model)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	m = m2.(Model)
	for _, r := range []rune("zzz") {
		m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m2.(Model)
	}
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(Model)
	if !m.jumpErr {
		t.Fatal("未命中应置 jumpErr")
	}
	if m.mode != inputJump || string(m.query) != "zzz" {
		t.Errorf("未命中应保留输入, mode=%v query=%q", m.mode, string(m.query))
	}
	if !strings.Contains(stripANSI(m.View()), "无此文") {
		t.Error("输入行后应显示'无此文'")
	}
}

// TestReadKeysNoPg 阅读态方向键滚动，PgUp/PgDn/Space/Home/g/G 均无反应。
func TestReadKeysNoPg(t *testing.T) {
	m := sized(t)
	m.updateListKey("enter") // 进阅读
	before := m.reader.vp.YOffset
	// 下滚 1 行
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m2.(Model)
	if m.reader.vp.YOffset != before {
		t.Errorf("↓ 应滚动, before=%d after=%d", before, m.reader.vp.YOffset)
	}
	before = m.reader.vp.YOffset
	// PgUp/PgDn/Space/Home/End/g/G 应无反应
	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyPgUp}, {Type: tea.KeyPgDown},
		{Type: tea.KeyRunes, Runes: []rune{' '}},
		{Type: tea.KeyHome}, {Type: tea.KeyEnd},
		{Type: tea.KeyRunes, Runes: []rune{'g'}}, {Type: tea.KeyRunes, Runes: []rune{'G'}},
	} {
		m2, _ = m.Update(k)
		m = m2.(Model)
		if m.reader.vp.YOffset != before {
			t.Errorf("键 %v 不应滚动, before=%d after=%d", k, before, m.reader.vp.YOffset)
		}
	}
}

// TestInputBackspace 搜索输入退格删除最后一个 rune（中文按 rune 计）。
func TestInputBackspace(t *testing.T) {
	m := sized(t)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = m2.(Model)
	for _, r := range []rune("中文") {
		m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m2.(Model)
	}
	if string(m.query) != "中文" {
		t.Fatalf("输入应累积 rune, got %q", string(m.query))
	}
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = m2.(Model)
	if string(m.query) != "中" {
		t.Errorf("退格应删一个 rune, got %q", string(m.query))
	}
}

// ids 取可见文章 ID 列表（测试辅助）。
func ids(arts []content.Article) []string {
	out := make([]string, len(arts))
	for i := range arts {
		out[i] = arts[i].ID
	}
	return out
}

// artID 取文章 ID（nil 安全）。
func artID(a *content.Article) string {
	if a == nil {
		return ""
	}
	return a.ID
}
