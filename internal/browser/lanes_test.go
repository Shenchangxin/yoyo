// 本文件锁住接管目标优先用预览文件的服务地址，没有预览时留下已有的 http 地址，空白页没有可打开的目标。
// 由 go test 调用。

package browser

import (
	"fmt"
	"testing"
)

// TestTakeoverTargetPrefersServedPreviewOverAboutBlank 锁住 about:blank 加预览路径时用 serve 返回的地址。
func TestTakeoverTargetPrefersServedPreviewOverAboutBlank(t *testing.T) {
	got := takeoverTarget("about:blank", `C:\Users\scx\Desktop\test\pelican-bike\index.html`, func(abs string) (string, error) {
		if abs == "" {
			t.Fatal("preview path empty")
		}
		return "http://127.0.0.1:9/index.html", nil
	})
	if got != "http://127.0.0.1:9/index.html" {
		t.Fatalf("got %q", got)
	}
}

// TestTakeoverTargetKeepsLiveHTTPWhenNoPreview 锁住没有预览时留下已有的 https 地址，并且不调用 serve。
func TestTakeoverTargetKeepsLiveHTTPWhenNoPreview(t *testing.T) {
	got := takeoverTarget("https://example.com/app", "", func(string) (string, error) {
		t.Fatal("should not serve")
		return "", fmt.Errorf("no")
	})
	if got != "https://example.com/app" {
		t.Fatalf("got %q", got)
	}
}

// TestTakeoverTargetEmptyWhenNothingToOpen 锁住只有 about:blank 且没有预览时结果是空字符串。
func TestTakeoverTargetEmptyWhenNothingToOpen(t *testing.T) {
	got := takeoverTarget("about:blank", "", nil)
	if got != "" {
		t.Fatalf("got %q", got)
	}
}

// TestIsBlankURL 锁住空字符串、about:blank 和 about:blank? 开头算空白，http 地址不算。
func TestIsBlankURL(t *testing.T) {
	if !isBlankURL("") || !isBlankURL("about:blank") || !isBlankURL("about:blank?foo") {
		t.Fatal("blank")
	}
	if isBlankURL("http://127.0.0.1:9/index.html") {
		t.Fatal("http")
	}
}
