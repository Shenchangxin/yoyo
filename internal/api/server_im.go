package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/im"
)

func mountIM(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /api/im", func(w http.ResponseWriter, r *http.Request) {
		if !imReady(w, a) {
			return
		}
		writeJSON(w, a.IM.Snapshot())
	})
	mux.HandleFunc("PUT /api/im/settings", func(w http.ResponseWriter, r *http.Request) {
		if !imReady(w, a) {
			return
		}
		var body im.Settings
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		if err := a.IM.SaveSettings(body); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("PUT /api/im/accounts", func(w http.ResponseWriter, r *http.Request) {
		if !imReady(w, a) {
			return
		}
		var body struct {
			Account im.Account        `json:"account"`
			Secrets map[string]string `json:"secrets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		acct, err := a.IM.SaveAccount(body.Account, body.Secrets)
		if err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, acct)
	})
	mux.HandleFunc("DELETE /api/im/accounts/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !imReady(w, a) {
			return
		}
		if err := a.IM.DeleteAccount(r.PathValue("id")); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /api/im/send", func(w http.ResponseWriter, r *http.Request) {
		if !imReady(w, a) {
			return
		}
		var body struct {
			AccountID string `json:"account_id"`
			ChatID    string `json:"chat_id"`
			Text      string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		if err := a.IM.Push(ctx, strings.TrimSpace(body.AccountID), body.ChatID, body.Text); err != nil {
			writeJSONStatus(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
}

func imReady(w http.ResponseWriter, a *app.App) bool {
	if a == nil || a.IM == nil {
		writeJSONStatus(w, 503, map[string]any{"error": "messaging is not running"})
		return false
	}
	return true
}
