package video

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

type CutProject struct {
	ID         string        `json:"id"`
	Title      string        `json:"title,omitempty"`
	SessionID  string        `json:"session_id,omitempty"`
	SourceHash string        `json:"source_hash,omitempty"`
	PosterHash string        `json:"poster_hash,omitempty"`
	ResultHash string        `json:"result_hash,omitempty"`
	Duration   float64       `json:"duration,omitempty"`
	Breakdown  cut.Breakdown `json:"breakdown"`
	EditList   cut.EditList  `json:"edit_list"`
	Stems      cut.Stems     `json:"stems"`
	Captions   cut.Captions  `json:"captions"`
	Note       string        `json:"note,omitempty"`
	Stale      bool          `json:"stale,omitempty"`
	CreatedAt  string        `json:"created_at,omitempty"`
	UpdatedAt  string        `json:"updated_at,omitempty"`
}

func (e *Engine) cutBin() cut.Bin {
	return cut.Bin{FFmpeg: e.FFMPEG, FFprobe: e.FFProbe, Whisper: e.WHISPER}
}

func (e *Engine) ListCutProjects() []CutProject {
	out := loadCol[CutProject](e, colCutProjects)
	if out == nil {
		return []CutProject{}
	}
	return out
}

func (e *Engine) GetCutProject(id string) (CutProject, error) {
	p, err := getDoc[CutProject](e, colCutProjects, strings.TrimSpace(id))
	if err != nil {
		return CutProject{}, fmt.Errorf("cut project not found")
	}
	return p, nil
}

type CutOpen struct {
	Title      string
	Hash       string
	ResourceID string
	SessionID  string
}

func parseCanvasResourceID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "resource:")
	for _, pref := range []string{"/resource/", "/resources/"} {
		if i := strings.LastIndex(s, pref); i >= 0 {
			s = s[i+len(pref):]
			if j := strings.IndexAny(s, "/?#"); j >= 0 {
				s = s[:j]
			}
			break
		}
	}
	return strings.TrimSpace(s)
}

func (e *Engine) OpenCut(in CutOpen) (CutProject, error) {
	hash := strings.TrimSpace(in.Hash)
	if hash == "" {
		rid := parseCanvasResourceID(in.ResourceID)
		if rid == "" {
			return CutProject{}, fmt.Errorf("source hash or canvas resource required")
		}
		res, err := e.getCanvasResource(rid)
		if err != nil {
			return CutProject{}, err
		}
		hash = strings.TrimSpace(res.CASHash)
	}
	if hash == "" {
		return CutProject{}, fmt.Errorf("resource has no CAS hash")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "成片"
	}
	sid := strings.TrimSpace(in.SessionID)
	if sid != "" {
		if id, ok := e.CutSessionBind(sid); ok {
			if cur, err := e.GetCutProject(id); err == nil && strings.TrimSpace(cur.SourceHash) == hash {
				e.ReleaseWorkshopBinds(sid)
				_ = e.BindCutSession(sid, cur.ID)
				return cur, nil
			}
		}
	}
	p, err := e.CreateCutProject(CutProject{Title: title})
	if err != nil {
		return CutProject{}, err
	}
	p, err = e.CutIngest(p.ID, hash, title)
	if err != nil {
		return p, err
	}
	if sid != "" {
		e.ReleaseWorkshopBinds(sid)
		if err := e.BindCutSession(sid, p.ID); err != nil {
			return p, err
		}
	}
	return p, nil
}

func (e *Engine) CreateCutProject(in CutProject) (CutProject, error) {
	now := Now()
	in.ID = NewID()
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		in.Title = "胶片"
	}
	in.CreatedAt = now
	in.UpdatedAt = now
	if err := e.putDoc(colCutProjects, in.ID, in); err != nil {
		return CutProject{}, err
	}
	e.emitCut(in)
	return in, nil
}

func (e *Engine) saveCut(p CutProject) (CutProject, error) {
	p.UpdatedAt = Now()
	if err := e.putDoc(colCutProjects, p.ID, p); err != nil {
		return CutProject{}, err
	}
	e.emitCut(p)
	return p, nil
}

func (e *Engine) DeleteCutProject(id string) error {
	id = strings.TrimSpace(id)
	m := e.loadCutBinds()
	for sid, pid := range m {
		if pid == id {
			delete(m, sid)
		}
	}
	_ = e.saveCutBinds(m)
	return e.delDoc(colCutProjects, id)
}

func (e *Engine) BindCutSession(sessionID, projectID string) error {
	sessionID = strings.TrimSpace(sessionID)
	projectID = strings.TrimSpace(projectID)
	if sessionID == "" || projectID == "" {
		return fmt.Errorf("session and project required")
	}
	if _, err := e.GetCutProject(projectID); err != nil {
		return err
	}
	m := e.loadCutBinds()
	next := make(map[string]string, len(m)+1)
	for sid, pid := range m {
		if sid == sessionID || pid == projectID {
			continue
		}
		next[sid] = pid
	}
	next[sessionID] = projectID
	return e.saveCutBinds(next)
}

func (e *Engine) ReleaseCutBind(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	m := e.loadCutBinds()
	if _, ok := m[sessionID]; !ok {
		return
	}
	delete(m, sessionID)
	_ = e.saveCutBinds(m)
}

func (e *Engine) CutSessionBind(sessionID string) (string, bool) {
	m := e.loadCutBinds()
	id := strings.TrimSpace(m[strings.TrimSpace(sessionID)])
	return id, id != ""
}

func (e *Engine) cutOf(sessionID, id string) (CutProject, error) {
	if id == "" {
		id, _ = e.CutSessionBind(sessionID)
	}
	if id == "" {
		return CutProject{}, fmt.Errorf("no cut project bound — pass id or bind first")
	}
	return e.GetCutProject(id)
}

func (e *Engine) emitCut(p CutProject) {
	if e.OnHub == nil {
		return
	}
	e.OnHub("cut_state", map[string]any{
		"id": p.ID, "title": p.Title, "source_hash": p.SourceHash,
		"result_hash": p.ResultHash, "duration": p.Duration,
		"clips": len(p.Breakdown.Clips), "cuts": len(p.EditList.Cuts),
	})
}

func (e *Engine) loadCutBinds() map[string]string {
	m := map[string]string{}
	_ = e.getJSON(fileCutBinds, &m)
	if m == nil {
		m = map[string]string{}
	}
	return m
}

func (e *Engine) saveCutBinds(m map[string]string) error {
	return e.putJSON(fileCutBinds, m)
}
