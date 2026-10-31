// app_test.go —— 根状态机测试：全部直接调 Update，标准库断言。
package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

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
	if !strings.Contains(a, "SKA-BLOG://GO") {
		t.Error("根框标题缺失")
	}
}

// TestMouseMoveList 滚轮在列表态移动聚焦。
func TestMouseMoveList(t *testing.T) {
	m := sized(t)
	m2, _ := m.Update(tea.MouseMsg{Type: tea.MouseWheelDown})
	if m2.(Model).list.focused != 1 {
		t.Errorf("滚轮下移应聚焦 1, got %d", m2.(Model).list.focused)
	}
}

// TestViewWellFormed 全屏行数与根框完整（上下边框/标题/提示/STATUS）。
func TestViewWellFormed(t *testing.T) {
	m := sized(t)
	v := m.View()
	lines := strings.Split(v, "\n")
	if len(lines) != twH {
		t.Errorf("应恰好 %d 行, got %d", twH, len(lines))
	}
	top := stripANSI(lines[0])
	if !strings.HasPrefix(top, "┌") || !strings.Contains(top, "SKA-BLOG://GO") {
		t.Errorf("顶边框应含标题, got %q", top)
	}
	last := stripANSI(lines[len(lines)-1])
	if !strings.HasPrefix(last, "└") || !strings.Contains(last, "退出") {
		t.Errorf("底边框应含键位提示, got %q", last)
	}
	joined := stripANSI(v)
	if !strings.Contains(joined, "STATUS") || !strings.Contains(joined, "当前位置：首页") {
		t.Error("STATUS 框首页态内容缺失")
	}
	if !strings.Contains(joined, "当前用户：tester") {
		t.Error("STATUS 应注入用户名")
	}
	if !strings.Contains(joined, "文章：6 篇") {
		t.Error("STATUS 应显示文章数")
	}
	if !strings.Contains(joined, "提示") || !strings.Contains(joined, "滚轮连滚会加速") {
		t.Error("提示框缺失")
	}
}
