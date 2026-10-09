package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/project"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

func mountPersonal(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("/api/inbox", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				ID string `json:"id"`
				Op string `json:"op"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Op == "dismiss" {
				writeJSON(w, map[string]any{"ok": a.InboxDismiss(body.ID)})
				return
			}
			a.InboxMarkRead(body.ID)
		}
		writeJSON(w, a.InboxList())
	})
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var p project.Project
			_ = json.NewDecoder(r.Body).Decode(&p)
			writeJSON(w, a.ProjectCreate(p))
			return
		}
		writeJSON(w, a.ProjectsList())
	})
	mux.HandleFunc("/api/memory", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Kind    string `json:"kind"`
				Text    string `json:"text"`
				Project string `json:"project"`
				ID      string `json:"id"`
				Op      string `json:"op"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			switch body.Op {
			case "promote":
				writeJSON(w, map[string]any{"ok": a.MemoryPromote(body.ID)})
				return
			case "forget":
				writeJSON(w, map[string]any{"ok": a.MemoryForget(body.ID)})
				return
			default:
				writeJSON(w, a.MemoryWrite(body.Kind, body.Text, body.Project))
				return
			}
		}
		writeJSON(w, a.MemoryList(r.URL.Query().Get("q"), r.URL.Query().Get("kind")))
	})
	mux.HandleFunc("/api/schedule", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var j schedule.Job
			_ = json.NewDecoder(r.Body).Decode(&j)
			writeJSON(w, a.ScheduleCreate(j))
			return
		}
		if r.Method == http.MethodDelete {
			id := r.URL.Query().Get("id")
			writeJSON(w, map[string]any{"ok": a.ScheduleCancel(id)})
			return
		}
		writeJSON(w, a.ScheduleList())
	})
	mux.HandleFunc("/api/connectors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var acct connector.Account
			_ = json.NewDecoder(r.Body).Decode(&acct)
			writeJSON(w, a.ConnectorConnect(acct))
			return
		}
		if r.Method == http.MethodDelete {
			id := r.URL.Query().Get("id")
			writeJSON(w, map[string]any{"ok": a.ConnectorDisconnect(id)})
			return
		}
		writeJSON(w, map[string]any{"accounts": a.ConnectorsList(), "catalog": a.ConnectorCatalog()})
	})
	mux.HandleFunc("/api/connectors/oauth", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Provider string `json:"provider"`
			ClientID string `json:"client_id"`
			Redirect string `json:"redirect"`
			Code     string `json:"code"`
			Secret   string `json:"secret"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Code != "" {
			acct, err := a.ConnectorComplete(body.Provider, body.Code, body.ClientID, body.Secret, body.Redirect)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, acct)
			return
		}
		m, err := a.ConnectorAuthURL(body.Provider, body.ClientID, body.Redirect)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, m)
	})
	mux.HandleFunc("/api/hooks/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/hooks/")
		if r.Method != http.MethodPost || id == "" {
			http.Error(w, "method", 405)
			return
		}
		if err := a.TriggerWebhook(id); err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "awake_required": true})
	})
	mux.HandleFunc("/api/isolation", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.IsolationReport())
	})
	mux.HandleFunc("/api/review/queue", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.ReviewQueue())
	})
	mux.HandleFunc("/api/browser/view", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.BrowserView())
	})
	mux.HandleFunc("/api/browser/takeover", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		var body struct {
			Path string `json:"path"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := a.BrowserTakeoverAt(body.Path); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/phone", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.PhoneStatus())
	})
	mux.HandleFunc("/api/phone/approve", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ID       string `json:"id"`
			Decision string `json:"decision"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := a.PhoneApprove(body.ID, body.Decision); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/phone/steer", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Session string `json:"session"`
			Text    string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := a.PhoneSteer(body.Session, body.Text); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/phone/schedule", func(w http.ResponseWriter, r *http.Request) {
		var j schedule.Job
		_ = json.NewDecoder(r.Body).Decode(&j)
		writeJSON(w, a.PhoneSchedule(j))
	})
	mux.HandleFunc("/api/mcp/http", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name     string `json:"name"`
			Endpoint string `json:"endpoint"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := a.StartMCPHTTP(body.Name, body.Endpoint); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/expert/admit", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Dir string `json:"dir"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		p, err := a.AdmitExpert(body.Dir)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, p)
	})
	mux.HandleFunc("/api/personal", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, a.PersonalSnapshot())
			return
		}
		var body struct {
			Op          string   `json:"op"`
			ID          string   `json:"id"`
			Text        string   `json:"text"`
			Hash        string   `json:"hash"`
			Approve     bool     `json:"approve"`
			Action      string   `json:"action"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Milestones  []string `json:"milestones"`
			URL         string   `json:"url"`
			Condition   string   `json:"condition"`
			Value       string   `json:"value"`
			Interval    int      `json:"interval_minutes"`
			Option      string   `json:"option"`
			Status      string   `json:"status"`
			GoalID      string   `json:"goal_id"`
			MilestoneID string   `json:"milestone_id"`
			Done        bool     `json:"done"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch body.Op {
		case "answer":
			t, err := a.PersonalAnswer(body.ID, body.Text)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, t)
		case "decide":
			p, err := a.PersonalDecide(body.ID, body.Hash, body.Approve)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, p)
		case "idea":
			it, err := a.PersonalIdea(body.ID, body.Action)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, it)
		case "goal":
			g, err := a.PersonalGoal(body.Title, body.Description, body.Milestones)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, g)
		case "watch":
			m, err := a.PersonalWatch(body.Title, body.URL, body.Condition, body.Value, body.Interval)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, m)
		case "cancel":
			t, err := a.PersonalCancel(body.ID)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, t)
		case "choice":
			c, err := a.PersonalChoice(body.ID, body.Option)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, c)
		case "ideas":
			if err := a.PersonalRefreshIdeas(); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, map[string]any{"ok": true})
		case "pause":
			t, err := a.PersonalPause(body.ID)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, t)
		case "resume":
			t, err := a.PersonalResume(body.ID)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, t)
		case "retry":
			t, err := a.PersonalRetry(body.ID)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, t)
		case "goal_status":
			g, err := a.PersonalGoalStatus(body.ID, body.Status)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, g)
		case "milestone":
			g, err := a.PersonalMilestone(body.GoalID, body.MilestoneID, body.Done)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, g)
		case "monitor_status":
			m, err := a.PersonalMonitorStatus(body.ID, body.Status)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			writeJSON(w, m)
		default:
			writeJSON(w, a.PersonalSnapshot())
		}
	})
	mux.HandleFunc("/api/expert/distill", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Body        string `json:"body"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		path, err := a.DistillExpert(body.Name, body.Description, body.Body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]any{"path": path})
	})
}
