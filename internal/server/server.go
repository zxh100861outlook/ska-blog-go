// server.go —— gliderlabs/ssh 服务器封装：PTY 会话暂给纯文本占位视图
//（Q2/Q3 接入 tui 包），无 PTY 请求打印纯文本文章列表。
package server

import (
	"crypto"
	"errors"
	"fmt"
	"strings"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"

	"skablog/internal/content"
	"skablog/internal/render"
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
		_, winch, isPTY := sess.Pty()
		if !isPTY {
			// ssh -T 无 PTY：输出纯文本列表后退出
			fmt.Fprint(sess, PlainList(articles))
			return
		}

		// TODO(Q3): 换回每个会话一个 bubbletea Program（tui 包在 Q2/Q3 编写）。
		// 原实现：io.Pipe 包一层输入 + prog.Run()（EOF 退出的完整资料在 git
		// 历史的 sshblog 仓库 internal/server/server.go，届时原样恢复）。
		//
		// Q1 临时实现：直接用 render 包把第一篇文章渲染成带 ANSI 颜色的
		// 纯文本打印给客户端，然后阻塞读取输入，EOF 或按 q 即退出 ——
		// 保证 stdin EOF 后会话不会挂死（P4.5 缺陷回归由 eof_test 把关）。
		if len(articles) > 0 {
			a := articles[0]
			fmt.Fprint(sess, render.Render("# "+a.Title+"\n\n"+a.Body)+"\r\n")
		}
		fmt.Fprint(sess, "\r\n(占位视图：Q3 恢复 TUI。按 q 或关闭输入退出)\r\n")

		// 必须消费 winch：channel 缓冲为 1，无人消费会卡住 SSH 请求循环。
		go func() {
			for range winch {
			}
		}()

		// 占位等待退出：读到 EOF（客户端关闭 stdin / 断开）或按键 q 即结束。
		buf := make([]byte, 256)
		for {
			n, err := sess.Read(buf)
			if err != nil {
				// EOF 或连接断开：正常结束，会话随之退出
				return
			}
			for _, b := range buf[:n] {
				if b == 'q' || b == 'Q' {
					return
				}
			}
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
