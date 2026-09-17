package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/runtime"
)

const spillAPIBytes = 80_000

type SessionTrace struct {
	SessionID string               `json:"session_id"`
	Title     string               `json:"title,omitempty"`
	Workspace string               `json:"workspace,omitempty"`
	Harness   string               `json:"harness,omitempty"`
	Model     string               `json:"model,omitempty"`
	CreatedAt time.Time            `json:"created_at,omitempty"`
	StartedAt time.Time            `json:"started_at,omitempty"`
	EndedAt   time.Time            `json:"ended_at,omitempty"`
	Stats     runtime.TraceStats   `json:"stats"`
	Events    []runtime.TraceEvent `json:"events"`
	Artifacts []TraceArtifact      `json:"artifacts"`
}

type TraceArtifact struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Label   string `json:"label"`
	Bytes   int    `json:"bytes,omitempty"`
	Preview string `json:"preview,omitempty"`
}

type SpillBlob struct {
	ID        string `json:"id"`
	Bytes     int    `json:"bytes"`
	Text      string `json:"text"`
	Truncated bool   `json:"truncated,omitempty"`
}

func (a *App) SessionTrace(id string) (SessionTrace, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return SessionTrace{}, fmt.Errorf("empty session id")
	}
	meta, _ := a.GetSession(id)
	evs, err := a.Trajectory(id)
	if err != nil {
		return SessionTrace{}, err
	}
	view := runtime.ProjectTrace(evs)
	if view.Events == nil {
		view.Events = []runtime.TraceEvent{}
	}
	if view.Stats.Tokens == 0 {
		if sh := a.ContextUsage(id); sh.Tokens > 0 {
			view.Stats.Tokens = sh.Tokens
		}
	}
	spill := runtime.BindSpill(a.Home.Root, meta.Workspace, id)
	arts := a.collectTraceArtifacts(id, meta, spill)
	if arts == nil {
		arts = []TraceArtifact{}
	}
	var spillBytes int
	for _, art := range arts {
		if art.Kind == "spill" {
			spillBytes += art.Bytes
		}
	}
	view.Stats.SpillBytes = spillBytes
	model := meta.Model
	if model == "" {
		model = a.Config.Model
	}
	return SessionTrace{
		SessionID: id,
		Title:     meta.Title,
		Workspace: meta.Workspace,
		Harness:   meta.Harness,
		Model:     model,
		CreatedAt: meta.CreatedAt,
		StartedAt: view.StartedAt,
		EndedAt:   view.EndedAt,
		Stats:     view.Stats,
		Events:    view.Events,
		Artifacts: arts,
	}, nil
}

func (a *App) SpillBlob(sessionID, blobID string) (SpillBlob, error) {
	sessionID = strings.TrimSpace(sessionID)
	blobID = strings.TrimSpace(blobID)
	if sessionID == "" || blobID == "" {
		return SpillBlob{}, fmt.Errorf("empty spill id")
	}
	meta, _ := a.GetSession(sessionID)
	spill := runtime.BindSpill(a.Home.Root, meta.Workspace, sessionID)
	if spill == nil {
		return SpillBlob{}, os.ErrNotExist
	}
	text, err := spill.Get(blobID)
	if err != nil {
		return SpillBlob{}, err
	}
	n := len(text)
	trunc := false
	if n > spillAPIBytes {
		text = text[:spillAPIBytes]
		trunc = true
	}
	return SpillBlob{ID: blobID, Bytes: n, Text: text, Truncated: trunc}, nil
}

func (a *App) collectTraceArtifacts(id string, meta SessionMeta, spill *runtime.Spill) []TraceArtifact {
	var out []TraceArtifact
	if art, ok := fileArtifact("trajectory", id+".jsonl", filepath.Join(a.Home.Sessions(), id+".jsonl")); ok {
		out = append(out, art)
	}
	if art, ok := fileArtifact("meta", id+".meta.json", filepath.Join(a.Home.Sessions(), id+".meta.json")); ok {
		out = append(out, art)
	}
	if meta.Harness != "" {
		out = append(out, TraceArtifact{Kind: "harness", ID: meta.Harness, Label: "harness " + shortHash(meta.Harness)})
	}
	if spill != nil {
		if notes := runtime.ReadNotes(spill); notes != "" {
			out = append(out, TraceArtifact{
				Kind:    "notes",
				ID:      "notes",
				Label:   "session notes",
				Bytes:   len(notes),
				Preview: capPreview(notes),
			})
		}
		ids := spill.ListIDs()
		sort.Strings(ids)
		for _, sid := range ids {
			if sid == "notes" {
				continue
			}
			out = append(out, TraceArtifact{
				Kind:  "spill",
				ID:    sid,
				Label: sid,
				Bytes: spill.Size(sid),
			})
		}
	}
	if dir := runtime.WorkspaceContextDir(meta.Workspace, id); dir != "" {
		if art, ok := fileArtifact("index", "INDEX.md", filepath.Join(dir, "INDEX.md")); ok {
			out = append(out, art)
		}
	}
	if meta.Workspace != "" {
		over := filepath.Join(meta.Workspace, ".yoyo", "overflow")
		ents, err := os.ReadDir(over)
		if err == nil {
			for _, e := range ents {
				if e.IsDir() {
					continue
				}
				if art, ok := fileArtifact("overflow", e.Name(), filepath.Join(over, e.Name())); ok {
					out = append(out, art)
				}
			}
		}
	}
	if a.Gate != nil {
		for _, o := range a.Gate.Pending() {
			if o.Request.SessionID != id {
				continue
			}
			label := strings.TrimSpace(o.Request.Action + " " + o.Request.Command)
			out = append(out, TraceArtifact{
				Kind:    "approval",
				ID:      o.ID,
				Label:   firstLabel(label, "pending approval"),
				Preview: strings.TrimSpace(o.Request.Path),
			})
		}
	}
	return out
}

func fileArtifact(kind, label, path string) (TraceArtifact, bool) {
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return TraceArtifact{}, false
	}
	art := TraceArtifact{Kind: kind, ID: label, Label: label, Bytes: int(fi.Size())}
	if fi.Size() > 0 && fi.Size() <= 8_000 && (strings.HasSuffix(label, ".md") || strings.HasSuffix(label, ".json") || strings.HasSuffix(label, ".txt")) {
		if b, err := os.ReadFile(path); err == nil {
			art.Preview = capPreview(string(b))
		}
	}
	return art, true
}

func capPreview(s string) string {
	const n = 4_000
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

func shortHash(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:12]
}

func firstLabel(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
