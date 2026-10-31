// server_test.go —— 纯文本列表、Start 构造与色彩档位探测测试（不真实监听）。
package server

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"skablog/internal/content"
)

func TestPlainList(t *testing.T) {
	arts := []content.Article{
		{ID: "wj1", Title: "hello,world", Body: "第一段\n\n第二段"},
		{ID: "wj2", Title: "github clone 加速", Body: "a\n\nb\n\nc"},
	}
	want := "[wj1] hello,world  (2 段)\n[wj2] github clone 加速  (3 段)\n"
	if got := PlainList(arts); got != want {
		t.Errorf("纯文本列表格式不符:\ngot:  %q\nwant: %q", got, want)
	}
	if got := PlainList(nil); got != "" {
		t.Errorf("空列表应输出空字符串, got %q", got)
	}
}

func TestStartRequiresGuest(t *testing.T) {
	_, err := Start(Config{Addr: "127.0.0.1:0", Guest: false})
	if err == nil {
		t.Fatalf("guest=false 应启动失败")
	}
	if !errors.Is(err, errGuestNotImplemented) || !strings.Contains(err.Error(), "非 guest 模式暂未实现鉴权") {
		t.Errorf("应返回占位错误提示, got: %v", err)
	}
}

// TestStartConstruction 只测试构造（加载 host key + 组装 Server），
// 不执行 ListenAndServe，避免 CI 无 ssh 客户端时挂起。
func TestStartConstruction(t *testing.T) {
	t.Chdir(t.TempDir()) // 让 data/host_key 落到临时目录
	srv, err := Start(Config{Addr: "127.0.0.1:0", Guest: true, Articles: []content.Article{
		{ID: "t1", Title: "标题", Body: "正文"},
	}})
	if err != nil {
		t.Fatalf("Start 失败: %v", err)
	}
	defer func() {
		if err := srv.Close(); err != nil {
			t.Errorf("Close 应干净返回: %v", err)
		}
	}()

	if _, err := os.Stat("data/host_key"); err != nil {
		t.Errorf("data/host_key 应已生成: %v", err)
	}
}

// TestApplySessionEnvProfile 表驱动验证 (term, colorterm) → 色彩档位映射。
//
// 重点：仅凭 TERM 真彩名单（kitty/alacritty/wezterm/foot/contour，无
// COLORTERM 转发）也应上 TrueColor；256color 后缀、其它非空、空兜底
// 保持原判定；显式 COLORTERM=truecolor/24bit 优先级最高。
func TestApplySessionEnvProfile(t *testing.T) {
	cases := []struct {
		name      string
		term      string // ptyTerm（OpenSSH 在 pty-req 携带，通常即客户端 TERM）
		colorTerm string // 会话环境里的 COLORTERM（sshd 默认不放行，通常缺省）
		want      termenv.Profile
	}{
		{"kitty 仅 TERM 名单上真彩", "xterm-kitty", "", termenv.TrueColor},
		{"alacritty 仅 TERM 名单上真彩", "alacritty", "", termenv.TrueColor},
		{"wezterm 仅 TERM 名单上真彩", "wezterm", "", termenv.TrueColor},
		{"foot 仅 TERM 名单上真彩", "foot", "", termenv.TrueColor},
		{"foot-direct 仅 TERM 名单上真彩", "foot-direct", "", termenv.TrueColor},
		{"contour 前缀上真彩", "contour", "", termenv.TrueColor},
		{"contour 变体前缀上真彩", "contour-iterm2", "", termenv.TrueColor},
		{"xterm-256color 走 ANSI256", "xterm-256color", "", termenv.ANSI256},
		{"screen-256color 走 ANSI256", "screen-256color", "", termenv.ANSI256},
		{"dumb 走 ANSI", "dumb", "", termenv.ANSI},
		{"vt100 不算真彩走 ANSI", "vt100", "", termenv.ANSI},
		{"rxvt-unicode 不算真彩走 ANSI", "rxvt-unicode", "", termenv.ANSI},
		{"空 TERM 兜底 ANSI256", "", "", termenv.ANSI256},
		{"显式 COLORTERM=truecolor 优先", "xterm-256color", "truecolor", termenv.TrueColor},
		{"显式 COLORTERM=24bit 优先", "xterm", "24bit", termenv.TrueColor},
		{"COLORTERM 非真彩不压 TERM 名单", "xterm-kitty", "false", termenv.TrueColor},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var env []string
			if tc.colorTerm != "" {
				env = append(env, "COLORTERM="+tc.colorTerm)
			}
			applySessionEnv(env, tc.term)
			if got := lipgloss.ColorProfile(); got != tc.want {
				t.Errorf("term=%q colorterm=%q: profile=%v, want %v",
					tc.term, tc.colorTerm, got, tc.want)
			}
		})
	}
}

// TestApplySessionEnvFromEnviron ptyTerm 缺失时退回会话环境 TERM，
// 且该 TERM 命中真彩名单也上 TrueColor（覆盖 env 通道取值路径）。
func TestApplySessionEnvFromEnviron(t *testing.T) {
	applySessionEnv([]string{"TERM=xterm-kitty"}, "")
	if got := lipgloss.ColorProfile(); got != termenv.TrueColor {
		t.Errorf("environ TERM=xterm-kitty 应上 TrueColor, got %v", got)
	}
}
