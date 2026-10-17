package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Shenchangxin/yoyo/internal/filestore"
	"github.com/Shenchangxin/yoyo/internal/game/ir"
)

const (
	colGames     = "games"
	fileBinds    = "_game_session_binds.json"
	StageDraft   = "draft"
	StageScene   = "scene"
	StageRules   = "rules"
	StageCands   = "candidates"
	StagePlaced  = "placed"
	StageAssets  = "assets"
	StageBuilt   = "compiled"
	StageChecked = "verified"
	StageFailed  = "failed"
)

type ImageEnqueuer func(prompt, projectID, assetID string) (jobID string, err error)

type Completer func(system, user string) (string, error)

type Engine struct {
	docs      *filestore.Store
	Workspace func() string
	Enqueue   ImageEnqueuer
	Complete  Completer
	SkipImage bool

	mu sync.Mutex
}

type Project struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Intent     string   `json:"intent,omitempty"`
	Genre      string   `json:"genre,omitempty"`
	Camera     string   `json:"camera,omitempty"`
	Slug       string   `json:"slug"`
	Stage      string   `json:"stage"`
	Status     string   `json:"status"`
	Error      string   `json:"error,omitempty"`
	NeedCollect int     `json:"need_collect,omitempty"`
	JobIDs     []string `json:"job_ids,omitempty"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

func Open(docsRoot string, workspace func() string) (*Engine, error) {
	docs, err := filestore.New(docsRoot)
	if err != nil {
		return nil, err
	}
	if workspace == nil {
		workspace = func() string { return "" }
	}
	return &Engine{docs: docs, Workspace: workspace}, nil
}

func (e *Engine) workspace() string {
	if e == nil || e.Workspace == nil {
		return ""
	}
	return strings.TrimSpace(e.Workspace())
}

func (p Project) Rel() string {
	slug := strings.TrimSpace(p.Slug)
	if slug == "" {
		slug = p.ID
	}
	return filepath.ToSlash(filepath.Join("game-worlds", slug))
}

func (e *Engine) Create(in Project) (Project, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		in.Title = "Game world"
	}
	if in.ID == "" {
		in.ID = newID()
	}
	if in.Slug == "" {
		in.Slug = slugify(in.Title, in.ID)
	}
	if in.Genre == "" {
		in.Genre = "tps"
	}
	if in.Camera == "" {
		in.Camera = normalizeCamera(in.Genre)
	}
	if in.Stage == "" {
		in.Stage = StageDraft
	}
	if in.Status == "" {
		in.Status = "open"
	}
	in.CreatedAt = now()
	in.UpdatedAt = in.CreatedAt
	if err := e.put(in); err != nil {
		return in, err
	}
	if err := e.initWorkspace(in); err != nil {
		return in, err
	}
	return in, nil
}

func (e *Engine) Get(id string) (Project, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Project{}, fmt.Errorf("id required")
	}
	var p Project
	if err := e.docs.Get(filestore.Rel(colGames, id), &p); err != nil {
		return Project{}, err
	}
	return p, nil
}

func (e *Engine) List() []Project {
	all, err := filestore.LoadAll[Project](e.docs, colGames)
	if err != nil {
		return []Project{}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].UpdatedAt > all[j].UpdatedAt })
	if all == nil {
		return []Project{}
	}
	return all
}

func (e *Engine) Update(p Project) (Project, error) {
	cur, err := e.Get(p.ID)
	if err != nil {
		return p, err
	}
	if strings.TrimSpace(p.Title) != "" {
		cur.Title = strings.TrimSpace(p.Title)
	}
	if p.Intent != "" {
		cur.Intent = p.Intent
	}
	if p.Genre != "" {
		cur.Genre = p.Genre
		cur.Camera = normalizeCamera(p.Genre)
	}
	if p.Camera != "" {
		cur.Camera = normalizeCamera(p.Camera)
	}
	if p.Stage != "" {
		cur.Stage = p.Stage
	}
	if p.Status != "" {
		cur.Status = p.Status
	}
	cur.Error = p.Error
	cur.NeedCollect = p.NeedCollect
	cur.JobIDs = p.JobIDs
	cur.UpdatedAt = now()
	return cur, e.put(cur)
}

func (e *Engine) Delete(id string) error {
	p, err := e.Get(id)
	if err != nil {
		return err
	}
	m := e.loadBinds()
	for sid, pid := range m {
		if pid == id {
			delete(m, sid)
		}
	}
	_ = e.saveBinds(m)
	if ws := e.workspace(); ws != "" && p.Slug != "" {
		_ = os.RemoveAll(filepath.Join(ws, "game-worlds", p.Slug))
	}
	return e.docs.Delete(filestore.Rel(colGames, id))
}

func (e *Engine) put(p Project) error {
	return e.docs.Put(filestore.Rel(colGames, p.ID), p)
}

func (e *Engine) Unbind(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session required")
	}
	m := e.loadBinds()
	if _, ok := m[sessionID]; !ok {
		return nil
	}
	next := make(map[string]string, len(m))
	for sid, pid := range m {
		if sid == sessionID {
			continue
		}
		next[sid] = pid
	}
	return e.saveBinds(next)
}

func (e *Engine) Bind(sessionID, projectID string) error {
	sessionID = strings.TrimSpace(sessionID)
	projectID = strings.TrimSpace(projectID)
	if sessionID == "" {
		return fmt.Errorf("session required")
	}
	if projectID == "" {
		return e.Unbind(sessionID)
	}
	if _, err := e.Get(projectID); err != nil {
		return err
	}
	m := e.loadBinds()
	next := make(map[string]string, len(m)+1)
	for sid, pid := range m {
		if sid == sessionID || pid == projectID {
			continue
		}
		next[sid] = pid
	}
	next[sessionID] = projectID
	return e.saveBinds(next)
}

func (e *Engine) SessionBind(sessionID string) (string, bool) {
	id := strings.TrimSpace(e.loadBinds()[strings.TrimSpace(sessionID)])
	return id, id != ""
}

func (e *Engine) HistoryRows() []map[string]any {
	binds := e.loadBinds()
	sess := map[string]string{}
	for sid, pid := range binds {
		if _, ok := sess[pid]; !ok {
			sess[pid] = sid
		}
	}
	var out []map[string]any
	for _, p := range e.List() {
		out = append(out, map[string]any{
			"kind":       "game",
			"id":         p.ID,
			"title":      p.Title,
			"updated_at": p.UpdatedAt,
			"session_id": sess[p.ID],
			"status":     p.Status,
			"gate":       p.Stage,
		})
	}
	return out
}

func (e *Engine) loadBinds() map[string]string {
	m := map[string]string{}
	_ = e.docs.Get(fileBinds, &m)
	if m == nil {
		m = map[string]string{}
	}
	return m
}

func (e *Engine) saveBinds(m map[string]string) error {
	return e.docs.Put(fileBinds, m)
}

func (e *Engine) projectDir(p Project) (string, error) {
	ws := e.workspace()
	if ws == "" {
		return "", fmt.Errorf("video workspace is not set")
	}
	dir := filepath.Join(ws, filepath.FromSlash(p.Rel()))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (e *Engine) initWorkspace(p Project) error {
	dir, err := e.projectDir(p)
	if err != nil {
		return err
	}
	for _, sub := range []string{"ir", "world", "playable", "qa"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return err
		}
	}
	progress := "# " + p.Title + "\n\nintent: " + p.Intent + "\nstage: " + p.Stage + "\n"
	return os.WriteFile(filepath.Join(dir, "_progress.md"), []byte(progress), 0o644)
}

func (e *Engine) writeIR(p Project, name string, v any) error {
	dir, err := e.projectDir(p)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(dir, "ir", name), raw, 0o644); err != nil {
		return err
	}
	_ = e.writeProgress(p)
	return nil
}

func (e *Engine) readIR(p Project, name string, v any) error {
	dir, err := e.projectDir(p)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(dir, "ir", name))
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

func (e *Engine) writeProgress(p Project) error {
	dir, err := e.projectDir(p)
	if err != nil {
		return err
	}
	body := fmt.Sprintf("# %s\n\nintent: %s\nstage: %s\nstatus: %s\n", p.Title, p.Intent, p.Stage, p.Status)
	return os.WriteFile(filepath.Join(dir, "_progress.md"), []byte(body), 0o644)
}

func (e *Engine) resolve(sessionID, id string) (Project, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		id, _ = e.SessionBind(sessionID)
	}
	if id == "" {
		return Project{}, fmt.Errorf("bind a game world first")
	}
	return e.Get(id)
}

func normalizeCamera(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "fps", "first", "first-person":
		return "fps"
	case "top", "topdown", "iso", "俯视":
		return "top"
	default:
		return "tps"
	}
}

func (e *Engine) syncQA(p Project) Project {
	dir, err := e.projectDir(p)
	if err != nil {
		return p
	}
	raw, err := os.ReadFile(filepath.Join(dir, "qa", "verification.json"))
	if err != nil {
		return p
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return p
	}
	status := strings.ToUpper(strings.TrimSpace(fmt.Sprint(doc["status"])))
	next := p
	switch status {
	case "PASS":
		if p.Stage == StageChecked && p.Status == "verified" && p.Error == "" {
			return p
		}
		next.Stage = StageChecked
		next.Status = "verified"
		next.Error = ""
	case "FAIL":
		if p.Status == "qa_failed" {
			return p
		}
		next.Status = "qa_failed"
	default:
		return p
	}
	if got, err := e.Update(next); err == nil {
		return got
	}
	return next
}

func (e *Engine) Receipt(p Project) map[string]any {
	p = e.syncQA(p)
	out := map[string]any{
		"id": p.ID, "title": p.Title, "intent": p.Intent, "genre": p.Genre,
		"camera": p.Camera, "slug": p.Slug, "stage": p.Stage, "status": p.Status,
		"error": p.Error, "rel": p.Rel(), "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		"need_collect": p.NeedCollect, "job_ids": p.JobIDs,
	}
	out["root"] = filepath.ToSlash(p.Rel())
	dir, err := e.projectDir(p)
	if err != nil {
		return out
	}
	playable := filepath.ToSlash(filepath.Join(p.Rel(), "playable", "index.html"))
	if _, err := os.Stat(filepath.Join(dir, "playable", "index.html")); err == nil {
		out["playable"] = playable
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "qa", "verification.json")); err == nil {
		var doc map[string]any
		if json.Unmarshal(raw, &doc) == nil {
			if st := strings.TrimSpace(fmt.Sprint(doc["status"])); st != "" && st != "<nil>" {
				out["qa"] = st
			}
		}
	}
	return out
}

func (e *Engine) Snapshot(p Project) map[string]any {
	out := e.Receipt(p)
	p = e.syncQA(p)
	dir, err := e.projectDir(p)
	if err != nil {
		return out
	}
	out["root"] = filepath.ToSlash(p.Rel())
	playable := filepath.ToSlash(filepath.Join(p.Rel(), "playable", "index.html"))
	if _, err := os.Stat(filepath.Join(dir, "playable", "index.html")); err == nil {
		out["playable"] = playable
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "playable", "index.html")); err == nil {
		out["preview_html"] = string(raw)
	}
	irFiles := []string{"scene.json", "rules.json", "candidates.json", "placement.json", "assets.json", "interactions.json", "correspondence.json"}
	docs := map[string]any{}
	for _, name := range irFiles {
		var v any
		if err := e.readIR(p, name, &v); err == nil {
			docs[strings.TrimSuffix(name, ".json")] = v
		}
	}
	if len(docs) > 0 {
		out["ir"] = docs
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "playable", "game.json")); err == nil {
		var g ir.Playable
		if json.Unmarshal(raw, &g) == nil {
			out["playable_spec"] = g
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "qa", "verification.json")); err == nil {
		var v any
		if json.Unmarshal(raw, &v) == nil {
			out["verification"] = v
		}
	}
	return out
}
