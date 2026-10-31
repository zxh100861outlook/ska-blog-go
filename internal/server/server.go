// server.go —— gliderlabs/ssh 服务器封装：每个会话一个 tea.Program，
// 或为无 PTY 请求打印纯文本文章列表。
package server

import (
	"crypto"
	"errors"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"

	"skablog/internal/content"
	"skablog/internal/tui"
)

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
		pipeReader, pipeWriter := io.Pipe()
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
