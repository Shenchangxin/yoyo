package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// SessionDump is the lossless on-disk bundle for one session identity.
// Trajectory is the raw JSONL event stream (not the inspector projection).
// Artifacts include full spill/context bodies, not previews.
type SessionDump struct {
	ID         string                `json:"id"`
	Query      string                `json:"query,omitempty"`
	Home       string                `json:"home"`
	Meta       SessionMeta           `json:"meta"`
	Paths      SessionDumpPaths      `json:"paths"`
	Trajectory []trace.Event         `json:"trajectory"`
	Artifacts  []SessionArtifactBody `json:"artifacts"`
}

type SessionDumpPaths struct {
	Home       string `json:"home"`
	Trajectory string `json:"trajectory"`
	Meta       string `json:"meta,omitempty"`
	SpillDir   string `json:"spill_dir"`
	ContextDir string `json:"context_dir,omitempty"`
	Dump       string `json:"dump"`
}

type SessionArtifactBody struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Path     string `json:"path,omitempty"`
	Bytes    int    `json:"bytes"`
	Encoding string `json:"encoding,omitempty"`
	Text     string `json:"text,omitempty"`
	DataB64  string `json:"data_b64,omitempty"`
}

// SessionIndexEntry is a locator row: enough to open trajectory + spill
// from a session id without loading bodies.
type SessionIndexEntry struct {
	ID             string `json:"id"`
	Title          string `json:"title,omitempty"`
	Workspace      string `json:"workspace,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	HasMeta        bool   `json:"has_meta"`
	Events         int    `json:"events"`
	SpillArtifacts int    `json:"spill_artifacts"`
	TrajectoryPath string `json:"trajectory_path"`
	SpillDir       string `json:"spill_dir"`
}

// ResolveSessionID maps an exact id or unique prefix to the durable session
// identity stored under YOYO_HOME/sessions.
func (a *App) ResolveSessionID(query string) (string, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "", fmt.Errorf("session id required")
	}
	ids := a.knownSessionIDs()
	for _, id := range ids {
		if id == q {
			return id, nil
		}
	}
	if a.sessionOnDisk(q) {
		return q, nil
	}
	var matches []string
	for _, id := range ids {
		if strings.HasPrefix(id, q) {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		shown := matches
		if len(shown) > 8 {
			shown = shown[:8]
		}
		return "", fmt.Errorf("session id %q is ambiguous (%d matches, e.g. %s)", q, len(matches), strings.Join(shown, ", "))
	}
	return "", fmt.Errorf("session %q not found under %s", q, a.Home.Sessions())
}

// DumpSession returns the complete trajectory and artifact bodies for a
// session id (exact or unique prefix) and writes sessions/{id}/dump.json.
func (a *App) DumpSession(query string) (SessionDump, error) {
	id, err := a.ResolveSessionID(query)
	if err != nil {
		return SessionDump{}, err
	}
	meta, metaErr := a.GetSession(id)
	if metaErr != nil {
		meta = SessionMeta{ID: id}
	}
	evs, err := a.Trajectory(id)
	if err != nil {
		return SessionDump{}, err
	}
	if evs == nil {
		evs = []trace.Event{}
	}
	paths := a.sessionDumpPaths(id, meta)
	dump := SessionDump{
		ID:         id,
		Query:      query,
		Home:       a.Home.Root,
		Meta:       meta,
		Paths:      paths,
		Trajectory: evs,
		Artifacts:  a.collectDumpArtifacts(id, meta),
	}
	if err := writeSessionDump(paths.Dump, dump); err != nil {
		return dump, err
	}
	return dump, nil
}

func (a *App) ListSessionIndex() []SessionIndexEntry {
	ids := a.knownSessionIDs()
	out := make([]SessionIndexEntry, 0, len(ids))
	for _, id := range ids {
		meta, metaErr := a.GetSession(id)
		if metaErr != nil {
			meta = SessionMeta{ID: id}
		}
		evs, _ := a.Trajectory(id)
		spill := runtime.BindSpill(a.Home.Root, meta.Workspace, id)
		nSpill := 0
		if spill != nil {
			nSpill = len(spill.ListIDs())
		}
		created := ""
		if !meta.CreatedAt.IsZero() {
			created = meta.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		out = append(out, SessionIndexEntry{
			ID:             id,
			Title:          meta.Title,
			Workspace:      meta.Workspace,
			CreatedAt:      created,
			HasMeta:        metaErr == nil,
			Events:         len(evs),
			SpillArtifacts: nSpill,
			TrajectoryPath: filepath.Join(a.Home.Sessions(), id+".jsonl"),
			SpillDir:       a.Home.SessionSpill(id),
		})
	}
	return out
}

func (a *App) sessionDumpPaths(id string, meta SessionMeta) SessionDumpPaths {
	p := SessionDumpPaths{
		Home:       a.Home.Root,
		Trajectory: filepath.Join(a.Home.Sessions(), id+".jsonl"),
		SpillDir:   a.Home.SessionSpill(id),
		Dump:       filepath.Join(a.Home.Sessions(), id, "dump.json"),
	}
	metaPath := filepath.Join(a.Home.Sessions(), id+".meta.json")
	if _, err := os.Stat(metaPath); err == nil {
		p.Meta = metaPath
	}
	if dir := runtime.WorkspaceContextDir(meta.Workspace, id); dir != "" {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			p.ContextDir = dir
		}
	}
	return p
}

func (a *App) collectDumpArtifacts(id string, meta SessionMeta) []SessionArtifactBody {
	seen := map[string]bool{}
	var out []SessionArtifactBody
	add := func(art SessionArtifactBody) {
		key := art.Kind + ":" + art.ID
		if art.ID == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, art)
	}

	spill := runtime.BindSpill(a.Home.Root, meta.Workspace, id)
	if spill != nil {
		ids := spill.ListIDs()
		sort.Strings(ids)
		for _, sid := range ids {
			text, err := spill.Get(sid)
			if err != nil {
				continue
			}
			kind := "spill"
			if sid == "notes" {
				kind = "notes"
			}
			path := filepath.Join(a.Home.SessionSpill(id), sid+".txt")
			if _, err := os.Stat(path); err != nil && meta.Workspace != "" {
				alt := filepath.Join(runtime.WorkspaceSpillDir(meta.Workspace, id), sid+".txt")
				if _, err2 := os.Stat(alt); err2 == nil {
					path = alt
				}
			}
			add(bytesArtifact(kind, sid, path, []byte(text)))
		}
	}

	if dir := runtime.WorkspaceContextDir(meta.Workspace, id); dir != "" {
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if rel == "dump.json" {
				return nil
			}
			if strings.HasPrefix(rel, "spill/") && strings.HasSuffix(rel, ".txt") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			add(bytesArtifact("context", rel, path, b))
			return nil
		})
	}
	if out == nil {
		return []SessionArtifactBody{}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func bytesArtifact(kind, id, path string, b []byte) SessionArtifactBody {
	art := SessionArtifactBody{Kind: kind, ID: id, Path: path, Bytes: len(b)}
	if utf8.Valid(b) {
		art.Encoding = "utf-8"
		art.Text = string(b)
		return art
	}
	art.Encoding = "base64"
	art.DataB64 = base64.StdEncoding.EncodeToString(b)
	return art
}

func writeSessionDump(path string, dump SessionDump) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(dump)
}

func (a *App) knownSessionIDs() []string {
	seen := map[string]bool{}
	var out []string
	add := func(id string) {
		id = strings.TrimSpace(strings.ReplaceAll(id, "\\", "/"))
		id = strings.TrimPrefix(id, "./")
		if id == "" || id == "." || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	if listed, err := a.Traces.ListSessions(); err == nil {
		for _, id := range listed {
			add(id)
		}
	}
	root := a.Home.Sessions()
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if d.Name() == "spill" {
				parent := filepath.ToSlash(filepath.Dir(rel))
				if parent != "." && parent != "" {
					add(parent)
				}
			}
			return nil
		}
		switch {
		case strings.HasSuffix(rel, ".jsonl"):
			add(strings.TrimSuffix(rel, ".jsonl"))
		case strings.HasSuffix(rel, ".meta.json"):
			add(strings.TrimSuffix(rel, ".meta.json"))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func (a *App) sessionOnDisk(id string) bool {
	if id == "" {
		return false
	}
	for _, p := range []string{
		filepath.Join(a.Home.Sessions(), id+".jsonl"),
		filepath.Join(a.Home.Sessions(), id+".meta.json"),
		a.Home.SessionSpill(id),
	} {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
