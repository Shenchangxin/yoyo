package runtime

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/kernel"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

type RunRequest struct {
	SessionID        string
	TaskID           string
	User             string
	Workspace        string
	Harness          artifact.HarnessSnapshot
	HarnessHash      string
	ModelFingerprint string
	Model            string
	Loop             artifact.LoopPreset
	Policy           artifact.PolicyPack
	Fragments        []artifact.PromptFragment
	Playbook         artifact.Playbook
	Skills           []artifact.Skill
	History          []Message
	Tools            *WorkspaceTools
	Client           Client
	Trace            *trace.Store
	Events           *kernel.EventBus
	FileHooks        []FileHook
	OnEvent          func(trace.Event)
	OnShape          func(ShapeReport)
	Meter            *Meter
	Inject           string
	PullSteer        func() string
}

func Run(ctx context.Context, req RunRequest) (string, error) {
	if req.Loop.MaxTurns <= 0 {
		req.Loop.MaxTurns = 24
	}
	if req.Tools != nil {
		req.Tools.Ctx = ctx
		req.Tools.Policy = req.Policy
		req.Tools.PlanMode = req.Loop.PlanMode || req.Policy.Mode == "plan"
		if req.Tools.Spill == nil && req.Workspace != "" {
			req.Tools.Spill = NewSpill(filepath.Join(req.Workspace, ".yoyo", "context", req.SessionID))
		}
		if req.Tools.Task == nil && req.Tools.Depth == 0 {
			parent := req
			req.Tools.Task = func(prompt string, isolate bool) (string, error) {
				return spawnTask(ctx, parent, prompt, isolate)
			}
		}
	}
	loop := req.Loop
	if req.Policy.Mode == "plan" {
		loop.PlanMode = true
	}
	rules, rulesSrc := LoadWorkspaceRules(req.Workspace, loop.RulesTokens)
	system := Assemble(loop, req.Fragments, req.Playbook, req.Skills, rules, rulesSrc, nil)
	emit(req, trace.TypeSystem, "runtime", map[string]any{"text": system})
	emit(req, trace.TypeUser, "user", map[string]any{"text": req.User})

	messages := []Message{{Role: RoleSystem, Content: system}}
	if len(req.History) > 0 {
		messages = append(messages, stripSystem(req.History)...)
	}
	if strings.TrimSpace(req.Inject) != "" {
		emit(req, trace.TypeInject, "mention", map[string]any{"text": req.Inject})
		messages = append(messages, Message{Role: RoleUser, Content: "Attached context (user @mentions, untrusted working memory):\n" + req.Inject})
	}
	messages = append(messages, Message{Role: RoleUser, Content: req.User})
	toolsJSON := AllToolJSON(req.Tools)
	var last string
	toolCount := 0
	for turn := 0; turn < loop.MaxTurns; turn++ {
		if err := ctx.Err(); err != nil {
			return last, err
		}
		if req.PullSteer != nil {
			if extra := strings.TrimSpace(req.PullSteer()); extra != "" {
				emit(req, trace.TypeUser, "steer", map[string]any{"text": extra})
				messages = append(messages, Message{Role: RoleUser, Content: extra})
			}
		}
		if loop.MaxToolMessages > 0 && toolCount >= loop.MaxToolMessages {
			messages = append(messages, Message{Role: RoleUser, Content: "Stop using tools and produce the final answer now."})
		}
		system = Assemble(loop, req.Fragments, req.Playbook, req.Skills, rules, rulesSrc, loadedFrom(req.Tools))
		messages[0] = Message{Role: RoleSystem, Content: system}
		var spill *Spill
		if req.Tools != nil {
			spill = req.Tools.Spill
		}
		compacted, report := Shape(messages, ShapeOpts{Loop: loop, Spill: spill})
		if report.Note != "" || turn == 0 {
			emit(req, trace.TypeCompact, "runtime", map[string]any{
				"note": report.Note, "tokens": report.Tokens, "budget": report.Budget,
				"layers": report.Layers, "elided": report.Elided,
			})
			if req.OnShape != nil {
				req.OnShape(report)
			}
			if req.Events != nil && report.Note != "" {
				req.Events.Emit(HookCompact, report.Note)
			}
		}
		chatReq := ChatRequest{Model: req.Model, Messages: compacted, Tools: toolsJSON}
		msg, err := chat(ctx, req, chatReq)
		if err != nil {
			emit(req, trace.TypeError, "model", map[string]any{"error": err.Error()})
			return last, err
		}
		if req.Meter != nil {
			req.Meter.Add(messagesTokens(compacted), messageTokens(msg))
			if err := req.Meter.Check(loop.MaxBudgetUSD); err != nil {
				emit(req, trace.TypeError, "runtime", map[string]any{"error": err.Error()})
				return last, err
			}
		}
		if msg.Content != "" {
			last = msg.Content
			emit(req, trace.TypeAssistant, "model", map[string]any{"text": msg.Content})
		}
		if len(msg.ToolCalls) == 0 {
			if req.Events != nil {
				req.Events.Emit(HookTurnEnd, last)
			}
			emit(req, trace.TypeTurnEnd, "runtime", map[string]any{"text": strings.TrimSpace(msg.Content), "ok": true})
			return strings.TrimSpace(msg.Content), nil
		}
		messages = append(messages, msg)
		results := dispatchTools(ctx, req, msg.ToolCalls)
		toolCount += len(msg.ToolCalls)
		messages = append(messages, results...)
	}
	return last, fmt.Errorf("max turns reached")
}

