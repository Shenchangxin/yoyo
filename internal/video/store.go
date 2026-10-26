// store.go 用文档库保存短剧、画布、短片和剪辑的集合，以及会话绑定和开关。

package video

import (
	"errors"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/filestore"
)

// 这些名字是文档库里的集合和单文件。
const (
	// colDramas 是短剧文档的集合名 dramas。
	colDramas          = "dramas"
	// colEpisodes 是分集文档的集合名 episodes。
	colEpisodes        = "episodes"
	// colSources 是原文文档的集合名 sources。
	colSources         = "sources"
	// colPlans 是分集方案的集合名 episode_plans。
	colPlans           = "episode_plans"
	// colCharacters 是角色文档的集合名 characters。
	colCharacters      = "characters"
	// colScenes 是场景文档的集合名 scenes。
	colScenes          = "scenes"
	// colProps 是道具文档的集合名 props。
	colProps           = "props"
	// colShots 是镜头文档的集合名 shots。
	colShots           = "shots"
	// colJobs 是生成任务的集合名 jobs。
	colJobs            = "jobs"
	// colStyles 是视觉风格的集合名 styles。
	colStyles          = "styles"
	// colMerges 是拼接结果的集合名 merges。
	colMerges          = "merges"
	// colProjects 是画布工程的集合名 canvas_projects。
	colProjects        = "canvas_projects"
	// colSnapshots 是画布快照的集合名 canvas_snapshots。
	colSnapshots       = "canvas_snapshots"
	// colResources 是画布资源的集合名 canvas_resources。
	colResources       = "canvas_resources"
	// colFolders 是画布文件夹的集合名 canvas_folders。
	colFolders         = "canvas_folders"
	// colAssets 是画布素材的集合名 canvas_assets。
	colAssets          = "canvas_assets"
	// colUnits 是画布单元的集合名 canvas_units。
	colUnits           = "canvas_units"
	// colLessons 是画布教训记录的集合名 canvas_lessons。
	colLessons         = "canvas_lessons"
	// colUploads 是画布分片上传的集合名 canvas_uploads。
	colUploads         = "canvas_uploads"
	// colTaskLogs 是画布任务日志的集合名 canvas_task_logs。
	colTaskLogs        = "canvas_task_logs"
	// colTextDeltas 是画布文本增量的集合名 canvas_text_deltas。
	colTextDeltas      = "canvas_text_deltas"
	// colUserSkills 是画布用户技能的集合名 canvas_user_skills。
	colUserSkills      = "canvas_user_skills"
	// colChannels 是画布频道的集合名 canvas_channels。
	colChannels        = "canvas_channels"
	// colStudioTopics 是短片选题的集合名 studio_topics。
	colStudioTopics    = "studio_topics"
	// colStudioProjects 是短片工程的集合名 studio_projects。
	colStudioProjects  = "studio_projects"
	// colStudioPublishes 是短片发布记录的集合名 studio_publishes。
	colStudioPublishes = "studio_publishes"
	// colCutProjects 是剪辑工程的集合名 cut_projects。
	colCutProjects     = "cut_projects"
	// colFilms 是翻拍工程的集合名 films。
	colFilms           = "films"
	// fileSettings 是键值设置文件 _settings.json。
	fileSettings       = "_settings.json"
	// fileSessionBinds 是短剧会话绑定文件 _session_binds.json。
	fileSessionBinds   = "_session_binds.json"
	// fileCanvasBinds 是画布会话绑定文件 _canvas_session_binds.json。
	fileCanvasBinds    = "_canvas_session_binds.json"
	// fileStudioBinds 是短片会话绑定文件 _studio_session_binds.json。
	fileStudioBinds    = "_studio_session_binds.json"
	// fileCutBinds 是剪辑会话绑定文件 _cut_session_binds.json。
	fileCutBinds       = "_cut_session_binds.json"
	// fileSkillFlags 是技能开关文件 _skill_flags.json。
	fileSkillFlags     = "_skill_flags.json"
	// filePluginFlags 是插件开关文件 _plugin_flags.json。
	filePluginFlags    = "_plugin_flags.json"
)

