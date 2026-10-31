// server.go —— gliderlabs/ssh 服务器封装：每个会话一个 tea.Program，
// 或为无 PTY 请求打印纯文本文章列表。
package server

import (
	"crypto"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"

	"skablog/internal/content"
	"skablog/internal/tui"
)

// envMu 串行化"会话终端环境覆写 + Program 创建"，见 makeHandler 内注释。
var envMu sync.Mutex

// gosshSigner 把 crypto.Signer 转成 x/crypto/ssh 的 Signer，
// 供 Server.AddHostKey 使用（gliderlabs/ssh.Signer 即其别名）。
func gosshSigner(key crypto.Signer) (ssh.Signer, error) {
	s, err := gossh.NewSignerFromKey(key)
	if err != nil {
		return nil, fmt.Errorf("server: 构造 host key signer 失败: %w", err)
	}
	return s, nil
}

// hostKeyPath 是 host key 的默认落盘位置（相对工作目录，PLAN 约定为 data/ 目录）。
const hostKeyPath = "data/host_key"

// errGuestNotImplemented 是 -guest=false 模式的占位错误。
var errGuestNotImplemented = errors.New("server: 非 guest 模式暂未实现鉴权")

// Config 是 SSH 服务器配置。
type Config struct {
	Addr     string            // 监听地址，如 127.0.0.1:2222
	Guest    bool              // guest 模式（允许匿名）；false 时启动即报错
	Articles []content.Article // 文章数据，会话间只读共享
}

// Start 按配置组装并返回一个已绑定好 handler 与 host key 的 *ssh.Server，
// 由调用方执行 ListenAndServe / Shutdown。
func Start(cfg Config) (*ssh.Server, error) {
	if !cfg.Guest {
		return nil, errGuestNotImplemented
	}
	if cfg.Addr == "" {
		cfg.Addr = ":2222"
	}

	signer, err := LoadOrCreate(hostKeyPath)
	if err != nil {
		return nil, err
	}
	hostSigner, err := gosshSigner(signer)
	if err != nil {
		return nil, err
	}

	srv := &ssh.Server{
		Addr:    cfg.Addr,
		Handler: makeHandler(cfg.Articles),
	}
	srv.AddHostKey(hostSigner)

	// guest 模式：故意不设置 PasswordHandler / PublicKeyHandler。
	// gliderlabs/ssh 文档明确：两者均为 nil 时不做任何客户端认证，
	// 客户端首个 "none" 请求即通过，不会弹出密码/密钥提示。
	return srv, nil
}

