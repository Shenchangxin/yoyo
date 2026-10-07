package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/runtime"
)

type ContextObject struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	Tokens int    `json:"tokens,omitempty"`
	Path   string `json:"path,omitempty"`
	Stale  bool   `json:"stale,omitempty"`
	Pinned bool   `json:"pinned,omitempty"`
}

type ContextInventory struct {
	Session  string              `json:"session"`
	Shape    runtime.ShapeReport `json:"shape"`
	Objects  []ContextObject     `json:"objects"`
	Plan     string              `json:"plan,omitempty"`
	PlanPath string              `json:"plan_path,omitempty"`
}

func (a *App) ContextInventory(sessionID string) ContextInventory {
	meta, _ := a.GetSession(sessionID)
	rep := a.measureSessionContext(sessionID)
	ws := meta.ToolRoot()
	if ws == "" {
		ws = meta.Workspace
	}
	inv := ContextInventory{
		Session:  sessionID,
		Shape:    rep,
		Plan:     sessionPlanText(meta, ws),
		PlanPath: runtime.PlanFilePath(ws, sessionID),
	}
	add := func(o ContextObject) {
		if strings.TrimSpace(o.Title) == "" && strings.TrimSpace(o.Path) == "" {
			return
		}
		if o.ID == "" {
			o.ID = o.Kind + ":" + o.Title
		}
		inv.Objects = append(inv.Objects, o)
	}
	if inv.Plan != "" {
		add(ContextObject{ID: "plan", Kind: "plan", Title: "Plan", Detail: capLine(inv.Plan, 80), Tokens: len([]rune(inv.Plan)) / 4, Path: inv.PlanPath, Pinned: true})
	}
	for _, name := range meta.PinnedSkills {
		add(ContextObject{ID: "skill:" + name, Kind: "skill", Title: name, Pinned: true})
	}
	for _, name := range meta.LoadedSkills {
		pinned := false
		for _, p := range meta.PinnedSkills {
			if p == name {
				pinned = true
				break
			}
		}
		if pinned {
			continue
		}
		add(ContextObject{ID: "skill:" + name, Kind: "skill", Title: name, Detail: "loaded this turn"})
	}
	for _, p := range meta.PinnedPaths {
		add(ContextObject{ID: "file:" + p, Kind: "file", Title: filepath.Base(p), Path: p, Pinned: true})
	}
	spill := runtime.BindSpill(a.Home.Root, ws, sessionID)
	if notes := runtime.ReadNotes(spill); notes != "" {
		parsed := runtime.ParseNotes(notes)
		if parsed.Objective != "" {
			add(ContextObject{ID: "objective", Kind: "pin", Title: "Objective", Detail: capLine(parsed.Objective, 120), Pinned: true})
		}
		for _, f := range parsed.Files {
			stale := false
			if ws != "" {
				if _, err := os.Stat(filepath.Join(ws, f)); err != nil {
					stale = true
				}
			}
			add(ContextObject{ID: "hot:" + f, Kind: "hot", Title: filepath.Base(f), Path: f, Stale: stale})
		}
	}
	if today := runtime.AssembleToday(&runtime.WorkspaceTools{
		Inbox:    a.Inbox,
		Schedule: a.Schedule,
		Projects: a.Projects,
	}); today != "" {
		add(ContextObject{ID: "today", Kind: "today", Title: "Today", Detail: capLine(today, 160)})
	}
	if spill != nil {
		ids := spill.ListIDs()
		sort.Strings(ids)
		if len(ids) > 12 {
			ids = ids[:12]
		}
		for _, id := range ids {
			add(ContextObject{ID: "spill:" + id, Kind: "spill", Title: id, Tokens: spill.Size(id), Path: id})
		}
	}
	mcpDir := runtime.WorkspaceMCPDir(ws)
	if mcpDir != "" {
		if raw, err := os.ReadFile(filepath.Join(mcpDir, "INDEX.md")); err == nil && len(raw) > 0 {
			add(ContextObject{ID: "mcp", Kind: "mcp", Title: "MCP catalog", Detail: "schemas delayed until checkpoint", Path: mcpDir})
		}
	}
	rules := filepath.Join(ws, ".yoyo", "rules", "INDEX.md")
	if _, err := os.Stat(rules); err == nil {
		add(ContextObject{ID: "rules", Kind: "rules", Title: "Rules index", Path: rules})
	}
	termDir := filepath.Join(runtime.WorkspaceContextDir(ws, sessionID), "terminals")
	if ents, err := os.ReadDir(termDir); err == nil {
		n := 0
		for _, e := range ents {
			if !e.IsDir() {
				n++
			}
		}
		if n > 0 {
			add(ContextObject{ID: "terminals", Kind: "terminal", Title: fmt.Sprintf("%d terminal logs", n), Path: termDir})
		}
	}
	return inv
}

func (a *App) SetSessionPlan(id, text string) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	m.PlanText = strings.TrimSpace(text)
	runtime.WritePlanFile(m.ToolRoot(), id, m.PlanText)
	if err := a.writeSession(m); err != nil {
		return m, err
	}
	return a.attachAuthMode(m), nil
}

func (a *App) RestoreSessionFiles(id, from string) (string, error) {
	meta, err := a.GetSession(id)
	if err != nil {
		return "", err
	}
	spill := runtime.BindSpill(a.Home.Root, meta.ToolRoot(), id)
	if spill == nil {
		return "no snapshots", nil
	}
	evs, _ := a.ReadTrace(id)
	after := eventTimeFrom(evs, from, a.Traces, id)
	n, err := runtime.RestoreFileSnapshotsAfter(spill.Dir, meta.ToolRoot(), after)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("restored %d files", n), nil
}

func (a *App) ContextPin(id, kind, key string, pinned bool) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	key = strings.TrimSpace(key)
	if key == "" {
		return m, fmt.Errorf("empty pin key")
	}
	switch kind {
	case "skill":
		m.PinnedSkills = toggleStr(m.PinnedSkills, key, pinned)
	case "file", "hot":
		m.PinnedPaths = toggleStr(m.PinnedPaths, key, pinned)
	default:
		return m, fmt.Errorf("unknown pin kind %s", kind)
	}
	if err := a.writeSession(m); err != nil {
		return m, err
	}
	return a.attachAuthMode(m), nil
}

func toggleStr(list []string, key string, on bool) []string {
	out := make([]string, 0, len(list)+1)
	seen := false
	for _, s := range list {
		if s == key {
			seen = true
			if on {
				out = append(out, s)
			}
			continue
		}
		out = append(out, s)
	}
	if on && !seen {
		out = append(out, key)
	}
	return out
}

func capLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if n <= 0 || len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
