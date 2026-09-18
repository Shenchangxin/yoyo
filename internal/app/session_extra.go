package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func bundledSkillsDir(evals string) string {
	if evals == "" {
		return "skills"
	}
	return filepath.Join(filepath.Dir(evals), "skills")
}

func toRuntimeAtts(atts []Attachment) []runtime.Attachment {
	out := make([]runtime.Attachment, 0, len(atts))
	for _, a := range atts {
		out = append(out, runtime.Attachment{Path: a.Path, Name: a.Name, MIME: a.MIME, DataB64: a.DataB64})
	}
	return out
}

func (a *App) DeleteSession(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("empty session id")
	}
	if a.Running(id) {
		_ = a.Interrupt(id)
	}
	if a.Threads != nil {
		a.Threads.Forget(id)
	}
	a.mu.Lock()
	if cancel := a.runs[id]; cancel != nil {
		cancel()
		delete(a.runs, id)
	}
	a.mu.Unlock()
	a.queueMu.Lock()
	delete(a.queue, id)
	delete(a.steers, id)
	a.queueMu.Unlock()
	ws := ""
	if m, err := a.GetSession(id); err == nil {
		ws = m.Workspace
	}
	root := a.Home.Sessions()
	var first error
	for _, name := range []string{id + ".meta.json", id + ".jsonl"} {
		err := os.Remove(filepath.Join(root, name))
		if err != nil && !os.IsNotExist(err) && first == nil {
			first = err
		}
	}
	if a.Traces != nil {
		if err := a.Traces.Remove(id); err != nil && first == nil {
			first = err
		}
	}
	runtime.RemoveSessionContext(a.Home.Root, ws, id)
	return first
}

func (a *App) ArchiveSession(id string, archived bool) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	m.Archived = archived
	return m, a.writeSession(m)
}

func (a *App) PinSession(id string, pinned bool) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	m.Pinned = pinned
	return m, a.writeSession(m)
}

func (a *App) SetSessionModel(id, model string) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	m.Model = strings.TrimSpace(model)
	return m, a.writeSession(m)
}

func (a *App) SearchSessions(q string, includeArchived bool) ([]SessionMeta, error) {
	list, err := a.ListSessions()
	if err != nil {
		return nil, err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	allowed := map[string]bool{}
	useIdx := false
	if a.Threads != nil && a.Threads.Index != nil && q != "" {
		useIdx = true
		for _, id := range a.Threads.Index.Match(q) {
			allowed[id] = true
		}
	}
	var out []SessionMeta
	for _, m := range list {
		if m.Archived && !includeArchived {
			continue
		}
		if q == "" {
			out = append(out, m)
			continue
		}
		if useIdx && allowed[m.ID] {
			out = append(out, m)
			continue
		}
		blob := strings.ToLower(m.Title + " " + m.ID + " " + m.Workspace)
		if strings.Contains(blob, q) {
			out = append(out, m)
			continue
		}
		if evs, err := a.Traces.Read(m.ID); err == nil {
			for _, ev := range evs {
				text, _ := ev.Payload["text"].(string)
				if strings.Contains(strings.ToLower(text), q) {
					out = append(out, m)
					break
				}
			}
		}
	}
	return out, nil
}

func (a *App) ResumeSession(id string) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	if a.Threads != nil && a.Caps != nil {
		st := a.Threads.Load(id)
		if len(st.SessionCaps) > 0 {
			var lv []capability.Level
			for _, c := range st.SessionCaps {
				lv = append(lv, capability.Level(c))
			}
			a.Caps.RestoreSession(id, lv)
		}
	}
	return m, nil
}

func (a *App) ExportSession(id string) (string, error) {
	evs, err := a.Trajectory(id)
	if err != nil {
		return "", err
	}
	m, _ := a.GetSession(id)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", strings.TrimSpace(m.Title))
	fmt.Fprintf(&b, "- id: `%s`\n- workspace: `%s`\n\n", m.ID, m.Workspace)
	for _, ev := range evs {
		text, _ := ev.Payload["text"].(string)
		switch ev.Type {
		case "user":
			fmt.Fprintf(&b, "## User\n\n%s\n\n", text)
		case "assistant":
			if delta, _ := ev.Payload["delta"].(bool); delta {
				continue
			}
			fmt.Fprintf(&b, "## Assistant\n\n%s\n\n", text)
		}
	}
	return b.String(), nil
}

func (a *App) CompactSession(id string) (string, error) {
	evs, err := a.Trajectory(id)
	if err != nil {
		return "", err
	}
	meta, err := a.GetSession(id)
	if err != nil {
		return "", err
	}
	hash := meta.Harness
	if hash == "" {
		hash = a.ActiveHash()
	}
	loop, frags, _, _, _, _, err := a.Materials(hash)
	if err != nil {
		return "", err
	}
	msgs := runtime.MessagesFromEvents(evs)
	spill := runtime.BindSpill(a.Home.Root, meta.Workspace, id)
	model := a.Config.Model
	if meta.Model != "" {
		model = meta.Model
	}
	window := runtime.ModelContextWindow(model)
	client, _ := a.Client()
	note := runtime.CompactHistory(a.Traces, id, msgs, loop, spill, client, model, window, frags)
	runtime.WriteDiscoverIndex(meta.Workspace, id, spill)
	if a.Hub != nil {
		a.Hub.Publish(trace.Event{
			Type:      trace.TypeCompact,
			Source:    "user",
			SessionID: id,
			Payload:   map[string]any{"note": note, "kind": "checkpoint"},
		})
	}
	return note, nil
}

func (a *App) Doctor() map[string]any {
	ws := a.Config.Workspace
	return map[string]any{
		"ok":              true,
		"version":         a.Health()["version"],
		"harness":         a.ActiveHash(),
		"workspace":       ws,
		"workspace_ready": WorkspaceReady(ws),
		"vault":           a.Vault.Status(),
		"isolated":        isolated(),
		"mcp":             a.MCP.List(),
		"isolation":       a.IsolationReport(),
	}
}

func (a *App) Logs(limit int) map[string]any {
	if limit <= 0 {
		limit = 80
	}
	recs, _ := a.Journal.ReadAll()
	if n := len(recs); n > limit {
		recs = recs[n-limit:]
	}
	return map[string]any{"journal": recs, "path": a.Journal.Path}
}

func (a *App) persistMCP() {
	var out []MCPServerConfig
	for _, info := range a.MCP.Info() {
		out = append(out, MCPServerConfig{Name: info.Name, Command: info.Command, Args: info.Args, Endpoint: info.Endpoint})
	}
	a.Config.MCP = out
	_ = a.SaveConfig()
}

func (a *App) ReplaceMCP(servers []MCPServerConfig) error {
	for _, info := range a.MCP.Info() {
		_ = a.MCP.Stop(info.Name)
	}
	for _, srv := range servers {
		if srv.Name == "" {
			continue
		}
		if srv.Endpoint != "" {
			if err := a.MCP.StartHTTP(srv.Name, srv.Endpoint); err != nil {
				return err
			}
			continue
		}
		if srv.Command == "" {
			continue
		}
		if err := a.MCP.Start(srv.Name, srv.Command, srv.Args); err != nil {
			return err
		}
	}
	a.persistMCP()
	return nil
}

func (a *App) StartMCP(name, command string, args []string) error {
	if err := a.MCP.Start(name, command, args); err != nil {
		return err
	}
	a.persistMCP()
	return nil
}

func (a *App) StopMCP(name string) error {
	if err := a.MCP.Stop(name); err != nil {
		return err
	}
	a.persistMCP()
	return nil
}