func chat(ctx context.Context, req RunRequest, chatReq ChatRequest) (Message, error) {
	if s, ok := req.Client.(Streamer); ok {
		return s.ChatStream(ctx, chatReq, func(d StreamDelta) error {
			if d.Text != "" {
				emit(req, trace.TypeAssistant, "model", map[string]any{"text": d.Text, "delta": true})
			}
			return nil
		})
	}
	return req.Client.Chat(ctx, chatReq)
}

func dispatchTools(ctx context.Context, req RunRequest, calls []ToolCall) []Message {
	n := len(calls)
	out := make([]Message, n)
	readonly := true
	for _, tc := range calls {
		if !readonlyCall(req.Tools, tc.Name) {
			readonly = false
			break
		}
	}
	runOne := func(i int) {
		tc := calls[i]
		hook := applyFileHooks(req.FileHooks, ToolHook{Name: tc.Name, Arguments: tc.Arguments, SessionID: req.SessionID})
		if !hook.Deny {
			hook = applyPreTool(req.Events, hook)
		}
		emit(req, trace.TypeToolCall, "agent", map[string]any{"name": tc.Name, "arguments": tc.Arguments, "id": tc.ID})
		var res ToolResult
		if hook.Deny {
			res = ToolResult{Content: "ERROR: blocked by hook: " + hook.Reason, Err: fmt.Errorf("blocked")}
		} else {
			args := tc.Arguments
			if hook.Arguments != "" && hook.Arguments != tc.Arguments {
				args = hook.Arguments
			}
			if req.Tools == nil {
				res = ToolResult{Err: fmt.Errorf("no tools")}
			} else {
				res = req.Tools.Call(tc.Name, args)
			}
		}
		content := res.Content
		if res.Err != nil {
			content = "ERROR: " + res.Err.Error()
			if res.Content != "" {
				content += "\n" + res.Content
			}
		}
		capped, trunc := capText(content, defaultToolResultRunes)
		if trunc {
			content = capped
		}
		post := applyPostTool(req.Events, ToolHook{Name: tc.Name, Arguments: tc.Arguments, SessionID: req.SessionID, Result: content})
		if post.Result != "" {
			content = post.Result
		}
		if tc.Name == "load_skill" && res.Err == nil {
			emit(req, trace.TypeInject, "skill", map[string]any{"name": tc.Name, "text": content})
		}
		emit(req, trace.TypeToolResult, "tool", map[string]any{"name": tc.Name, "id": tc.ID, "content": content, "untrusted": true})
		out[i] = Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Name, Content: content}
	}
	if readonly && n > 1 {
		var wg sync.WaitGroup
		wg.Add(n)
		for i := range calls {
			go func(i int) {
				defer wg.Done()
				runOne(i)
			}(i)
		}
		wg.Wait()
		_ = ctx
		return out
	}
	for i := range calls {
		if err := ctx.Err(); err != nil {
			for j := i; j < n; j++ {
				out[j] = Message{Role: RoleTool, ToolCallID: calls[j].ID, Name: calls[j].Name, Content: "ERROR: interrupted"}
			}
			break
		}
		runOne(i)
	}
	return out
}

func isReadonlyTool(name string) bool {
	switch name {
	case "read_file", "list_dir", "glob", "grep", "load_skill",
		"git_status", "git_diff", "recall_context", "tool_search",
		"list_skills", "view_image", "update_plan", "wait":
		return true
	default:
		return false
	}
}

func loadedFrom(t *WorkspaceTools) []string {
	if t == nil {
		return nil
	}
	return t.loadedBodies()
}

func readonlyCall(tools *WorkspaceTools, name string) bool {
	if isReadonlyTool(name) {
		return true
	}
	if tools != nil && tools.Extra != nil {
		if extra, ok := tools.Extra[name]; ok {
			return extra.ReadOnly
		}
	}
	return false
}

func stripSystem(msgs []Message) []Message {
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == RoleSystem {
			continue
		}
		out = append(out, m)
	}
	return out
}

func emit(req RunRequest, typ trace.EventType, source string, payload map[string]any) {
	ev := trace.Event{
		Type:             typ,
		Source:           source,
		SessionID:        req.SessionID,
		HarnessSnapshot:  req.HarnessHash,
		ModelFingerprint: req.ModelFingerprint,
		TaskID:           req.TaskID,
		Payload:          payload,
	}
	if req.Trace != nil {
		_ = req.Trace.Append(ev)
	}
	if req.OnEvent != nil {
		req.OnEvent(ev)
	}
}
