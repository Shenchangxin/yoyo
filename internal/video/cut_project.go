// cut_project.go 保存剪辑工程，并把会话绑到工程上。

package video

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	// 下面是成片的配方。每次渲染都从源片按配方重做，调色和烧字幕不会叠加，重新合成也不会丢掉配音。

	// UseCuts 为真时成片按剪辑表拼。合成过一次之后保持为真。
	UseCuts bool `json:"use_cuts,omitempty"`
	// Grade 是调色预设。
	Grade string `json:"grade,omitempty"`
	// Enhance 是原声增强预设。
	Enhance string `json:"enhance,omitempty"`
	// Burn 为真时把字幕烧进成片。
	Burn bool `json:"burn,omitempty"`
	// DubReplace 为真时配音替换原声，否则原声压低垫在配音下面。
	DubReplace bool `json:"dub_replace,omitempty"`
	// DuckOff 为真时混音不在人声下压低音乐和原声。
	DuckOff   bool   `json:"duck_off,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// cutBin 用引擎上的 ffmpeg、ffprobe、whisper 和它的模型组成剪辑命令。命令跟着引擎的上下文，引擎关闭时一起停下。WHISPER_LANG 可以指定转写语言。
func (e *Engine) cutBin() cut.Bin {
	return cut.Bin{
		FFmpeg: e.FFMPEG, FFprobe: e.FFProbe, Whisper: e.WHISPER,
		WhisperModel: e.whisperModel(),
		WhisperLang:  strings.TrimSpace(os.Getenv("WHISPER_LANG")),
		Ctx:          e.ctx,
	}
}

// whisperModel 找 whisper.cpp 的模型。没有 whisper 时返回空串。视频目录下的 models 也会找。
func (e *Engine) whisperModel() string {
	if strings.TrimSpace(e.WHISPER) == "" {
		return ""
	}
	var extra []string
	if e.Dir != "" {
		extra = append(extra, filepath.Join(e.Dir, "models"))
	}
	return LookWhisperModel(e.WHISPER, extra...)
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
// id 去掉空白后传给 getDoc，从 colCutProjects 取值。
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
// in 的 Hash 去掉空白后为空时改用 ResourceID 解析出的画布资源 CAS 哈希；Title 去掉空白后为空时写成成片；SessionID 去掉空白后非空时查已有绑定。
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
// in 的 ID 换成 NewID，Title 去掉空白后为空时写成胶片，再整份传给 putDoc 和 emitCut。
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

// saveCut 更新时间后写回工程，并通知 cut_state。只用在新建的工程，或已经拿着这条工程锁的地方；其余改动走 mutateCut。
// p 写上 UpdatedAt 后，用 p.ID 调用 putDoc，并传给 emitCut。
func (e *Engine) saveCut(p CutProject) (CutProject, error) {
	p.UpdatedAt = Now()
	if err := e.putDoc(colCutProjects, p.ID, p); err != nil {
		return CutProject{}, err
	}
	e.emitCut(p)
	return p, nil
}

// mutateCut 在这条剪辑工程的锁里读出最新的一份，交给 fn 改，再写回。fn 返回错误时不写，返回读到的那份和错误。ffmpeg 这类长活要在锁外做完，再用它把结果写上。fn 里不能再调用 mutateCut。
// id 去掉首尾空白后用来加锁和读取。
func (e *Engine) mutateCut(id string, fn func(p *CutProject) error) (CutProject, error) {
	id = strings.TrimSpace(id)
	unlock := e.lockDoc(colCutProjects, id)
	defer unlock()
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	if err := fn(&p); err != nil {
		if errors.Is(err, errUnchanged) {
			return p, nil
		}
		return p, err
	}
	return e.saveCut(p)
}

// DeleteCutProject 删掉指向该工程的会话绑定，再删除工程文档。
// id 去掉空白后，从绑定表里删掉值等于它的项，并传给 delDoc。
func (e *Engine) DeleteCutProject(id string) error {
	id = strings.TrimSpace(id)
	e.bindMu.Lock()
	m := e.loadCutBinds()
	for sid, pid := range m {
		if pid == id {
			delete(m, sid)
		}
	}
	_ = e.saveCutBinds(m)
	e.bindMu.Unlock()
	unlock := e.lockDoc(colCutProjects, id)
	defer unlock()
	return e.delDoc(colCutProjects, id)
}

// BindCutSession 把会话绑到工程，并去掉这份绑定表里同一会话或同一工程的旧行。会话或工程为空时失败。
// sessionID 去掉空白后为空，或 projectID 去掉空白后为空，则返回 session and project required。
// projectID 去掉空白后传给 GetCutProject，并作为绑定值；与旧行的工程标识相等的绑定会被丢掉。
func (e *Engine) BindCutSession(sessionID, projectID string) error {
	sessionID = strings.TrimSpace(sessionID)
	projectID = strings.TrimSpace(projectID)
	if sessionID == "" || projectID == "" {
		return fmt.Errorf("session and project required")
	}
	if _, err := e.mutateCut(projectID, func(p *CutProject) error {
		if p.SessionID == sessionID {
			return errUnchanged
		}
		p.SessionID = sessionID
		return nil
	}); err != nil {
		return err
	}
	e.bindMu.Lock()
	defer e.bindMu.Unlock()
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
// sessionID 去掉空白后为空则返回，否则从绑定表删除该键。
func (e *Engine) ReleaseCutBind(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	e.bindMu.Lock()
	defer e.bindMu.Unlock()
	m := e.loadCutBinds()
	if _, ok := m[sessionID]; !ok {
		return
	}
	delete(m, sessionID)
	_ = e.saveCutBinds(m)
}

// LatestCutSession 返回绑定表里第一个指向该工程的会话。
// projectID 去掉空白后为空，或引擎为 nil 时返回空串，否则用来匹配绑定值。
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
// sessionID 去掉空白后作为绑定表的键。
func (e *Engine) CutSessionBind(sessionID string) (string, bool) {
	m := e.loadCutBinds()
	id := strings.TrimSpace(m[strings.TrimSpace(sessionID)])
	return id, id != ""
}

// cutOf 标识为空时改用会话绑定。仍没有时返回 no cut project bound。
// sessionID 在 id 为空时传给 CutSessionBind。
// id 为空时改用会话绑定，仍为空则返回 no cut project bound，否则传给 GetCutProject。
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
// p 的 ID、Title、SourceHash、ResultHash、Duration、拆解片段数和剪辑条数放进 cut_state。
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
// m 传给 putJSON，键是 fileCutBinds。
func (e *Engine) saveCutBinds(m map[string]string) error {
	return e.putJSON(fileCutBinds, m)
}
