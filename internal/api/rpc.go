package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	goruntime "runtime"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/project"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type RPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

func ServeRPC(ctx context.Context, a *app.App, r io.Reader, w io.Writer) error {
	dec := json.NewDecoder(r)
	var mu sync.Mutex
	enc := json.NewEncoder(w)
	ch, unsub := a.Hub.Subscribe("*")
	defer unsub()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				mu.Lock()
				_ = enc.Encode(map[string]any{
					"jsonrpc": "2.0",
					"method":  "item.event",
					"params": map[string]any{
						"type":      string(ev.Type),
						"item_kind": ev.ItemKind,
						"session":   ev.SessionID,
						"source":    ev.Source,
						"payload":   ev.Payload,
					},
				})
				if ev.Type == "approval" && ev.Source == "gate" {
					_ = enc.Encode(map[string]any{
						"jsonrpc": "2.0",
						"method":  "approval.request",
						"params": map[string]any{
							"id":      ev.Payload["id"],
							"session": ev.SessionID,
							"action":  ev.Payload["action"],
							"command": ev.Payload["command"],
							"path":    ev.Payload["path"],
							"level":   ev.Payload["level"],
						},
					})
				}
				if ev.Type == "turn_end" {
					_ = enc.Encode(map[string]any{
						"jsonrpc": "2.0",
						"method":  "turn.completed",
						"params": map[string]any{
							"session": ev.SessionID,
							"stop":    ev.Payload["stop"],
							"ok":      ev.Payload["ok"],
							"text":    ev.Payload["text"],
						},
					})
				}
				mu.Unlock()
			}
		}
	}()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var req RPCRequest
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		mu.Lock()
		err := enc.Encode(Dispatch(ctx, a, req))
		mu.Unlock()
		if err != nil {
			return err
		}
	}
}

func handleRPC(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST", 405)
			return
		}
		var req RPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONStatus(w, 400, RPCResponse{JSONRPC: "2.0", Error: &RPCError{Code: -32700, Message: err.Error()}})
			return
		}
		writeJSON(w, Dispatch(r.Context(), a, req))
	}
}

func Dispatch(ctx context.Context, a *app.App, req RPCRequest) RPCResponse {
	res := RPCResponse{JSONRPC: "2.0", ID: req.ID}
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		res.Error = &RPCError{Code: -32600, Message: "jsonrpc must be 2.0"}
		return res
	}
	params := req.Params
	if len(params) == 0 {
		params = []byte("{}")
	}
	result, err := callMethod(ctx, a, req.Method, params)
	if err != nil {
		code := -32000
		var data any
		if e, ok := app.AsL3(err); ok {
			code = 403
			data = e
		}
		var b runtime.ErrBudget
		if errors.As(err, &b) {
			code = 429
			data = b
		}
		res.Error = &RPCError{Code: code, Message: err.Error(), Data: data}
		return res
	}
	res.Result = result
	return res
}

