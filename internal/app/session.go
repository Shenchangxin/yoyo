package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/eval"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

type SessionMeta struct {
	ID               string    `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	Workspace        string    `json:"workspace"`
	Harness          string    `json:"harness"`
	ModelFingerprint string    `json:"model_fingerprint"`
	Title            string    `json:"title,omitempty"`
	Archived         bool      `json:"archived"`
	Pinned           bool      `json:"pinned"`
	Model            string    `json:"model,omitempty"`
	LoadedSkills     []string  `json:"loaded_skills,omitempty"`
	PlanText         string    `json:"plan_text,omitempty"`
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
		ModelFingerprint: Fingerprint(a.Config),
		Title:            "session",
	}
	b, _ := json.MarshalIndent(meta, "", "  ")
	path := filepath.Join(a.Home.Sessions(), id+".meta.json")
	return meta, os.WriteFile(path, b, 0o644)
}

func (a *App) GetSession(id string) (SessionMeta, error) {
	b, err := os.ReadFile(filepath.Join(a.Home.Sessions(), id+".meta.json"))
	if err != nil {
		return SessionMeta{ID: id}, err
	}
	var m SessionMeta
	return m, json.Unmarshal(b, &m)
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
	if a.Running(sessionID) {
		a.Enqueue(sessionID, QueuedTurn{Text: message, Plan: plan, Attachments: atts})
		return ErrQueued
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	ctx, release, err := a.acquireRun(sessionID, ctx)
	if err != nil {
		cancel()
		if err == errBusy {
			a.Enqueue(sessionID, QueuedTurn{Text: message, Plan: plan, Attachments: atts})
			return ErrQueued
		}
		return err
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
		goruntime.LockOSThread()
		_, _ = a.sendLocked(ctx, sessionID, message, nil, nil, plan, atts)
	}()
	return nil
}

func (a *App) SendOpts(ctx context.Context, sessionID, message string, client runtime.Client, onEvent func(trace.Event), plan bool) (string, error) {
	ctx, release, err := a.acquireRun(sessionID, ctx)
	if err != nil {
		return "", err
	}
	defer release()
	return a.sendLocked(ctx, sessionID, message, client, onEvent, plan, nil)
}

func (a *App) sendLocked(ctx context.Context, sessionID, message string, client runtime.Client, onEvent func(trace.Event), plan bool, atts []Attachment) (string, error) {
	meta, err := a.GetSession(sessionID)
	if err != nil {
		meta, err = a.NewSession("")
		if err != nil {
			return "", err
		}
		sessionID = meta.ID
	}
	hash := a.ActiveHash()
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
	for _, s := range disk {
		skillBodies[s.Name] = s.Body
	}
	model := a.Config.Model
	if meta.Model != "" {
		model = meta.Model
	}
	tools := &runtime.WorkspaceTools{
		Workspace: meta.Workspace,
		SessionID: sessionID,
		Caps:      a.Caps,
		Skills:    skillBodies,
		Loaded:    append([]string(nil), meta.LoadedSkills...),
		Policy:    pol,
		PlanMode:  loop.PlanMode,
		Ctx:       ctx,
		Extra:     a.extraTools(sessionID),
		Spill:     runtime.BindSpill(a.Home.Root, meta.Workspace, sessionID),
		PlanText:  meta.PlanText,
	}
	hist := []runtime.Message{}
	if evs, err := a.Traces.Read(sessionID); err == nil {
		hist = runtime.MessagesFromEvents(evs)
	}
	window := runtime.ModelContextWindow(model)
	hist = runtime.MaybeCheckpoint(a.Traces, sessionID, hist, loop, tools.Spill, client, model, window)
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

	if meta.Title == "" || meta.Title == "session" {
		meta.Title = titleFrom(message)
		_ = a.writeSession(meta)
	}
	note := hash
	if snap.Note != "" {
		note = hash + " " + snap.Note
	}
	inject, _ := runtime.ExpandMentionsSkills(meta.Workspace, message, note, skillBodies, 2400)
	if extra := runtime.ExpandAttachments(meta.Workspace, toRuntimeAtts(atts), 2400); extra != "" {
		if inject != "" {
			inject += "\n"
		}
		inject += extra
	}
	out, runErr := runtime.Run(ctx, runtime.RunRequest{
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
		Tools:            tools,
		Client:           client,
		Trace:            a.Traces,
		Events:           a.Kernel.Events(),
		FileHooks:        runtime.LoadFileHooks(meta.Workspace),
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
					"description": "Call admitted WASM export with integer arguments a,b (L2 harness plugin).",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"a": map[string]any{"type": "integer"},
							"b": map[string]any{"type": "integer"},
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

func (a *App) RunEval(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, nil)
}

func (a *App) RunEvalOpts(ctx context.Context, client runtime.Client, safety bool) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, func(suite *artifact.EvalSuite) {
		if safety {
			suite.Safety = append(append([]string{}, suite.Safety...), "no-escape")
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
	return a.EvolveK(ctx, client, failed, 3)
}

func (a *App) EvolveK(ctx context.Context, client runtime.Client, failed map[string]string, k int) (evolve.CycleResult, error) {
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
		report, err := a.RunEval(ctx, client)
		if err == nil {
			for _, r := range report.Results {
				if !r.Pass {
					failed[r.ID] = r.Error
					if failed[r.ID] == "" {
						failed[r.ID] = "verifier_fail"
					}
				}
			}
		}
	}
	_, _, _, _, suite, _, err := a.Materials(activeHash)
	if err != nil {
		return evolve.CycleResult{}, err
	}
	sid := "evolve-" + newID()[:8]
	return a.Evolve.Cycle(ctx, evolve.CycleOpts{
		Parent:        parentSet,
		Baseline:      baseSet,
		Suite:         suite,
		Client:        client,
		Trace:         a.Traces,
		SessionID:     sid,
		Model:         a.Config.Model,
		FailedTasks:   failed,
		K:             k,
		PromoteCanary: true,
		HeldOut:       suite.HeldOut,
		StagingHash:   a.Refs.GetOrEmpty(artifact.RefStaging),
	})
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
	active := a.ActiveHash()
	if active == "" {
		return "", nil
	}
	events, err := a.Traces.Read(sessionID)
	if err != nil || len(events) == 0 {
		return "", err
	}
	base, err := a.LoadSnapshot(active)
	if err != nil {
		return "", err
	}
	_, _, pb, _, _, _, err := a.Materials(active)
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
	next.Parent = active
	next.Playbook = pbHash
	next.Note = "online-ace-stage"
	hash, err := a.CAS.PutSnapshot(next)
	if err != nil {
		return "", err
	}
	if err := a.Refs.Set(artifact.RefStaging, hash); err != nil {
		return "", err
	}
	_ = a.Archive.Add(evolve.Node{
		ID: hash, Parent: active, Snapshot: hash, ProposalID: "online-ace-stage",
		Note: "online-ace-stage",
	})
	_, _ = a.Journal.Append("ace.stage", map[string]string{"hash": hash, "session": sessionID})
	return hash, nil
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

func (a *App) LoadWASM(id string, bin []byte, export string) error {
	if err := a.Caps.Check(capability.Request{Level: capability.HighRisk, Action: "load_wasm", SessionID: "ui"}); err != nil {
		if !a.Config.AutoAllow {
			return err
		}
	}
	_, err := evolve.AdmitWASM(a.Kernel, a.WASM, id, bin, export)
	return err
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
