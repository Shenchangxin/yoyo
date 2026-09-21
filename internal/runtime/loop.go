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
	Checkpoint       string
	Stop             *StopReason
	StopHooks        []FileHook
	PreCompactHooks  []FileHook
	ProfileMemory    string
	// RoundSeq is the highest assistant round already on this session's
	// JSONL (rN). The next chat call uses N+1 so live UI keys never collide
	// with bubbles still on screen after a checkpoint rebuild.
	RoundSeq int
	// SoftHorizon is desktop/CLI chat only (I6). Harbor eval and subagents
	// leave it false so MaxTurns / MaxToolMessages stay hard bounds.
	// Chat stops on end_turn (with an open-plan continue), cancel, budget,
	// or overflow — not because a counter ran out while work was progressing.
	SoftHorizon bool
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
		if req.Tools.OperatorVoice == "" {
			req.Tools.OperatorVoice = OperatorVoice(req.User, req.History)
		}
		if req.Tools.Spill == nil {
			req.Tools.Spill = BindSpill(req.Home, req.Workspace, req.SessionID)
		}
		if req.Tools.Task == nil && req.Tools.Depth < MaxTaskDepth {
			parent := req
			req.Tools.Task = func(prompt string, isolate bool) (string, error) {
				return spawnTask(ctx, parent, prompt, isolate)
			}
		}
		if req.Tools.MaxParallel <= 0 {
			req.Tools.MaxParallel = req.Loop.MaxParallel
		}
	}
	loop := req.Loop
	if req.Policy.Mode == "plan" {
		loop.PlanMode = true
		req.Loop = loop
	}
	kernel := newContextKernel(&req)
	emit(req, trace.TypeSystem, "runtime", map[string]any{"text": kernel.prefix})

	messages, err := kernel.seed()
	if err != nil {
		return "", err
	}
	var last string
	toolCount := 0
	planContinues := 0
	toolsAtPlanContinue := -1
	overflowFails := 0
	roundSeq := req.RoundSeq
	writeHits := map[string]int{}
	lastPlan := planTextOf(req.Tools)
	if roundSeq <= 0 {
		for _, m := range req.History {
			if m.Role == RoleAssistant {
				roundSeq++
			}
		}
	}
	turnLimit := loop.MaxTurns
	if turnLimit <= 0 {
		turnLimit = 24
	}
	for turn := 0; req.SoftHorizon || turn < turnLimit; turn++ {
		if err := ctx.Err(); err != nil {
			// Interrupt landed between a tool batch and the next model call.
			// The UI only learns a turn is over from a terminal event, so do
			// not return silently — same card the mid-stream cancel path shows.
			emit(req, trace.TypeError, "runtime", ClassifyError(err).Payload())
			setStop(&req, StopCancelled)
			return last, stopErr(StopCancelled, err.Error(), err)
		}
		if req.PullSteer != nil {
			if extra := strings.TrimSpace(req.PullSteer()); extra != "" {
				emit(req, trace.TypeUser, "steer", map[string]any{"text": extra, "name": "steer"})
				messages = append(messages, Message{Role: RoleUser, Content: extra})
			}
		}
		if !req.SoftHorizon && loop.MaxToolMessages > 0 && toolCount >= loop.MaxToolMessages {
			nudge := toolBudgetNudge
			if !loop.PlanMode && PlanOpen(planTextOf(req.Tools)) {
				nudge = planContinueNudge
			}
			if !lastUserIs(messages, nudge) {
				messages = append(messages, Message{Role: RoleUser, Content: nudge})
			}
		}
		compacted, toolsJSON, report := kernel.prompt(messages, false)
		if kernel.shouldCheckpoint(report) {
			messages = kernel.checkpoint(messages, "budget", "")
			compacted, toolsJSON, report = kernel.prompt(messages, false)
			report.Trigger = "budget"
		}
		if req.OnShape != nil {
			req.OnShape(report)
		}
		if report.Note != "" {
			emit(req, trace.TypeCompact, "runtime", reportPayload(report, nil))
			if req.Events != nil {
				req.Events.Emit(HookCompact, report.Note)
			}
		}
		chatReq := ChatRequest{Model: req.Model, Messages: compacted, Tools: toolsJSON, CacheKey: PromptCacheKey(req.HarnessHash, kernel.prefix, toolsJSON)}
		roundSeq++
		roundID := fmt.Sprintf("%s:r%d", req.SessionID, roundSeq)
		msg, err := chat(ctx, req, chatReq, roundID)
		if err != nil && isStreamErr(err) {
			msg, err = req.Client.Chat(ctx, chatReq)
		}
		if err != nil && IsContextOverflow(err) {
			overflowFails++
			if overflowFails >= 2 {
				err = stopErr(StopOverflow, "context overflow circuit breaker", err)
			} else {
				messages = kernel.checkpoint(messages, "overflow", "")
				compacted, toolsJSON, report = kernel.prompt(messages, true)
				report.Layers = appendLayer(report.Layers, "overflow")
				report.Note = strings.Join(report.Layers, "+")
				report.Trigger = "overflow"
				emit(req, trace.TypeCompact, "runtime", reportPayload(report, map[string]any{"overflow": overflowFails}))
				if req.OnShape != nil {
					req.OnShape(report)
				}
				chatReq.Messages = compacted
				chatReq.Tools = toolsJSON
				chatReq.CacheKey = PromptCacheKey(req.HarnessHash, kernel.prefix, toolsJSON)
				msg, err = chat(ctx, req, chatReq, roundID)
				if err != nil && IsContextOverflow(err) {
					err = stopErr(StopOverflow, "context overflow circuit breaker", err)
				}
			}
		}
		if err != nil {
			reason := StopModelError
			classErr := err
			if IsContextOverflow(err) {
				reason = StopOverflow
			}
			if ctx.Err() != nil {
				reason = StopCancelled
				classErr = ctx.Err()
			}
			emit(req, trace.TypeError, "model", ClassifyError(classErr).Payload())
			setStop(&req, reason)
			return last, stopErr(reason, "", err)
		}
		overflowFails = 0
		if msg.PromptTokens > 0 {
			report.ProviderPrompt = msg.PromptTokens
			if msg.CachedTokens > 0 {
				report.CachedTokens = msg.CachedTokens
				report.CacheReported = true
			} else if msg.CacheReported {
				report.CacheReported = true
			}
			if req.OnShape != nil {
				req.OnShape(report)
			}
		}
		if req.Meter != nil {
			in, out := messagesTokens(compacted)+toolsJSONTokens(toolsJSON), messageTokens(msg)
			if msg.PromptTokens > 0 {
				in = msg.PromptTokens
			}
			if msg.CompletionTokens > 0 {
				out = msg.CompletionTokens
			}
			req.Meter.Add(in, out)
			if err := req.Meter.Check(loop.MaxBudgetUSD); err != nil {
				emit(req, trace.TypeError, "runtime", ClassifyError(err).Payload())
				setStop(&req, StopBudget)
				return last, stopErr(StopBudget, err.Error(), err)
			}
		}
		if msg.Content != "" {
			last = msg.Content
			emit(req, trace.TypeAssistant, "model", map[string]any{"text": msg.Content, "id": roundID, "round": roundID})
		}
		if len(msg.ToolCalls) == 0 {
			kernel.notes = persistWorkingMemory(req, messages, kernel.notes)
			if extra := planContinueIfOpen(req, loop.PlanMode, toolCount, &planContinues, &toolsAtPlanContinue); extra != "" {
				if req.Tools != nil {
					req.Tools.VerifyHint = true
				}
				if strings.TrimSpace(msg.Content) != "" || len(msg.ToolCalls) > 0 {
					messages = append(messages, msg)
				}
				if !lastUserIs(messages, extra) {
					messages = append(messages, Message{Role: RoleUser, Content: extra})
				}
				continue
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
			if report.CachedTokens > 0 || report.CacheReported {
				end["cached_tokens"] = report.CachedTokens
				end["cache_reported"] = report.CacheReported
			}
			end["cache_stable"] = report.CacheStable
			end["prefix_hash"] = report.PrefixHash
			end["dynamic_at"] = report.DynamicAt
			if len(report.Layers) > 0 {
				end["layers"] = report.Layers
			}
			end["stop"] = string(StopEndTurn)
			setStop(&req, StopEndTurn)
			if block, extra := applyStopHook(req); block {
				messages = append(messages, Message{Role: RoleUser, Content: extra})
				continue
			}
			emit(req, trace.TypeTurnEnd, "runtime", end)
			consolidateMemory(req, ParseNotes(kernel.notes))
			return strings.TrimSpace(msg.Content), nil
		}
		messages = append(messages, msg)
		results := dispatchTools(ctx, req, msg.ToolCalls, roundID, true)
		toolCount += len(msg.ToolCalls)
		messages = append(messages, results...)
		recordProgress(writeHits, msg.ToolCalls, req.Tools, &lastPlan)
		kernel.notes = persistWorkingMemory(req, messages, kernel.notes)
		if stall := rewriteStallNudge(req, writeHits); stall != "" && !lastUserIs(messages, stall) {
			messages = append(messages, Message{Role: RoleUser, Content: stall})
		}
		if inj := middlewareNudge(loop, results); inj != "" {
			messages = append(messages, Message{Role: RoleUser, Content: inj})
		}
	}
	emit(req, trace.TypeError, "runtime", maxTurnsInfo(loop.MaxTurns).Payload())
	setStop(&req, StopMaxTurns)
	return last, maxTurnsErr()
}