// dramaRec 在短剧上多记删除时间。
type dramaRec struct {
	Drama
	DeletedAt string `json:"deleted_at,omitempty"`
}

// episodeRec 在分集上多记删除时间，以及挂上的角色、场景和道具标识。
type episodeRec struct {
	Episode
	DeletedAt    string   `json:"deleted_at,omitempty"`
	CharacterIDs []string `json:"character_ids,omitempty"`
	SceneIDs     []string `json:"scene_ids,omitempty"`
	PropIDs      []string `json:"prop_ids,omitempty"`
}

// characterRec 在角色上多记创建、更新和删除时间。
type characterRec struct {
	Character
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	DeletedAt string `json:"deleted_at,omitempty"`
}

// sceneRec 在场景上多记创建、更新和删除时间。
type sceneRec struct {
	Scene
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	DeletedAt string `json:"deleted_at,omitempty"`
}

// propRec 在道具上多记创建、更新和删除时间。
type propRec struct {
	Prop
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	DeletedAt string `json:"deleted_at,omitempty"`
}

// shotRec 在镜头上多记删除、创建和更新时间。
type shotRec struct {
	Shot
	DeletedAt string `json:"deleted_at,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// styleRec 在风格上多记是否为种子数据，以及创建和更新时间。
type styleRec struct {
	Style
	Seeded    bool   `json:"seeded,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// canvasSnapshotRec 是一张画布快照的修订、标题和节点计数。
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

// canvasResourceRec 是一条画布资源的哈希、种类、尺寸和播放状态。
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

// canvasFolderRec 是一个画布文件夹的名字和顺序。
type canvasFolderRec struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Position  int    `json:"position"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	DeletedAt string `json:"deleted_at,omitempty"`
}

// canvasAssetRec 是一条画布素材，指向文件夹、资源和载荷。
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

// canvasUnitRec 是画布工程里的一个有序单元。
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

// canvasLessonRec 记下一条画布教训的主题、情境和步骤。
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

// canvasUploadRec 记下一次分片上传的块数、已收块数和临时目录。
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

// canvasTaskLogRec 是一条画布任务日志的级别、阶段和消息。
type canvasTaskLogRec struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Level     string `json:"level"`
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

// canvasDeltaRec 是一条按序号排列的画布文本增量。
type canvasDeltaRec struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Seq       int    `json:"seq"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// canvasUserSkillRec 保存一条用户技能的载荷和删除时间。
type canvasUserSkillRec struct {
	ID          string `json:"id"`
	PayloadJSON string `json:"payload_json"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	DeletedAt   string `json:"deleted_at,omitempty"`
}

// skillFlagRec 记下技能是否已添加、是否喜欢。
type skillFlagRec struct {
	Added     bool   `json:"added"`
	Liked     bool   `json:"liked"`
	UpdatedAt string `json:"updated_at"`
}

// pluginFlagRec 记下插件是否启用。
type pluginFlagRec struct {
	Enabled   bool   `json:"enabled"`
	UpdatedAt string `json:"updated_at"`
}

// getDoc 按集合和标识读取文档。文档库为空或标识为空白时返回 ErrNotFound。
func getDoc[T any](e *Engine, col, id string) (T, error) {
	var v T
	if e.docs == nil || strings.TrimSpace(id) == "" {
		return v, filestore.ErrNotFound
	}
	err := e.docs.Get(filestore.Rel(col, id), &v)
	return v, err
}

// putDoc 把文档写入集合。文档库未打开或标识为空时返回 store not open。
func (e *Engine) putDoc(col, id string, v any) error {
	if e.docs == nil || id == "" {
		return errors.New("store not open")
	}
	return e.docs.Put(filestore.Rel(col, id), v)
}

// delDoc 删除集合里的文档。文档库为空时直接返回 nil。
func (e *Engine) delDoc(col, id string) error {
	if e.docs == nil {
		return nil
	}
	return e.docs.Delete(filestore.Rel(col, id))
}

// loadCol 读出整个集合。文档库为空或读取失败时返回 nil。
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

// getJSON 按相对路径读取一份 JSON。文档库为空时返回 ErrNotFound。
func (e *Engine) getJSON(rel string, v any) error {
	if e.docs == nil {
		return filestore.ErrNotFound
	}
	return e.docs.Get(rel, v)
}

// putJSON 按相对路径写入一份 JSON。文档库为空时返回 store not open。
func (e *Engine) putJSON(rel string, v any) error {
	if e.docs == nil {
		return errors.New("store not open")
	}
	return e.docs.Put(rel, v)
}

// kvSettings 读出键值设置。没有文档时返回空 map。
func (e *Engine) kvSettings() map[string]string {
	m := map[string]string{}
	_ = e.getJSON(fileSettings, &m)
	if m == nil {
		m = map[string]string{}
	}
	return m
}

// loadBinds 读出短剧会话绑定。没有文档时返回空 map。
func (e *Engine) loadBinds() map[string]Bind {
	m := map[string]Bind{}
	_ = e.getJSON(fileSessionBinds, &m)
	if m == nil {
		m = map[string]Bind{}
	}
	return m
}

// saveBinds 写回短剧会话绑定。
func (e *Engine) saveBinds(m map[string]Bind) error {
	return e.putJSON(fileSessionBinds, m)
}

// loadCanvasBinds 读出会话到画布标识的映射。没有文档时返回空 map。
func (e *Engine) loadCanvasBinds() map[string]string {
	m := map[string]string{}
	_ = e.getJSON(fileCanvasBinds, &m)
	if m == nil {
		m = map[string]string{}
	}
	return m
}

// saveCanvasBinds 写回画布会话绑定。
func (e *Engine) saveCanvasBinds(m map[string]string) error {
	return e.putJSON(fileCanvasBinds, m)
}

// loadStudioBinds 读出会话到短片工程的映射。没有文档时返回空 map。
func (e *Engine) loadStudioBinds() map[string]string {
	m := map[string]string{}
	_ = e.getJSON(fileStudioBinds, &m)
	if m == nil {
		m = map[string]string{}
	}
	return m
}

// saveStudioBinds 写回短片会话绑定。
func (e *Engine) saveStudioBinds(m map[string]string) error {
	return e.putJSON(fileStudioBinds, m)
}

// loadSkillFlags 读出技能开关。没有文档时返回空 map。
func (e *Engine) loadSkillFlags() map[string]skillFlagRec {
	m := map[string]skillFlagRec{}
	_ = e.getJSON(fileSkillFlags, &m)
	if m == nil {
		m = map[string]skillFlagRec{}
	}
	return m
}

// saveSkillFlags 写回技能开关。
func (e *Engine) saveSkillFlags(m map[string]skillFlagRec) error {
	return e.putJSON(fileSkillFlags, m)
}

// loadPluginFlags 读出插件开关。没有文档时返回空 map。
func (e *Engine) loadPluginFlags() map[string]pluginFlagRec {
	m := map[string]pluginFlagRec{}
	_ = e.getJSON(filePluginFlags, &m)
	if m == nil {
		m = map[string]pluginFlagRec{}
	}
	return m
}

// savePluginFlags 写回插件开关。
func (e *Engine) savePluginFlags(m map[string]pluginFlagRec) error {
	return e.putJSON(filePluginFlags, m)
}

// containsID 报告切片里是否已有这个标识。
func containsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// addID 在标识非空且尚未出现时把它加到切片末尾。
func addID(ids []string, id string) []string {
	id = strings.TrimSpace(id)
	if id == "" || containsID(ids, id) {
		return ids
	}
	return append(ids, id)
}

// removeID 去掉切片里等于该标识的项。
func removeID(ids []string, id string) []string {
	var out []string
	for _, x := range ids {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}
