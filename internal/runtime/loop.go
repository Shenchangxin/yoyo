// Package runtime 是聊天和评测共用的模型循环。两边走同一个 Run，差别在请求标志，不是两套循环。
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

// RunRequest 是一次 Run 的输入。SoftHorizon 为假时 MaxTurns 和 MaxToolMessages 是硬边界；为真时是桌面或命令行聊天。
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
	Stop             *StopReason // 非 nil 时 Run 把停止原因写回这里；nil 表示调用方不接收原因
	StopHooks        []FileHook
	PreCompactHooks  []FileHook
	ProfileMemory    string
	PageContext      string
	// RoundSeq 是这条会话 JSONL 上已经有过的最大助手回合号 rN。下一次模型调用用 N+1，避免检查点重建后和界面上还在的气泡撞号。小于等于 0 时，Run 改按 History 里的助手消息条数起算。
	RoundSeq int
	// SoftHorizon 只给桌面和命令行聊天。评测和子代理保持 false，MaxTurns 与 MaxToolMessages 仍是硬边界。
	// 为真时，停止来自结束回合、取消、预算、溢出、观察没有变化，或 ChatSoftGuardTurns。观察仍在变化时，不会因为评测那档 MaxTurns 用尽而停。相同工具参数只发提醒。
	SoftHorizon bool
	// ShowThinking 为真时把供应商推理流进记录。零值 false：不少 OpenAI 兼容端点会拒绝思考字段。
	ShowThinking bool
	// MaxOutputTokens 限制一次模型回复的输出 token。零值交给供应商默认。回复仍被切断时，循环拒绝半截工具调用，并要求分段再写，见 truncation.go。
	MaxOutputTokens int
	// UserCommitted 为真表示操作者那一行已经在 JSONL 上。seed 仍把它放进模型记录，但不能再发第二条 TypeUser。零值 false 时 seed 会发出 TypeUser。
	UserCommitted bool
	// RequestID 是合成器乐观发送的 id。seed 把它盖在 TypeUser 上，渲染器用来替换本地回声，避免气泡出现两次。空串表示没有这张本地回声。
	RequestID string
}

var emitSeq atomic.Int64