func middlewareNudge(loop artifact.LoopPreset, results []Message) string {
	n := loop.MaxRecentToolErrors
	if n <= 0 || strings.TrimSpace(loop.ToolErrorInstruction) == "" {
		return ""
	}
	errs := 0
	for _, m := range results {
		if strings.HasPrefix(m.Content, "ERROR:") {
			errs++
		}
	}
	if errs < n {
		return ""
	}
	return loop.ToolErrorInstruction
}

func spillOf(req RunRequest) *Spill {
	if req.Tools != nil {
		return req.Tools.Spill
	}
	return nil
}

func chat(ctx context.Context, req RunRequest, chatReq ChatRequest, roundID string) (Message, error) {
	if s, ok := req.Client.(Streamer); ok {
		var wg sync.WaitGroup
		msg, err := s.ChatStream(ctx, chatReq, func(d StreamDelta) error {
			if d.Text != "" {
				emit(req, trace.TypeAssistant, "model", map[string]any{"text": d.Text, "delta": true, "id": roundID, "round": roundID})
			}
			if d.ToolDone && d.Tool.Name != "" && readonlyCall(req.Tools, d.Tool.Name) {
				tc := d.Tool
				wg.Add(1)
				go func() {
					defer wg.Done()
					_ = dispatchTools(ctx, req, []ToolCall{tc}, roundID, false)
				}()
			}
			return nil
		})
		wg.Wait()
		return msg, err
	}
	return req.Client.Chat(ctx, chatReq)
}

