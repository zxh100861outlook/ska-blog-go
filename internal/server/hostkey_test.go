// hostkey_test.go —— host key 生成/重载往返测试。
package server

import (
	"crypto"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadOrCreateGenerate 首次调用：自动建父目录、落盘 0600、返回可用 Signer。
func TestLoadOrCreateGenerate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "host_key")

	signer, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("生成 host key 失败: %v", err)
	}
	if signer == nil {
		t.Fatalf("返回的 Signer 不应为 nil")
	}

	// 文件应已生成，且权限 0600
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("host key 文件应已生成: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("文件权限应为 0600, got %o", got)
	}

	// 内容应为 OpenSSH 私钥 PEM
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 host key 失败: %v", err)
	}
	if !strings.Contains(string(data), "OPENSSH PRIVATE KEY") {
		t.Errorf("应为 OpenSSH 私钥 PEM, 内容前几行: %q", string(data[:min(64, len(data))]))
	}
	// 父目录 sub 应被自动创建
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Errorf("父目录应被自动创建: %v", err)
	}
}

// TestLoadOrCreateReloadRoundTrip 生成后重新加载：公钥必须一致（round trip）。
func TestLoadOrCreateReloadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host_key")

	first, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("首次生成失败: %v", err)
	}
	firstPub, ok := first.Public().(ed25519.PublicKey)
	if !ok {
		t.Fatalf("应为 ed25519 公钥, got %T", first.Public())
	}

	second, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("重载失败: %v", err)
	}
	if _, ok := second.(crypto.Signer); !ok {
		t.Errorf("重载返回值应满足 crypto.Signer, got %T", second)
	}
	secondPub, ok := second.Public().(ed25519.PublicKey)
	if !ok {
		t.Fatalf("重载后应为 ed25519 公钥, got %T", second.Public())
	}
	if string(firstPub) != string(secondPub) {
		t.Errorf("重载后的公钥应一致\nfirst:  %x\nsecond: %x", firstPub, secondPub)
	}
}

// TestLoadOrCreateInvalidFile 已存在但内容非法时应报错，而不是静默覆盖。
func TestLoadOrCreateInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host_key")
	if err := os.WriteFile(path, []byte("不是 PEM"), 0o600); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	if _, err := LoadOrCreate(path); err == nil {
		t.Errorf("非法 PEM 应返回错误")
	}
}