// Run 跑一轮模型循环。Loop.MaxTurns 小于等于 0 时写成 24，后面的 turnLimit 再小于等于 0 时也写成 24。SoftHorizon 为真时 for 不看 turnLimit。
// 上下文溢出只重试当前这一步，见下面的溢出分支。Depth 小于 MaxTaskDepth 且 Task 还是 nil 时，把 Task 设为 spawnTask。子代理的循环预设在 tools_task.go。
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
	if req.Tools != nil {
		req.Tools.setRepeatBlock("", "")
	}
	if req.SoftHorizon {
		seedCheckWatch(req, &watch, messages)
		if req.Tools != nil {
			script, refusal := watch.repeatBlock()
			req.Tools.setRepeatBlock(script, refusal)
		}
	}
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
	// sinceProgress 计的是自上次计划步骤或文件真正变化以来的模型轮数，没有工具调用的轮也算。softGuard 在 turn 到达 192 时才看它：小于 24 就复位并做检查点，不小于 24 就停止。只计工具轮会让被截断的回复把运行拖住不结束。
	sinceProgress := 0
	repeats := newRepeatTracker()
	for turn := 0; req.SoftHorizon || turn < turnLimit; turn++ {
		if req.SoftHorizon {
			// stop 时按 ChatSoftGuardTurns 发轮数用尽，原因是 StopMaxTurns。reset 时发出 kind 为 budget_continue 的 compact，并把 turn 置 0；for 在本轮结束时还会加一，下一轮从 1 计。
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
			// 打断落在一批工具和下一次模型调用之间。界面只从终局事件知道这一轮结束，所以这里不能悄悄返回，卡片和流中途取消是同一张。
			emit(req, trace.TypeError, "runtime", ClassifyError(err).Payload())
			setStop(&req, StopCancelled)
			return last, stopErr(StopCancelled, err.Error(), err)
		}
		if req.PullSteer != nil {
			if extra := strings.TrimSpace(req.PullSteer()); extra != "" {
				repeats.reset()
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
			// 每轮投影账本抽样写入。每轮都记会让每次模型调用都落一条，会话 facb2e21ba0b31a3 里有 257 条，并不增加信息。检查点（Trigger 非空）始终写入。roundSeq 能被 shapeLogEvery（16）整除时也写入。
			if roundSeq%shapeLogEvery == 0 || report.Trigger != "" {
				emit(req, trace.TypeCompact, "runtime", reportPayload(report, nil))
			}
			if req.Events != nil {
				req.Events.Emit(HookCompact, report.Note)
			}
		}
		chatReq := ChatRequest{Model: req.Model, Messages: compacted, Tools: toolsJSON, CacheKey: PromptCacheKey(req.HarnessHash, kernel.cachePrefix(), toolsJSON), ShowThinking: req.ShowThinking, MaxTokens: req.MaxOutputTokens}
		roundSeq++
		roundID := fmt.Sprintf("%s:r%d", req.SessionID, roundSeq)
		emitStepStart(req, turnNo, roundSeq, roundID, roundID)
		msg, att, err := completeChat(ctx, req, chatReq, roundID, turnNo, roundSeq)
		// 上下文溢出时 overflowFails 加一。这次调用最终没有错误时，计数清零。
		// 加一后仍小于 2：做一次 kind 为 overflow 的检查点，再用同一 roundSeq、把 roundID 加上 :retry，重试当前这一步，不进入下一轮 for。这次重试若仍是上下文溢出，改成 StopOverflow。
		// 加一后已经不小于 2：不再检查点，直接改成 StopOverflow。两种停止的消息都是 context overflow circuit breaker。重试若是别的错误，按那个错误返回，不改成溢出。
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
				chatReq.CacheKey = PromptCacheKey(req.HarnessHash, kernel.cachePrefix(), toolsJSON)
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
			// 每次模型调用一条用量。turn_end 只带最后一次调用，看不出提示缓存有没有跨过一整轮保住。
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
		// SoftHorizon 下可见语言最多改写 2 次。没有工具调用且没有截断催促时，把语言提示追加进消息并进入下一轮，不把这条回复当成结束。
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
			// 回复停在输出上限。留下已经说出的内容并要求续写，不把这次切断当成回合结束。
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
			// 计划模式以外，askNudges 还小于 1，且 waitsForNod 为真时，只催这一次并进入下一轮。
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
			// 催促次数已经用尽，或催促之后模型改用散文回答。这里若只发普通的 end_turn，步骤还开着，看起来却像任务做完了。聊天回合（SoftHorizon 且 Depth 为 0）仍按结束回合记账，原因改成 plan_open，并在 turn_end 之后再发一张可重试的卡片。
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
				end["identity_tokens"] = report.IdentityTokens
				end["rules_tokens"] = report.RulesTokens
				end["catalog_tokens"] = report.CatalogTokens
				end["skill_body_tokens"] = report.SkillBodyTokens
				end["memory_tokens"] = report.MemoryTokens
				end["today_tokens"] = report.TodayTokens
				end["notes_tokens"] = report.NotesTokens
				end["chat_tokens"] = report.ChatTokens
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
		if stall := repeats.observe(req, msg.ToolCalls); stall != "" && !lastUserIs(messages, stall) {
			emit(req, trace.TypeInject, "repeat-tool-reminder", map[string]any{"text": stall, "name": "repeat-tool-reminder"})
			messages = append(messages, Message{Role: RoleUser, Content: stall})
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
		if req.Tools != nil {
			script, refusal := watch.repeatBlock()
			req.Tools.setRepeatBlock(script, refusal)
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

// middlewareNudge 在本批结果里以前缀 ERROR: 开头的条数达到 MaxRecentToolErrors 时，返回 ToolErrorInstruction。上限小于等于 0 或说明为空时返回空串。计数要求冒号，和诊断日志里的 ERROR 前缀不是同一条规则。
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

// spillOf 返回工具上的 Spill。Tools 为 nil 时返回 nil。
func spillOf(req RunRequest) *Spill {
	if req.Tools != nil {
		return req.Tools.Spill
	}
	return nil
}

// completeChat 先走 chat。错误文本含 stream 或 sse 时，再改用 Client.Chat 试当前这一步一次。这次成功就用整段回复合成 streamAttempt，不再保留失败的流。
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

// elapsedMsSince 返回自 started 起的毫秒数。结果小于等于 0 时返回 1。
func elapsedMsSince(started time.Time) int {
	n := int(time.Since(started).Milliseconds())
	if n <= 0 {
		return 1
	}
	return n
}

// chat 在 Client 实现 Streamer 时走流。ShowThinking 为假时丢掉推理增量。只读工具的块一结束就在后台预取，recordTrace 为 false，正式 dispatchTools 再发一次轨迹。
// 上下文取消且已经组装出块时，尝试记为 committed；其他错误记为 abandoned。没有 Streamer 时直接 Client.Chat。
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

// dispatchTools 执行一批工具调用。调用数大于 1，并且（整批都是只读，或 parallelOK 为真）时并行。parallelOK 从调用数大于 1 起算，任一 exclusiveCall 或任一 parallelizableCall 为假就把它关掉。整批都是只读时，即使调用也标了 exclusive，仍走并行。
// 信号量容量先取 8。Tools 非空、MaxParallel 大于 0 且小于 8 时，容量改成 MaxParallel。
// 没进并行时，批次里有 exclusive 就串行。shell 和 run_skill_script 按名字就是 exclusive。web_fetch 和 web_search 的注解 Exclusive 与 OpenWorld 都为真，exclusiveCall 因此为真，而且它们不是只读，所以不能并行。exclusive 只是串行，避免工作目录和环境变量竞态。前面的 shell 失败后，同批后面的写入仍然执行。
// 其余情况也串行。上下文已取消时，还没执行的调用写成 toolAbortedBeforeDispatch。recordTrace 为 false 时把结果放进 prefetch，不发工具轨迹。
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
		// 串行是为了避开工作目录和环境变量竞态。同批前面的调用失败后，后面的调用仍执行。
		for i := range jobs {
			if err := ctx.Err(); err != nil {
				for j := i; j < n; j++ {
					jobs[j].content = toolAbortedBeforeDispatch
					jobs[j].ok = false
				}
				break
			}
			runExec(i)
		}
	} else {
		for i := range jobs {
			if err := ctx.Err(); err != nil {
				for j := i; j < n; j++ {
					jobs[j].content = toolAbortedBeforeDispatch
					jobs[j].ok = false
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

// isReadonlyTool 按内置名单判断只读。命中的名字可以并行，也可以在流式块结束时预取。名单之外再看 Extra 和 HostAnn。
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

// loadedFrom 返回已加载技能正文，并在有计划或验证提示时附上对应覆盖层。t 为 nil 时返回 nil。
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

// planTextOf 返回当前计划文本。t 为 nil 时返回空串。
func planTextOf(t *WorkspaceTools) string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.PlanText
}

// readonlyCall 在内置只读名单、Extra.ReadOnly 或 HostAnn.ReadOnly 任一为真时返回 true。web_fetch 和 web_search 不在此列。
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

// exclusiveCall 对 shell 和 run_skill_script 按名字返回 true。其他名字再看 Extra 或 HostAnn 的 Exclusive、OpenWorld，有一个为真就是 exclusive。web_fetch 和 web_search 因此返回 true。
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

// parallelizableCall 在 exclusive 时返回 false。否则只读或名字是 task 时返回 true，再不然看 HostAnn.ConcurrencySafe。
func parallelizableCall(tools *WorkspaceTools, name string) bool {
	if exclusiveCall(tools, name) {
		return false
	}
	if readonlyCall(tools, name) || name == "task" {
		return true
	}
	return HostAnn(name).ConcurrencySafe
}

// readOnlySubagentCall 给子代理筛工具。readonlyCall 传入 nil，所以只看内置名单和 HostAnn。此外 web_fetch、web_search、reach_status 也返回 true。这三名在这里算只读，不改变 exclusiveCall：web_fetch 和 web_search 在 dispatchTools 里仍然 exclusive。
func readOnlySubagentCall(name string) bool {
	if readonlyCall(nil, name) {
		return true
	}
	switch name {
	case "web_fetch", "web_search", "reach_status":
		return true
	default:
		return false
	}
}

// setStop 把原因写进 req.Stop。req 或 Stop 指针为 nil 时什么都不做。
func setStop(req *RunRequest, r StopReason) {
	if req != nil && req.Stop != nil {
		*req.Stop = r
	}
}

// isStreamErr 在错误文本的小写里含有 stream 或 sse 时返回 true。err 为 nil 时返回 false。
func isStreamErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "stream") || strings.Contains(s, "sse")
}

// applyStopHook 在停止钩子要求继续时返回 true 和一段用户消息。Run 把这段消息追加进记录并进入下一轮，不把原因写成 StopHook。
// StopHooks 里第一个 Deny 就返回。Reason 为空时用 blocked by .yoyo/hooks.json stop hook。事件总线的 HookStop 出错，或结果 Block 为真，也返回 true。
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

// maxPlanContinues 是 6。计划仍开着时，planContinueIfOpen 最多催这么多次。
const maxPlanContinues = 6

// planContinueIfOpen 在计划模式、没有工具或计划已完成时返回空串。催促次数已经不小于 6，或上次催促之后工具条数没变，也返回空串。否则次数加一，并记下当时的工具条数。
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

// PlanOpen 在计划里至少有一条清单、且还有未完成步骤时返回 true。
func PlanOpen(plan string) bool {
	total, open := planSteps(plan)
	return total > 0 && open > 0
}

// planSteps 把计划里的清单行数成总数和未完成数。认得出的完成状态不计入 open。
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

// planLineStatus 从清单行取出方括号里的状态。序号点号必须出现在前四个字节内，点号后要有 [status]。对不上时返回空串和 false。
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

// planStatusDone 把状态收成小写并把空格、连字符换成下划线。complete、completed、done、finished 返回 true，其余返回 false。
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

// lastUserIs 在最后一条消息是用户且 Content 与 text 全等时返回 true。text 去掉空白后为空，或消息列表为空，返回 false。比较本身不去掉 Content 的空白。
func lastUserIs(msgs []Message, text string) bool {
	if len(msgs) == 0 || strings.TrimSpace(text) == "" {
		return false
	}
	last := msgs[len(msgs)-1]
	return last.Role == RoleUser && last.Content == text
}

// lastStopPayload 是发给 HookStop 的空载荷。
type lastStopPayload struct{}

// StopHookResult 是停止钩子的返回。Block 为真时 Run 把 Message 追加进记录并继续。Message 为空时 applyStopHook 改用英文句 Stop hook asked to continue。
type StopHookResult struct {
	Block   bool
	Message string
}

// stripSystem 丢掉 system、developer 和 memory 角色，其余原样保留。
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

// emit 写轨迹并通知实时总线。payload 为 nil 时换成空 map。用户、注入、错误、压缩和审批在没有 id 时生成一个。
// assistant_stream 不落盘。delta 为真的助手、推理和工具结果不落盘。progress 为真的 tool_call 不落盘。OnEvent 不接收 kind 为空或 shape 的压缩、file_change 和 usage，其余先经过 slimLiveEvent。
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