func dispatchTools(ctx context.Context, req RunRequest, calls []ToolCall, roundID string, recordTrace bool) []Message {
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
		change    *FileChange
		parts     []ContentPart
	}
	jobs := make([]job, n)
	readonly := true
	exclusive := false
	for _, tc := range calls {
		if !readonlyCall(req.Tools, tc.Name) {
			readonly = false
		}
		if exclusiveCall(req.Tools, tc.Name) {
			exclusive = true
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
		if recordTrace {
			emit(req, trace.TypeToolCall, "agent", map[string]any{
				"name": tc.Name, "arguments": tc.Arguments, "id": tc.ID, "round": roundID,
			})
		}
	}
	runExec := func(i int) {
		if req.Tools != nil {
			req.Tools.mu.Lock()
			pre, ok := req.Tools.prefetch[jobs[i].tc.ID]
			if ok {
				delete(req.Tools.prefetch, jobs[i].tc.ID)
			}
			req.Tools.mu.Unlock()
			if ok {
				jobs[i].content = pre.content
				jobs[i].ok = pre.ok
				jobs[i].spillID = pre.spillID
				jobs[i].elapsedMs = pre.elapsedMs
				jobs[i].change = pre.change
				jobs[i].parts = pre.parts
				jobs[i].bytes = len(pre.content)
				return
			}
		}
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
		preview, spillID := ingestToolResult(spillOf(req), j.tc.ID, j.tc.Name, content, ingestBudget(req.Loop), recallNoStub(j.tc.Name, content))
		if j.tc.Name == "shell" {
			writeTerminalFile(req.Workspace, req.SessionID, j.tc.ID, content)
		}
		post := applyPostTool(req.Events, ToolHook{Name: j.tc.Name, Arguments: j.tc.Arguments, SessionID: req.SessionID, Result: preview})
		if post.Result != "" {
			preview = post.Result
		}
		jobs[i].content = preview
		jobs[i].spillID = spillID
		jobs[i].elapsedMs = int(time.Since(started).Milliseconds())
		jobs[i].ok = res.Err == nil
		jobs[i].change = res.FileChange
		jobs[i].parts = res.Parts
	}
	if readonly && n > 1 {
		sem := make(chan struct{}, 8)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := range jobs {
			go func(i int) {
				defer wg.Done()
				sem <- struct{}{}
				runExec(i)
				<-sem
			}(i)
		}
		wg.Wait()
	} else if exclusive {
		// Exclusive means serial (cwd/env races), not a transaction.
		// A failed shell must not cancel a sibling write in the same batch.
		for i := range jobs {
			if err := ctx.Err(); err != nil {
				for j := i; j < n; j++ {
					jobs[j].content = "ERROR: interrupted"
				}
				break
			}
			runExec(i)
		}
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
		if !recordTrace {
			if req.Tools != nil {
				req.Tools.mu.Lock()
				if req.Tools.prefetch == nil {
					req.Tools.prefetch = map[string]prefetchHit{}
				}
				req.Tools.prefetch[j.tc.ID] = prefetchHit{
					content:   j.content,
					elapsedMs: j.elapsedMs,
					spillID:   j.spillID,
					ok:        j.ok,
					change:    j.change,
					parts:     j.parts,
				}
				req.Tools.mu.Unlock()
			}
			out[i] = Message{Role: RoleTool, ToolCallID: j.tc.ID, Name: j.tc.Name, Content: j.content, Parts: j.parts}
			continue
		}
		if j.tc.Name == "load_skill" && j.ok {
			emit(req, trace.TypeInject, "skill", map[string]any{"name": j.tc.Name, "text": j.content, "round": roundID})
		}
		if j.tc.Name == "update_plan" && j.ok {
			emit(req, trace.TypePlan, "agent", map[string]any{"name": j.tc.Name, "text": j.content, "round": roundID, "id": j.tc.ID})
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
		if j.change != nil {
			payload["paths"] = j.change.Paths
			emit(req, trace.TypeFileChange, "tool", map[string]any{
				"id": j.tc.ID, "name": j.tc.Name, "paths": j.change.Paths, "round": roundID,
			})
		}
		emit(req, trace.TypeToolResult, "tool", payload)
		out[i] = Message{Role: RoleTool, ToolCallID: j.tc.ID, Name: j.tc.Name, Content: j.content, Parts: j.parts}
	}
	return out
}

func isReadonlyTool(name string) bool {
	switch name {
	case "read_file", "list_dir", "glob", "grep", "load_skill",
		"git_status", "git_diff", "recall_context", "tool_search",
		"list_skills", "view_image", "update_plan", "wait", "ask_user",
		"office_query", "office_render", "memory_search", "schedule_list",
		"browser_snapshot", "clipboard_read", "project_list", "connector_read",
		"read_thread":
		return true
	default:
		return false
	}
}

func loadedFrom(t *WorkspaceTools) []string {
	if t == nil {
		return nil
	}
	out := t.loadedBodies()
	if body := planFirstOverlay(t); body != "" {
		out = append(out, body)
	}
	if body := verifyArtifactOverlay(t); body != "" {
		out = append(out, body)
	}
	return out
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
	ann := HostAnn(name)
	return ann.ReadOnly
}

func exclusiveCall(tools *WorkspaceTools, name string) bool {
	if name == "shell" || name == "run_skill_script" {
		return true
	}
	if tools != nil && tools.Extra != nil {
		if extra, ok := tools.Extra[name]; ok {
			return extra.Exclusive || extra.OpenWorld
		}
	}
	ann := HostAnn(name)
	return ann.Exclusive || ann.OpenWorld
}

func setStop(req *RunRequest, r StopReason) {
	if req != nil && req.Stop != nil {
		*req.Stop = r
	}
}

func isStreamErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "stream") || strings.Contains(s, "sse")
}

