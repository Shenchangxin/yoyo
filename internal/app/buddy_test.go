// buddy_test.go 锁住 app 对 Buddy 的两处接线：发到 Hub 的事件小到能内联进桌面通知，借给 agent 的 yoyo 钥匙只认官方端点。

package app

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/buddy"
	"github.com/Shenchangxin/yoyo/internal/connection"
)

// TestBuddyNoticeInlines 锁住带长进度行和大图标的视图发出去后，桌面通知仍带着编号、状态和进度。
func TestBuddyNoticeInlines(t *testing.T) {
	a := &App{Hub: NewHub()}
	ch, cancel := a.Hub.Subscribe("")
	defer cancel()
	a.publishBuddy(buddy.View{
		ID: "gemini", Name: "Gemini CLI", State: buddy.StateInstalling, Icon: "data:image/svg+xml;base64," + strings.Repeat("A", 8000),
		Progress: &buddy.Progress{Phase: "deps", Percent: 40, Line: strings.Repeat("npm http fetch GET 200 https://registry.npmjs.org/x ", 40)},
	})
	ev := <-ch
	n := NoticeOf(ev)
	if n.Event == nil {
		t.Fatal("buddy notice was not inlined")
	}
	if n.Event.ItemKind != "buddy" || n.Event.Payload["id"] != "gemini" || n.Event.Payload["state"] != "installing" {
		t.Fatalf("payload %+v", n.Event.Payload)
	}
	if _, ok := n.Event.Payload["icon"]; ok {
		t.Fatal("full view in the notice")
	}
}

// TestOfficialFor 锁住复用只认官方端点：代理或自建端点的同名提供商不算。
func TestOfficialFor(t *testing.T) {
	cases := []struct {
		provider string
		c        connection.Connection
		want     bool
	}{
		{"anthropic", connection.Connection{Vendor: "claude"}, true},
		{"anthropic", connection.Connection{Vendor: "claude", Endpoint: "https://api.anthropic.com/v1"}, true},
		{"anthropic", connection.Connection{Vendor: "claude", Endpoint: "https://proxy.example.com/v1"}, false},
		{"openai", connection.Connection{Vendor: "openai", Endpoint: "https://api.openai.com/v1"}, true},
		{"openai", connection.Connection{Vendor: "openai", Endpoint: "https://api.deepseek.com/v1"}, false},
		{"gemini", connection.Connection{Vendor: "gemini"}, true},
		{"gemini", connection.Connection{Vendor: "openai"}, false},
	}
	for _, c := range cases {
		if got := officialFor(c.provider, c.c); got != c.want {
			t.Fatalf("%s %+v => %v", c.provider, c.c, got)
		}
	}
}
