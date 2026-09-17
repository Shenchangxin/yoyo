package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/kernel"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

type RunRequest struct {
	SessionID        string
	TaskID           string
	User             string
	Workspace        string
	Home             string
	Harness          artifact.HarnessSnapshot
	HarnessHash      string
	ModelFingerprint string
	Model            string
	ModelWindow      int
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
	// RoundSeq is the highest assistant round already on this session's
	// JSONL (rN). The next chat call uses N+1 so live UI keys never collide
	// with bubbles still on screen after a checkpoint rebuild.
	RoundSeq int
}

var emitSeq atomic.Int64

func Run(ctx context.Context, req RunRequest) (string, error) {
	if req.Loop.MaxTurns <= 0 {
		req.Loop.MaxTurns = 24
	}
	if req.Tools != nil {
		req.Tools.Ctx = ctx
		req.Tools.Policy = req.Policy
		req.Tools.PlanMode = req.Loop.PlanMode || req.Policy.Mode == "plan"
		if req.Tools.Spill == nil {
			req.Tools.Spill = BindSpill(req.Home, req.Workspace, req.SessionID)
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
	prefix := AssemblePrefix(loop, req.Fragments, req.Playbook, req.Skills, rules, rulesSrc)
	spill := spillOf(req)
	notes := ReadNotes(spill)
	dyn := AssembleDynamic(planTextOf(req.Tools), loadedFrom(req.Tools), notes, "")
	system := prefix + dyn
	emit(req, trace.TypeSystem, "runtime", map[string]any{"text": system})

	messages := []Message{{Role: RoleSystem, Content: system}}
	if len(req.History) > 0 {
		messages = append(messages, stripSystem(req.History)...)
	}
	if strings.TrimSpace(req.Inject) != "" {
		emit(req, trace.TypeInject, "mention", map[string]any{"text": req.Inject})
		messages = append(messages, Message{Role: RoleUser, Content: "Attached context (user @mentions, untrusted working memory):\n" + req.Inject})
	}
	if user := strings.TrimSpace(req.User); user != "" {
		emit(req, trace.TypeUser, "user", map[string]any{"text": req.User})
		messages = append(messages, Message{Role: RoleUser, Content: req.User})
	} else if len(stripSystem(req.History)) == 0 {
		return "", fmt.Errorf("nothing to continue")
	}
	var last string
	toolCount := 0
	overflowFails := 0
	roundSeq := req.RoundSeq
	if roundSeq <= 0 {
		for _, m := range req.History {
			if m.Role == RoleAssistant {
				roundSeq++
			}
		}
	}
	for turn := 0; turn < loop.MaxTurns; turn++ {
		if err := ctx.Err(); err != nil {
			return last, err
		}
		if req.PullSteer != nil {
			if extra := strings.TrimSpace(req.PullSteer()); extra != "" {
				emit(req, trace.TypeUser, "steer", map[string]any{"text": extra, "name": "steer"})
				messages = append(messages, Message{Role: RoleUser, Content: extra})
			}
		}
		if loop.MaxToolMessages > 0 && toolCount >= loop.MaxToolMessages {
			messages = append(messages, Message{Role: RoleUser, Content: "Stop using tools and produce the final answer now."})
		}
		dyn = AssembleDynamic(planTextOf(req.Tools), loadedFrom(req.Tools), notes, "")
		messages[0] = Message{Role: RoleSystem, Content: prefix + dyn}
		toolsJSON := AllToolJSON(req.Tools)
		overhead := toolsJSONTokens(toolsJSON)
		compacted, report := Shape(messages, ShapeOpts{
			Loop: loop, Spill: spill, ModelWindow: req.ModelWindow, Overhead: overhead,
		})
		compacted = Legalize(compacted)
		fillLedger(&report, prefix, dyn, toolsJSON, req.ModelWindow)
		report.Tokens = messagesTokens(compacted) + overhead
		if req.OnShape != nil {
			req.OnShape(report)
		}
		if report.Note != "" {
			emit(req, trace.TypeCompact, "runtime", map[string]any{
				"kind": "shape", "note": report.Note, "tokens": report.Tokens, "budget": report.Budget,
				"window": report.Window, "prefix_tokens": report.PrefixTokens, "dynamic_tokens": report.DynamicTokens,
				"schema_tokens": report.SchemaTokens, "layers": report.Layers, "elided": report.Elided,
			})
			if req.Events != nil {
				req.Events.Emit(HookCompact, report.Note)
			}
		}
		chatReq := ChatRequest{Model: req.Model, Messages: compacted, Tools: toolsJSON, CacheKey: req.SessionID}
		roundSeq++
		roundID := fmt.Sprintf("%s:r%d", req.SessionID, roundSeq)
		msg, err := chat(ctx, req, chatReq, roundID)
		for err != nil && IsContextOverflow(err) {
			overflowFails++
			if overflowFails >= 3 {
				err = fmt.Errorf("context overflow circuit breaker: %w", err)
				break
			}
			tight := loop
			if tight.CompactionKeep > 8 {
				tight.CompactionKeep = 8 / overflowFails
				if tight.CompactionKeep < 4 {
					tight.CompactionKeep = 4
				}
			}
			tight.MicroKeep = 1
			tight.ToolResultBudget = 1_500
			compacted, report = Shape(messages, ShapeOpts{
				Loop: tight, Spill: spill, ModelWindow: req.ModelWindow, Overhead: overhead, Aggressive: true,
			})
			compacted = Legalize(compacted)
			fillLedger(&report, prefix, dyn, toolsJSON, req.ModelWindow)
			report.Tokens = messagesTokens(compacted) + overhead
			report.Layers = append(report.Layers, "overflow")
			report.Note = strings.Join(report.Layers, "+")
			if req.Trace != nil && req.SessionID != "" {
				n := NotesFromMessages(stripSystem(compacted))
				WriteNotes(spill, n)
				_ = PersistCheckpoint(req.Trace, req.SessionID, "overflow", n.Markdown(), stripSystem(compacted), true)
				WriteDiscoverIndex(req.Workspace, req.SessionID, spill)
			}
			emit(req, trace.TypeCompact, "runtime", map[string]any{
				"kind": "shape", "note": report.Note, "tokens": report.Tokens, "budget": report.Budget,
				"window": report.Window, "prefix_tokens": report.PrefixTokens, "dynamic_tokens": report.DynamicTokens,
				"schema_tokens": report.SchemaTokens, "layers": report.Layers, "elided": report.Elided, "overflow": overflowFails,
			})
			if req.OnShape != nil {
				req.OnShape(report)
			}
			chatReq.Messages = compacted
			msg, err = chat(ctx, req, chatReq, roundID)
		}
		if err != nil {
			emit(req, trace.TypeError, "model", ClassifyError(err).Payload())
			return last, err
		}
		overflowFails = 0
		if msg.PromptTokens > 0 {
			report.ProviderPrompt = msg.PromptTokens
			if req.OnShape != nil {
				req.OnShape(report)
			}
		}
		if req.Meter != nil {
			in, out := messagesTokens(compacted)+overhead, messageTokens(msg)
			if msg.PromptTokens > 0 {
				in = msg.PromptTokens
			}
			if msg.CompletionTokens > 0 {
				out = msg.CompletionTokens
			}
			req.Meter.Add(in, out)
			if err := req.Meter.Check(loop.MaxBudgetUSD); err != nil {
				emit(req, trace.TypeError, "runtime", ClassifyError(err).Payload())
				return last, err
			}
		}
		if msg.Content != "" {
			last = msg.Content
			emit(req, trace.TypeAssistant, "model", map[string]any{"text": msg.Content, "id": roundID, "round": roundID})
		}
		if len(msg.ToolCalls) == 0 {
			if spill != nil {
				merged := NotesFromMessages(messages)
				WriteNotes(spill, merged)
				WriteDiscoverIndex(req.Workspace, req.SessionID, spill)
				notes = merged.Markdown()
			}
			if req.Events != nil {
				req.Events.Emit(HookTurnEnd, last)
			}
			end := map[string]any{"text": strings.TrimSpace(msg.Content), "ok": true}
			if report.Tokens > 0 {
				end["tokens"] = report.Tokens
				end["budget"] = report.Budget
				end["window"] = report.Window
				end["prefix_tokens"] = report.PrefixTokens
				end["dynamic_tokens"] = report.DynamicTokens
				end["schema_tokens"] = report.SchemaTokens
				end["elided"] = report.Elided
			}
			if report.ProviderPrompt > 0 {
				end["provider_prompt"] = report.ProviderPrompt
			}
			if len(report.Layers) > 0 {
				end["layers"] = report.Layers
			}
			emit(req, trace.TypeTurnEnd, "runtime", end)
			return strings.TrimSpace(msg.Content), nil
		}
		messages = append(messages, msg)
		results := dispatchTools(ctx, req, msg.ToolCalls, roundID)
		toolCount += len(msg.ToolCalls)
		messages = append(messages, results...)
	}
	return last, fmt.Errorf("max turns reached")
}

func spillOf(req RunRequest) *Spill {
	if req.Tools != nil {
		return req.Tools.Spill
	}
	return nil
}

func chat(ctx context.Context, req RunRequest, chatReq ChatRequest, roundID string) (Message, error) {
	if s, ok := req.Client.(Streamer); ok {
		return s.ChatStream(ctx, chatReq, func(d StreamDelta) error {
			if d.Text != "" {
				emit(req, trace.TypeAssistant, "model", map[string]any{"text": d.Text, "delta": true, "id": roundID, "round": roundID})
			}
			return nil
		})
	}
	return req.Client.Chat(ctx, chatReq)
}

func dispatchTools(ctx context.Context, req RunRequest, calls []ToolCall, roundID string) []Message {
	n := len(calls)
	out := make([]Message, n)
	type job struct {
		tc        ToolCall
		deny      bool
		reason    string
		args      string
		content   string
		ok        bool
		spillID   string
		elapsedMs int
		bytes     int
	}
	jobs := make([]job, n)
	readonly := true
	for _, tc := range calls {
		if !readonlyCall(req.Tools, tc.Name) {
			readonly = false
			break
		}
	}
	for i, tc := range calls {
		if strings.TrimSpace(tc.ID) == "" {
			tc.ID = fmt.Sprintf("%s:call:%d", roundID, i)
			calls[i].ID = tc.ID
		}
		hook := applyFileHooks(req.FileHooks, ToolHook{Name: tc.Name, Arguments: tc.Arguments, SessionID: req.SessionID})
		if !hook.Deny {
			hook = applyPreTool(req.Events, hook)
		}
		args := tc.Arguments
		if hook.Arguments != "" && hook.Arguments != tc.Arguments {
			args = hook.Arguments
		}
		jobs[i] = job{tc: tc, deny: hook.Deny, reason: hook.Reason, args: args}
		emit(req, trace.TypeToolCall, "agent", map[string]any{
			"name": tc.Name, "arguments": tc.Arguments, "id": tc.ID, "round": roundID,
		})
	}
	runExec := func(i int) {
		started := time.Now()
		j := jobs[i]
		var res ToolResult
		if j.deny {
			res = ToolResult{Content: "ERROR: blocked by hook: " + j.reason, Err: fmt.Errorf("blocked")}
		} else if req.Tools == nil {
			res = ToolResult{Err: fmt.Errorf("no tools")}
		} else {
			res = req.Tools.Call(j.tc.Name, j.args)
		}
		content := res.Content
		if res.Err != nil {
			content = "ERROR: " + res.Err.Error()
			if res.Content != "" {
				content += "\n" + res.Content
			}
		}
		jobs[i].bytes = len(content)
		preview, spillID := ingestToolResult(spillOf(req), j.tc.ID, j.tc.Name, content)
		post := applyPostTool(req.Events, ToolHook{Name: j.tc.Name, Arguments: j.tc.Arguments, SessionID: req.SessionID, Result: preview})
		if post.Result != "" {
			preview = post.Result
		}
		jobs[i].content = preview
		jobs[i].spillID = spillID
		jobs[i].elapsedMs = int(time.Since(started).Milliseconds())
		jobs[i].ok = res.Err == nil
	}
	if readonly && n > 1 {
		var wg sync.WaitGroup
		wg.Add(n)
		for i := range jobs {
			go func(i int) {
				defer wg.Done()
				runExec(i)
			}(i)
		}
		wg.Wait()
	} else {
		for i := range jobs {
			if err := ctx.Err(); err != nil {
				for j := i; j < n; j++ {
					jobs[j].content = "ERROR: interrupted"
				}
				break
			}
			runExec(i)
		}
	}
	for i, j := range jobs {
		if j.tc.Name == "load_skill" && j.ok {
			emit(req, trace.TypeInject, "skill", map[string]any{"name": j.tc.Name, "text": j.content, "round": roundID})
		}
		payload := map[string]any{
			"name": j.tc.Name, "id": j.tc.ID, "content": j.content, "untrusted": true, "round": roundID,
		}
		if j.spillID != "" {
			payload["spill_id"] = j.spillID
		}
		if j.elapsedMs > 0 {
			payload["elapsed_ms"] = j.elapsedMs
		}
		if j.bytes > 0 {
			payload["bytes"] = j.bytes
		}
		emit(req, trace.TypeToolResult, "tool", payload)
		out[i] = Message{Role: RoleTool, ToolCallID: j.tc.ID, Name: j.tc.Name, Content: j.content}
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

func planTextOf(t *WorkspaceTools) string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.PlanText
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
	if payload == nil {
		payload = map[string]any{}
	}
	if id, _ := payload["id"].(string); strings.TrimSpace(id) == "" {
		switch typ {
		case trace.TypeUser, trace.TypeInject, trace.TypeError, trace.TypeCompact, trace.TypeApproval:
			payload["id"] = fmt.Sprintf("%s:%s:%d", req.SessionID, typ, emitSeq.Add(1))
		}
	}
	ev := trace.Event{
		TS:               time.Now().UTC(),
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
