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
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func Handler(a *app.App, static http.Handler) http.Handler {
	mux := http.NewServeMux()
	mountPersonal(mux, a)
	mountVideo(mux, a)
	mux.HandleFunc("/rpc", handleRPC(a))
	mux.HandleFunc("/api/rpc", handleRPC(a))
	mux.HandleFunc("/ws", handleWS(a))
	mux.HandleFunc("/api/ws", handleWS(a))
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		restRPC(a, w, r, "health", nil)
	})
	mux.HandleFunc("/api/running", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ids": a.RunningIDs(), "runs": a.RunningStatus()})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&a.Config)
			_ = a.SaveConfig()
		}
		writeJSON(w, a.Config)
	})
	mux.HandleFunc("/api/provider/test", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.TestProvider())
	})
	mux.HandleFunc("/api/mcp/replace", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Servers []app.MCPServerConfig `json:"servers"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := a.ReplaceMCP(body.Servers); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Workspace string `json:"workspace"`
				Channel   string `json:"channel"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			m, err := a.NewSessionOn(body.Workspace, body.Channel)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			writeJSON(w, m)
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q != "" {
			list, err := a.SearchSessions(q, r.URL.Query().Get("archived") == "1")
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			writeJSON(w, list)
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
		if len(parts) > 1 && parts[1] == "retry" && r.Method == http.MethodPost {
			if err := a.RetrySession(id); err != nil {
				http.Error(w, err.Error(), 409)
				return
			}
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
		if resolved, err := a.ResolveSessionID(id); err == nil {
			id = resolved
		}
		if len(parts) > 1 && parts[1] == "dump" {
			dump, err := a.DumpSession(id)
			if err != nil {
				http.Error(w, err.Error(), 404)
				return
			}
			writeJSON(w, dump)
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
		if len(parts) > 1 && parts[1] == "trace" {
			tr, err := a.SessionTrace(id)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			writeJSON(w, tr)
			return
		}
		if len(parts) > 2 && parts[1] == "spill" {
			blob, err := a.SpillBlob(id, parts[2])
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					http.Error(w, "not found", 404)
					return
				}
				http.Error(w, err.Error(), 500)
				return
			}
			writeJSON(w, blob)
			return
		}
		if len(parts) > 1 && parts[1] == "messages" && r.Method == http.MethodPost {
			var body struct {
				Text        string           `json:"text"`
				Async       bool             `json:"async"`
				Plan        bool             `json:"plan"`
				Attachments []app.Attachment `json:"attachments"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Async {
				if err := a.StartSendOpts(id, body.Text, body.Plan, body.Attachments); err != nil {
					if errors.Is(err, app.ErrQueued) {
						writeJSON(w, map[string]any{"ok": true, "queued": true})
						return
					}
					http.Error(w, err.Error(), 409)
					return
				}
				writeJSON(w, map[string]any{"ok": true, "async": true})
				return
			}
			out, err := a.SendOpts(r.Context(), id, body.Text, nil, a.Hub.Publish, body.Plan)
			if err != nil {
				writeAppErr(w, err)
				return
			}
			writeJSON(w, map[string]any{"text": out})
			return
		}
		if r.Method == http.MethodDelete {
			if err := a.DeleteSession(id); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		if len(parts) > 1 && parts[1] == "archive" && r.Method == http.MethodPost {
			var body struct {
				Archived *bool `json:"archived"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			archived := true
			if body.Archived != nil {
				archived = *body.Archived
			}
			m, err := a.ArchiveSession(id, archived)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, m)
			return
		}
		if len(parts) > 1 && parts[1] == "pin" && r.Method == http.MethodPost {
			var body struct {
				Pinned bool `json:"pinned"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			m, err := a.PinSession(id, body.Pinned)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, m)
			return
		}
		if len(parts) > 1 && parts[1] == "export" {
			text, err := a.ExportSession(id)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, map[string]any{"markdown": text})
			return
		}
		if len(parts) > 1 && parts[1] == "steer" && r.Method == http.MethodPost {
			var body struct {
				Text string `json:"text"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if err := a.Steer(id, body.Text); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		if len(parts) > 1 && parts[1] == "queue" {
			writeJSON(w, a.QueueList(id))
			return
		}
		if len(parts) > 1 && parts[1] == "compact" && r.Method == http.MethodPost {
			var body struct {
				Focus string `json:"focus"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			note, err := a.CompactSessionFocus(id, body.Focus)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, map[string]any{"note": note})
			return
		}
		if len(parts) > 1 && parts[1] == "model" && r.Method == http.MethodPost {
			var body struct {
				Model string `json:"model"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			m, err := a.SetSessionModel(id, body.Model)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, m)
			return
		}
		if len(parts) > 1 && parts[1] == "auth" && r.Method == http.MethodPost {
			var body struct {
				Mode string `json:"mode"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			m, err := a.SetSessionAuthMode(id, body.Mode)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, m)
			return
		}
		if len(parts) > 1 && parts[1] == "workspace" && r.Method == http.MethodPost {
			var body struct {
				Workspace string `json:"workspace"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			m, err := a.SetSessionWorkspace(id, body.Workspace)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, m)
			return
		}
		if len(parts) > 1 && parts[1] == "isolate" && r.Method == http.MethodPost {
			var body struct {
				Isolate bool `json:"isolate"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			m, err := a.SetSessionIsolate(id, body.Isolate)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, m)
			return
		}
		if len(parts) > 1 && parts[1] == "skills" && r.Method == http.MethodPost {
			var body struct {
				Names []string `json:"names"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			m, err := a.SetSessionPinnedSkills(id, body.Names)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, m)
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
		if err := a.StartMCP(body.Name, body.Command, body.Args); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "tools": a.MCP.Tools()})
	})
	mux.HandleFunc("/api/mcp/stop", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := a.StopMCP(body.Name); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/fs/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		ws := q.Get("workspace")
		if ws == "" {
			ws = a.Workspace()
		}
		writeJSON(w, runtime.FuzzySearch(ws, q.Get("q"), 40))
	})
	mux.HandleFunc("/api/skills", func(w http.ResponseWriter, r *http.Request) {
		ws := r.URL.Query().Get("workspace")
		if name := strings.TrimSpace(r.URL.Query().Get("name")); name != "" {
			sk := a.GetSkill(ws, name)
			if sk == nil {
				http.Error(w, "unknown skill", 404)
				return
			}
			writeJSON(w, sk)
			return
		}
		writeJSON(w, a.ListSkills(ws))
	})
	mux.HandleFunc("/api/skills/market", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Slug string `json:"slug"`
				Op   string `json:"op"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Op == "uninstall" {
				if err := a.UninstallMarketSkill(body.Slug); err != nil {
					http.Error(w, err.Error(), 400)
					return
				}
				writeJSON(w, map[string]any{"ok": true})
				return
			}
			out, err := a.InstallMarketSkill(body.Slug)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, out)
			return
		}
		c, err := a.SkillMarket(r.URL.Query().Get("refresh") == "1")
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		writeJSON(w, c)
	})
	mux.HandleFunc("/api/open", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		var err error
		switch body.Kind {
		case "editor":
			err = a.OpenInEditor(body.Path)
		case "terminal":
			err = a.OpenWorkspaceTerminal(body.Path)
		default:
			err = a.OpenPath(body.Path)
		}
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/about", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.Health())
	})
	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Events []diaglog.FrontendEvent `json:"events"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if a.Log != nil {
				_ = a.Log.AppendFrontend(body.Events)
			}
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		writeJSON(w, a.Logs(80))
	})
	mux.HandleFunc("/api/journal", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.JournalTail(80))
	})
	mux.HandleFunc("/api/logs/export", func(w http.ResponseWriter, r *http.Request) {
		path, err := a.ExportDiagnostics("")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, map[string]any{"path": path})
	})
	mux.HandleFunc("/api/logs/diagnose", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Session string `json:"session"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		note, err := a.DiagnoseSession(body.Session, 80)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"text": note})
	})
	mux.HandleFunc("/api/doctor", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.Doctor())
	})
	mux.HandleFunc("/api/key/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.Vault.Status())
	})
	mux.HandleFunc("/api/harness", func(w http.ResponseWriter, r *http.Request) {
		st, err := a.HarnessState()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, st)
	})
	mux.HandleFunc("/api/harness/lineage", func(w http.ResponseWriter, r *http.Request) {
		lin, err := a.HarnessLineage()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, lin)
	})
	mux.HandleFunc("/api/harness/reveal", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Hash string `json:"hash"`
		}
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&body)
		} else {
			body.Hash = r.URL.Query().Get("hash")
		}
		if err := a.RevealHarness(body.Hash); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
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
	mux.HandleFunc("/api/workspace/file", func(w http.ResponseWriter, r *http.Request) {
		ws := r.URL.Query().Get("workspace")
		path := r.URL.Query().Get("path")
		prev, err := a.PreviewWorkspaceFile(ws, path)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, prev)
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
	mux.HandleFunc("/api/eval/sealed", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Minute)
		defer cancel()
		rep, err := a.RunEvalSealed(ctx, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, rep)
	})
	mux.HandleFunc("/api/eval/transfer", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Minute)
		defer cancel()
		rep, err := a.RunEvalTransfer(ctx, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, rep)
	})
	mux.HandleFunc("/api/eval/index", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Minute)
		defer cancel()
		rep, err := a.RunEvalIndex(ctx, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, rep)
	})
	mux.HandleFunc("/api/eval/behavior", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()
		rep, err := a.RunEvalBehavior(ctx, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, rep)
	})
	mux.HandleFunc("/api/eval/last", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.LastEval())
	})
	mux.HandleFunc("/api/evolve/last", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.LastEvolve())
	})
	mux.HandleFunc("/api/eval/baselines", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			N      int     `json:"n"`
			Sealed bool    `json:"sealed"`
			MaxUSD float64 `json:"max_usd"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Minute)
		defer cancel()
		rep, err := a.SearchBaselines(ctx, nil, body.N, app.EvolveRun{Sealed: body.Sealed, MaxUSD: body.MaxUSD, Baselines: body.N})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, rep)
	})
	mux.HandleFunc("/api/evolve", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			K         int     `json:"k"`
			Rounds    int     `json:"rounds"`
			Promote   bool    `json:"promote"`
			Sealed    bool    `json:"sealed"`
			Behavior  bool    `json:"behavior"`
			Index     bool    `json:"index"`
			Baselines int     `json:"baselines"`
			MaxUSD    float64 `json:"max_usd"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		timeout := 15 * time.Minute
		if body.Rounds > 1 {
			timeout = time.Duration(body.Rounds) * 15 * time.Minute
			if timeout > 2*time.Hour {
				timeout = 2 * time.Hour
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		res, err := a.EvolveWith(ctx, nil, nil, app.EvolveRun{
			K: body.K, Rounds: body.Rounds, PromoteActive: body.Promote, Sealed: body.Sealed,
			Behavior: body.Behavior, IndexTransfer: body.Index, Baselines: body.Baselines, MaxUSD: body.MaxUSD,
		})
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
			"mcp":    a.MCP.Info(),
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
		writeJSON(w, a.CheckUpdate())
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

func restRPC(a *app.App, w http.ResponseWriter, r *http.Request, method string, params any) {
	raw := []byte("{}")
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		raw = b
	}
	res := Dispatch(r.Context(), a, RPCRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: raw})
	if res.Error != nil {
		code := 500
		if res.Error.Code == 403 {
			code = 403
		}
		if res.Error.Code == 429 {
			code = 429
		}
		http.Error(w, res.Error.Message, code)
		return
	}
	writeJSON(w, res.Result)
}