func applyStopHook(req RunRequest) (bool, string) {
	for _, r := range req.StopHooks {
		if r.Deny {
			msg := r.Reason
			if msg == "" {
				msg = "blocked by .yoyo/hooks.json stop hook"
			}
			return true, msg
		}
	}
	if req.Events == nil {
		return false, ""
	}
	out, err := req.Events.Serial(HookStop, lastStopPayload{})
	if err != nil {
		return true, "Stop hook blocked completion: " + err.Error()
	}
	if h, ok := out.(StopHookResult); ok && h.Block {
		msg := h.Message
		if msg == "" {
			msg = "Stop hook asked to continue."
		}
		return true, msg
	}
	return false, ""
}

const maxPlanContinues = 6

func planContinueIfOpen(req RunRequest, planMode bool, toolCount int, continues *int, toolsAt *int) string {
	if planMode || req.Tools == nil {
		return ""
	}
	if !PlanOpen(planTextOf(req.Tools)) {
		return ""
	}
	if continues != nil && *continues >= maxPlanContinues {
		return ""
	}
	if continues != nil && *continues > 0 && toolsAt != nil && toolCount == *toolsAt {
		return ""
	}
	if continues != nil {
		*continues++
	}
	if toolsAt != nil {
		*toolsAt = toolCount
	}
	return planContinueNudge
}

