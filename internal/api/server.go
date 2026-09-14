package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/app"
)

func Handler(a *app.App, static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "harness": a.ActiveHash(), "model": a.Config.Model})
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
				Text string `json:"text"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
			defer cancel()
			out, err := a.Send(ctx, id, body.Text, nil, nil)
			if err != nil {
				http.Error(w, err.Error(), 500)
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
	mux.HandleFunc("/api/harness", func(w http.ResponseWriter, r *http.Request) {
		refs, _ := a.ListHarnesses()
		snap, _ := a.LoadSnapshot(a.ActiveHash())
		writeJSON(w, map[string]any{"active": a.ActiveHash(), "refs": refs, "snapshot": snap})
	})
	mux.HandleFunc("/api/harness/checkout", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Hash string `json:"hash"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := a.Checkout(body.Hash); err != nil {
			http.Error(w, err.Error(), 400)
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
		d, err := a.Diff(q.Get("a"), q.Get("b"))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"diff": d})
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
	mux.HandleFunc("/api/evolve", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
		defer cancel()
		res, err := a.EvolveOnce(ctx, nil, nil)
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
		})
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

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
