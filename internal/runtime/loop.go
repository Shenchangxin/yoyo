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
	"github.com/Shenchangxin/yoyo/internal/observe"
	"github.com/Shenchangxin/yoyo/internal/skillpack"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

type RunRequest struct {
	SessionID        string
	TurnID           string
	TraceID          string
	TaskID           string
	User             string
	UserParts        []ContentPart
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
	OnChild          func(id, prompt string, done bool)
	OnShape          func(ShapeReport)
	Meter            *Meter
	Observe          *observe.Tracer
	Inject           string
	PullSteer        func() string
	Checkpoint       string
	Stop             *StopReason
	StopHooks        []FileHook
	PreCompactHooks  []FileHook
	ProfileMemory    string
	PageContext      string
	// RoundSeq is the highest assistant round already on this session's
	// JSONL (rN). The next chat call uses N+1 so live UI keys never collide
	// with bubbles still on screen after a checkpoint rebuild.
	RoundSeq int
	// SoftHorizon is desktop/CLI chat only (I6). Harbor eval and subagents
	// leave it false so MaxTurns / MaxToolMessages stay hard bounds.
	// Chat stops on end_turn (with an open-plan continue), cancel, budget,
	// overflow, a repeating-tool / unchanged-observation circuit breaker,
	// or ChatSoftGuardTurns — not because Harbor MaxTurns ran out while
	// the observation was still changing.
	SoftHorizon bool
	// ShowThinking streams provider reasoning into the transcript. Off by
	// default: many OpenAI-compatible endpoints reject thinking fields.
	ShowThinking bool
	// MaxOutputTokens caps one model reply. Zero leaves the provider default.
	// Desktop chat sets it from the model catalog so a long file body fits one
	// reply; when a reply is still cut off, the loop refuses the partial tool
	// call and asks for it in parts (see truncation.go).
	MaxOutputTokens int
	// UserCommitted means the operator row is already on JSONL. seed() still
	// appends it to the model transcript but must not emit a second TypeUser.
	UserCommitted bool
	// RequestID is the compositor's optimistic send id. Seed stamps it on TypeUser
	// so the renderer can replace the local echo instead of doubling the bubble.
	RequestID string
}

var emitSeq atomic.Int64

