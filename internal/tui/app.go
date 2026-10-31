// app.go —— 根 model：视图状态机（列表/阅读）、尺寸、主题、用户名注入。
//
// Update 的消息分类：
//   - tea.WindowSizeMsg：记录尺寸、触发 ready、通知子组件 resize；
//   - tea.KeyMsg：按视图状态分发（列表态移动/进阅读，阅读态滚动/返回）；
//   - ctrl+t：循环主题（列表态/阅读态都可用，不退出）；
//   - tab：切换侧栏收起/展开（全局）；
//   - / 与 :：列表态进入搜索/编号跳转输入态（手写 rune 级输入）。
//
// 鼠标不接管（见 server.go 去掉 WithMouseCellMotion）：终端拖拽选中即可复制。
package tui

import (
	"strings"

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

// inputMode 是列表态下的"输入态"：搜索 / 编号跳转。
type inputMode int

const (
	inputNone inputMode = iota // 无输入态
	inputSearch                // / 搜索：实时过滤列表
	inputJump                  // : 编号跳转：精确后前缀匹配文章 ID
)

// Model 实现 bubbletea.Model。
type Model struct {
	articles    []content.Article // 全部文章（共享只读）
	visible     []content.Article // 当前可见文章（搜索过滤后，等于 articles 或子集）
	username    string            // 当前用户（SSH 会话用户名 / 本地USER）
	view        viewState         // 当前视图
	width       int               // 终端宽
	height      int               // 终端高
	theme       string            // 当前主题名（注册表键，View 时现取 palette）
	list        listModel         // 列表子模型
	reader      readerModel       // 阅读子模型
	sidebarOpen bool              // 侧栏展开（Tab 收起/展开）
	mode        inputMode         // 当前输入态（搜索/跳转/无）
	query       []rune            // 输入缓冲（rune 级，中文按字节累积为 rune）
	jumpErr     bool              // 跳转未命中标记（输入行后显示"无此文"）
}

// New 构造根 model：username 取 SSH 会话 sess.User()（或本地 USER）。
func New(arts []content.Article, username string) Model {
	return Model{
		articles:    arts,
		visible:     arts,
		username:    username,
		view:        viewList,
		theme:       theme.Names()[0],
		list:        listModel{width: 0},
		sidebarOpen: true,
	}
}

// Init 目前没有启动期任务。
func (m Model) Init() tea.Cmd { return nil }

// Update 是状态机核心，只分发消息、不改样式。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resume() // resize 子组件
		return m, nil

	case tea.KeyMsg:
		// 全局键：Ctrl+T 循环主题；Tab 切换侧栏；Q / Ctrl+C 退出（模式无关）
		switch s := msg.String(); s {
		case "ctrl+t":
			m.theme = theme.Cycle(m.theme)
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.sidebarOpen = !m.sidebarOpen
			m.resume() // 主列宽变化，重排列表/正文
			return m, nil
		}

		// 输入态（搜索/跳转）优先处理
		if m.mode != inputNone {
			return m, m.updateInput(msg)
		}

		switch m.view {
		case viewList:
			return m, m.updateListKey(msg.String())
		case viewRead:
			if m.reader.keys(msg) {
				return m, tea.Quit // 阅读态 q 也退出
			}
			switch msg.String() {
			case "left", "h", "esc":
				m.view = viewList // ←/h/Esc 返回列表
			}
			return m, nil
		}
	}
	return m, nil
}

// updateListKey 处理列表态键位（指针接收者：就地改 view/reader 状态）。
func (m *Model) updateListKey(s string) tea.Cmd {
	rows := listRows(m.height)
	switch s {
	case "up", "k":
		m.list.move(m.visible, -1, rows)
	case "down", "j":
		m.list.move(m.visible, 1, rows)
	case "enter":
		return m.openFocused()
	case "/":
		// 进入搜索态：显示输入行，空查询命中全部（可见列表不变）
		m.mode = inputSearch
		m.query = nil
		m.jumpErr = false
		m.refreshVisible()
	case ":":
		// 进入编号跳转态：显示输入行，不过滤列表
		m.mode = inputJump
		m.query = nil
		m.jumpErr = false
	case "q":
		return tea.Quit
	}
	return nil
}

