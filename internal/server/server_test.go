// server_test.go —— 纯文本列表与 Start 构造测试（不真实监听，避免 CI 挂起）。
package server

import (
	"errors"
	"os"
	"strings"
	"testing"

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