// makeHandler 构造会话处理器。
func makeHandler(articles []content.Article) ssh.Handler {
	return func(sess ssh.Session) {
		pty, winch, isPTY := sess.Pty()
		if !isPTY {
			// ssh -T 无 PTY：输出纯文本列表后退出
			fmt.Fprint(sess, PlainList(articles))
			return
		}

		// 有 PTY：每个会话独立跑一个 bubbletea Program。
		// 输入不能直接绑 sess：当客户端关闭 stdin（脚本/管道用法），
		// bubbletea 的 readLoop 读到 EOF 只会静默结束，程序不会退出，
		// SSH 通道随之挂死。因此用 io.Pipe 包一层：
		//   转发 goroutine 从 sess 读字节写 pipe；io.Copy 保证 EOF 前
		//   已读到的字节先完整写入 pipe（按键不丢），Copy 返回后
		//   （EOF 或对端断开）再向 program 发送 QuitMsg 促其退出。
		// 会话级终端环境：lipgloss/termenv 的 ColorProfile() 在进程内
		// 首次被调用时按 TERM/COLORTERM 探测并缓存（sync.Once）。服务端
		// 进程常无 TERM（nohup/systemd/无 TTY 启动），首帧渲染会退化为
		// Ascii 无色档位，所有 Foreground 样式变成空操作，Ctrl+T 切主题
		// 便无任何视觉变化。因此必须在创建 Program 前按本会话携带的
		// 环境覆写全局 TERM/COLORTERM。
		//
		// 并发会话互相覆写全局 env 是博客场景的可接受取舍：会话短促、
		// 目标一致（都是让各自界面有颜色）。用包级 mutex 把"设 env +
		// NewProgram + 触发首次尺寸"圈进同一临界区，把互相影响的窗口
		// 缩到最小。Send 在 Program 未启动前会阻塞（见 bubbletea 文档），
		// 故仍放 goroutine 发起——它不占锁，主 goroutine 随即 Run 无死锁。
		pipeReader, pipeWriter := io.Pipe()
		envMu.Lock()
		// TERM 由客户端在 pty-req 里携带（OpenSSH 不把 TERM 放 env 通道），
		// Environ() 里通常只有 LANG/LC_* 等；两者都传给 applySessionEnv。
		applySessionEnv(sess.Environ(), pty.Term)
		prog := tea.NewProgram(tui.New(articles, sess.User()),
			tea.WithInput(pipeReader),
			tea.WithOutput(sess),
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(), // 滚轮加速需要鼠标事件
		)
		go func() {
			defer pipeWriter.Close()
			// io.Copy 语义：读到 EOF 之前的内容全部写完才返回 —— 保证按键不丢
			_, _ = io.Copy(pipeWriter, sess)
			// 读侧结束（EOF / 客户端断开），正式退出 TUI；
			// Program 已终止时 Send 是 no-op，安全。
			prog.Send(tea.QuitMsg{})
		}()

		// Send 在 Program 未启动前会阻塞到启动完成（见 bubbletea 文档），
		// 所以放在 goroutine 里发，主 goroutine 随即进入 Run，无死锁。
		go prog.Send(tea.WindowSizeMsg{Width: pty.Window.Width, Height: pty.Window.Height})
		envMu.Unlock()

		// 持续转发客户端窗口变化；winch 在会话结束时由库关闭，
		// range 循环自然退出。该 channel 缓冲为 1，若无人消费会
		// 卡住 SSH 请求循环，因此必须消费。
		go func() {
			for win := range winch {
				prog.Send(tea.WindowSizeMsg{Width: win.Width, Height: win.Height})
			}
		}()

		// 注意：本版 bubbletea 的 Wait() 无返回值，最终错误来自 Run()。
		if _, err := prog.Run(); err != nil {
			fmt.Fprintf(sess, "会话异常退出: %v\r\n", err)
			_ = sess.Exit(1)
		}
	}
}

// applySessionEnv 把本会话的终端类型/环境写入进程环境，供
// lipgloss/termenv 在首次渲染探测颜色档位时读取。
//
//   - TERM：优先取 pty 请求携带的终端类型（sess.Pty().Term，OpenSSH
//     只在 pty-req 里发 TERM，不走 env 通道），其次取会话环境里的
//     TERM；两者皆无时兜底 xterm-256color，保证至少 ANSI256 档
//     （服务端进程环境常缺 TERM）。
//   - COLORTERM：会话环境携带才设置；否则清除，避免把宿主进程残留
//     的 COLORTERM 误带入本会话。
func applySessionEnv(environ []string, ptyTerm string) {
	term := ptyTerm
	colorTerm := ""
	for _, kv := range environ {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		switch k {
		case "TERM":
			if term == "" {
				term = v // ptyTerm 缺失时退回 env 通道的 TERM
			}
		case "COLORTERM":
			colorTerm = v
		}
	}
	if term == "" {
		term = "xterm-256color"
	}
	_ = os.Setenv("TERM", term)
	if colorTerm != "" {
		_ = os.Setenv("COLORTERM", colorTerm)
	} else {
		_ = os.Unsetenv("COLORTERM")
	}
}

// PlainList 渲染纯文本文章列表，格式与 main.go 的 -list 完全一致：
//
//	[wj2] github clone 加速  (14 段)
func PlainList(articles []content.Article) string {
	var b strings.Builder
	for _, a := range articles {
		fmt.Fprintf(&b, "[%s] %s  (%d 段)\n", a.ID, a.Title, a.ParagraphCount())
	}
	return b.String()
}