func Run(ctx context.Context, req RunRequest) (string, error) {
	if req.Loop.MaxTurns <= 0 {
		req.Loop.MaxTurns = 24
	}
	if req.Tools != nil {
		req.Tools.Ctx = ctx
		req.Tools.OnLive = req.OnEvent
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
			req.Tools.Task = func(prompt string, isolate bool, profile string) (string, error) {
				return spawnTask(ctx, parent, prompt, isolate, profile)
			}
		}
		if req.Tools.MaxParallel <= 0 {
			req.Tools.MaxParallel = req.Loop.MaxParallel
		}
		ApplyPackBootstrap(req.Tools)
		if extra := applySessionStart(req.Events, SessionStart{
			SessionID: req.SessionID,
			Workspace: req.Workspace,
			Depth:     req.Tools.Depth,
		}); extra.AdditionalContext != "" {
			if req.Inject != "" {
				req.Inject = extra.AdditionalContext + "\n" + req.Inject
			} else {
				req.Inject = extra.AdditionalContext
			}
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
	const turnNo = 1
	emitTurnStart(req, turnNo)
	var last string
	toolCount := 0
	planContinues := 0
	askNudges := 0
	languageRewrites := 0
	clickHits := 0
	toolsAtPlanContinue := -1
	overflowFails := 0
	roundSeq := req.RoundSeq
	writeHits := map[string]int{}
	failHits := map[string]int{}
	errorHits := map[string]int{}
	waitHits := map[string]int{}
	idleHits := map[string]int{}
	var watch checkWatch
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
	// sinceProgress counts model rounds since the last plan step or changed file,
	// including rounds that produced no tool call. The desktop cap is a
	// checkpoint while that number stays inside ChatStallWindow, and a stop
	// once it does not. Counting only tool rounds let a cut-off reply extend
	// the run forever.
	sinceProgress := 0
	repeats := newRepeatTracker()
	for turn := 0; req.SoftHorizon || turn < turnLimit; turn++ {
		if req.SoftHorizon {
			reset, stop := softGuard(turn, sinceProgress)
			if stop {
				emit(req, trace.TypeError, "runtime", maxTurnsInfo(ChatSoftGuardTurns).Payload())
				setStop(&req, StopMaxTurns)
				return last, maxTurnsErr()
			}
			if reset {
				emit(req, trace.TypeCompact, "runtime", map[string]any{"kind": "budget_continue", "since_progress": sinceProgress, "round": roundSeq})
				turn = 0
			}
		}
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
				emit(req, trace.TypeUser, "steer", map[string]any{"text": extra, "name": "steer", "lane": "next-step"})
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
			// The per-round projection ledger is sampled. Logging it every round
			// wrote one record per model call (257 in session facb2e21ba0b31a3)
			// without adding information. Checkpoints always log.
			if roundSeq%shapeLogEvery == 0 || report.Trigger != "" {
				emit(req, trace.TypeCompact, "runtime", reportPayload(report, nil))
			}
			if req.Events != nil {
				req.Events.Emit(HookCompact, report.Note)
			}
		}
		chatReq := ChatRequest{Model: req.Model, Messages: compacted, Tools: toolsJSON, CacheKey: PromptCacheKey(req.HarnessHash, kernel.prefix, toolsJSON), ShowThinking: req.ShowThinking, MaxTokens: req.MaxOutputTokens}
		roundSeq++
		roundID := fmt.Sprintf("%s:r%d", req.SessionID, roundSeq)
		emitStepStart(req, turnNo, roundSeq, roundID, roundID)
		msg, att, err := completeChat(ctx, req, chatReq, roundID, turnNo, roundSeq)
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
				retryID := roundID + ":retry"
				msg, att, err = completeChat(ctx, req, chatReq, retryID, turnNo, roundSeq)
				if err != nil && IsContextOverflow(err) {
					err = stopErr(StopOverflow, "context overflow circuit breaker", err)
				}
			}
		}
		if err != nil {
			emitStepEnd(req, turnNo, roundSeq, roundID)
			logModelRound(ctx, roundSeq, msg, err)
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
		logModelRound(ctx, roundSeq, msg, nil)
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
			// One record per model call. The turn_end row carries only the last
			// call, so it cannot show whether the prompt cache held across a turn.
			emit(req, trace.TypeUsage, "model", map[string]any{
				"round":             roundID,
				"prompt_tokens":     msg.PromptTokens,
				"completion_tokens": msg.CompletionTokens,
				"cached_tokens":     msg.CachedTokens,
				"cache_reported":    report.CacheReported,
				"cache_stable":      report.CacheStable,
				"elapsed_ms":        msg.ElapsedMs,
				"ttft_ms":           msg.FirstTokenMs,
			})
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
				if msg.Content != "" {
					last = msg.Content
				}
				emitSettlement(req, att, msg)
				emitStepEnd(req, turnNo, roundSeq, roundID)
				emit(req, trace.TypeError, "runtime", ClassifyError(err).Payload())
				setStop(&req, StopBudget)
				return last, stopErr(StopBudget, err.Error(), err)
			}
		}
		msg, cutNudge := trimTruncatedReply(msg)
		langNudge := ""
		if req.SoftHorizon && languageRewrites < 2 {
			if pin := pinVisibleLanguage(&msg, OperatorVoice(req.User, req.History)); pin != "" {
				languageRewrites++
				langNudge = pin
				if len(msg.ToolCalls) == 0 && cutNudge == "" {
					if msg.Content != "" {
						last = msg.Content
					}
					emitSettlement(req, att, msg)
					if !lastUserIs(messages, pin) {
						messages = append(messages, Message{Role: RoleUser, Content: pin})
					}
					emitStepEnd(req, turnNo, roundSeq, roundID)
					sinceProgress++
					continue
				}
			}
		}
		if msg.Content != "" {
			last = msg.Content
		}
		emitSettlement(req, att, msg)
		if len(msg.ToolCalls) == 0 && cutNudge != "" {
			// The reply was cut at the output limit. Keep what was said, ask for
			// the rest, and do not treat the cut as a finished turn.
			if strings.TrimSpace(msg.Content) != "" {
				messages = append(messages, msg)
			}
			messages = append(messages, Message{Role: RoleUser, Content: cutNudge})
			emitStepEnd(req, turnNo, roundSeq, roundID)
			sinceProgress++
			continue
		}
		if len(msg.ToolCalls) == 0 {
			kernel.notes = persistWorkingMemory(req, messages, kernel.notes)
			if !loop.PlanMode && askNudges < 1 && waitsForNod(msg.Content) {
				askNudges++
				if strings.TrimSpace(msg.Content) != "" {
					messages = append(messages, msg)
				}
				nudge := voiceNudge(req, askContinueNudge, askContinueNudgeZH)
				if !lastUserIs(messages, nudge) {
					messages = append(messages, Message{Role: RoleUser, Content: nudge})
				}
				emitStepEnd(req, turnNo, roundSeq, roundID)
				sinceProgress++
				continue
			}
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
				emitStepEnd(req, turnNo, roundSeq, roundID)
				sinceProgress++
				continue
			}
			// The nudge budget is spent, or the model answered in prose after a
			// nudge. A bare end_turn here reads as a finished task while steps are
			// still open: the "ghost" stop. Chat turns (SoftHorizon, top level only)
			// keep the end_turn bookkeeping, tag it, and emit a retryable plan_open
			// card after it so the operator sees why the run stopped.
			var planOpen *ErrorInfo
			if req.SoftHorizon && !loop.PlanMode && req.Tools != nil && req.Tools.Depth == 0 {
				if total, open := planSteps(planTextOf(req.Tools)); open > 0 {
					info := planOpenInfo(open, total)
					planOpen = &info
				}
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
			stop := StopEndTurn
			if planOpen != nil {
				stop = StopPlanOpen
			}
			end["stop"] = string(stop)
			setStop(&req, stop)
			if block, extra := applyStopHook(req); block {
				messages = append(messages, Message{Role: RoleUser, Content: extra})
				emitStepEnd(req, turnNo, roundSeq, roundID)
				sinceProgress++
				continue
			}
			emitStepEnd(req, turnNo, roundSeq, roundID)
			emit(req, trace.TypeTurnEnd, "runtime", end)
			if planOpen != nil {
				emit(req, trace.TypeError, "runtime", planOpen.Payload())
			}
			consolidateMemory(req, ParseNotes(kernel.notes))
			return strings.TrimSpace(msg.Content), nil
		}
		messages = append(messages, msg)
		results := dispatchTools(ctx, req, msg.ToolCalls, roundID, true)
		toolCount += len(msg.ToolCalls)
		messages = append(messages, results...)
		logToolRound(ctx, roundSeq, results)
		emitStepEnd(req, turnNo, roundSeq, roundID)
		if cutNudge != "" {
			messages = append(messages, Message{Role: RoleUser, Content: cutNudge})
		}
		progressed := recordProgress(writeHits, failHits, msg.ToolCalls, results, req.Tools, &lastPlan)
		if progressed {
			sinceProgress = 0
		} else {
			sinceProgress++
		}
		if repeats.note(msg.ToolCalls, results, progressed) {
			if stall := repeats.nudge(req); stall != "" && !lastUserIs(messages, stall) {
				messages = append(messages, Message{Role: RoleUser, Content: stall})
			}
			if repeats.stuck() {
				emit(req, trace.TypeError, "runtime", stuckLoopInfo("the same call kept returning the same result").Payload())
				setStop(&req, StopStuck)
				return last, stuckLoopErr("the same call kept returning the same result")
			}
		}
		if !inspectsOnly(msg.ToolCalls) {
			recordToolErrors(errorHits, results)
		}
		recordWaits(waitHits, msg.ToolCalls)
		recordIdleActions(idleHits, msg.ToolCalls, results)
		recordClickMicro(&clickHits, msg.ToolCalls)
		if req.SoftHorizon {
			watch.note(msg.ToolCalls, results)
		}
		kernel.notes = persistWorkingMemory(req, messages, kernel.notes)
		if req.SoftHorizon && watch.stuck() {
			emit(req, trace.TypeError, "runtime", stuckLoopInfo(watch.reason()).Payload())
			setStop(&req, StopStuck)
			return last, stuckLoopErr(watch.reason())
		}
		if req.SoftHorizon && hitMax(failHits) >= rewriteStallStopHits {
			emit(req, trace.TypeError, "runtime", stuckLoopInfo("rewrote the same file without progress").Payload())
			setStop(&req, StopStuck)
			return last, stuckLoopErr("rewrote the same file without progress")
		}
		if req.SoftHorizon && hitMax(errorHits) >= errorRepeatStopHits {
			emit(req, trace.TypeError, "runtime", stuckLoopInfo("the same tool failure kept repeating").Payload())
			setStop(&req, StopStuck)
			return last, stuckLoopErr("the same tool failure kept repeating")
		}
		if req.SoftHorizon && hitMax(waitHits) >= waitLoopStopHits {
			emit(req, trace.TypeError, "runtime", stuckLoopInfo("wait/poll loop").Payload())
			setStop(&req, StopStuck)
			return last, stuckLoopErr("wait/poll loop")
		}
		if req.SoftHorizon && hitMax(idleHits) >= idleActionStopHits {
			emit(req, trace.TypeError, "runtime", stuckLoopInfo("opened the same page without interacting").Payload())
			setStop(&req, StopStuck)
			return last, stuckLoopErr("opened the same page without interacting")
		}
		if stall := rewriteStallNudge(req, writeHits); stall != "" && !lastUserIs(messages, stall) {
			messages = append(messages, Message{Role: RoleUser, Content: stall})
		}
		if stall := errorRepeatNudge(req, errorHits); stall != "" && !lastUserIs(messages, stall) {
			messages = append(messages, Message{Role: RoleUser, Content: stall})
		}
		if stall := waitLoopNudge(req, waitHits); stall != "" && !lastUserIs(messages, stall) {
			messages = append(messages, Message{Role: RoleUser, Content: stall})
		}
		if stall := idleActionNudge(req, idleHits); stall != "" && !lastUserIs(messages, stall) {
			messages = append(messages, Message{Role: RoleUser, Content: stall})
		}
		if stall := observationNudge(req, &watch, msg.ToolCalls); stall != "" && !lastUserIs(messages, stall) {
			messages = append(messages, Message{Role: RoleUser, Content: stall})
		}
		if stall := clickMicroNudge(req, clickHits); stall != "" && !lastUserIs(messages, stall) {
			messages = append(messages, Message{Role: RoleUser, Content: stall})
		}
		if langNudge != "" && !lastUserIs(messages, langNudge) {
			messages = append(messages, Message{Role: RoleUser, Content: langNudge})
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

func completeChat(ctx context.Context, req RunRequest, chatReq ChatRequest, roundID string, turn, step int) (Message, *streamAttempt, error) {
	msg, att, err := chat(ctx, req, chatReq, roundID, turn, step)
	if err != nil && isStreamErr(err) {
		started := time.Now()
		msg, err = req.Client.Chat(ctx, chatReq)
		if msg.ElapsedMs <= 0 {
			msg.ElapsedMs = elapsedMsSince(started)
		}
		if err == nil {
			att = &streamAttempt{ID: roundID, Turn: turn, Step: step, RoundID: roundID, Blocks: blocksFromMessage(msg)}
		}
	}
	return msg, att, err
}

func elapsedMsSince(started time.Time) int {
	n := int(time.Since(started).Milliseconds())
	if n <= 0 {
		return 1
	}
	return n
}

func chat(ctx context.Context, req RunRequest, chatReq ChatRequest, roundID string, turn, step int) (msg Message, att *streamAttempt, err error) {
	started := time.Now()
	firstMs := 0
	markFirst := func() {
		if firstMs == 0 {
			firstMs = elapsedMsSince(started)
		}
	}
	defer func() {
		if msg.ElapsedMs <= 0 {
			msg.ElapsedMs = elapsedMsSince(started)
		}
		if msg.FirstTokenMs <= 0 && firstMs > 0 {
			msg.FirstTokenMs = firstMs
		}
	}()
	var sp observe.Span
	if req.Observe != nil {
		parent := observe.SpanFrom(ctx)
		sp = req.Observe.StartChild(parent, "model.complete", req.SessionID, map[string]any{"round": roundID, "model": req.Model})
		sp.TurnID = req.TurnID
		defer func() {
			if sp.Attrs == nil {
				sp.Attrs = map[string]any{}
			}
			if msg.PromptTokens > 0 {
				sp.Attrs["gen_ai.usage.input_tokens"] = msg.PromptTokens
			}
			if msg.CompletionTokens > 0 {
				sp.Attrs["gen_ai.usage.output_tokens"] = msg.CompletionTokens
			}
			if msg.CachedTokens > 0 || msg.CacheReported {
				sp.Attrs["gen_ai.usage.cache_read_tokens"] = msg.CachedTokens
			}
			req.Observe.End(sp, err)
		}()
	}
	if s, ok := req.Client.(Streamer); ok {
		att = &streamAttempt{ID: roundID, Turn: turn, Step: step, RoundID: roundID}
		asm := NewBlockAssembler()
		acc := NewStreamAccumulator()
		var wg sync.WaitGroup
		var prog toolProgressGate
		emitAttemptStart(req, att)
		msg, err = s.ChatStream(ctx, chatReq, func(ch StreamChunk) error {
			if ch.Type == ChunkReasoningDelta && !req.ShowThinking {
				return nil
			}
			if ch.IsToken() {
				markFirst()
			}
			acc.Push(nowMillis(), ch)
			asm.Push(ch)
			emitStreamChunk(req, att, ch)
			if ch.Type == ChunkToolCallDelta && ch.Name != "" && prog.allow(ch.ID, len(ch.ArgumentsDelta), time.Now()) {
				emitToolProgress(req, roundID, ToolCall{ID: ch.ID, Name: ch.Name, Arguments: ch.ArgumentsDelta})
			}
			if ch.Type == ChunkBlockEnd && ch.Block != nil && ch.Block.Type == BlockToolCall && readonlyCall(req.Tools, ch.Block.Name) {
				tc := ToolCall{ID: ch.Block.ID, Name: ch.Block.Name, Arguments: ch.Block.Arguments}
				wg.Add(1)
				go func() {
					defer wg.Done()
					_ = dispatchTools(ctx, req, []ToolCall{tc}, roundID, false)
				}()
			}
			return nil
		})
		wg.Wait()
		if err != nil {
			if ctx.Err() != nil {
				blocks := asm.InterruptedBlocks()
				if len(blocks) > 0 {
					att.Blocks = blocks
					att.Stream = acc.Snapshot()
					att.Interrupted = true
					emitSettlement(req, att, Message{Role: RoleAssistant, Content: textFromBlocks(blocks)})
					emitAttemptEnd(req, att, StreamOutcomeCommitted)
				} else {
					emitAttemptEnd(req, att, StreamOutcomeAbandoned)
				}
			} else {
				emitAttemptEnd(req, att, StreamOutcomeAbandoned)
			}
			return msg, att, err
		}
		if msg.Truncated {
			asm.Push(StreamChunk{Type: ChunkFinish, Reason: &FinishReason{Kind: FinishMaxTokens}})
		} else if len(msg.ToolCalls) > 0 {
			asm.Push(StreamChunk{Type: ChunkFinish, Reason: &FinishReason{Kind: FinishToolCalls}})
		} else {
			asm.Push(StreamChunk{Type: ChunkFinish, Reason: &FinishReason{Kind: FinishStop}})
		}
		att.Blocks = asm.Blocks()
		att.Stream = acc.Snapshot()
		emitAttemptEnd(req, att, StreamOutcomeCommitted)
		return msg, att, err
	}
	msg, err = req.Client.Chat(ctx, chatReq)
	att = &streamAttempt{ID: roundID, Turn: turn, Step: step, RoundID: roundID, Blocks: blocksFromMessage(msg)}
	return msg, att, err
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
	parallelOK := n > 1
	for _, tc := range calls {
		if !readonlyCall(req.Tools, tc.Name) {
			readonly = false
		}
		if exclusiveCall(req.Tools, tc.Name) {
			exclusive = true
			parallelOK = false
		}
		if !parallelizableCall(req.Tools, tc.Name) {
			parallelOK = false
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
			call := map[string]any{
				"name": tc.Name, "arguments": tc.Arguments, "id": tc.ID, "round": roundID,
			}
			if label := skillpack.CallLabel(tc.Name, tc.Arguments); label != "" {
				call["skill"] = label
			}
			emit(req, trace.TypeToolCall, "agent", call)
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
		if req.Tools != nil {
			req.Tools.setLiveCall(j.tc.ID, roundID)
			defer req.Tools.setLiveCall("", "")
		}
		var res ToolResult
		var toolSpan observe.Span
		if req.Observe != nil {
			parent := observe.SpanFrom(ctx)
			toolSpan = req.Observe.StartChild(parent, "tool.exec", req.SessionID, map[string]any{"tool": j.tc.Name})
			toolSpan.TurnID = req.TurnID
		}
		if j.deny {
			res = ToolResult{Content: "ERROR: blocked by hook: " + j.reason, Err: fmt.Errorf("blocked")}
		} else if req.Tools == nil {
			res = ToolResult{Err: fmt.Errorf("no tools")}
		} else {
			res = req.Tools.Call(j.tc.Name, j.args)
		}
		if req.Observe != nil {
			req.Observe.End(toolSpan, res.Err)
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
	if (readonly || parallelOK) && n > 1 {
		limit := 8
		if req.Tools != nil && req.Tools.MaxParallel > 0 && req.Tools.MaxParallel < limit {
			limit = req.Tools.MaxParallel
		}
		sem := make(chan struct{}, limit)
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
			label := skillpack.CallLabel(j.tc.Name, j.args)
			if label == "" {
				label = j.tc.Name
			}
			emit(req, trace.TypeInject, "skill", map[string]any{"name": label, "text": j.content, "round": roundID})
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
			if j.change.Patch != "" {
				payload["patch"] = j.change.Patch
			}
			if j.change.Added != 0 {
				payload["added"] = j.change.Added
			}
			if j.change.Removed != 0 {
				payload["removed"] = j.change.Removed
			}
			if j.change.Created {
				payload["created"] = true
			}
			filePay := map[string]any{
				"id": j.tc.ID, "name": j.tc.Name, "paths": j.change.Paths, "round": roundID,
			}
			if j.change.Added != 0 {
				filePay["added"] = j.change.Added
			}
			if j.change.Removed != 0 {
				filePay["removed"] = j.change.Removed
			}
			if j.change.Created {
				filePay["created"] = true
			}
			emit(req, trace.TypeFileChange, "tool", filePay)
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
		"read_thread", "read_skill_file", "personal_status", "present_choices",
		"read_page", "list_pages":
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

func parallelizableCall(tools *WorkspaceTools, name string) bool {
	if exclusiveCall(tools, name) {
		return false
	}
	if readonlyCall(tools, name) || name == "task" {
		return true
	}
	return HostAnn(name).ConcurrencySafe
}

func readOnlySubagentCall(name string) bool {
	if readonlyCall(nil, name) {
		return true
	}
	switch name {
	case "web_fetch", "web_search":
		return true
	default:
		return false
	}
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
	return voiceNudge(req, planContinueNudge, planContinueNudgeZH)
}

func PlanOpen(plan string) bool {
	total, open := planSteps(plan)
	return total > 0 && open > 0
}

// planSteps counts checklist lines in a plan as (total, open).
func planSteps(plan string) (total, open int) {
	for _, line := range strings.Split(plan, "\n") {
		st, ok := planLineStatus(line)
		if !ok {
			continue
		}
		total++
		if !planStatusDone(st) {
			open++
		}
	}
	return total, open
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
		TurnID:           req.TurnID,
		HarnessSnapshot:  req.HarnessHash,
		ModelFingerprint: req.ModelFingerprint,
		TaskID:           req.TaskID,
		Payload:          payload,
	}
	if req.TurnID != "" {
		if _, ok := payload["turn_id"]; !ok {
			payload["turn_id"] = req.TurnID
		}
	}
	persist := req.Trace != nil
	if persist {
		if typ == trace.TypeAssistantStream {
			persist = false
		}
		if delta, _ := payload["delta"].(bool); delta {
			switch typ {
			case trace.TypeAssistant, trace.TypeReasoning, trace.TypeToolResult:
				persist = false
			}
		}
		if typ == trace.TypeToolCall && payloadProgress(payload) {
			persist = false
		}
	}
	if persist {
		_ = req.Trace.Append(ev)
	}
	if req.OnEvent != nil {
		if typ == trace.TypeCompact {
			kind, _ := payload["kind"].(string)
			if kind == "" || kind == "shape" {
				return
			}
		}
		if typ == trace.TypeFileChange || typ == trace.TypeUsage {
			return
		}
		req.OnEvent(slimLiveEvent(ev))
	}
}
