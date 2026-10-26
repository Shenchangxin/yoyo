// types.go 定义短剧、分集、资产、镜头和排队任务的数据结构。

package video

import "encoding/json"

// Mode 是一种工作台的标识、标题、提示和是否就绪。
type Mode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Hint  string `json:"hint"`
	Ready bool   `json:"ready"`
}

// Modes 返回短剧、画布和短片三种工作台，Ready 都为真。
func Modes() []Mode {
	return []Mode{
		{ID: "drama", Title: "Short drama", Hint: "Novel to episode: script, assets, shots, stitch.", Ready: true},
		{ID: "canvas", Title: "Infinite canvas", Hint: "Spatial generation workspace: nodes, tools, timeline, director.", Ready: true},
		{ID: "creative", Title: "Shorts", Hint: "Hot lists to a finished short: topics, voice, assemble, handoff.", Ready: true},
	}
}

// Provider 是一条供应商配置，含服务类型、基址、保险库键名和模型。
type Provider struct {
	ID          string `json:"id"`
	ServiceType string `json:"service_type"`
	Provider    string `json:"provider"`
	Name        string `json:"name"`
	BaseURL     string `json:"base_url"`
	VaultKey    string `json:"vault_key"`
	Model       string `json:"model"`
	Models      string `json:"models"`
	Priority    int    `json:"priority"`
	IsDefault   bool   `json:"is_default"`
	IsActive    bool   `json:"is_active"`
	Settings    string `json:"settings"`
	HasKey      bool   `json:"has_key"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Style 是一条视觉风格的名字、取值和提示词。
type Style struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Value       string `json:"value"`
	Prompt      string `json:"prompt"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

// Job 是一次生成任务，记下供应商、远程标识、结果哈希和所属的剧、集、镜头或资产。
type Job struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	VaultKey     string `json:"vault_key"`
	RemoteID     string `json:"remote_id"`
	Prompt       string `json:"prompt"`
	Params       string `json:"params"`
	ResultHash   string `json:"result_hash"`
	PosterHash   string `json:"poster_hash"`
	Error        string `json:"error"`
	DramaID      string `json:"drama_id"`
	EpisodeID    string `json:"episode_id"`
	StoryboardID string `json:"storyboard_id"`
	CharacterID  string `json:"character_id"`
	SceneID      string `json:"scene_id"`
	PropID       string `json:"prop_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	CompletedAt  string `json:"completed_at"`
}

// Drama 是一部短剧的标题、类型、画幅和集数。
type Drama struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Genre         string `json:"genre"`
	Style         string `json:"style"`
	AspectRatio   string `json:"aspect_ratio"`
	Status        string `json:"status"`
	ThumbnailHash string `json:"thumbnail_hash"`
	EpisodeCount  int    `json:"episode_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// Episode 是一集的正文、剧本、供应商、分辨率和流水线状态。
type Episode struct {
	ID              string `json:"id"`
	DramaID         string `json:"drama_id"`
	EpisodeNumber   int    `json:"episode_number"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	ScriptContent   string `json:"script_content"`
	Status          string `json:"status"`
	VideoHash       string `json:"video_hash"`
	PosterHash      string `json:"poster_hash"`
	ImageProviderID string `json:"image_provider_id"`
	VideoProviderID string `json:"video_provider_id"`
	ImageModel      string `json:"image_model"`
	VideoModel      string `json:"video_model"`
	TTSProviderID   string `json:"tts_provider_id"`
	TTSModel        string `json:"tts_model"`
	Resolution      string `json:"resolution"`
	Pipeline        string `json:"pipeline"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// Character 是短剧里的一个角色，含外形、定妆提示词和静帧哈希。
type Character struct {
	ID               string `json:"id"`
	DramaID          string `json:"drama_id"`
	Name             string `json:"name"`
	Role             string `json:"role"`
	Appearance       string `json:"appearance"`
	Styling          string `json:"styling"`
	FinalPrompt      string `json:"final_prompt"`
	ImageHash        string `json:"image_hash"`
	SortOrder        int    `json:"sort_order"`
	Linked           bool   `json:"linked"`
	FirstSeenEpisode int    `json:"first_seen_episode,omitempty"`
}

// Scene 是一个场景的地点、时段、光线和静帧哈希。
type Scene struct {
	ID          string `json:"id"`
	DramaID     string `json:"drama_id"`
	Location    string `json:"location"`
	TimeOfDay   string `json:"time_of_day"`
	Prompt      string `json:"prompt"`
	Lighting    string `json:"lighting"`
	FinalPrompt string `json:"final_prompt"`
	ImageHash   string `json:"image_hash"`
	Linked      bool   `json:"linked"`
}

// Prop 是一件道具的名字、类型和静帧哈希。
type Prop struct {
	ID          string `json:"id"`
	DramaID     string `json:"drama_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	FinalPrompt string `json:"final_prompt"`
	ImageHash   string `json:"image_hash"`
	Linked      bool   `json:"linked"`
}

// Shot 是一个镜头的景别、运动、时长、视频哈希，以及挂上的角色和道具。
type Shot struct {
	ID           string   `json:"id"`
	EpisodeID    string   `json:"episode_id"`
	SceneID      string   `json:"scene_id"`
	ShotNumber   int      `json:"shot_number"`
	Title        string   `json:"title"`
	ShotType     string   `json:"shot_type"`
	Angle        string   `json:"angle"`
	Movement     string   `json:"movement"`
	Atmosphere   string   `json:"atmosphere"`
	Description  string   `json:"description"`
	VideoPrompt  string   `json:"video_prompt"`
	Duration     int      `json:"duration"`
	VideoHash    string   `json:"video_hash"`
	PosterHash   string   `json:"poster_hash"`
	Status       string   `json:"status"`
	CharacterIDs []string `json:"character_ids"`
	PropIDs      []string `json:"prop_ids"`
}

// EpisodeBundle 把一集和它的剧、资产、镜头、任务、时长计划放在一起。
type EpisodeBundle struct {
	Drama      Drama        `json:"drama"`
	Episode    Episode      `json:"episode"`
	Characters []Character  `json:"characters"`
	Scenes     []Scene      `json:"scenes"`
	Props      []Prop       `json:"props"`
	Shots      []Shot       `json:"shots"`
	Jobs       []Job        `json:"jobs"`
	Plan       DurationPlan `json:"plan"`
}

// SourceSlice 是原文的一段，含起止下标、标题和预览。
type SourceSlice struct {
	Index   int    `json:"index"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Heading string `json:"heading"`
	Peek    string `json:"peek"`
}

// Source 是一部剧的原文哈希、字数和切片。
type Source struct {
	ID        string        `json:"id"`
	DramaID   string        `json:"drama_id"`
	Hash      string        `json:"hash"`
	Title     string        `json:"title"`
	RuneCount int           `json:"rune_count"`
	Slices    []SourceSlice `json:"slices"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// PlanEpisode 是分集方案里的一集，含钩子、原文区间和目标秒数。
type PlanEpisode struct {
	N             int      `json:"n"`
	Title         string   `json:"title"`
	Logline       string   `json:"logline"`
	Hook          string   `json:"hook"`
	Start         int      `json:"start"`
	End           int      `json:"end"`
	CharacterIDs  []string `json:"character_ids"`
	TargetSeconds int      `json:"target_seconds"`
	EpisodeID     string   `json:"episode_id,omitempty"`
}

// EpisodePlan 是一部剧的分集方案及其状态。
type EpisodePlan struct {
	ID         string        `json:"id"`
	DramaID    string        `json:"drama_id"`
	Status     string        `json:"status"`
	SourceID   string        `json:"source_id"`
	Episodes   []PlanEpisode `json:"episodes"`
	QueueKind  string        `json:"queue_kind,omitempty"`
	QueueIndex int           `json:"queue_index,omitempty"`
	CreatedAt  string        `json:"created_at"`
	UpdatedAt  string        `json:"updated_at"`
}

// JobParams 是排队任务携带的尺寸、参考、画布节点和重试信息。
type JobParams struct {
	Size               string          `json:"size,omitempty"`
	ReferenceImages    []string        `json:"reference_images,omitempty"`
	ReferenceImageURLs []string        `json:"reference_image_urls,omitempty"`
	ReferenceVideoURLs []string        `json:"reference_video_urls,omitempty"`
	FirstFrameURL      string          `json:"first_frame_url,omitempty"`
	LastFrameURL       string          `json:"last_frame_url,omitempty"`
	GenerateAudio      bool            `json:"generate_audio,omitempty"`
	Duration           int             `json:"duration,omitempty"`
	AspectRatio        string          `json:"aspect_ratio,omitempty"`
	Resolution         string          `json:"resolution,omitempty"`
	ShotIDs            []string        `json:"shot_ids,omitempty"`
	CanvasProjectID    string          `json:"canvas_project_id,omitempty"`
	NodeID             string          `json:"node_id,omitempty"`
	Mode               string          `json:"mode,omitempty"`
	ChannelID          string          `json:"channel_id,omitempty"`
	PluginID           string          `json:"plugin_id,omitempty"`
	ResourceID         string          `json:"resource_id,omitempty"`
	GenerationSpec     json.RawMessage `json:"generation_spec,omitempty"`
	RecoverAttempts    int             `json:"recover_attempts,omitempty"`
	ClientContext      json.RawMessage `json:"client_context,omitempty"`
	TaskType           string          `json:"task_type,omitempty"`
	InputJSON          string          `json:"input_json,omitempty"`
	ResultJSON         string          `json:"result_json,omitempty"`
	ResultState        string          `json:"result_state,omitempty"`
	TextDraft          string          `json:"text_draft,omitempty"`
	MediaStage         string          `json:"media_stage,omitempty"`
	Operation          string          `json:"operation,omitempty"`
	AttemptGroupID     string          `json:"attempt_group_id,omitempty"`
	ClientOperationID  string          `json:"client_operation_id,omitempty"`
	RetryOf            string          `json:"retry_of,omitempty"`
	RemotePayload      json.RawMessage `json:"remote_payload,omitempty"`
	StudioProjectID    string          `json:"studio_project_id,omitempty"`
	StudioShotID       string          `json:"studio_shot_id,omitempty"`
	StudioKind         string          `json:"studio_kind,omitempty"`
}

// EnqueueImage 是入队一张静帧时的提示词、模型、尺寸和引用。
type EnqueueImage struct {
	Prompt          string
	Model           string
	ProviderID      string
	Size            string
	AspectRatio     string
	Refs            []string
	DramaID         string
	EpisodeID       string
	CharacterID     string
	SceneID         string
	PropID          string
	StudioProjectID string
	StudioShotID    string
	StudioKind      string
}

// EnqueueVideo 是入队一段视频时的提示词、时长、画幅和引用。
type EnqueueVideo struct {
	Prompt      string
	Model       string
	ProviderID  string
	Duration    int
	AspectRatio string
	Resolution  string
	Audio       bool
	Refs        []string
	DramaID     string
	EpisodeID   string
	ShotID      string
}
