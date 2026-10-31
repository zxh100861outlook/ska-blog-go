// ska-blog-go —— SSH 博客（Go 重写版）。main 提供的运行模式：
//
//	go run .                 默认: 启动 SSH 服务器（tui 在 Q2/Q3 接入）, SIGINT/SIGTERM 优雅关闭
//	go run . -list           调试: 打印文章索引：[id] 标题  (N 段)
//	go run . -render wj2     调试: 打印指定 ID 文章渲染后的 ANSI 文本
//	go run . -local          调试: 在本终端运行 TUI（Q2 实现）
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	gossh "github.com/gliderlabs/ssh"

	"skablog/internal/content"
	"skablog/internal/render"
	"skablog/internal/server"
	"skablog/internal/tui"
)

func main() {
	var (
		addr     = flag.String("addr", "127.0.0.1:2222", "SSH 监听地址")
		articles = flag.String("articles", "articles/articles.md", "文章数据文件路径")
		guest    = flag.Bool("guest", true, "允许匿名访客")
		list     = flag.Bool("list", false, "调试：打印文章索引")
		renderID = flag.String("render", "", "调试：渲染指定 ID 的文章并打印")
		local    = flag.Bool("local", false, "在本终端运行 TUI（Q2 实现）")
	)
	flag.Parse()

	arts, err := loadArticles(*articles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}

	switch {
	case *list:
		// 与纯文本 SSH 列表共用同一格式函数，保证两端一致
		fmt.Print(server.PlainList(arts))
	case *renderID != "":
		renderArticle(arts, *renderID)
	case *local:
		runLocal(arts)
	default:
		runServer(*addr, *guest, arts)
	}
}

// runLocal 在本终端运行 TUI（Q2/Q3：布局+交互）。
func runLocal(arts []content.Article) {
	user := os.Getenv("USER")
	if user == "" {
		user = "local"
	}
	p := tea.NewProgram(tui.New(arts, user),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "TUI 运行错误:", err)
		os.Exit(1)
	}
}

// runServer 启动 SSH 服务器并阻塞，直到收到 SIGINT/SIGTERM 优雅关闭。
func runServer(addr string, guest bool, arts []content.Article) {
	srv, err := server.Start(server.Config{Addr: addr, Guest: guest, Articles: arts})
	if err != nil {
		fmt.Fprintln(os.Stderr, "服务器启动失败:", err)
		os.Exit(1)
	}

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "收到 %v，正在优雅关闭...\n", sig)
		// Shutdown 优雅等待活动连接；给个短超时防止卡死
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "优雅关闭未完成，强制关闭:", err)
			if err := srv.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "强制关闭失败:", err)
				os.Exit(1)
			}
		}
	case err := <-serveDone:
		// 服务器自行退出（通常是启动失败）；ErrServerClosed 不算错误
		if err != nil && !errors.Is(err, gossh.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, "服务器运行错误:", err)
			os.Exit(1)
		}
	}
}

// loadArticles 读取并解析文章文件。
func loadArticles(path string) ([]content.Article, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文章文件失败: %w", err)
	}
	arts, err := content.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("解析文章失败: %w", err)
	}
	return arts, nil
}

// printList 已由 server.PlainList 取代（保证 SSH 与 -list 格式一致）。

// renderArticle 渲染并打印一篇文章；找不到 ID 时报错退出。
func renderArticle(arts []content.Article, id string) {
	a := content.Find(arts, id)
	if a == nil {
		fmt.Fprintf(os.Stderr, "错误: 找不到文章 %q\n", id)
		os.Exit(1)
	}
	// 文章标题按一级标题渲染，正文原样交给渲染器；按 80 列折行（调试模式默认宽）
	fmt.Println(render.Render("# "+a.Title+"\n\n"+a.Body, 80))
}
