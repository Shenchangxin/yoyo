// Package ir is the Yoyo game-world shared representation.
// Coordinates stay host-owned; models pick candidate ids.
package ir

const SchemaVersion = 1

type Vec3 [3]float64

type Status struct {
	Usable     bool    `json:"usable"`
	Confidence float64 `json:"confidence,omitempty"`
	Reason     string  `json:"reason,omitempty"`
}

type ScenePacket struct {
	Status      Status           `json:"view_status"`
	Summary     string           `json:"overall_summary,omitempty"`
	Environment string           `json:"environment_visual_type,omitempty"`
	Terrain     Terrain          `json:"terrain"`
	Regions     []Region         `json:"regions,omitempty"`
	Notes       []string         `json:"visual_notes,omitempty"`
	Camera      Camera           `json:"camera"`
	Genre       string           `json:"genre,omitempty"`
	Seed        int64            `json:"seed,omitempty"`
	Surfaces    map[string][]any `json:"surface_distribution,omitempty"`
}

type Terrain struct {
	Width    int       `json:"width"`
	Depth    int       `json:"depth"`
	CellSize float64   `json:"cell_size"`
	Heights  []float64 `json:"heights"`
	Surfaces []string  `json:"surfaces"`
}

type Region struct {
	ID             string   `json:"id"`
	Category       string   `json:"category"`
	Bounds         [4]float64 `json:"bounds"`
	Accessibility  string   `json:"accessibility,omitempty"`
	Affordances    []string `json:"affordances,omitempty"`
	Walkability    string   `json:"walkability,omitempty"`
}

type Camera struct {
	Preset   string  `json:"preset,omitempty"`
	Position Vec3    `json:"position,omitempty"`
	Yaw      float64 `json:"yaw,omitempty"`
	Pitch    float64 `json:"pitch,omitempty"`
}

type RulePlan struct {
	Status         Status          `json:"game_rule_status"`
	Title          string          `json:"title,omitempty"`
	Description    string          `json:"description,omitempty"`
	CoreLoop       string          `json:"core_loop,omitempty"`
	Role           string          `json:"player_or_npc_role,omitempty"`
	Objective      string          `json:"objective,omitempty"`
	Win            string          `json:"win_condition,omitempty"`
	Fail           string          `json:"fail_condition,omitempty"`
	TimeLimit      int             `json:"time_limit_seconds,omitempty"`
	AllowedActions []string        `json:"allowed_actions,omitempty"`
	Randomness     Randomness      `json:"randomness_policy"`
	Required       []ElementReq    `json:"required_gameplay_elements_for_placement,omitempty"`
	Scoring        []string        `json:"scoring_rules,omitempty"`
}

type Randomness struct {
	FreeNavigation    bool   `json:"free_navigation"`
	FixedPathRequired bool   `json:"fixed_path_required"`
	RandomChoice      bool   `json:"random_choice_allowed"`
	Description       string `json:"description,omitempty"`
}

type ElementReq struct {
	ElementType   string `json:"element_type"`
	Purpose       string `json:"purpose,omitempty"`
	PlacementHint string `json:"placement_hint_from_scene,omitempty"`
	RequiredCount int    `json:"required_count"`
}

type Candidate struct {
	ID            string   `json:"candidate_id"`
	GridCell      string   `json:"grid_cell_3x3,omitempty"`
	WorldXYZ      Vec3     `json:"world_xyz"`
	NormalXYZ     Vec3     `json:"normal_xyz"`
	SurfaceType   string   `json:"surface_type,omitempty"`
	Walkability   string   `json:"walkability,omitempty"`
	Kind          string   `json:"candidate_kind,omitempty"`
	SupportRadius float64  `json:"max_supported_footprint_radius_m,omitempty"`
	Tags          []string `json:"placement_tags,omitempty"`
	Distance      float64  `json:"distance_from_origin,omitempty"`
}

type CandidateSet struct {
	SchemaVersion int         `json:"candidate_schema_version"`
	Candidates    []Candidate `json:"candidates"`
}

