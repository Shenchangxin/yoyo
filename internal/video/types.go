package video

type Mode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Hint  string `json:"hint"`
	Ready bool   `json:"ready"`
}

func Modes() []Mode {
	return []Mode{
		{ID: "drama", Title: "Short drama", Hint: "Novel to episode: script, assets, shots, stitch.", Ready: true},
		{ID: "canvas", Title: "Infinite canvas", Hint: "Spatial generation. Not wired yet.", Ready: false},
		{ID: "creative", Title: "Creative", Hint: "Single-clip play. Not wired yet.", Ready: false},
	}
}

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

type Style struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Value       string `json:"value"`
	Prompt      string `json:"prompt"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

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

type Character struct {
	ID          string `json:"id"`
	DramaID     string `json:"drama_id"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Appearance  string `json:"appearance"`
	Styling     string `json:"styling"`
	FinalPrompt string `json:"final_prompt"`
	ImageHash   string `json:"image_hash"`
	SortOrder   int    `json:"sort_order"`
	Linked      bool   `json:"linked"`
}

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

type JobParams struct {
	Size               string   `json:"size,omitempty"`
	ReferenceImages    []string `json:"reference_images,omitempty"`
	ReferenceImageURLs []string `json:"reference_image_urls,omitempty"`
	ReferenceVideoURLs []string `json:"reference_video_urls,omitempty"`
	FirstFrameURL      string   `json:"first_frame_url,omitempty"`
	LastFrameURL       string   `json:"last_frame_url,omitempty"`
	GenerateAudio      bool     `json:"generate_audio,omitempty"`
	Duration           int      `json:"duration,omitempty"`
	AspectRatio        string   `json:"aspect_ratio,omitempty"`
	Resolution         string   `json:"resolution,omitempty"`
	ShotIDs            []string `json:"shot_ids,omitempty"`
}

type EnqueueImage struct {
	Prompt      string
	Model       string
	ProviderID  string
	Size        string
	AspectRatio string
	Refs        []string
	DramaID     string
	EpisodeID   string
	CharacterID string
	SceneID     string
	PropID      string
}

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
