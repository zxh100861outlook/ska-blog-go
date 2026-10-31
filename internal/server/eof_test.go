// eof_test.go —— P4.5 缺陷回归：SSH 会话 stdin EOF 必须能让 TUI 退出。
// 旧缺陷：stdin 关闭后 bubbletea readLoop 读到 EOF 静默结束，会话挂死，
// `printf "\\rqq" | ssh ... -tt` 超时（exit 124）。
package server

import (
	"net"
	"testing"
	"time"

	gossh "golang.org/x/crypto/ssh"

	"skablog/internal/content"
)

// TestSessionEOFQuits 程序化起 server（随机端口），用 x/crypto/ssh 客户端
// 开 PTY session，写入 q 后关闭 stdin，断言远端 handler 在超时前返回。
func TestSessionEOFQuits(t *testing.T) {
	t.Chdir(t.TempDir()) // host key 落临时目录

	srv, err := Start(Config{Addr: "127.0.0.1:0", Guest: true, Articles: []content.Article{
		{ID: "t1", Title: "标题", Body: "正文"},
	}})
	if err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("随机端口监听失败: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(l) }()
	defer func() {
		if err := srv.Close(); err != nil {
			t.Errorf("Close 失败: %v", err)
		}
	}()

	addr := l.Addr().String()
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	cconfig := &gossh.ClientConfig{
		User:            "guest",
		Auth:            []gossh.AuthMethod{}, // 无凭证，guest 通道
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         3 * time.Second,
	}
	cConn, chans, reqs, err := gossh.NewClientConn(conn, addr, cconfig)
	if err != nil {
		t.Fatalf("SSH 握手失败: %v", err)
	}
	client := gossh.NewClient(cConn, chans, reqs)
	defer func() { _ = client.Close() }()

	sess, err := client.NewSession()
	if err != nil {
		t.Fatalf("开 session 失败: %v", err)
	}

	// stdin pipe 必须在 Shell 启动之前获取
	stdin, err := sess.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe 失败: %v", err)
	}

	// 申请 PTY（80x24），与真机 -tt 场景一致
	if err := sess.RequestPty("xterm", 80, 24, gossh.TerminalModes{}); err != nil {
		t.Fatalf("申请 PTY 失败: %v", err)
	}

	if err := sess.Shell(); err != nil {
		t.Fatalf("shell 请求失败: %v", err)
	}

	// 写入 q 并关闭 stdin —— 触发服务端读侧 EOF
	if _, err := stdin.Write([]byte("qq")); err != nil {
		t.Fatalf("写入 q 失败: %v", err)
	}
	if err := stdin.Close(); err != nil {
		t.Fatalf("关闭 stdin 失败: %v", err)
	}

	// 断言：会话在 8 秒内结束（旧缺陷会挂到超时）
	waitErr := make(chan error, 1)
	go func() { waitErr <- sess.Wait() }()
	select {
	case err := <-waitErr:
		if err != nil {
			t.Logf("会话退出状态: %v", err) // exit status 0 即 nil，其余记录不判死
		}
	case <-time.After(8 * time.Second):
		t.Fatalf("stdin EOF 后 handler 未在超时前返回（回归缺陷未修复，P4.5）")
	}
	// srv.Serve 的返回值经缓冲 channel 丢弃不读（defer 已 Close），
	// 避免 Close 之前在测试体内阻塞。
}
