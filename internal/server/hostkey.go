// hostkey.go —— SSH 服务器 host key 的加载与生成。
package server

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	gossh "golang.org/x/crypto/ssh"
)

// LoadOrCreate 读取指定路径的 ed25519 私钥（OpenSSH PEM 格式）；
// 文件不存在时自动生成一把新密钥并落盘。
// 返回值满足 crypto.Signer（ed25519.PrivateKey 同时实现了 Public/Sign）。
func LoadOrCreate(path string) (crypto.Signer, error) {
	// 先尝试读已有密钥
	data, err := os.ReadFile(path)
	if err == nil {
		key, parseErr := parsePrivateKey(data)
		if parseErr != nil {
			return nil, fmt.Errorf("server: 解析 host key %s 失败: %w", path, parseErr)
		}
		return key, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("server: 读取 host key %s 失败: %w", path, err)
	}

	// 不存在则生成新 ed25519 密钥
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("server: 生成 ed25519 密钥失败: %w", err)
	}

	// 序列化为 OpenSSH 私钥 PEM（x/crypto/ssh 提供的官方 API）
	block, err := gossh.MarshalPrivateKey(crypto.PrivateKey(priv), "sshblog host key")
	if err != nil {
		return nil, fmt.Errorf("server: 编码私钥 PEM 失败: %w", err)
	}
	pemBytes := pem.EncodeToMemory(block)

	// 自动创建父目录，文件落盘，权限严格 0600
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("server: 创建 host key 目录失败: %w", err)
	}
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return nil, fmt.Errorf("server: 写入 host key 失败: %w", err)
	}

	return priv, nil
}

// parsePrivateKey 从 PEM 字节解析 ed25519 私钥。
// ParseRawPrivateKey 按 PEM 头自动识别格式，OpenSSH 格式的 ed25519
// 密钥会以 ed25519.PrivateKey 类型返回。
func parsePrivateKey(pemBytes []byte) (ed25519.PrivateKey, error) {
	parsed, err := gossh.ParseRawPrivateKey(pemBytes)
	if err != nil {
		return nil, err
	}
	switch key := parsed.(type) {
	case ed25519.PrivateKey:
		return key, nil
	case *ed25519.PrivateKey:
		return *key, nil
	default:
		return nil, fmt.Errorf("server: 不是 ed25519 私钥（得到 %T）", parsed)
	}
}
