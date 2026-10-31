// app.go —— 根 model：视图状态机（列表/阅读）、尺寸、主题、用户名注入。
//
// Update 的消息分类：
//   - tea.WindowSizeMsg：记录尺寸、触发 ready、通知子组件 resize；
//   - tea.KeyMsg：按视图状态分发（列表态移动/进阅读，阅读态滚动/返回）；
//   - tea.MouseMsg：滚轮行数由 accel 加速计算后作用到当前视图；
//   - ctrl+t：循环主题（列表态/阅读态都可用，不退出）。
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"skablog/internal/content"
	"skablog/internal/theme"
)

// viewState 是根视图状态机。
type viewState int

const (
	viewList viewState = iota // 列表态
	viewRead                  // 阅读态（正文滚动）
)

// Model 实现 bubbletea.Model。
type Model struct {
	articles []content.Article // 全部文章（共享只读）
	username string            // 当前用户（SSH 会话用户名 / 本地USER）
	view     viewState         // 当前视图
	width    int               // 终端宽
	height   int               // 终端高
	theme    string            // 当前主题名（注册表键，View 时现取 palette）
	list     listModel         // 列表子模型
	reader   readerModel       // 阅读子模型
}

// New 构造根 model：username 取 SSH 会话 sess.User()（或本地 USER）。
func New(arts []content.Article, username string) Model {
	return Model{
		articles: arts,
		username: username,
		view:     viewList,
		theme:    theme.Names()[0],
		list:     listModel{width: 0},
	}
}

// Init 目前没有启动期任务（鼠标由 ProgramOption 开启）。
func (m Model) Init() tea.Cmd { return nil }

// Update 是状态机核心，只分发消息、不改样式。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resume() // resize 子组件
		return m, nil

	case tea.KeyMsg:
		// 全局键：Ctrl+T 循环主题；Q / Ctrl+C 退出（模式无关）
		switch s := msg.String(); s {
		case "ctrl+t":
			m.theme = theme.Cycle(m.theme)
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		}

		switch m.view {
		case viewList:
			return m, m.updateListKey(msg.String())
		case viewRead:
			if m.reader.keys(msg) {
				return m, tea.Quit // 阅读态 q 也退出
			}
			switch s := msg.String(); s {
			case "left", "h", "esc":
				m.view = viewList // ←/h/Esc 返回列表
			}
			return m, nil
		}

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp, tea.MouseWheelDown:
			return m.updateWheel(msg.Type == tea.MouseWheelUp)
		}
	}
	return m, nil
}

// updateListKey 处理列表态键位（指针接收者：就地改 view/reader 状态）。
func (m *Model) updateListKey(s string) tea.Cmd {
	rows := listRows(m.height)
	switch s {
	case "up", "k":
		m.list.move(m.articles, -1, rows)
	case "down", "j":
		m.list.move(m.articles, 1, rows)
	case "enter":
		if len(m.articles) > 0 {
			arts := m.articles
			m.reader.open(arts[m.list.focused], listMainW(m.width-2, sidebarWidth(m.width-2)), readerRows(m.height))
			m.view = viewRead
		}
	case "q":
		return tea.Quit
	}
	return nil
}

// updateWheel 处理滚轮：列表态移动聚焦，阅读态按加速行数滚动。
func (m Model) updateWheel(up bool) (tea.Model, tea.Cmd) {
	switch m.view {
	case viewList:
		if up {
			m.list.move(m.articles, -1, listRows(m.height))
		} else {
			m.list.move(m.articles, 1, listRows(m.height))
		}
	case viewRead:
		m.reader.wheel(time.Now(), up)
	}
	return m, nil
}

// resume 在窗口尺寸变化/进入模式时同步子组件。
func (m *Model) resume() {
	inner := m.width - 2
	if inner < 0 {
		inner = 0
	}
	m.list.resize(listMainW(inner, sidebarWidth(inner)))
	if m.view == viewRead && m.reader.article != nil {
		art := *m.reader.article
		m.reader.open(art, listMainW(inner, sidebarWidth(inner)), readerRows(m.height))
	}
}

// View 折行渲染整屏。WindowSizeMsg 到达前返回 "loading..."（绝不返回空串）。
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "loading..."
	}
	w, rows := m.width, m.height-2
	if rows < 1 {
		rows = 1
	}
	return renderRoot(m.articles, m.username, m.list, m.reader, m.view, w, rows, theme.Get(m.theme))
}

// 以下是几处布局行数换算的小助手（集中管理，测试同包可见）。

// listRows 列表态可用行数（根框上下边各占 1）。
func listRows(height int) int {
	if n := height - 2; n > 0 {
		return n
	}
	return 1
}

// readerRows 阅读态正文行数（再扣掉顶部进度行 1 行）。
func readerRows(height int) int {
	if n := height - 3; n > 0 {
		return n
	}
	return 1
}

// listMainW 主列宽 = 内宽 - sidebar。
func listMainW(inner, sbW int) int {
	if n := inner - sbW; n > 1 {
		return n
	}
	return 1
}
