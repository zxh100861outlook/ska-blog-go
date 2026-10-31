// Package theme 提供 TUI 的 3 套内置配色（palette）与切换注册表。
//
// Palette 是纯数据结构，样式在 View 时由 tui 包从当前 palette 现取
// （禁止预烘焙 lipgloss.Style），因此 Ctrl+T 切主题后全界面即时生效。
package theme

import "github.com/charmbracelet/lipgloss"

// Palette 是一套主题配色的完整集合。
type Palette struct {
	Name       string         // 主题名（注册表键）
	Primary    lipgloss.Color // 主色：选中态边框/强调文字
	Background lipgloss.Color // 背景色
	Text       lipgloss.Color // 正文文字
	TextMuted  lipgloss.Color // 暗淡文字：摘要/键位提示
	Border     lipgloss.Color // 边框
}

var registry = []Palette{
	{
		Name:       "默认",
		Primary:    "#8b5cf6", // 紫
		Background: "#1a1a24",
		Text:       "#dedede",
		TextMuted:  "#8a8a96",
		Border:     "#5b5b70",
	},
	{
		Name:       "aura",
		Primary:    "#5cb66b", // 绿 aura 风
		Background: "#0f1a10",
		Text:       "#d9e6d9",
		TextMuted:  "#77a077",
		Border:     "#3c6b44",
	},
	{
		Name:       "冷青",
		Primary:    "#22d3ee", // 冷青色
		Background: "#0b1822",
		Text:       "#d5e6ee",
		TextMuted:  "#6d92a6",
		Border:     "#2a5a70",
	},
}

// Names 返回全部内置主题名（注册顺序，供测试/调试）。
func Names() []string {
	out := make([]string, 0, len(registry))
	for _, p := range registry {
		out = append(out, p.Name)
	}
	return out
}

// Get 按名字取 palette；未知名字回退到第一套（默认紫）。
func Get(name string) Palette {
	for _, p := range registry {
		if p.Name == name {
			return p
		}
	}
	return registry[0]
}

// Cycle 返回 next 主题：在注册表里循环（默认→aura→冷青→默认...）。
// 状态仅由当前主题名承载，无副作用，可直接在 Ctrl+T 分支调用。
func Cycle(cur string) string {
	for i, p := range registry {
		if p.Name == cur {
			return registry[(i+1)%len(registry)].Name
		}
	}
	// 未知名字从默认开始
	return registry[0].Name
}
