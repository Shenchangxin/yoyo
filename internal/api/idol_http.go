package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Shenchangxin/yoyo/internal/app"
)

func mountIdol(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("/api/idol", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(io.LimitReader(r.Body, 12<<20))
		var p struct {
			Op      string          `json:"op"`
			Payload json.RawMessage `json:"payload"`
		}
		_ = json.Unmarshal(body, &p)
		payload := string(p.Payload)
		if payload == "" || payload == "null" {
			payload = string(body)
		}
		raw, err := a.IdolCall(p.Op, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	})
}