func callMethod(ctx context.Context, a *app.App, method string, params json.RawMessage) (any, error) {
	if method == "" {
		return map[string]any{"ok": true}, nil
	}
	switch method {
	case "health":
		return a.Health(), nil
	case "thread.start":
		var p struct {
			Workspace string `json:"workspace"`
		}
		_ = json.Unmarshal(params, &p)
		return a.NewSession(p.Workspace)
	case "turn.start":
		var p struct {
			Session     string `json:"session"`
			Text        string `json:"text"`
			Plan        bool   `json:"plan"`
			Wait        bool   `json:"wait"`
			Attachments []any  `json:"attachments"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if !p.Wait {
			var atts []app.Attachment
			for _, raw := range p.Attachments {
				b, _ := json.Marshal(raw)
				var att app.Attachment
				_ = json.Unmarshal(b, &att)
				atts = append(atts, att)
			}
			if err := a.StartSendOpts(p.Session, p.Text, p.Plan, atts); err != nil {
				if errors.Is(err, app.ErrQueued) {
					return map[string]any{"ok": true, "queued": true}, nil
				}
				return nil, err
			}
			return map[string]any{"ok": true, "async": true}, nil
		}
		out, err := a.SendOpts(ctx, p.Session, p.Text, nil, a.Hub.Publish, p.Plan)
		if err != nil {
			return nil, err
		}
		return map[string]any{"text": out}, nil
	case "turn.retry":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		if err := a.RetrySession(p.Session); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	case "turn.interrupt":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.Interrupt(p.Session)
	case "trajectory.get":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		id, err := a.ResolveSessionID(p.Session)
		if err != nil {
			return nil, err
		}
		return a.Trajectory(id)
	case "trace.get":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		id, err := a.ResolveSessionID(p.Session)
		if err != nil {
			return nil, err
		}
		return a.SessionTrace(id)
	case "spill.get":
		var p struct {
			Session string `json:"session"`
			ID      string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		id, err := a.ResolveSessionID(p.Session)
		if err != nil {
			return nil, err
		}
		return a.SpillBlob(id, p.ID)
	case "session.dump":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.DumpSession(p.Session)
	case "session.list":
		return a.ListSessionIndex(), nil
	case "session.resolve":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		id, err := a.ResolveSessionID(p.Session)
		if err != nil {
			return nil, err
		}
		return map[string]any{"id": id}, nil
	case "approvals.list":
		return a.Gate.Pending(), nil
	case "approvals.resolve", "approval.respond":
		var p struct {
			ID       string `json:"id"`
			Decision string `json:"decision"`
			Answer   string `json:"answer"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ResolveApprovalAnswer(p.ID, p.Decision, p.Answer)
	case "harness.get":
		return a.HarnessState()
	case "harness.lineage":
		return a.HarnessLineage()
	case "harness.reveal":
		var p struct {
			Hash string `json:"hash"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.RevealHarness(p.Hash)
	case "harness.checkout":
		var p struct {
			Hash string `json:"hash"`
			L3   bool   `json:"l3"`
		}
		_ = json.Unmarshal(params, &p)
		if err := a.CheckoutOpts(p.Hash, app.CheckoutOpts{ConfirmL3: p.L3}); err != nil {
			return nil, err
		}
		return map[string]any{"active": a.ActiveHash()}, nil
	case "harness.rollback":
		if err := a.Rollback(); err != nil {
			return nil, err
		}
		return map[string]any{"active": a.ActiveHash()}, nil
	case "harness.diff":
		var p struct {
			A string `json:"a"`
			B string `json:"b"`
		}
		_ = json.Unmarshal(params, &p)
		return a.DiffDetail(p.A, p.B)
	case "thread.list":
		return a.ListSessions()
	case "thread.resume":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ResumeSession(p.Session)
	case "thread.rename":
		var p struct {
			Session string `json:"session"`
			Title   string `json:"title"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.RenameSession(p.Session, p.Title)
	case "thread.fork":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ForkSession(p.Session)
	case "thread.delete":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.DeleteSession(p.Session)
	case "thread.archive":
		var p struct {
			Session  string `json:"session"`
			Archived *bool  `json:"archived"`
		}
		_ = json.Unmarshal(params, &p)
		archived := true
		if p.Archived != nil {
			archived = *p.Archived
		}
		return a.ArchiveSession(p.Session, archived)
	case "thread.pin":
		var p struct {
			Session string `json:"session"`
			Pinned  bool   `json:"pinned"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PinSession(p.Session, p.Pinned)
	case "thread.search":
		var p struct {
			Query           string `json:"query"`
			IncludeArchived bool   `json:"include_archived"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SearchSessions(p.Query, p.IncludeArchived)
	case "thread.export":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		text, err := a.ExportSession(p.Session)
		return map[string]any{"markdown": text}, err
	case "thread.model.set":
		var p struct {
			Session string `json:"session"`
			Model   string `json:"model"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionModel(p.Session, p.Model)
	case "thread.auth.set":
		var p struct {
			Session string `json:"session"`
			Mode    string `json:"mode"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionAuthMode(p.Session, p.Mode)
	case "thread.workspace.set":
		var p struct {
			Session   string `json:"session"`
			Workspace string `json:"workspace"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionWorkspace(p.Session, p.Workspace)
	case "thread.isolate.set":
		var p struct {
			Session string `json:"session"`
			Isolate bool   `json:"isolate"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionIsolate(p.Session, p.Isolate)
	case "thread.skills.pin":
		var p struct {
			Session string   `json:"session"`
			Names   []string `json:"names"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SetSessionPinnedSkills(p.Session, p.Names)
	case "thread.compact":
		var p struct {
			Session string `json:"session"`
			Focus   string `json:"focus"`
		}
		_ = json.Unmarshal(params, &p)
		note, err := a.CompactSessionFocus(p.Session, p.Focus)
		return map[string]any{"note": note}, err
	case "thread.queue.list":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.QueueList(p.Session), nil
	case "turn.steer":
		var p struct {
			Session string `json:"session"`
			Text    string `json:"text"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.Steer(p.Session, p.Text)
	case "fs.search":
		var p struct {
			Workspace string `json:"workspace"`
			Query     string `json:"query"`
			Limit     int    `json:"limit"`
		}
		_ = json.Unmarshal(params, &p)
		ws := p.Workspace
		if ws == "" {
			ws = a.Workspace()
		}
		return runtime.FuzzySearch(ws, p.Query, p.Limit), nil
	case "skills.list":
		var p struct {
			Workspace string `json:"workspace"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ListSkills(p.Workspace), nil
	case "skills.get":
		var p struct {
			Workspace string `json:"workspace"`
			Name      string `json:"name"`
		}
		_ = json.Unmarshal(params, &p)
		sk := a.GetSkill(p.Workspace, p.Name)
		if sk == nil {
			return nil, fmt.Errorf("unknown skill")
		}
		return sk, nil
	case "skills.market":
		var p struct {
			Refresh bool `json:"refresh"`
		}
		_ = json.Unmarshal(params, &p)
		return a.SkillMarket(p.Refresh)
	case "skills.install":
		var p struct {
			Slug string `json:"slug"`
		}
		_ = json.Unmarshal(params, &p)
		return a.InstallMarketSkill(p.Slug)
	case "skills.uninstall":
		var p struct {
			Slug string `json:"slug"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.UninstallMarketSkill(p.Slug)
	case "host.open":
		var p struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
		}
		_ = json.Unmarshal(params, &p)
		switch p.Kind {
		case "editor":
			return map[string]any{"ok": true}, a.OpenInEditor(p.Path)
		case "terminal":
			return map[string]any{"ok": true}, a.OpenWorkspaceTerminal(p.Path)
		default:
			return map[string]any{"ok": true}, a.OpenPath(p.Path)
		}
	case "key.status":
		return a.Vault.Status(), nil
	case "mcp.stop":
		var p struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.StopMCP(p.Name)
	case "mcp.list":
		return map[string]any{"servers": a.MCP.Info(), "tools": a.MCP.Tools()}, nil
	case "logs.tail":
		var p struct {
			Limit int `json:"limit"`
		}
		_ = json.Unmarshal(params, &p)
		return a.Logs(p.Limit), nil
	case "doctor":
		return a.Doctor(), nil
	case "about":
		h := a.Health()
		h["goos"] = goruntime.GOOS
		return h, nil
	case "update.check":
		return a.CheckUpdate(), nil
	case "playbook.get":
		return a.Playbook()
	case "playbook.rate":
		var p struct {
			ID      string `json:"id"`
			Helpful bool   `json:"helpful"`
		}
		_ = json.Unmarshal(params, &p)
		return a.RatePlaybook(p.ID, p.Helpful)
	case "workspace.diff":
		var p struct {
			Workspace string `json:"workspace"`
		}
		_ = json.Unmarshal(params, &p)
		text, err := a.WorkspaceDiff(p.Workspace)
		if err != nil {
			return nil, err
		}
		return map[string]any{"diff": text}, nil
	case "eval.run_safety":
		tctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		return a.RunEvalOpts(tctx, nil, true)
	case "eval.run":
		tctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		return a.RunEval(tctx, nil)
	case "eval.best_of_n":
		var p struct {
			N int `json:"n"`
		}
		_ = json.Unmarshal(params, &p)
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.BestOfN(tctx, p.N, nil)
	case "eval.best_of_models":
		var p struct {
			Models []string `json:"models"`
		}
		_ = json.Unmarshal(params, &p)
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.BestOfModels(tctx, p.Models, nil)
	case "eval.run_tb":
		tctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		return a.RunEvalTB(tctx, nil)
	case "eval.run_sealed":
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.RunEvalSealed(tctx, nil)
	case "eval.run_transfer":
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.RunEvalTransfer(tctx, nil)
	case "eval.run_index":
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.RunEvalIndex(tctx, nil)
	case "eval.run_behavior":
		tctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		return a.RunEvalBehavior(tctx, nil)
	case "eval.last":
		return a.LastEval(), nil
	case "evolve.last":
		return a.LastEvolve(), nil
	case "eval.baselines":
		var p struct {
			N      int     `json:"n"`
			Sealed bool    `json:"sealed"`
			MaxUSD float64 `json:"max_usd"`
		}
		_ = json.Unmarshal(params, &p)
		tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
		defer cancel()
		return a.SearchBaselines(tctx, nil, p.N, app.EvolveRun{Sealed: p.Sealed, MaxUSD: p.MaxUSD, Baselines: p.N})
	case "evolve.run":
		var p struct {
			K         int     `json:"k"`
			Rounds    int     `json:"rounds"`
			Promote   bool    `json:"promote"`
			Sealed    bool    `json:"sealed"`
			Behavior  bool    `json:"behavior"`
			Index     bool    `json:"index"`
			Baselines int     `json:"baselines"`
			MaxUSD    float64 `json:"max_usd"`
		}
		_ = json.Unmarshal(params, &p)
		timeout := 15 * time.Minute
		if p.Rounds > 1 {
			timeout = time.Duration(p.Rounds) * 15 * time.Minute
			if timeout > 2*time.Hour {
				timeout = 2 * time.Hour
			}
		}
		tctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return a.EvolveWith(tctx, nil, nil, app.EvolveRun{
			K: p.K, Rounds: p.Rounds, PromoteActive: p.Promote, Sealed: p.Sealed,
			Behavior: p.Behavior, IndexTransfer: p.Index, Baselines: p.Baselines, MaxUSD: p.MaxUSD,
		})
	case "turn.running":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"running": a.Running(p.Session)}, nil
	case "turn.running_ids":
		return a.RunningIDs(), nil
	case "turn.running_status":
		return a.RunningStatus(), nil
	case "context.get":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ContextUsage(p.Session), nil
	case "config.get":
		return a.Config, nil
	case "config.set":
		_ = json.Unmarshal(params, &a.Config)
		return map[string]any{"ok": true}, a.SaveConfig()
	case "provider.test":
		return a.TestProvider(), nil
	case "mcp.replace":
		var p struct {
			Servers []app.MCPServerConfig `json:"servers"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ReplaceMCP(p.Servers)
	case "key.set":
		var p struct {
			Value string `json:"value"`
		}
		_ = json.Unmarshal(params, &p)
		a.Vault.Set("default", p.Value)
		return map[string]any{"ok": true}, nil
	case "mcp.start":
		var p struct {
			Name    string   `json:"name"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true, "tools": a.MCP.Tools()}, a.StartMCP(p.Name, p.Command, p.Args)
	case "archive.list":
		return a.Archive.List(), nil
	case "plugins.list":
		return map[string]any{"fibers": a.Kernel.Fibers(), "wasm": a.WASM.List(), "mcp": a.MCP.Info(), "tools": a.MCP.Tools()}, nil
	case "fiber.unload":
		var p struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(params, &p)
		if f := a.Kernel.Fiber(p.Name); f != nil {
			return map[string]any{"ok": true}, f.Dispose()
		}
		return map[string]any{"ok": true}, a.WASM.Unload(p.Name)
	case "workspace.hunks":
		var p struct {
			Workspace string `json:"workspace"`
		}
		_ = json.Unmarshal(params, &p)
		return a.WorkspaceHunks(p.Workspace)
	case "workspace.preview":
		var p struct {
			Workspace string `json:"workspace"`
			Path      string `json:"path"`
		}
		_ = json.Unmarshal(params, &p)
		return a.PreviewWorkspaceFile(p.Workspace, p.Path)
	case "workspace.apply_hunks":
		var p struct {
			Workspace string   `json:"workspace"`
			IDs       []string `json:"ids"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ApplyWorkspaceHunks(p.Workspace, p.IDs)
	case "workspace.reverse_hunks":
		var p struct {
			Workspace string   `json:"workspace"`
			IDs       []string `json:"ids"`
			Snapshot  string   `json:"snapshot"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ReverseWorkspaceHunks(p.Workspace, p.IDs, p.Snapshot)
	case "update.status":
		p := a.StagingPath()
		st, err := os.Stat(p)
		size := int64(0)
		if err == nil {
			size = st.Size()
		}
		return map[string]any{"staging": p, "exists": err == nil, "size": size}, nil
	case "update.apply":
		var p struct {
			Exe string `json:"exe"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ApplyStagedUpdate(p.Exe)
	case "inbox.list":
		return a.InboxList(), nil
	case "inbox.dismiss":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": a.InboxDismiss(p.ID)}, nil
	case "inbox.read":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		a.InboxMarkRead(p.ID)
		return map[string]any{"ok": true}, nil
	case "projects.list":
		return a.ProjectsList(), nil
	case "projects.create":
		var p project.Project
		_ = json.Unmarshal(params, &p)
		return a.ProjectCreate(p), nil
	case "memory.list":
		var p struct {
			Q    string `json:"q"`
			Kind string `json:"kind"`
		}
		_ = json.Unmarshal(params, &p)
		return a.MemoryList(p.Q, p.Kind), nil
	case "memory.promote":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.MemoryPromote(p.ID), nil
	case "memory.forget":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.MemoryForget(p.ID), nil
	case "memory.write":
		var p struct {
			Kind    string `json:"kind"`
			Text    string `json:"text"`
			Project string `json:"project"`
		}
		_ = json.Unmarshal(params, &p)
		return a.MemoryWrite(p.Kind, p.Text, p.Project), nil
	case "schedule.list":
		return a.ScheduleList(), nil
	case "schedule.create":
		var p schedule.Job
		_ = json.Unmarshal(params, &p)
		return a.ScheduleCreate(p), nil
	case "schedule.cancel":
		var p struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ScheduleCancel(p.ID), nil
	case "connectors.list":
		return a.ConnectorsList(), nil
	case "connectors.catalog":
		return a.ConnectorCatalog(), nil
	case "connectors.connect":
		var p connector.Account
		_ = json.Unmarshal(params, &p)
		return a.ConnectorConnect(p), nil
	case "connectors.auth_url":
		var p struct {
			Provider string `json:"provider"`
			ClientID string `json:"client_id"`
			Redirect string `json:"redirect"`
		}
		_ = json.Unmarshal(params, &p)
		return a.ConnectorAuthURL(p.Provider, p.ClientID, p.Redirect)
	case "connectors.token":
		var p struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ConnectorStoreToken(p.ID, p.Token)
	case "isolation.report":
		return a.IsolationReport(), nil
	case "review.queue":
		return a.ReviewQueue(), nil
	case "phone.status":
		return a.PhoneStatus(), nil
	case "phone.approve":
		var p struct {
			ID       string `json:"id"`
			Decision string `json:"decision"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.PhoneApprove(p.ID, p.Decision)
	case "phone.steer":
		var p struct {
			Session string `json:"session"`
			Text    string `json:"text"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.PhoneSteer(p.Session, p.Text)
	case "mcp.start_http":
		var p struct {
			Name     string `json:"name"`
			Endpoint string `json:"endpoint"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.StartMCPHTTP(p.Name, p.Endpoint)
	case "expert.admit":
		var p struct {
			Dir string `json:"dir"`
		}
		_ = json.Unmarshal(params, &p)
		return a.AdmitExpert(p.Dir)
	case "expert.distill":
		var p struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Body        string `json:"body"`
		}
		_ = json.Unmarshal(params, &p)
		path, err := a.DistillExpert(p.Name, p.Description, p.Body)
		return map[string]any{"path": path}, err
	case "computer.allow":
		var p struct {
			App string `json:"app"`
		}
		_ = json.Unmarshal(params, &p)
		a.ComputerAllow(p.App)
		return map[string]any{"ok": true}, nil
	default:
		return nil, errMethod(method)
	}
}

type errMethod string

func (e errMethod) Error() string { return "method not found: " + string(e) }
