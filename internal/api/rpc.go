package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/runtime"
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
						"type":    string(ev.Type),
						"session": ev.SessionID,
						"source":  ev.Source,
						"payload": ev.Payload,
					},
				})
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
			Session string `json:"session"`
			Text    string `json:"text"`
			Plan    bool   `json:"plan"`
			Wait    bool   `json:"wait"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if !p.Wait {
			if err := a.StartSend(p.Session, p.Text, p.Plan); err != nil {
				return nil, err
			}
			return map[string]any{"ok": true, "async": true}, nil
		}
		tctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		out, err := a.SendOpts(tctx, p.Session, p.Text, nil, a.Hub.Publish, p.Plan)
		if err != nil {
			return nil, err
		}
		return map[string]any{"text": out}, nil
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
		return a.Trajectory(p.Session)
	case "approvals.list":
		return a.Gate.Pending(), nil
	case "approvals.resolve":
		var p struct {
			ID       string `json:"id"`
			Decision string `json:"decision"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ResolveApproval(p.ID, p.Decision)
	case "harness.get":
		refs, err := a.ListHarnesses()
		if err != nil {
			return nil, err
		}
		snap, _ := a.LoadSnapshot(a.ActiveHash())
		return map[string]any{"active": a.ActiveHash(), "refs": refs, "snapshot": snap}, nil
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
	case "evolve.run":
		var p struct {
			K int `json:"k"`
		}
		_ = json.Unmarshal(params, &p)
		tctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()
		return a.EvolveK(tctx, nil, nil, p.K)
	case "turn.running":
		var p struct {
			Session string `json:"session"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"running": a.Running(p.Session)}, nil
	case "turn.running_ids":
		return a.RunningIDs(), nil
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
		return map[string]any{"ok": true, "tools": a.MCP.Tools()}, a.MCP.Start(p.Name, p.Command, p.Args)
	case "archive.list":
		return a.Archive.List(), nil
	case "plugins.list":
		return map[string]any{"fibers": a.Kernel.Fibers(), "wasm": a.WASM.List(), "mcp": a.MCP.List(), "tools": a.MCP.Tools()}, nil
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
	case "workspace.apply_hunks":
		var p struct {
			Workspace string   `json:"workspace"`
			IDs       []string `json:"ids"`
		}
		_ = json.Unmarshal(params, &p)
		return map[string]any{"ok": true}, a.ApplyWorkspaceHunks(p.Workspace, p.IDs)
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
	default:
		return nil, errMethod(method)
	}
}

type errMethod string

func (e errMethod) Error() string { return "method not found: " + string(e) }
