package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/pages"
	"github.com/Shenchangxin/yoyo/internal/profile"
)

func mountSurfaces(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("/api/pause", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Paused bool `json:"paused"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if err := a.SetPaused(body.Paused); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
		}
		writeJSON(w, map[string]any{"paused": a.IsPaused()})
	})
	mux.HandleFunc("/api/pages", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var p struct {
				pages.Page
				Expected int `json:"expected_revision"`
			}
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			out, err := a.PagesSave(p.Page, p.Expected)
			if err != nil {
				var conflict *pages.ConflictError
				if errors.As(err, &conflict) {
					writeJSONStatus(w, 409, map[string]any{"error": err.Error(), "page": conflict.Page})
					return
				}
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, out)
			return
		default:
			writeJSON(w, a.PagesList(r.URL.Query().Get("q"), r.URL.Query().Get("space_id")))
		}
	})
	mux.HandleFunc("/api/pages/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/pages/")
		if id == "reviews" {
			if r.Method == http.MethodPost {
				var body struct {
					ID      string `json:"id"`
					Hash    string `json:"hash"`
					Approve bool   `json:"approve"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				pg, rev, err := a.PagesDecide(body.ID, body.Hash, body.Approve)
				if err != nil {
					http.Error(w, err.Error(), 400)
					return
				}
				writeJSON(w, map[string]any{"page": pg, "review": rev})
				return
			}
			writeJSON(w, a.PagesReviews())
			return
		}
		pg, err := a.PagesGet(r.URL.Query().Get("space_id"), id)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		writeJSON(w, pg)
	})
	mux.HandleFunc("/api/pages/propose", func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			Session  string `json:"session"`
			Title    string `json:"title"`
			Content  string `json:"content"`
			ID       string `json:"id"`
			Space    string `json:"space_id"`
			Parent   string `json:"parent_id"`
			Expected int    `json:"expected_revision"`
		}
		_ = json.NewDecoder(r.Body).Decode(&p)
		out, err := a.PagesPropose(p.Session, p.Title, p.Content, p.ID, p.Space, p.Parent, p.Expected)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, out)
	})
	mux.HandleFunc("/api/profiles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var p profile.Profile
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			out, err := a.ProfilesSave(p)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, out)
			return
		}
		writeJSON(w, a.ProfilesList())
	})
	mux.HandleFunc("/api/voice/transcribe", func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			Filename string `json:"filename"`
			Audio    string `json:"audio"`
		}
		_ = json.NewDecoder(r.Body).Decode(&p)
		raw, err := decodeSkinB64(p.Audio)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		text, err := a.VoiceTranscribe(p.Filename, raw)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"text": text})
	})
	mux.HandleFunc("/api/voice/speak", func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			Text string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&p)
		b, ctype, err := a.VoiceSpeak(p.Text)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"audio": encodeSkinB64(b), "content_type": ctype})
	})
	mux.HandleFunc("/api/voice/receipt", func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			Session  string `json:"session"`
			Duration int    `json:"duration_sec"`
			Text     string `json:"transcript"`
		}
		_ = json.NewDecoder(r.Body).Decode(&p)
		a.VoiceReceipt(p.Session, p.Duration, p.Text)
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/connectors/mention", func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			Account string `json:"account"`
			Text    string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&p)
		if err := a.ConnectorMention(p.Account, p.Text); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/schedule/retry", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if r.Method == http.MethodPost {
			var body struct {
				ID string `json:"id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.ID != "" {
				id = body.ID
			}
		}
		j, err := a.ScheduleRetry(id)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, j)
	})
}
