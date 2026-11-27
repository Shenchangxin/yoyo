// subagent.go 给桌面回合挂上 task 子代理。
// 子代理是一份独立的 agentloop 代理，跑在子会话上，轨迹写进子会话自己的 JSONL。
// 父会话只拿到摘要（由 WorkspaceTools.task 截断并加上 SUBAGENT_SUMMARY 前缀），以及 subagent 卡片的 start 和 complete 两条事件。

package desk

import (
	"context"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/worktree"
)

// attachTask 在 turn.Tools 还没有 Task、且深度没到 runtime.MaxTaskDepth 时，挂上走 agentloop 的子代理。
// 调用方已经设置过 Task（例如测试替身）时不覆盖。ctx 是这一轮的上下文，取消时子代理一起停。
// 没有这一步时，桌面清单里的 task 一调用就返回 subagent not available。
func attachTask(ctx context.Context, turn Turn) {
	tools := turn.Tools
	if tools == nil || tools.Task != nil || tools.Depth >= runtime.MaxTaskDepth {
		return
	}
	tools.Task = func(prompt string, isolate bool, profile string) (string, error) {
		return runTask(ctx, turn, prompt, isolate, profile)
	}
}

// runTask 在 parent 的会话上发出 subagent 卡片的开始事件，跑完子代理后再发完成事件，返回子代理最后一条助手正文。
// 子代理失败时完成事件带上 error，隔离目录建不起来也算失败，卡片照样收尾，不会一直停在进行中。
func runTask(ctx context.Context, parent Turn, prompt string, isolate bool, profile string) (string, error) {
	childID := runtime.ChildSessionID(parent.SessionID)
	parent.emit(trace.Event{Type: trace.TypeSubagent, Source: "runtime", Payload: map[string]any{
		"prompt": prompt, "isolate": isolate, "child": childID, "phase": "start", "profile": profile,
	}}, true)
	out, err := runChild(ctx, parent, childID, prompt, isolate, profile)
	payload := map[string]any{
		"prompt": prompt, "isolate": isolate, "summary": out, "child": childID, "phase": "complete", "profile": profile,
	}
	if err != nil {
		payload["error"] = err.Error()
	}
	parent.emit(trace.Event{Type: trace.TypeSubagent, Source: "runtime", Payload: payload}, true)
	return out, err
}

// runChild 用一个新的 Host 在 childID 上跑子代理。新 Host 不登记在父 Host 里，父会话的 Steer 和 Interrupt 不会找到它；父这一轮的 ctx 取消时它一起停。
// isolate 为真且父工作区非空时，先用 worktree.Isolate 复制一份工作区，跑完清理。
// 子代理沿用父这一轮的系统提示、模型、客户端、计量、轨迹和停止钩子；工具集按 runtime.ChildTools 裁剪；第一条消息是 profile 的角色说明加上 prompt。
// 不传 OnPrompt，子会话的 token 不记到父会话的上下文形状上；费用仍经共享的 Meter 记到父这一轮。
func runChild(ctx context.Context, parent Turn, childID, prompt string, isolate bool, profile string) (string, error) {
	ws := parent.Tools.Workspace
	if isolate && ws != "" {
		iso, cleanup, err := worktree.Isolate(ws)
		if err != nil {
			return "", err
		}
		defer cleanup()
		ws = iso
	}
	return New().Run(ctx, Turn{
		SessionID:        childID,
		TurnID:           diaglog.NewTurnID(),
		Prompt:           parent.Prompt,
		User:             runtime.TaskUser(runtime.TaskProfileInstruction(profile), prompt),
		Model:            parent.Model,
		MaxTokens:        parent.MaxTokens,
		Client:           parent.Client,
		Tools:            runtime.ChildTools(parent.Tools, childID, ws, profile),
		Trace:            parent.Trace,
		OnEvent:          parent.OnEvent,
		ShowThinking:     parent.ShowThinking,
		HarnessHash:      parent.HarnessHash,
		ModelFingerprint: parent.ModelFingerprint,
		Meter:            parent.Meter,
		StopHooks:        parent.StopHooks,
		MaxSteps:         parent.MaxSteps,
		TaskID:           parent.TaskID,
	})
}
