// subagent_test.go 锁住桌面回合里的 task 子代理：父会话拿到摘要和两条 subagent 事件，子会话留下自己的轨迹。

package desk

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// delegateClient 第一次在父会话上要 task，子会话里回一句摘要，父会话看到工具结果后收尾。
type delegateClient struct {
	mu         sync.Mutex
	toolResult string
}

// Chat 按 req 里的消息判断是哪一方在问。ctx 取消时返回 ctx.Err。
func (c *delegateClient) Chat(ctx context.Context, req runtime.ChatRequest) (runtime.Message, error) {
	if err := ctx.Err(); err != nil {
		return runtime.Message{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range req.Messages {
		if m.Role == runtime.RoleTool {
			c.toolResult = m.Content
			return runtime.Message{Role: runtime.RoleAssistant, Content: "parent done"}, nil
		}
	}
	for _, m := range req.Messages {
		if m.Role == runtime.RoleUser && strings.Contains(m.Content, "child-job") {
			return runtime.Message{Role: runtime.RoleAssistant, Content: "child summary"}, nil
		}
	}
	return runtime.Message{Role: runtime.RoleAssistant, ToolCalls: []runtime.ToolCall{{
		ID: "t1", Name: "task", Arguments: `{"prompt":"child-job","profile":"explore"}`,
	}}}, nil
}

// TestDeskTaskRunsSubagent 确认桌面上的 task 不再返回 subagent not available：父会话收到 SUBAGENT_SUMMARY，轨迹里有 start 和 complete 两条 subagent 事件，子会话有自己的助手正文。
func TestDeskTaskRunsSubagent(t *testing.T) {
	store := trace.NewStore(t.TempDir())
	client := &delegateClient{}
	out, err := New().Run(context.Background(), Turn{
		SessionID: "s",
		TurnID:    "tid",
		User:      "delegate",
		Client:    client,
		Tools:     &runtime.WorkspaceTools{Workspace: t.TempDir()},
		Trace:     store,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "parent done" {
		t.Fatalf("out %q", out)
	}
	if !strings.Contains(client.toolResult, "SUBAGENT_SUMMARY") || !strings.Contains(client.toolResult, "child summary") {
		t.Fatalf("tool result %q", client.toolResult)
	}
	saved, err := store.Read("s")
	if err != nil {
		t.Fatal(err)
	}
	var phases []string
	child := ""
	for _, ev := range saved {
		if ev.Type != trace.TypeSubagent {
			continue
		}
		phase, _ := ev.Payload["phase"].(string)
		phases = append(phases, phase)
		child, _ = ev.Payload["child"].(string)
		if ev.Payload["profile"] != "explore" {
			t.Fatalf("profile %+v", ev.Payload)
		}
	}
	if strings.Join(phases, ",") != "start,complete" {
		t.Fatalf("phases %v", phases)
	}
	if !runtime.IsChildSessionID(child) {
		t.Fatalf("child id %q", child)
	}
	childEvents, err := store.Read(child)
	if err != nil {
		t.Fatal(err)
	}
	sawSummary := false
	for _, ev := range childEvents {
		if ev.Type == trace.TypeAssistant && ev.Payload["text"] == "child summary" {
			sawSummary = true
		}
	}
	if !sawSummary {
		t.Fatal("child session has no assistant summary")
	}
}

// TestAttachTaskKeepsCallerTask 确认调用方已经设置的 Task 不被覆盖，深度到上限时不挂子代理。
func TestAttachTaskKeepsCallerTask(t *testing.T) {
	called := false
	mine := func(string, bool, string) (string, error) { called = true; return "mine", nil }
	tools := &runtime.WorkspaceTools{Task: mine}
	attachTask(context.Background(), Turn{SessionID: "s", Tools: tools})
	if _, err := tools.Task("x", false, ""); err != nil || !called {
		t.Fatalf("caller task replaced, called %v err %v", called, err)
	}
	deep := &runtime.WorkspaceTools{Depth: runtime.MaxTaskDepth}
	attachTask(context.Background(), Turn{SessionID: "s", Tools: deep})
	if deep.Task != nil {
		t.Fatal("task attached at max depth")
	}
}
