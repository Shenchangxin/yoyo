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
	b.WriteString("你是工人。默认只在这张工单的工作目录里读写。任务说明里写出的目录也可以读写，直接去做，不要再问能不能动。只写出文件时，只能改这个文件所在的目录。说明里没点名、又必须改到这些路径以外时，才用 orchestrator_report 报 blocked，并在 question 里写明路径。\n")
	b.WriteString("任务说明里已经写明的选择不要再问。说明没写到的细节，按已有文件自己定，做完再核对。一直做到说明里的完成标准。不要在一个小进展之后用 ready、waiting 或 blocked 把下一步交还给人。\n")
	b.WriteString("后来补来的消息可能只是一个变化、一个答案或下一步，不是一份新的完整说明。沿着已有的日志、计划和目录继续，不要因为这句短就停下来要求把整张工单重写一遍。\n")
	b.WriteString("只有三种情况才停下来问：说明没授权的破坏性操作、必须改说明和工单目录以外的文件、缺少密钥或登录。\n")
	b.WriteString("操作者用 @skill:名字 指定技能时，先 load_skill 这个名字。@file:路径 是附上的文件。\n")
	b.WriteString("用 orchestrator_report 报告 progress、waiting、ready、blocked、failed 或 dropped。progress 报完继续做。ready 只在完成标准达到、并且计划里没有 pending 或 in_progress 时使用。计划还开着就先 update_plan。question 只在上面三种硬性挡住时填写。\n")
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
