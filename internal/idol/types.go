package idol

import "time"

// Services is what the idol desk calls. Empty endpoints fall back to the
// operator's existing model and generation connections. Keys live in the vault.
type Services struct {
	ChatBaseURL   string `json:"chat_base_url"`
	ChatModel     string `json:"chat_model"`
	JudgeModel    string `json:"judge_model"`
	SpeechBaseURL string `json:"speech_base_url"`
	SpeechModel   string `json:"speech_model"`
	SpeechVoice   string `json:"speech_voice"`
	ASRBaseURL    string `json:"asr_base_url"`
	ASRModel      string `json:"asr_model"`
	ImageBaseURL  string `json:"image_base_url"`
	ImageModel    string `json:"image_model"`
	ReachOut      bool   `json:"reach_out"`
	TextOnly      bool   `json:"text_only"`
	PhoneTexture  bool   `json:"phone_texture"`
	WorkEnabled   bool   `json:"work_enabled"`
	WorkProfileID string `json:"work_profile_id"`
}

// UserCard is how every idol addresses this operator. It is not a harness profile.
type UserCard struct {
	Address  string `json:"address"`
	Gender   string `json:"gender"`
	Timezone string `json:"timezone"`
	Birthday string `json:"birthday"` // MM-DD
}

type Outfit struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Blob string `json:"blob"`
}

type Idol struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Intro         string    `json:"intro"`
	Color         string    `json:"color"`
	SpokenLang    string    `json:"spoken_lang"`
	SubtitleLang  string    `json:"subtitle_lang"`
	Sheet         string    `json:"sheet"`
	Facts         string    `json:"facts"`
	Readings      string    `json:"readings"`
	VoiceID       string    `json:"voice_id"`
	Portrait      string    `json:"portrait"`
	OutfitID      string    `json:"outfit_id"`
	Outfits       []Outfit  `json:"outfits,omitempty"`
	LikedGifts    []string  `json:"liked_gifts,omitempty"`
	CircleID      string    `json:"circle_id"`
	WorkEnabled   bool      `json:"work_enabled"`
	WorkProfileID string    `json:"work_profile_id"`
	WorkSessionID string    `json:"work_session_id"`
	Pinned        bool      `json:"pinned"`
	Hidden        bool      `json:"hidden"`
	Kind          string    `json:"kind,omitempty"` // pet, catalog, or empty for someone the user made
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Circle struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Members []string `json:"members,omitempty"`
}

type Segment struct {
	Spoken   string `json:"spoken"`
	Subtitle string `json:"subtitle"`
}

type Record struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"` // turn, reach, event, work_said
	At          time.Time `json:"at"`
	User        string    `json:"user,omitempty"`
	Segments    []Segment `json:"segments,omitempty"`
	Emotion     string    `json:"emotion,omitempty"`
	PhotoID     string    `json:"photo_id,omitempty"`
	WorkSession string    `json:"work_session,omitempty"`
	WorkError   string    `json:"work_error,omitempty"`
	GiftID      string    `json:"gift_id,omitempty"`
	Delta       int       `json:"delta,omitempty"`
}

type Conversation struct {
	ID        string    `json:"id"`
	IdolID    string    `json:"idol_id"`
	Title     string    `json:"title,omitempty"`
	Current   bool      `json:"current"`
	Ended     bool      `json:"ended"`
	EndReason string    `json:"end_reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	OutOfLine int       `json:"out_of_line,omitempty"`
	NegStreak int       `json:"neg_streak,omitempty"`
	Records   []Record  `json:"records,omitempty"`
}

type Memory struct {
	ID           string    `json:"id"`
	Text         string    `json:"text"`
	SourceIdol   string    `json:"source_idol"`
	CircleID     string    `json:"circle_id,omitempty"`
	SupersededBy string    `json:"superseded_by,omitempty"`
	At           time.Time `json:"at"`
}

type Affinity struct {
	IdolID        string    `json:"idol_id"`
	Score         int       `json:"score"`
	Tier          string    `json:"tier"`
	CooldownUntil time.Time `json:"cooldown_until,omitempty"`
}

type Stock struct {
	GiftID string `json:"gift_id"`
	Count  int    `json:"count"`
}

type Lore struct {
	ID     string `json:"id"`
	IdolID string `json:"idol_id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type CallLog struct {
	IdolID    string    `json:"idol_id"`
	At        time.Time `json:"at"`
	Seconds   int       `json:"seconds"`
	Connected bool      `json:"connected"`
}

type DayMark struct {
	Day          string `json:"day"`
	Talk         bool   `json:"talk"`
	Call         bool   `json:"call"`
	Gift         bool   `json:"gift"`
	TalkRewarded bool   `json:"talk_rewarded"`
	CallRewarded bool   `json:"call_rewarded"`
	GiftRewarded bool   `json:"gift_rewarded"`
}

type ReachMark struct {
	IdolID   string    `json:"idol_id"`
	At       time.Time `json:"at"`
	Answered bool      `json:"answered"`
}

type State struct {
	Services      Services       `json:"services"`
	User          UserCard       `json:"user"`
	Idols         []Idol         `json:"idols"`
	Circles       []Circle       `json:"circles"`
	Conversations []Conversation `json:"conversations"`
	Memories      []Memory       `json:"memories"`
	Affinity      []Affinity     `json:"affinity"`
	Inventory     []Stock        `json:"inventory"`
	Lore          []Lore         `json:"lore"`
	Calls         []CallLog      `json:"calls"`
	Reaches       []ReachMark    `json:"reaches"`
	Achievements  []string       `json:"achievements"`
	Draws         int            `json:"draws"`
	BonusDraws    int            `json:"bonus_draws"`
	DrawDay       string         `json:"draw_day"`
	ResetCards    int            `json:"reset_cards"`
	Tasks         DayMark        `json:"tasks"`
	PresetRev     int            `json:"preset_rev,omitempty"`
}