// updateInput 处理搜索/跳转输入态的键：字符累积、退格、Esc、Enter。
func (m *Model) updateInput(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = inputNone
		m.query = nil
		m.jumpErr = false
		m.refreshVisible() // 退搜索/跳转恢复全量
		return nil
	case "enter":
		if m.mode == inputSearch {
			return m.openFocused() // 搜索：直接进焦点文章阅读
		}
		if len(m.query) > 0 {
			if a := matchID(m.articles, string(m.query)); a != nil {
				// 跳转命中：直达阅读
				m.mode = inputNone
				m.query = nil
				m.jumpErr = false
				m.refreshVisible()
				m.reader.open(*a, m.mainW(), readerRows(m.height))
				m.view = viewRead
				return nil
			}
			m.jumpErr = true // 未命中：显示"无此文"，保留输入
		}
		return nil
	// 方向键移动光标：仅搜索态可移动（列表已过滤，在命中项间移动）；
	// 跳转态列表未过滤，移动无意义——up/down 非字符，直接吞掉。
	// 注意：j/k 是查询字符（搜 jdk/kotlin 不能废），不得截获，落入下方 rune 累积分支。
	case "up":
		if m.mode == inputSearch {
			m.list.move(m.visible, -1, listRows(m.height))
		}
		return nil
	case "down":
		if m.mode == inputSearch {
			m.list.move(m.visible, 1, listRows(m.height))
		}
		return nil
	case "backspace", "ctrl+h":
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.jumpErr = false
			if m.mode == inputSearch {
				m.refreshVisible()
				m.list.focused, m.list.offset = 0, 0 // 焦点回首个命中
			}
		}
		return nil
	}
	// 普通字符：rune 级累积（中文按客户端送来的字节解码为 rune 追加）
	if len(msg.Runes) > 0 {
		for _, r := range msg.Runes {
			m.query = append(m.query, r)
		}
		m.jumpErr = false
		if m.mode == inputSearch {
			m.refreshVisible()
			m.list.focused, m.list.offset = 0, 0 // 实时过滤，焦点在首个命中
		}
	}
	return nil
}

// openFocused 打开当前可见列表的焦点文章进阅读，并退出搜索态。
func (m *Model) openFocused() tea.Cmd {
	arts := m.visible
	if len(arts) == 0 {
		return nil
	}
	art := arts[m.list.focused]
	m.mode = inputNone
	m.query = nil
	m.jumpErr = false
	m.refreshVisible()
	m.reader.open(art, m.mainW(), readerRows(m.height))
	m.view = viewRead
	return nil
}

// refreshVisible 按当前输入态重算可见文章集（搜索过滤/全量），并夹紧聚焦。
func (m *Model) refreshVisible() {
	if m.mode == inputSearch {
		m.visible = filterArticles(m.articles, string(m.query))
	} else {
		m.visible = m.articles
	}
	if m.list.focused >= len(m.visible) {
		if len(m.visible) == 0 {
			m.list.focused = 0
		} else {
			m.list.focused = len(m.visible) - 1
		}
	}
	if m.list.focused < 0 {
		m.list.focused = 0
	}
}

// filterArticles 按标题+正文+ID 做不区分大小写的子串匹配（空查询命中全部）。
func filterArticles(arts []content.Article, q string) []content.Article {
	q = strings.ToLower(q)
	if q == "" {
		return arts
	}
	var out []content.Article
	for _, a := range arts {
		if strings.Contains(strings.ToLower(a.Title), q) ||
			strings.Contains(strings.ToLower(a.Body), q) ||
			strings.Contains(strings.ToLower(a.ID), q) {
			out = append(out, a)
		}
	}
	return out
}

// matchID 按文章 ID 匹配：先精确（不区分大小写），后前缀。
func matchID(arts []content.Article, q string) *content.Article {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil
	}
	for i := range arts {
		if strings.EqualFold(arts[i].ID, q) {
			return &arts[i]
		}
	}
	for i := range arts {
		if strings.HasPrefix(strings.ToLower(arts[i].ID), q) {
			return &arts[i]
		}
	}
	return nil
}

// resume 在窗口尺寸变化/侧栏切换/进入模式时同步子组件。
func (m *Model) resume() {
	m.list.resize(m.mainW())
	if m.view == viewRead && m.reader.article != nil {
		art := *m.reader.article
		m.reader.open(art, m.mainW(), readerRows(m.height))
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
	return renderRoot(m, w, rows)
}

// 以下是几处布局行数换算的小助手（集中管理，测试同包可见）。

// innerW 根框内宽（左右各 1 列边框）。
func (m Model) innerW() int {
	if n := m.width - 2; n > 0 {
		return n
	}
	return 0
}

// sbW 侧栏宽：收起（Tab 后）为 0，展开按 30% 规则计算。
func (m Model) sbW() int {
	if !m.sidebarOpen {
		return 0
	}
	return sidebarWidth(m.innerW())
}

// mainW 主列宽 = 内宽 - 侧栏宽（至少 1 列）。
func (m Model) mainW() int {
	if n := m.innerW() - m.sbW(); n > 1 {
		return n
	}
	return 1
}

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
