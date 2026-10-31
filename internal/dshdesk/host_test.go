// host_test.go 锁住桌面回合能走 dsh 代理：纯文本回复，以及一次写文件会留下工具事件。

package dshdesk

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TestDeskReturnsAssistant 确认没有工具调用时，Run 返回模型正文。
func TestDeskReturnsAssistant(t *testing.T) {
	host := New()
	out, err := host.Run(context.Background(), Turn{
		SessionID: "s",
		User:      "ping",
		Client:    &runtime.ScriptedClient{Steps: []runtime.Message{{Role: runtime.RoleAssistant, Content: "pong"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "pong" {
		t.Fatalf("out %q", out)
	}
}

// TestDeskWritesFile 确认启发式模型的 write_file 会落盘，并发出工具调用事件。
func TestDeskWritesFile(t *testing.T) {
	dir := t.TempDir()
	host := New()
	var saw bool
	out, err := host.Run(context.Background(), Turn{
		SessionID:   "s",
		User:        "Write hello.txt containing hello",
		HarnessHash: "h1",
		Client:      runtime.HeuristicSolver{},
		Tools:       &runtime.WorkspaceTools{Workspace: dir, ChatOverlay: true},
		OnEvent: func(ev trace.Event) {
			if ev.Type == trace.TypeToolCall {
				if ev.HarnessSnapshot != "h1" {
					t.Errorf("hash %q", ev.HarnessSnapshot)
				}
				saw = true
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "done" {
		t.Fatalf("out %q", out)
	}
	body, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" {
		t.Fatalf("body %q", body)
	}
	if !saw {
		t.Fatal("no tool call")
	}
}
