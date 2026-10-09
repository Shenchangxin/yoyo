package browser

import (
	"fmt"
	"testing"
)

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

func TestTakeoverTargetKeepsLiveHTTPWhenNoPreview(t *testing.T) {
	got := takeoverTarget("https://example.com/app", "", func(string) (string, error) {
		t.Fatal("should not serve")
		return "", fmt.Errorf("no")
	})
	if got != "https://example.com/app" {
		t.Fatalf("got %q", got)
	}
}

func TestTakeoverTargetEmptyWhenNothingToOpen(t *testing.T) {
	got := takeoverTarget("about:blank", "", nil)
	if got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestIsBlankURL(t *testing.T) {
	if !isBlankURL("") || !isBlankURL("about:blank") || !isBlankURL("about:blank?foo") {
		t.Fatal("blank")
	}
	if isBlankURL("http://127.0.0.1:9/index.html") {
		t.Fatal("http")
	}
}
