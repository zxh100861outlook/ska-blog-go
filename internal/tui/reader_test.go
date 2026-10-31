// reader_test.go —— 阅读视图：glamour 渲染 + 兜底路径单测。
package tui

import (
	"errors"
	"strings"
	"testing"

	"skablog/internal/content"
	"skablog/internal/theme"
)

// TestReaderGlamourFallback glamour 失败时应回退 internal/render（fallback 置位、
// 内容仍渲染出正文），恢复后不再兜底。
//
// 说明：glamour 底层 goldmark 对非法 markdown 几乎不报错，错误面在选项构造，
// 因此这里注入一个必然失败的 glamourRender 模拟"构造非法输入"失败路径。
func TestReaderGlamourFallback(t *testing.T) {
	old := glamourRender
	art := content.Article{ID: "t", Title: "标题", Body: "段落**粗体**`码`"}

	// 注入失败渲染器 → 兜底
	glamourRender = func(md string, w int, p theme.Palette) (string, error) {
		return "", errors.New("glamour: 构造失败")
	}
	r := readerModel{}
	r.open(art, 40, 10, theme.Get("默认"))
	if !r.fallback {
		t.Fatal("glamour 失败应触发兜底")
	}
	if got := stripANSI(r.vp.View()); !strings.Contains(got, "段落") || !strings.Contains(got, "粗体") {
		t.Errorf("兜底应输出 internal/render 正文, got %q", got)
	}

	// 恢复真实渲染器 → 走 glamour（不兜底）
	glamourRender = old
	r2 := readerModel{}
	r2.open(art, 40, 10, theme.Get("默认"))
	if r2.fallback {
		t.Error("glamour 正常时应不兜底")
	}
	if got := stripANSI(r2.vp.View()); !strings.Contains(got, "段落") {
		t.Errorf("glamour 输出应含正文, got %q", got)
	}
}
