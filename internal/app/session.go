package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/eval"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

type SessionMeta struct {
	ID               string    `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	Workspace        string    `json:"workspace"`
	Harness          string    `json:"harness"`
	HarnessPolicy    string    `json:"harness_policy,omitempty"`
	ModelFingerprint string    `json:"model_fingerprint"`
	Title            string    `json:"title,omitempty"`
	Archived         bool      `json:"archived"`
	Pinned           bool      `json:"pinned"`
	Model            string    `json:"model,omitempty"`
	LoadedSkills     []string  `json:"loaded_skills,omitempty"`
	PlanText         string    `json:"plan_text,omitempty"`
	AuthMode         string    `json:"auth_mode,omitempty"`
}

func (a *App) NewSession(workspace string) (SessionMeta, error) {
	if workspace == "" {
		workspace = a.Workspace()
	}
	id := newID()
	meta := SessionMeta{
		ID:               id,
		CreatedAt:        time.Now().UTC(),
		Workspace:        workspace,
		Harness:          a.ActiveHash(),
		HarnessPolicy:    session.FollowActive,
		ModelFingerprint: Fingerprint(a.Config),
		Title:            "session",
	}
	b, _ := json.MarshalIndent(meta, "", "  ")
	path := filepath.Join(a.Home.Sessions(), id+".meta.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return meta, err
	}
	a.applySessionAuth(id, capability.AuthDefault)
	return a.attachAuthMode(meta), nil
}

func (a *App) GetSession(id string) (SessionMeta, error) {
	b, err := os.ReadFile(filepath.Join(a.Home.Sessions(), id+".meta.json"))
	if err != nil {
		return SessionMeta{ID: id}, err
	}
	var m SessionMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return m, err
	}
	return a.attachAuthMode(m), nil
}

func (a *App) attachAuthMode(m SessionMeta) SessionMeta {
	mode := capability.AuthDefault
	if a != nil && a.Threads != nil && m.ID != "" {
		if st := a.Threads.Load(m.ID); strings.TrimSpace(st.AuthMode) != "" {
			mode = capability.ParseAuthMode(st.AuthMode)
		} else if strings.TrimSpace(m.AuthMode) != "" {
			mode = capability.ParseAuthMode(m.AuthMode)
		}
	} else if strings.TrimSpace(m.AuthMode) != "" {
		mode = capability.ParseAuthMode(m.AuthMode)
	}
	m.AuthMode = mode
	return m
}

func (a *App) applySessionAuth(id, mode string) {
	mode = capability.ParseAuthMode(mode)
	caps := capability.AuthModeCapStrings(mode)
	if a.Threads != nil {
		a.Threads.SetAuthMode(id, mode, caps)
	}
	if a.Caps != nil {
		a.Caps.ApplyAuthMode(id, mode)
	}
}

func (a *App) writeSession(m SessionMeta) error {
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(filepath.Join(a.Home.Sessions(), m.ID+".meta.json"), b, 0o644)
}

func titleFrom(message string) string {
	s := strings.TrimSpace(strings.ReplaceAll(message, "\n", " "))
	if s == "" {
		return "session"
	}
	r := []rune(s)
	if len(r) > 42 {
		return string(r[:42]) + "…"
	}
	return s
}

func (a *App) Trajectory(id string) ([]trace.Event, error) {
	return a.Traces.Read(id)
}

func (a *App) Client() (runtime.Client, error) {
	key, _ := a.Vault.Lease("default")
	return runtime.NewOpenAIClient(a.Config.BaseURL, key), nil
}

func (a *App) Send(ctx context.Context, sessionID, message string, client runtime.Client, onEvent func(trace.Event)) (string, error) {
	return a.SendOpts(ctx, sessionID, message, client, onEvent, false)
}

func (a *App) Running(sessionID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.runs[sessionID] != nil
}

func (a *App) RunningIDs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := make([]string, 0, len(a.runs))
	for id := range a.runs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (a *App) acquireRun(sessionID string, ctx context.Context) (context.Context, func(), error) {
	ctx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	if a.runs[sessionID] != nil {
		a.mu.Unlock()
		cancel()
		return nil, nil, errBusy
	}
	a.runs[sessionID] = cancel
	a.mu.Unlock()
	return ctx, func() {
		cancel()
		a.mu.Lock()
		delete(a.runs, sessionID)
		a.mu.Unlock()
	}, nil
}

func (a *App) StartSend(sessionID, message string, plan bool) error {
	return a.StartSendOpts(sessionID, message, plan, nil)
}

func (a *App) StartSendOpts(sessionID, message string, plan bool, atts []Attachment) error {
	atts, resume := takeResume(message, atts)
	if resume {
		message = ""
	}
	if resume && a.Running(sessionID) {
		return errBusy
	}
	if a.Running(sessionID) {
		a.Enqueue(sessionID, QueuedTurn{Text: message, Plan: plan, Attachments: atts})
		return ErrQueued
	}
	return a.launchSend(sessionID, message, plan, atts, resume)
}

// RetrySession continues the current turn from existing history. It does not
// append a new user message (unlike Send).
func (a *App) RetrySession(sessionID string) error {
	if a.Running(sessionID) {
		return errBusy
	}
	return a.launchSend(sessionID, "", false, nil, true)
}

func takeResume(message string, atts []Attachment) ([]Attachment, bool) {
	if strings.TrimSpace(message) != "" {
		return atts, false
	}
	kept := make([]Attachment, 0, len(atts))
	resume := false
	for _, a := range atts {
		if a.Name == "__resume__" || a.Path == "__resume__" {
			resume = true
			continue
		}
		kept = append(kept, a)
	}
	return kept, resume
}

func (a *App) launchSend(sessionID, message string, plan bool, atts []Attachment, resume bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	ctx, release, err := a.acquireRun(sessionID, ctx)
	if err != nil {
		cancel()
		if err == errBusy {
			if resume {
				return errBusy
			}
			a.Enqueue(sessionID, QueuedTurn{Text: message, Plan: plan, Attachments: atts})
			return ErrQueued
		}
		return err
	}
	if a.Threads != nil && strings.TrimSpace(message) != "" {
		a.Threads.SetResume(sessionID, message, plan)
	}
	go func() {
		defer cancel()
		defer release()
		defer func() {
			if rec := recover(); rec != nil {
				a.Hub.Publish(trace.Event{
					Type: trace.TypeError, Source: "runtime", SessionID: sessionID,
					Payload: map[string]any{"error": fmt.Sprintf("loop panic: %v", rec)},
				})
			}
		}()
		defer a.kickQueue(sessionID)
		_, _ = a.sendLocked(ctx, sessionID, message, nil, nil, plan, atts, resume)
	}()
	return nil
}

func (a *App) SendOpts(ctx context.Context, sessionID, message string, client runtime.Client, onEvent func(trace.Event), plan bool) (string, error) {
	ctx, release, err := a.acquireRun(sessionID, ctx)
	if err != nil {
		return "", err
	}
	defer release()
	return a.sendLocked(ctx, sessionID, message, client, onEvent, plan, nil, false)
}

func (a *App) sendLocked(ctx context.Context, sessionID, message string, client runtime.Client, onEvent func(trace.Event), plan bool, atts []Attachment, resume bool) (out string, runErr error) {
	if a.Observe != nil {
		sp := a.Observe.Start("turn", sessionID, map[string]any{"plan": plan, "resume": resume})
		defer func() { a.Observe.End(sp, runErr) }()
	}
	if strings.TrimSpace(sessionID) == "" {
		return "", fmt.Errorf("empty session id")
	}
	meta, err := a.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("unknown session %s: %w", sessionID, err)
	}
	if meta.HarnessPolicy == "" {
		meta.HarnessPolicy = session.FollowActive
	}
	hash := session.ResolveHarness(meta.HarnessPolicy, meta.Harness, a.ActiveHash())
	if hash == "" {
		return "", fmt.Errorf("session %s has no harness", sessionID)
	}
	snap, err := a.LoadSnapshot(hash)
	if err != nil {
		return "", err
	}
	loop, frags, pb, skills, _, pol, err := a.Materials(hash)
	if err != nil {
		return "", err
	}
	if plan {
		loop.PlanMode = true
		pol.Mode = "plan"
	}
	if a.Config.MaxBudgetUSD > 0 {
		if loop.MaxBudgetUSD <= 0 || a.Config.MaxBudgetUSD < loop.MaxBudgetUSD {
			loop.MaxBudgetUSD = a.Config.MaxBudgetUSD
		}
	}
	meter := &runtime.Meter{USDPerMTok: a.Config.USDPerMTok}
	if client == nil {
		client, err = a.Client()
		if err != nil {
			return "", err
		}
	}
	skillBodies := map[string]string{}
	for _, s := range skills {
		skillBodies[s.Name] = s.Body
	}
	disk := runtime.LoadSkillDirs(runtime.SkillRoots(a.Home.Root, meta.Workspace, bundledSkillsDir(a.BundledEvals))...)
	skills = runtime.MergeSkills(skills, disk)
	skills = runtime.FilterSkills(skills, loop.PlanMode || plan)
	skillDirs := map[string]string{}
	skillMeta := map[string]artifact.Skill{}
	for _, s := range skills {
		skillBodies[s.Name] = s.Body
		if s.Dir != "" {
			skillDirs[s.Name] = s.Dir
		}
		skillMeta[s.Name] = s
	}
	model := a.Config.Model
	if meta.Model != "" {
		model = meta.Model
	}
	preHooks, stopHooks := runtime.LoadHookFile(meta.Workspace)
	tools := &runtime.WorkspaceTools{
		Workspace:  meta.Workspace,
		SessionID:  sessionID,
		Caps:       a.Caps,
		Skills:     skillBodies,
		SkillDirs:  skillDirs,
		SkillMeta:  skillMeta,
		Loaded:     append([]string(nil), meta.LoadedSkills...),
		Policy:     pol,
		PlanMode:   loop.PlanMode,
		Ctx:        ctx,
		Extra:      a.extraTools(sessionID),
		Spill:      runtime.BindSpill(a.Home.Root, meta.Workspace, sessionID),
		PlanText:   meta.PlanText,
		Advertised: append([]string(nil), snap.Tools...),
		AskUser:    a.askUserFn(ctx, sessionID),
	}
	a.attachPersonal(tools)
	hist := []runtime.Message{}
	roundSeq := 0
	if evs, err := a.Traces.Read(sessionID); err == nil {
		hist = runtime.MessagesFromEvents(evs)
		roundSeq = runtime.MaxAssistantRound(evs)
	}
	window := runtime.ModelContextWindow(model)
	hist = runtime.MaybeCheckpoint(a.Traces, sessionID, hist, loop, tools.Spill, client, model, window, frags)
	wrapped := onEvent
	if a.Hub != nil {
		prev := wrapped
		wrapped = func(ev trace.Event) {
			a.Hub.Publish(ev)
			if prev != nil {
				prev(ev)
			}
		}
	}

	inject := ""
	if resume {
		message = ""
		atts = nil
	} else {
		if meta.Title == "" || meta.Title == "session" {
			meta.Title = titleFrom(message)
			_ = a.writeSession(meta)
		}
		note := hash
		if snap.Note != "" {
			note = hash + " " + snap.Note
		}
		inject, _ = runtime.ExpandMentionsSkills(meta.Workspace, message, note, skillBodies, 2400)
		if extra := runtime.ExpandAttachments(meta.Workspace, toRuntimeAtts(atts), 2400); extra != "" {
			if inject != "" {
				inject += "\n"
			}
			inject += extra
		}
	}
	profile := ""
	if a.Memory != nil {
		profile = a.Memory.ProfilePin(800)
	}
	out, runErr = runtime.Run(ctx, runtime.RunRequest{
		SessionID:        sessionID,
		User:             message,
		Inject:           inject,
		Workspace:        meta.Workspace,
		Home:             a.Home.Root,
		Harness:          snap,
		HarnessHash:      hash,
		ModelFingerprint: Fingerprint(a.Config),
		Model:            model,
		ModelWindow:      window,
		Loop:             loop,
		Policy:           pol,
		Fragments:        frags,
		Playbook:         pb,
		Skills:           skills,
		History:          hist,
		RoundSeq:         roundSeq,
		Tools:            tools,
		Client:           client,
		Trace:            a.Traces,
		Events:           a.Kernel.Events(),
		FileHooks:        preHooks,
		StopHooks:        stopHooks,
		ProfileMemory:    profile,
		OnEvent:          wrapped,
		OnShape:          func(r runtime.ShapeReport) { a.RememberShape(sessionID, r) },
		Meter:            meter,
		PullSteer:        func() string { return a.pullSteer(sessionID) },
	})
	loaded, planText := tools.Pins()
	meta.LoadedSkills = loaded
	meta.PlanText = planText
	_ = a.writeSession(meta)
	a.RememberUsage(meter.Snapshot())
	_, _ = a.StageACE(sessionID)
	if a.Threads != nil {
		var blob strings.Builder
		blob.WriteString(meta.Title)
		blob.WriteByte(' ')
		blob.WriteString(message)
		a.Threads.IndexBlob(sessionID, blob.String())
		if runErr == nil {
			a.Threads.SetResume(sessionID, "", false)
		}
	}
	return out, runErr
}

const errBusy errString = "session already running"

func (a *App) extraTools(sessionID string) map[string]runtime.ExtraTool {
	extra := map[string]runtime.ExtraTool{}
	if a.MCP != nil {
		for _, t := range a.MCP.Tools() {
			t := t
			name := "mcp__" + t.Server + "__" + t.Name
			params := t.InputSchema
			if params == nil {
				params = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			extra[name] = runtime.ExtraTool{
				JSON: runtime.ToolJSON{Type: "function", Function: map[string]any{
					"name": name, "description": t.Description, "parameters": params,
				}},
				ReadOnly:  t.ReadOnlyHint,
				OpenWorld: t.OpenWorldHint,
				Exclusive: t.OpenWorldHint || t.DestructiveHint,
				Call: func(argsJSON string) runtime.ToolResult {
					if err := a.Caps.Check(capability.Request{
						Level: capability.Network, Action: name, SessionID: sessionID, ForceAsk: true,
					}); err != nil {
						return runtime.ToolResult{Err: err}
					}
					out, err := a.MCP.Call(t.Server, t.Name, argsJSON)
					return runtime.ToolResult{Content: out, Err: err}
				},
			}
		}
	}
	if a.WASM != nil {
		for _, id := range a.WASM.List() {
			id := id
			name := "wasm__" + id
			extra[name] = runtime.ExtraTool{
				JSON: runtime.ToolJSON{Type: "function", Function: map[string]any{
					"name":        name,
					"description": "Call admitted WASM plugin with JSON in/out (no WASI FS).",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"input": map[string]any{"type": "string"},
							"a":     map[string]any{"type": "integer"},
							"b":     map[string]any{"type": "integer"},
						},
					},
				}},
				Call: func(argsJSON string) runtime.ToolResult {
					if err := a.Caps.Check(capability.Request{
						Level: capability.HighRisk, Action: name, SessionID: sessionID, ForceAsk: true,
					}); err != nil {
						return runtime.ToolResult{Err: err}
					}
					mod := a.WASM.Get(id)
					if mod == nil {
						return runtime.ToolResult{Err: fmt.Errorf("wasm %s not loaded", id)}
					}
					if out, err := mod.CallJSON([]byte(argsJSON)); err == nil {
						text := string(out)
						if logs := mod.Logs(); logs != "" {
							text += "\nlog:" + logs
						}
						return runtime.ToolResult{Content: text}
					}
					var args struct {
						A uint64 `json:"a"`
						B uint64 `json:"b"`
					}
					_ = json.Unmarshal([]byte(argsJSON), &args)
					n, err := mod.CallI32(args.A, args.B)
					if err != nil {
						return runtime.ToolResult{Err: err}
					}
					out := fmt.Sprintf("%d", n)
					if logs := mod.Logs(); logs != "" {
						out += "\nlog:" + logs
					}
					return runtime.ToolResult{Content: out}
				},
			}
		}
	}
	return extra
}

func (a *App) askUserFn(ctx context.Context, sessionID string) func(string) (string, error) {
	return func(question string) (string, error) {
		ev := trace.Event{
			TS:        time.Now().UTC(),
			Type:      trace.TypeAsk,
			Source:    "agent",
			SessionID: sessionID,
			Payload:   map[string]any{"question": question, "text": question},
		}
		if a.Traces != nil {
			_ = a.Traces.Append(ev)
		}
		if a.Hub != nil {
			a.Hub.Publish(ev)
		}
		if a.Gate == nil {
			return "no operator; continue with a reasonable default", nil
		}
		dec, offer, err := a.Gate.AskOffer(ctx, capability.Request{
			Level: capability.ReadWorkspace, Action: "ask_user", Command: question,
			SessionID: sessionID, ForceAsk: true,
		})
		if err != nil {
			return "", err
		}
		a.askMu.Lock()
		ans := a.askAns[offer.ID]
		a.askMu.Unlock()
		if strings.TrimSpace(ans) == "" {
			return "operator decision: " + string(dec), nil
		}
		return ans, nil
	}
}

func (a *App) RunEval(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, nil)
}

func (a *App) RunEvalOpts(ctx context.Context, client runtime.Client, safety bool) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, func(suite *artifact.EvalSuite) {
		if safety {
			already := false
			for _, id := range suite.Safety {
				if id == "no-escape" {
					already = true
					break
				}
			}
			if !already {
				suite.Safety = append(append([]string{}, suite.Safety...), "no-escape")
			}
		}
	})
}

func (a *App) RunEvalTB(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, func(suite *artifact.EvalSuite) {
		suite.HeldIn = append(append([]string{}, suite.HeldIn...), "mkdir-note")
		suite.HeldOut = append(append([]string{}, suite.HeldOut...), "copy-seed")
		if suite.Repeats < 2 {
			suite.Repeats = 2
		}
	})
}

func (a *App) RunEvalSealed(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, eval.ApplySealed)
}

func (a *App) RunEvalTransfer(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, func(suite *artifact.EvalSuite) {
		eval.ApplySealed(suite)
		suite.HeldIn = nil
		suite.HeldOut = append([]string{}, suite.Transfer...)
		suite.Transfer = nil
		suite.Safety = nil
	})
}

func (a *App) RunEvalMut(ctx context.Context, client runtime.Client, mut func(*artifact.EvalSuite)) (eval.RunReport, error) {
	return a.runEval(ctx, client, "", mut)
}

func (a *App) runEval(ctx context.Context, client runtime.Client, model string, mut func(*artifact.EvalSuite)) (eval.RunReport, error) {
	hash := a.ActiveHash()
	snap, err := a.LoadSnapshot(hash)
	if err != nil {
		return eval.RunReport{}, err
	}
	loop, frags, pb, skills, suite, _, err := a.Materials(hash)
	if err != nil {
		return eval.RunReport{}, err
	}
	if mut != nil {
		mut(&suite)
	}
	if client == nil {
		client, err = a.Client()
		if err != nil {
			return eval.RunReport{}, err
		}
	}
	if model == "" {
		model = a.Config.Model
	}
	return a.Eval.Run(ctx, eval.RunOpts{
		Suite:     suite,
		Snapshot:  snap,
		Hash:      hash,
		Client:    client,
		Loop:      loop,
		Fragments: frags,
		Playbook:  pb,
		Skills:    skills,
		Trace:     a.Traces,
		SessionID: "eval-" + newID()[:8],
		Model:     model,
	})
}

func (a *App) EvolveOnce(ctx context.Context, client runtime.Client, failed map[string]string) (evolve.CycleResult, error) {
	return a.EvolveWith(ctx, client, failed, EvolveRun{K: 3})
}

func (a *App) EvolveK(ctx context.Context, client runtime.Client, failed map[string]string, k int) (evolve.CycleResult, error) {
	return a.EvolveWith(ctx, client, failed, EvolveRun{K: k})
}

// EvolveRun is the operator-facing evolve control surface. PromoteActive is
// off by default: Harbor-accepted edits land on refs/canary only.
type EvolveRun struct {
	K              int
	Rounds         int
	PromoteActive  bool
	Sealed         bool
	MaxWall        time.Duration
	MaxUSD         float64
	PromoteRepeats int
}

func (a *App) EvolveWith(ctx context.Context, client runtime.Client, failed map[string]string, run EvolveRun) (evolve.CycleResult, error) {
	if run.Rounds > 1 {
		all, err := a.EvolveLoop(ctx, client, failed, run)
		if err != nil {
			return evolve.CycleResult{}, err
		}
		if len(all) == 0 {
			return evolve.CycleResult{}, fmt.Errorf("no evolve rounds")
		}
		return all[len(all)-1], nil
	}
	return a.evolveOnce(ctx, client, failed, run)
}

func (a *App) EvolveLoop(ctx context.Context, client runtime.Client, failed map[string]string, run EvolveRun) ([]evolve.CycleResult, error) {
	n := run.Rounds
	if n <= 0 {
		n = 1
	}
	start := time.Now()
	var out []evolve.CycleResult
	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if run.MaxWall > 0 && time.Since(start) > run.MaxWall {
			break
		}
		if run.MaxUSD > 0 && a.spentUSD() >= run.MaxUSD {
			break
		}
		res, err := a.evolveOnce(ctx, client, failed, run)
		if err != nil {
			return out, err
		}
		out = append(out, res)
		failed = nil
	}
	return out, nil
}

func (a *App) spentUSD() float64 {
	a.usageMu.Lock()
	defer a.usageMu.Unlock()
	if a.lastUsage == nil {
		return 0
	}
	v, _ := a.lastUsage["usd"].(float64)
	return v
}

func (a *App) evolveOnce(ctx context.Context, client runtime.Client, failed map[string]string, run EvolveRun) (evolve.CycleResult, error) {
	activeHash := a.ActiveHash()
	parentHash := evolve.SelectParent(a.Archive.List(), activeHash)
	if parentHash == "" {
		parentHash = activeHash
	}
	if _, err := a.LoadSnapshot(parentHash); err != nil {
		parentHash = activeHash
	}
	parentSet, err := a.materialSet(parentHash)
	if err != nil {
		return evolve.CycleResult{}, err
	}
	baseSet, err := a.materialSet(activeHash)
	if err != nil {
		return evolve.CycleResult{}, err
	}
	if client == nil {
		client, err = a.Client()
		if err != nil {
			return evolve.CycleResult{}, err
		}
	}
	if failed == nil {
		failed = map[string]string{}
	}
	_, _, _, _, suite, _, err := a.Materials(activeHash)
	if err != nil {
		return evolve.CycleResult{}, err
	}
	if run.Sealed {
		eval.ApplySealed(&suite)
	}
	k := run.K
	if k <= 0 {
		k = 3
	}
	sid := "evolve-" + newID()[:8]
	res, err := a.Evolve.Cycle(ctx, evolve.CycleOpts{
		Parent:         parentSet,
		Baseline:       baseSet,
		Suite:          suite,
		Client:         client,
		Trace:          a.Traces,
		SessionID:      sid,
		Model:          a.Config.Model,
		FailedTasks:    failed,
		K:              k,
		PromoteActive:  run.PromoteActive,
		HeldOut:        suite.HeldOut,
		StagingHash:    a.Refs.GetOrEmpty(artifact.RefStaging),
		PromoteRepeats: run.PromoteRepeats,
	})
	if err != nil {
		return res, err
	}
	if res.Promoted != "" && len(suite.Transfer) > 0 {
		a.markTransfer(ctx, client, activeHash, res.Promoted, suite)
	}
	return res, nil
}

func (a *App) markTransfer(ctx context.Context, client runtime.Client, baseHash, canary string, suite artifact.EvalSuite) {
	baseSet, err := a.materialSet(baseHash)
	if err != nil {
		return
	}
	candSet, err := a.materialSet(canary)
	if err != nil {
		return
	}
	xfer := suite
	xfer.HeldIn = nil
	xfer.HeldOut = append([]string{}, suite.Transfer...)
	xfer.Transfer = nil
	xfer.Safety = nil
	baseRep, err := a.Eval.Run(ctx, eval.RunOpts{
		Suite: xfer, Snapshot: baseSet.Snap, Hash: baseSet.Hash, Client: client,
		Loop: baseSet.Loop, Fragments: baseSet.Fragments, Playbook: baseSet.Playbook, Skills: baseSet.Skills,
		SessionID: "xfer-base", Model: a.Config.Model,
	})
	if err != nil {
		return
	}
	candRep, err := a.Eval.Run(ctx, eval.RunOpts{
		Suite: xfer, Snapshot: candSet.Snap, Hash: candSet.Hash, Client: client,
		Loop: candSet.Loop, Fragments: candSet.Fragments, Playbook: candSet.Playbook, Skills: candSet.Skills,
		SessionID: "xfer-cand", Model: a.Config.Model,
	})
	if err != nil {
		return
	}
	fail := candRep.Metrics.HeldOutPass < baseRep.Metrics.HeldOutPass
	_ = a.Archive.FlagTransfer(canary, fail)
}

func (a *App) materialSet(hash string) (evolve.MaterialSet, error) {
	snap, err := a.LoadSnapshot(hash)
	if err != nil {
		return evolve.MaterialSet{}, err
	}
	loop, frags, pb, skills, _, _, err := a.Materials(hash)
	if err != nil {
		return evolve.MaterialSet{}, err
	}
	return evolve.MaterialSet{
		Snap: snap, Hash: hash, Loop: loop, Fragments: frags, Playbook: pb, Skills: skills,
	}, nil
}

// StageACE writes an online playbook delta to refs/staging. It never moves
// refs/active — Harbor still owns promotion. This is the ACE online path.
func (a *App) StageACE(sessionID string) (string, error) {
	meta, err := a.GetSession(sessionID)
	if err != nil {
		return "", nil
	}
	hash := session.ResolveHarness(session.NormalizePolicy(meta.HarnessPolicy), meta.Harness, a.ActiveHash())
	if hash == "" {
		return "", nil
	}
	events, err := a.Traces.Read(sessionID)
	if err != nil || len(events) == 0 {
		return "", err
	}
	var lastEnd *trace.Event
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == trace.TypeTurnEnd {
			lastEnd = &events[i]
			break
		}
	}
	if lastEnd == nil {
		return "", nil
	}
	ok, _ := lastEnd.Payload["ok"].(bool)
	if !ok {
		return "", nil
	}
	base, err := a.LoadSnapshot(hash)
	if err != nil {
		return "", err
	}
	_, _, pb, _, _, _, err := a.Materials(hash)
	if err != nil {
		return "", err
	}
	if st := a.Refs.GetOrEmpty(artifact.RefStaging); st != "" {
		if ss, err := a.LoadSnapshot(st); err == nil && ss.Playbook != "" {
			if v, _, err := artifact.Decode[artifact.Playbook](a.CAS, ss.Playbook); err == nil {
				pb = v
			}
		}
	}
	stripped := stripACEBodies(events)
	ws := ""
	if m, err := a.GetSession(sessionID); err == nil {
		ws = m.Workspace
	}
	if notes := runtime.ReadNotes(runtime.BindSpill(a.Home.Root, ws, sessionID)); notes != "" {
		stripped = append([]trace.Event{{
			Type:      trace.TypeUser,
			SessionID: sessionID,
			Payload:   map[string]any{"text": notes},
		}}, stripped...)
	}
	ref := evolve.Reflect(stripped, nil, pb)
	curated := evolve.Curate(pb, ref)
	if evolve.PlaybooksEqual(pb, curated) {
		return "", nil
	}
	pbHash, err := a.CAS.Put(artifact.KindPlaybook, curated.ID, curated)
	if err != nil {
		return "", err
	}
	next := artifact.CloneSnapshot(base)
	next.Parent = hash
	next.Playbook = pbHash
	next.Note = "online-ace-stage"
	staged, err := a.CAS.PutSnapshot(next)
	if err != nil {
		return "", err
	}
	if err := a.Refs.Set(artifact.RefStaging, staged); err != nil {
		return "", err
	}
	_ = a.Archive.Add(evolve.Node{
		ID: staged, Parent: hash, Snapshot: staged, ProposalID: "online-ace-stage",
		Note: "online-ace-stage",
	})
	_, _ = a.Journal.Append("ace.stage", map[string]string{"hash": staged, "session": sessionID})
	return staged, nil
}

func stripACEBodies(evs []trace.Event) []trace.Event {
	out := make([]trace.Event, len(evs))
	copy(out, evs)
	for i := range out {
		if out[i].Type != trace.TypeToolResult {
			continue
		}
		p := map[string]any{}
		for k, v := range out[i].Payload {
			p[k] = v
		}
		name, _ := p["name"].(string)
		id, _ := p["id"].(string)
		content, _ := p["content"].(string)
		if strings.HasPrefix(content, "ERROR:") {
			if len(content) > 240 {
				content = content[:240]
			}
			p["content"] = content
		} else {
			p["content"] = name + " " + id
		}
		out[i].Payload = p
	}
	return out
}

func (a *App) ListHarnesses() (map[string]string, error) {
	return a.Refs.List()
}

func (a *App) BestOfN(ctx context.Context, n int, client runtime.Client) (BestOfNReport, error) {
	if n < 1 {
		n = 3
	}
	rep := BestOfNReport{N: n, Kind: "repeat"}
	for i := 0; i < n; i++ {
		r, err := a.RunEval(ctx, client)
		if err != nil {
			return rep, err
		}
		rep.Reports = append(rep.Reports, r)
		if i == 0 || evalScore(r) > evalScore(rep.Best) {
			rep.Best = r
		}
	}
	return rep, nil
}

func (a *App) BestOfModels(ctx context.Context, models []string, clients map[string]runtime.Client) (BestOfNReport, error) {
	if len(models) == 0 {
		models = append([]string{a.Config.Model}, a.Config.Models...)
	}
	seen := map[string]bool{}
	var clean []string
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		clean = append(clean, m)
	}
	if len(clean) == 0 {
		clean = []string{a.Config.Model}
	}
	type row struct {
		model string
		r     eval.RunReport
		err   error
	}
	ch := make(chan row, len(clean))
	for _, m := range clean {
		m := m
		go func() {
			cl := clients[m]
			if cl == nil && clients != nil {
				cl = clients["*"]
			}
			r, err := a.runEval(ctx, cl, m, nil)
			ch <- row{model: m, r: r, err: err}
		}()
	}
	rep := BestOfNReport{N: len(clean), Kind: "models"}
	for i := 0; i < len(clean); i++ {
		got := <-ch
		if got.err != nil {
			return rep, got.err
		}
		rep.Models = append(rep.Models, ModelRun{Model: got.model, Report: got.r})
		rep.Reports = append(rep.Reports, got.r)
		if i == 0 || evalScore(got.r) > evalScore(rep.Best) {
			rep.Best = got.r
			rep.BestModel = got.model
		}
	}
	return rep, nil
}

type BestOfNReport struct {
	N         int              `json:"n"`
	Kind      string           `json:"kind,omitempty"`
	Reports   []eval.RunReport `json:"reports"`
	Best      eval.RunReport   `json:"best"`
	BestModel string           `json:"best_model,omitempty"`
	Models    []ModelRun       `json:"models,omitempty"`
}

type ModelRun struct {
	Model  string         `json:"model"`
	Report eval.RunReport `json:"report"`
}

func evalScore(r eval.RunReport) int {
	return r.Metrics.HeldInPass + r.Metrics.HeldOutPass
}

func (a *App) DiffDetail(aHash, bHash string) (map[string]any, error) {
	left, err := a.LoadSnapshot(aHash)
	if err != nil {
		return nil, err
	}
	right, err := a.LoadSnapshot(bHash)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"text":   artifact.SnapshotDiff(left, right),
		"fields": artifact.SnapshotDiffFields(left, right),
		"a":      left,
		"b":      right,
	}, nil
}

func (a *App) Diff(aHash, bHash string) (string, error) {
	d, err := a.DiffDetail(aHash, bHash)
	if err != nil {
		return "", err
	}
	text, _ := d["text"].(string)
	return text, nil
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