type Placement struct {
	PlacementID      string  `json:"placement_id"`
	ElementType      string  `json:"element_type"`
	SourceReq        int     `json:"source_requirement_index"`
	CandidateID      string  `json:"candidate_id"`
	WorldXYZ         Vec3    `json:"world_xyz"`
	NormalXYZ        Vec3    `json:"normal_xyz"`
	HeightOffset     float64 `json:"height_offset,omitempty"`
	Mode             string  `json:"placement_mode,omitempty"`
	Purpose          string  `json:"gameplay_purpose,omitempty"`
	Reason           string  `json:"placement_reason,omitempty"`
}

type PlacementPlan struct {
	Status   Status      `json:"placement_status"`
	Elements []Placement `json:"elements"`
}

type Asset struct {
	AssetID      string   `json:"asset_id"`
	Name         string   `json:"asset_name,omitempty"`
	Role         string   `json:"asset_role"`
	PlacementIDs []string `json:"placement_ids,omitempty"`
	Primitive    string   `json:"primitive,omitempty"`
	SizeMeters   Vec3     `json:"expected_size_meters,omitempty"`
	Color        string   `json:"color,omitempty"`
	TextureHash  string   `json:"texture_hash,omitempty"`
	Prompt       string   `json:"generation_prompt_en,omitempty"`
	SkipMesh     bool     `json:"skip_mesh,omitempty"`
	JobID        string   `json:"job_id,omitempty"`
}

type AssetPlan struct {
	Assets []Asset `json:"assets"`
	Notes  []string `json:"notes,omitempty"`
}

type StateDelta struct {
	Score      int    `json:"score,omitempty"`
	HP         int    `json:"hp,omitempty"`
	Flag       string `json:"flag,omitempty"`
	Inventory  string `json:"inventory,omitempty"`
	Win        bool   `json:"win,omitempty"`
	Fail       bool   `json:"fail,omitempty"`
	Teleport   bool   `json:"teleport,omitempty"`
	TeleportTo Vec3   `json:"teleport_to,omitempty"`
}

type Interaction struct {
	ID           string     `json:"id"`
	Participants []string   `json:"participants"`
	Trigger      string     `json:"trigger"`
	Precondition string     `json:"precondition,omitempty"`
	Delta        StateDelta `json:"delta"`
}

type Correspondence struct {
	ID             string   `json:"id"`
	Kappa          string   `json:"kappa"`
	SceneRef       string   `json:"scene_ref,omitempty"`
	RuntimeRef     string   `json:"runtime_ref,omitempty"`
	InteractionIDs []string `json:"interaction_ids,omitempty"`
	LogicIDs       []string `json:"logic_ids,omitempty"`
}

// Playable is the sealed runtime contract written to playable/game.json.
type Playable struct {
	SchemaVersion int               `json:"schema_version"`
	Title         string            `json:"title,omitempty"`
	Genre         string            `json:"genre,omitempty"`
	Objective     string            `json:"objective,omitempty"`
	Win           string            `json:"win,omitempty"`
	Fail          string            `json:"fail,omitempty"`
	TimeLimit     int               `json:"time_limit_seconds,omitempty"`
	NeedCollect   int               `json:"need_collect,omitempty"`
	Camera        string            `json:"camera,omitempty"`
	Player        PlayableActor     `json:"player"`
	Scene         Terrain           `json:"scene"`
	Elements      []PlayableElement `json:"elements"`
	Interactions  []Interaction     `json:"interactions"`
}

type PlayableActor struct {
	Position Vec3    `json:"position"`
	Yaw      float64 `json:"yaw,omitempty"`
	Radius   float64 `json:"radius,omitempty"`
	Height   float64 `json:"height,omitempty"`
	Speed    float64 `json:"speed,omitempty"`
}

type PlayableElement struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Position  Vec3    `json:"position"`
	Size      Vec3    `json:"size,omitempty"`
	Color     string  `json:"color,omitempty"`
	Primitive string  `json:"primitive,omitempty"`
	Trigger   string  `json:"trigger,omitempty"`
	Solid     bool    `json:"solid,omitempty"`
}
