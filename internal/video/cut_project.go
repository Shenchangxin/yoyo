// cut_project.go 保存剪辑工程，并把会话绑到工程上。

package video

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// CutProject 是一条剪辑工程，含源片、参考片、拆解、剪辑表、音轨和字幕。
type CutProject struct {
	ID            string        `json:"id"`
	Title         string        `json:"title,omitempty"`
	SessionID     string        `json:"session_id,omitempty"`
	SourceHash    string        `json:"source_hash,omitempty"`
	Kind          string        `json:"kind,omitempty"`
	ReferenceHash string        `json:"reference_hash,omitempty"`
	PosterHash    string        `json:"poster_hash,omitempty"`
	ResultHash    string        `json:"result_hash,omitempty"`
	Duration      float64       `json:"duration,omitempty"`
	Breakdown     cut.Breakdown `json:"breakdown"`
	EditList      cut.EditList  `json:"edit_list"`
	Stems         cut.Stems     `json:"stems"`
	Captions      cut.Captions  `json:"captions"`
	Note          string        `json:"note,omitempty"`
	Stale         bool          `json:"stale,omitempty"`
	CreatedAt     string        `json:"created_at,omitempty"`
	UpdatedAt     string        `json:"updated_at,omitempty"`
}

// cutBin 用引擎上的 ffmpeg、ffprobe 和 whisper 路径组成剪辑命令。
func (e *Engine) cutBin() cut.Bin {
	return cut.Bin{FFmpeg: e.FFMPEG, FFprobe: e.FFProbe, Whisper: e.WHISPER}
}

// ListCutProjects 读出全部剪辑工程。没有时返回空切片。
func (e *Engine) ListCutProjects() []CutProject {
	out := loadCol[CutProject](e, colCutProjects)
	if out == nil {
		return []CutProject{}
	}
	return out
}

// GetCutProject 按标识读取工程。没有时返回 cut project not found。
func (e *Engine) GetCutProject(id string) (CutProject, error) {
	p, err := getDoc[CutProject](e, colCutProjects, strings.TrimSpace(id))
	if err != nil {
		return CutProject{}, fmt.Errorf("cut project not found")
	}
	return p, nil
}

// CutOpen 是打开剪辑时的标题、源哈希、画布资源和会话。
type CutOpen struct {
	Title      string
	Hash       string
	ResourceID string
	SessionID  string
}

// parseCanvasResourceID 去掉 resource: 前缀，或从 /resource/、/resources/ 后面取出标识，并截掉 /、?、#。
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

// OpenCut 用哈希或画布资源的 CAS 哈希建工程并摄入。标题空着时用「成片」。同一会话已绑着同一源片时，先放开短片、短剧和画布绑定，再绑回这条剪辑。
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
				cur.SessionID = sid
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
		p.SessionID = sid
	}
	return p, nil
}

// CreateCutProject 新建工程。标题空着时用「胶片」，并通知 cut_state。
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

// saveCut 更新时间后写回工程，并通知 cut_state。
func (e *Engine) saveCut(p CutProject) (CutProject, error) {
	p.UpdatedAt = Now()
	if err := e.putDoc(colCutProjects, p.ID, p); err != nil {
		return CutProject{}, err
	}
	e.emitCut(p)
	return p, nil
}

// DeleteCutProject 删掉指向该工程的会话绑定，再删除工程文档。
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

// BindCutSession 把会话绑到工程，并去掉这份绑定表里同一会话或同一工程的旧行。会话或工程为空时失败。
func (e *Engine) BindCutSession(sessionID, projectID string) error {
	sessionID = strings.TrimSpace(sessionID)
	projectID = strings.TrimSpace(projectID)
	if sessionID == "" || projectID == "" {
		return fmt.Errorf("session and project required")
	}
	p, err := e.GetCutProject(projectID)
	if err != nil {
		return err
	}
	if p.SessionID != sessionID {
		p.SessionID = sessionID
		if _, err := e.saveCut(p); err != nil {
			return err
		}
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

// ReleaseCutBind 去掉该会话的剪辑绑定。
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

// LatestCutSession 返回绑定表里第一个指向该工程的会话。
func (e *Engine) LatestCutSession(projectID string) string {
	projectID = strings.TrimSpace(projectID)
	if e == nil || projectID == "" {
		return ""
	}
	for sessionID, id := range e.loadCutBinds() {
		if id == projectID {
			return sessionID
		}
	}
	return ""
}

// CutSessionBind 返回会话绑着的工程标识。
func (e *Engine) CutSessionBind(sessionID string) (string, bool) {
	m := e.loadCutBinds()
	id := strings.TrimSpace(m[strings.TrimSpace(sessionID)])
	return id, id != ""
}

// cutOf 标识为空时改用会话绑定。仍没有时返回 no cut project bound。
func (e *Engine) cutOf(sessionID, id string) (CutProject, error) {
	if id == "" {
		id, _ = e.CutSessionBind(sessionID)
	}
	if id == "" {
		return CutProject{}, fmt.Errorf("no cut project bound — pass id or bind first")
	}
	return e.GetCutProject(id)
}

// emitCut 在 OnHub 非空时送出 cut_state，带上标识、标题、哈希、时长和片段数。
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

// loadCutBinds 读出剪辑会话绑定。没有文档时返回空 map。
func (e *Engine) loadCutBinds() map[string]string {
	m := map[string]string{}
	_ = e.getJSON(fileCutBinds, &m)
	if m == nil {
		m = map[string]string{}
	}
	return m
}

// saveCutBinds 写回剪辑会话绑定。
func (e *Engine) saveCutBinds(m map[string]string) error {
	return e.putJSON(fileCutBinds, m)
}
