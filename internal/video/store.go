package video

import (
	"errors"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/filestore"
)

const (
	colDramas        = "dramas"
	colEpisodes      = "episodes"
	colSources       = "sources"
	colPlans         = "episode_plans"
	colCharacters    = "characters"
	colScenes        = "scenes"
	colProps         = "props"
	colShots         = "shots"
	colJobs          = "jobs"
	colStyles        = "styles"
	colMerges        = "merges"
	colProjects      = "canvas_projects"
	colSnapshots     = "canvas_snapshots"
	colResources     = "canvas_resources"
	colFolders       = "canvas_folders"
	colAssets        = "canvas_assets"
	colUnits         = "canvas_units"
	colLessons       = "canvas_lessons"
	colUploads       = "canvas_uploads"
	colTaskLogs      = "canvas_task_logs"
	colTextDeltas    = "canvas_text_deltas"
	colUserSkills    = "canvas_user_skills"
	colChannels      = "canvas_channels"
	fileSettings     = "_settings.json"
	fileSessionBinds = "_session_binds.json"
	fileCanvasBinds  = "_canvas_session_binds.json"
	fileSkillFlags   = "_skill_flags.json"
)

type dramaRec struct {
	Drama
	DeletedAt string `json:"deleted_at,omitempty"`
}

type episodeRec struct {
	Episode
	DeletedAt    string   `json:"deleted_at,omitempty"`
	CharacterIDs []string `json:"character_ids,omitempty"`
	SceneIDs     []string `json:"scene_ids,omitempty"`
	PropIDs      []string `json:"prop_ids,omitempty"`
}

type characterRec struct {
	Character
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	DeletedAt string `json:"deleted_at,omitempty"`
}

type sceneRec struct {
	Scene
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	DeletedAt string `json:"deleted_at,omitempty"`
}

type propRec struct {
	Prop
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	DeletedAt string `json:"deleted_at,omitempty"`
}

type shotRec struct {
	Shot
	DeletedAt string `json:"deleted_at,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type styleRec struct {
	Style
	Seeded    bool   `json:"seeded,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type canvasSnapshotRec struct {
	ID              string `json:"id"`
	CanvasID        string `json:"canvas_id"`
	Revision        int64  `json:"revision"`
	Title           string `json:"title"`
	PayloadJSON     string `json:"payload_json"`
	Reason          string `json:"reason"`
	NodeCount       int    `json:"node_count"`
	ConnectionCount int    `json:"connection_count"`
	CreatedAt       string `json:"created_at"`
}

type canvasResourceRec struct {
	ID             string `json:"id"`
	CASHash        string `json:"cas_hash"`
	Kind           string `json:"kind"`
	Mime           string `json:"mime"`
	Bytes          int64  `json:"bytes"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	DurationMs     int    `json:"duration_ms"`
	PosterHash     string `json:"poster_hash"`
	PlaybackStatus string `json:"playback_status"`
	FileName       string `json:"file_name"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	DeletedAt      string `json:"deleted_at,omitempty"`
}

type canvasFolderRec struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Position  int    `json:"position"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	DeletedAt string `json:"deleted_at,omitempty"`
}

type canvasAssetRec struct {
	ID          string `json:"id"`
	FolderID    string `json:"folder_id"`
	Kind        string `json:"kind"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	ResourceID  string `json:"resource_id"`
	PayloadJSON string `json:"payload_json"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	DeletedAt   string `json:"deleted_at,omitempty"`
}

type canvasUnitRec struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	PayloadJSON string `json:"payload_json"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	DeletedAt   string `json:"deleted_at,omitempty"`
}

type canvasLessonRec struct {
	ID        string `json:"id"`
	Topic     string `json:"topic"`
	Category  string `json:"category"`
	Situation string `json:"situation"`
	Lesson    string `json:"lesson"`
	StepsJSON string `json:"steps_json"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type canvasUploadRec struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	FileName   string `json:"file_name"`
	Mime       string `json:"mime"`
	Size       int    `json:"size"`
	ChunkSize  int    `json:"chunk_size"`
	ChunkCount int    `json:"chunk_count"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	DurationMs int    `json:"duration_ms"`
	Received   int    `json:"received"`
	Dir        string `json:"dir"`
	CreatedAt  string `json:"created_at"`
}

type canvasTaskLogRec struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Level     string `json:"level"`
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type canvasDeltaRec struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Seq       int    `json:"seq"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type canvasUserSkillRec struct {
	ID          string `json:"id"`
	PayloadJSON string `json:"payload_json"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	DeletedAt   string `json:"deleted_at,omitempty"`
}

type skillFlagRec struct {
	Added     bool   `json:"added"`
	Liked     bool   `json:"liked"`
	UpdatedAt string `json:"updated_at"`
}

func getDoc[T any](e *Engine, col, id string) (T, error) {
	var v T
	if e.docs == nil || strings.TrimSpace(id) == "" {
		return v, filestore.ErrNotFound
	}
	err := e.docs.Get(filestore.Rel(col, id), &v)
	return v, err
}

func (e *Engine) putDoc(col, id string, v any) error {
	if e.docs == nil || id == "" {
		return errors.New("store not open")
	}
	return e.docs.Put(filestore.Rel(col, id), v)
}

func (e *Engine) delDoc(col, id string) error {
	if e.docs == nil {
		return nil
	}
	return e.docs.Delete(filestore.Rel(col, id))
}

func loadCol[T any](e *Engine, col string) []T {
	if e.docs == nil {
		return nil
	}
	all, err := filestore.LoadAll[T](e.docs, col)
	if err != nil {
		return nil
	}
	return all
}

func (e *Engine) getJSON(rel string, v any) error {
	if e.docs == nil {
		return filestore.ErrNotFound
	}
	return e.docs.Get(rel, v)
}

func (e *Engine) putJSON(rel string, v any) error {
	if e.docs == nil {
		return errors.New("store not open")
	}
	return e.docs.Put(rel, v)
}

func (e *Engine) kvSettings() map[string]string {
	m := map[string]string{}
	_ = e.getJSON(fileSettings, &m)
	if m == nil {
		m = map[string]string{}
	}
	return m
}

func (e *Engine) loadBinds() map[string]Bind {
	m := map[string]Bind{}
	_ = e.getJSON(fileSessionBinds, &m)
	if m == nil {
		m = map[string]Bind{}
	}
	return m
}

func (e *Engine) saveBinds(m map[string]Bind) error {
	return e.putJSON(fileSessionBinds, m)
}

func (e *Engine) loadCanvasBinds() map[string]string {
	m := map[string]string{}
	_ = e.getJSON(fileCanvasBinds, &m)
	if m == nil {
		m = map[string]string{}
	}
	return m
}

func (e *Engine) saveCanvasBinds(m map[string]string) error {
	return e.putJSON(fileCanvasBinds, m)
}

func (e *Engine) loadSkillFlags() map[string]skillFlagRec {
	m := map[string]skillFlagRec{}
	_ = e.getJSON(fileSkillFlags, &m)
	if m == nil {
		m = map[string]skillFlagRec{}
	}
	return m
}

func (e *Engine) saveSkillFlags(m map[string]skillFlagRec) error {
	return e.putJSON(fileSkillFlags, m)
}

func containsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func addID(ids []string, id string) []string {
	id = strings.TrimSpace(id)
	if id == "" || containsID(ids, id) {
		return ids
	}
	return append(ids, id)
}

func removeID(ids []string, id string) []string {
	var out []string
	for _, x := range ids {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}
