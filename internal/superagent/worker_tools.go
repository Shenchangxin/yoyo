// worker_tools.go 拥有工人代理上的回复、交还和目录准备。
// 不拥有工作区工具本身。那一套由 Mount 登记。

package superagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
)

// ensureWorkDir 准备 ticketID 的目录。没有准备命令时建一个空目录。准备失败时返回错误，调用方结束工单。
func (e *Executor) ensureWorkDir(ticketID string) (string, error) {
	if e == nil || strings.TrimSpace(e.Root) == "" {
		return "", nil
	}
	id := strings.TrimSpace(ticketID)
	if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return "", errCode("bad_ticket_dir")
	}
	if ticket, ok := e.Engine.Ticket(ticketID); ok && ticket.Directory != "" {
		return ticket.Directory, nil
	}
	dir := filepath.Join(e.Root, "tickets", id)
	if e.PrepareCmd != "" {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			e.Engine.SetDirectory(ticketID, dir, id)
			return dir, nil
		}
		if e.RunPrepare == nil {
			return "", errCode("no_prepare")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		env := map[string]string{}
		if e.Env != nil {
			for k, v := range e.Env(dir, ticketID) {
				env[k] = v
			}
		}
		env["DSH_SUPERAGENT_DIR"] = dir
		env["DSH_SUPERAGENT_NAME"] = id
		env["DSH_SUPERAGENT_TICKET"] = ticketID
		out, err := e.RunPrepare(ctx, e.Root, e.PrepareCmd, env)
		if err != nil {
			text := strings.TrimSpace(out)
			if text == "" {
				text = err.Error()
			}
			return "", fmt.Errorf("%s", clip(text, 500))
		}
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	e.Engine.SetDirectory(ticketID, dir, id)
	return dir, nil
}

// workDir 返回已经记下的目录。还没有时退回 ticketDir。
func (e *Executor) workDir(ticketID string) (string, error) {
	if ticket, ok := e.Engine.Ticket(ticketID); ok && ticket.Directory != "" {
		return ticket.Directory, nil
	}
	return e.ticketDir(ticketID)
}

// workerPrompt 是工人的系统提示。env 里的值写出来，钥匙不写。
func workerPrompt(e *Executor, dir, ticketID string) string {
	var b strings.Builder
	b.WriteString("你是工人。在工单目录里完成技术工作，自己判断做法并核对结果。\n")
	b.WriteString("用 orchestrator_report 报告 progress、waiting、ready、blocked、failed 或 dropped。提问放在 question。\n")
	b.WriteString("回复原处用 orchestrator_reply。超出范围用 orchestrator_handback。只跟随上下文用 orchestrator_follow。无关线程用 orchestrator_unrelated。\n")
	if dir != "" {
		fmt.Fprintf(&b, "工作目录 %s\n", dir)
	}
	if e != nil && e.Env != nil {
		for _, key := range []string{
			"DSH_SUPERAGENT_DIR", "DSH_SUPERAGENT_NAME", "DSH_SUPERAGENT_TICKET",
			"DSH_SUPERAGENT_GITHUB_REPOSITORY", "DSH_SUPERAGENT_GITHUB_ACCOUNT",
			"DSH_SUPERAGENT_FEISHU_BOT", "DSH_SUPERAGENT_FEISHU_CHAT",
		} {
			env := e.Env(dir, ticketID)
			if env[key] != "" {
				fmt.Fprintf(&b, "%s=%s\n", key, env[key])
			}
		}
	}
	return b.String()
}

// replyTool 把正文交回触发原来的渠道。
func (e *Executor) replyTool(ticketID string) dsh.Tool {
	return dsh.Tool{
		Name:        "orchestrator_reply",
		Description: "把回复交回触发原来的渠道。purpose 取 acknowledgement、answer、result 或 attention。trigger 是相关触发编号。text 是正文。",
		Mode:        func(dsh.ToolCall) string { return dsh.ModeExclusive },
		Run: func(_ context.Context, call dsh.ToolCall) (dsh.Result, error) {
			var body struct {
				Purpose string `json:"purpose"`
				Trigger string `json:"trigger"`
				Text    string `json:"text"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &body); err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			if _, err := e.Engine.Reply(ticketID, body.Purpose, body.Trigger, body.Text); err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			return dsh.Result{Text: "queued " + body.Purpose}, nil
		},
	}
}

// handbackTool 把超出范围的触发交还主循环。
func (e *Executor) handbackTool(ticketID string) dsh.Tool {
	return dsh.Tool{
		Name:        "orchestrator_handback",
		Description: "这条人的消息超出本工单范围时交还。trigger 是触发编号。reason 说明为什么。",
		Mode:        func(dsh.ToolCall) string { return dsh.ModeExclusive },
		Run: func(_ context.Context, call dsh.ToolCall) (dsh.Result, error) {
			var body struct {
				Trigger string `json:"trigger"`
				Reason  string `json:"reason"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &body); err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			if _, err := e.Engine.Handback(ticketID, body.Trigger, body.Reason); err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			return dsh.Result{Text: "handed back"}, nil
		},
	}
}

// followTool 只为上下文跟随一条线程，不取得人的后续消息。
func (e *Executor) followTool(ticketID string) dsh.Tool {
	return dsh.Tool{
		Name:        "orchestrator_follow",
		Description: "只为上下文跟随 thread。这不让工单拥有该线程上的人的后续消息。",
		Mode:        func(dsh.ToolCall) string { return dsh.ModeExclusive },
		Run: func(_ context.Context, call dsh.ToolCall) (dsh.Result, error) {
			var body struct {
				Thread string `json:"thread"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &body); err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			if _, err := e.Engine.FollowForContext(ticketID, body.Thread); err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			return dsh.Result{Text: "following " + body.Thread}, nil
		},
	}
}

// unrelatedTool 标记一条线程和这张工单无关。
func (e *Executor) unrelatedTool(ticketID string) dsh.Tool {
	return dsh.Tool{
		Name:        "orchestrator_unrelated",
		Description: "标记 thread 与本工单无关。reason 说明原因。",
		Mode:        func(dsh.ToolCall) string { return dsh.ModeExclusive },
		Run: func(_ context.Context, call dsh.ToolCall) (dsh.Result, error) {
			var body struct {
				Thread string `json:"thread"`
				Reason string `json:"reason"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &body); err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			if err := e.Engine.MarkUnrelated(ticketID, body.Thread, body.Reason); err != nil {
				return dsh.Result{Text: err.Error(), IsError: true}, nil
			}
			return dsh.Result{Text: "unrelated"}, nil
		},
	}
}