func PlanOpen(plan string) bool {
	n, open := 0, 0
	for _, line := range strings.Split(plan, "\n") {
		st, ok := planLineStatus(line)
		if !ok {
			continue
		}
		n++
		if !planStatusDone(st) {
			open++
		}
	}
	return n > 0 && open > 0
}

func planLineStatus(line string) (string, bool) {
	line = strings.TrimSpace(line)
	dot := strings.IndexByte(line, '.')
	if dot < 0 || dot > 3 {
		return "", false
	}
	rest := strings.TrimSpace(line[dot+1:])
	if !strings.HasPrefix(rest, "[") {
		return "", false
	}
	end := strings.IndexByte(rest, ']')
	if end < 1 {
		return "", false
	}
	return rest[1:end], true
}

func planStatusDone(raw string) bool {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	switch s {
	case "complete", "completed", "done", "finished":
		return true
	default:
		return false
	}
}

func lastUserIs(msgs []Message, text string) bool {
	if len(msgs) == 0 || strings.TrimSpace(text) == "" {
		return false
	}
	last := msgs[len(msgs)-1]
	return last.Role == RoleUser && last.Content == text
}

type lastStopPayload struct{}

type StopHookResult struct {
	Block   bool
	Message string
}

func stripSystem(msgs []Message) []Message {
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == RoleSystem || m.Role == RoleDeveloper || m.Role == RoleMemory {
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
	persist := req.Trace != nil
	if persist && typ == trace.TypeAssistant {
		if delta, _ := payload["delta"].(bool); delta {
			persist = false
		}
	}
	if persist {
		_ = req.Trace.Append(ev)
	}
	if req.OnEvent != nil {
		req.OnEvent(ev)
	}
}
