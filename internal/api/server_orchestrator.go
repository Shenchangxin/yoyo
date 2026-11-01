// server_orchestrator.go 把编排台的快照、决定和模式接到 HTTP。
// 不创建引擎。桌面 Service 走同一套 App 方法。

package api

import (
	"encoding/json"
	"net/http"

	"github.com/Shenchangxin/yoyo/internal/app"
)

// mountOrchestrator 把快照和决定接到 mux。a 为空时这些路由返回 503。
func mountOrchestrator(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /api/orchestrator", func(w http.ResponseWriter, r *http.Request) {
		if a == nil {
			writeJSONStatus(w, 503, map[string]any{"error": "app is not running"})
			return
		}
		writeJSON(w, a.OrchestratorView())
	})
	mux.HandleFunc("POST /api/orchestrator/decide", func(w http.ResponseWriter, r *http.Request) {
		if a == nil {
			writeJSONStatus(w, 503, map[string]any{"error": "app is not running"})
			return
		}
		var body struct {
			TriggerID string `json:"triggerId"`
			Kind      string `json:"kind"`
			Reason    string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		if err := a.OrchestratorDecide(body.TriggerID, body.Kind, body.Reason); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /api/orchestrator/deliver", func(w http.ResponseWriter, r *http.Request) {
		if a == nil {
			writeJSONStatus(w, 503, map[string]any{"error": "app is not running"})
			return
		}
		writeJSON(w, map[string]any{"n": a.OrchestratorDeliver()})
	})
	mux.HandleFunc("POST /api/orchestrator/act", func(w http.ResponseWriter, r *http.Request) {
		if a == nil {
			writeJSONStatus(w, 503, map[string]any{"error": "app is not running"})
			return
		}
		var body struct {
			Kind      string `json:"kind"`
			TicketID  string `json:"ticketId"`
			TriggerID string `json:"triggerId"`
			Text      string `json:"text"`
			Reason    string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		if err := a.OrchestratorAct(body.Kind, body.TicketID, body.TriggerID, body.Text, body.Reason); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /api/orchestrator/mode", func(w http.ResponseWriter, r *http.Request) {
		if a == nil {
			writeJSONStatus(w, 503, map[string]any{"error": "app is not running"})
			return
		}
		var body struct {
			Mode string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		if err := a.OrchestratorMode(body.Mode); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /api/orchestrator/paused", func(w http.ResponseWriter, r *http.Request) {
		if a == nil {
			writeJSONStatus(w, 503, map[string]any{"error": "app is not running"})
			return
		}
		var body struct {
			Paused bool `json:"paused"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		if err := a.OrchestratorPaused(body.Paused); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /api/orchestrator/admission", func(w http.ResponseWriter, r *http.Request) {
		if a == nil {
			writeJSONStatus(w, 503, map[string]any{"error": "app is not running"})
			return
		}
		var body struct {
			ListedUnaddressed   string `json:"listedUnaddressed"`
			UnlistedAddressed   string `json:"unlistedAddressed"`
			UnlistedUnaddressed string `json:"unlistedUnaddressed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		if err := a.OrchestratorAdmission(body.ListedUnaddressed, body.UnlistedAddressed, body.UnlistedUnaddressed); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
}
