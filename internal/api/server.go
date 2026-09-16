package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func Handler(a *app.App, static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", handleRPC(a))
	mux.HandleFunc("/api/rpc", handleRPC(a))
	mux.HandleFunc("/ws", handleWS(a))
	mux.HandleFunc("/api/ws", handleWS(a))
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.Health())
	})
	mux.HandleFunc("/api/running", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ids": a.RunningIDs()})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&a.Config)
			_ = a.SaveConfig()
		}
		writeJSON(w, a.Config)
	})
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Workspace string `json:"workspace"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			m, err := a.NewSession(body.Workspace)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			writeJSON(w, m)
			return
		}
		list, err := a.ListSessions()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
		parts := strings.Split(rest, "/")
		id := parts[0]
		if len(parts) > 1 && parts[1] == "events" {
			serveSSE(w, r, a, id)
			return
		}
		if len(parts) > 1 && parts[1] == "context" {
			writeJSON(w, a.ContextUsage(id))
			return
		}
		if len(parts) > 1 && parts[1] == "running" {
			writeJSON(w, map[string]any{"running": a.Running(id)})
			return
		}
		if len(parts) > 1 && parts[1] == "interrupt" && r.Method == http.MethodPost {
			_ = a.Interrupt(id)
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		if len(parts) > 1 && parts[1] == "fork" && r.Method == http.MethodPost {
			m, err := a.ForkSession(id)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, m)
			return
		}
		if len(parts) > 1 && parts[1] == "title" && r.Method == http.MethodPost {
			var body struct {
				Title string `json:"title"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if err := a.RenameSession(id, body.Title); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		if len(parts) > 1 && parts[1] == "trajectory" {
			evs, err := a.Trajectory(id)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			writeJSON(w, evs)
			return
		}
		if len(parts) > 1 && parts[1] == "messages" && r.Method == http.MethodPost {
			var body struct {
				Text  string `json:"text"`
				Async bool   `json:"async"`
				Plan  bool   `json:"plan"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Async {
				if err := a.StartSend(id, body.Text, body.Plan); err != nil {
					http.Error(w, err.Error(), 409)
					return
				}
				writeJSON(w, map[string]any{"ok": true, "async": true})
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
			defer cancel()
			out, err := a.SendOpts(ctx, id, body.Text, nil, a.Hub.Publish, body.Plan)
			if err != nil {
				writeAppErr(w, err)
				return
			}
			writeJSON(w, map[string]any{"text": out})
			return
		}
		m, err := a.GetSession(id)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		writeJSON(w, m)
	})
	mux.HandleFunc("/api/approvals", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				ID       string `json:"id"`
				Decision string `json:"decision"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if err := a.ResolveApproval(body.ID, body.Decision); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		writeJSON(w, a.Gate.Pending())
	})
	mux.HandleFunc("/api/mcp/start", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name    string   `json:"name"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := a.MCP.Start(body.Name, body.Command, body.Args); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "tools": a.MCP.Tools()})
	})
	mux.HandleFunc("/api/harness", func(w http.ResponseWriter, r *http.Request) {
		refs, _ := a.ListHarnesses()
		snap, _ := a.LoadSnapshot(a.ActiveHash())
		writeJSON(w, map[string]any{"active": a.ActiveHash(), "refs": refs, "snapshot": snap})
	})
	mux.HandleFunc("/api/harness/checkout", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Hash string `json:"hash"`
			L3   bool   `json:"l3"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := a.CheckoutOpts(body.Hash, app.CheckoutOpts{ConfirmL3: body.L3}); err != nil {
			writeAppErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"active": a.ActiveHash()})
	})
	mux.HandleFunc("/api/harness/rollback", func(w http.ResponseWriter, r *http.Request) {
		if err := a.Rollback(); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"active": a.ActiveHash()})
	})
	mux.HandleFunc("/api/harness/diff", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		d, err := a.DiffDetail(q.Get("a"), q.Get("b"))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, d)
	})
	mux.HandleFunc("/api/playbook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				ID      string `json:"id"`
				Helpful bool   `json:"helpful"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			pb, err := a.RatePlaybook(body.ID, body.Helpful)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, pb)
			return
		}
		pb, err := a.Playbook()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, pb)
	})
	mux.HandleFunc("/api/workspace/diff", func(w http.ResponseWriter, r *http.Request) {
		ws := r.URL.Query().Get("workspace")
		d, err := a.WorkspaceDiff(ws)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"diff": d})
	})
	mux.HandleFunc("/api/workspace/hunks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Workspace string   `json:"workspace"`
				IDs       []string `json:"ids"`
				Snapshot  string   `json:"snapshot"`
				Reverse   bool     `json:"reverse"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			var err error
			if body.Reverse {
				err = a.ReverseWorkspaceHunks(body.Workspace, body.IDs, body.Snapshot)
			} else {
				err = a.ApplyWorkspaceHunks(body.Workspace, body.IDs)
			}
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		ws := r.URL.Query().Get("workspace")
		h, err := a.WorkspaceHunks(ws)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, h)
	})
	mux.HandleFunc("/api/eval/safety", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()
		rep, err := a.RunEvalOpts(ctx, nil, true)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, rep)
	})
	mux.HandleFunc("/api/eval", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()
		rep, err := a.RunEval(ctx, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, rep)
	})
	mux.HandleFunc("/api/eval/best", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			N      int      `json:"n"`
			Models []string `json:"models"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Minute)
		defer cancel()
		if len(body.Models) > 0 {
			rep, err := a.BestOfModels(ctx, body.Models, nil)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			writeJSON(w, rep)
			return
		}
		rep, err := a.BestOfN(ctx, body.N, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, rep)
	})
	mux.HandleFunc("/api/eval/tb", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()
		rep, err := a.RunEvalTB(ctx, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, rep)
	})
	mux.HandleFunc("/api/evolve", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			K int `json:"k"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
		defer cancel()
		res, err := a.EvolveK(ctx, nil, nil, body.K)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, res)
	})
	mux.HandleFunc("/api/archive", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.Archive.List())
	})
	mux.HandleFunc("/api/plugins", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"fibers": a.Kernel.Fibers(),
			"wasm":   a.WASM.List(),
			"mcp":    a.MCP.List(),
			"tools":  a.MCP.Tools(),
		})
	})
	mux.HandleFunc("/api/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Exe string `json:"exe"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if err := a.ApplyStagedUpdate(body.Exe); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		p := a.StagingPath()
		st, err := os.Stat(p)
		size := int64(0)
		if err == nil {
			size = st.Size()
		}
		writeJSON(w, map[string]any{"staging": p, "exists": err == nil, "size": size})
	})
	mux.HandleFunc("/api/key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST", 405)
			return
		}
		b, _ := io.ReadAll(r.Body)
		var body struct {
			Value string `json:"value"`
		}
		_ = json.Unmarshal(b, &body)
		a.Vault.Set("default", body.Value)
		writeJSON(w, map[string]any{"ok": true})
	})
	if static != nil {
		mux.Handle("/", static)
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("yoyo api\n"))
		})
	}
	return mux
}

func serveSSE(w http.ResponseWriter, r *http.Request, a *app.App, sessionID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "sse unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, unsub := a.Hub.Subscribe(sessionID)
	defer unsub()
	if evs, err := a.Trajectory(sessionID); err == nil {
		for _, ev := range evs {
			writeSSE(w, ev)
		}
		flusher.Flush()
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(w, ev)
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, ev trace.Event) {
	b, _ := json.Marshal(ev)
	fmt.Fprintf(w, "data: %s\n\n", b)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAppErr(w http.ResponseWriter, err error) {
	if e, ok := app.AsL3(err); ok {
		writeJSONStatus(w, 403, map[string]any{"error": e.Error(), "l3": true, "surfaces": e.Surfaces, "from": e.From, "to": e.To})
		return
	}
	var b runtime.ErrBudget
	if errors.As(err, &b) {
		writeJSONStatus(w, 429, map[string]any{"error": b.Error(), "budget": true, "used": b.Used, "cap": b.Cap})
		return
	}
	http.Error(w, err.Error(), 500)
}
