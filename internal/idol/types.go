// 本文件拥有偶像桌面落盘的数据结构：服务端点、用户卡片、偶像、对话、记忆、好感和库存。
// 不读写文件，也不调用模型。Store 在 store.go 把 State 写成 state.json。

package idol

import "time"

// Services 是偶像桌面要调用的模型服务。ChatModel 是会话文本模型，形如 connectionId::model，空则用当前聊天连接。
// 语音、转写和图像可以写内置供应商；端点为空时回落到生成连接。密钥在保险库里。ChatBaseURL 是旧的端点覆盖。
type Services struct {
	ChatBaseURL    string `json:"chat_base_url"`
	ChatModel      string `json:"chat_model"`
	JudgeModel     string `json:"judge_model"`
	SpeechProvider string `json:"speech_provider,omitempty"`
	SpeechBaseURL  string `json:"speech_base_url"`
	SpeechModel    string `json:"speech_model"`
	SpeechVoice    string `json:"speech_voice"`
	ASRProvider    string `json:"asr_provider,omitempty"`
	ASRBaseURL     string `json:"asr_base_url"`
	ASRModel       string `json:"asr_model"`
	ImageProvider  string `json:"image_provider,omitempty"`
	ImageBaseURL   string `json:"image_base_url"`
	ImageModel     string `json:"image_model"`
	ReachOut       bool   `json:"reach_out"`
	TextOnly       bool   `json:"text_only"`
	PhoneTexture   bool   `json:"phone_texture"`
	WorkEnabled    bool   `json:"work_enabled"`
	WorkProfileID  string `json:"work_profile_id"`
}

// UserCard 是每位偶像称呼这位操作者的方式。它不是工作区里的 harness 配置。
type UserCard struct {
	Address  string `json:"address"`
	Gender   string `json:"gender"`
	Timezone string `json:"timezone"`
	Birthday string `json:"birthday"` // 月-日，形如 MM-DD
}

// Outfit 是一套立绘。Blob 是头像或贴图在 blobs/ 里的文件名。
type Outfit struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Blob string `json:"blob"`
}

// Idol 是一位偶像。Kind 为 pet、catalog，或空表示用户自己建的。OutfitID 形如 stand/<编号>，立绘文件由 StandCache 另取。
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
	Kind          string    `json:"kind,omitempty"` // pet、catalog，或空表示用户自己建的
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Circle 是一组偶像。Members 存成员偶像的 id。
type Circle struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Members []string `json:"members,omitempty"`
}

// Segment 是一轮回复里的一句。Spoken 给语音，Subtitle 给字幕。
type Segment struct {
	Spoken   string `json:"spoken"`
	Subtitle string `json:"subtitle"`
}

// Record 是对话里的一条。Kind 取 turn、reach、event、work_said。
type Record struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"` // turn、reach、event、work_said
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

// Conversation 是一位偶像的一段对话。Current 为真表示当前这段；Ended 为真时 Turn 拒绝再写。
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

// Memory 是一句关于用户的记忆。SupersededBy 非空表示已被后来的一句盖过。同一 CircleID 的偶像可以看见。
type Memory struct {
	ID           string    `json:"id"`
	Text         string    `json:"text"`
	SourceIdol   string    `json:"source_idol"`
	CircleID     string    `json:"circle_id,omitempty"`
	SupersededBy string    `json:"superseded_by,omitempty"`
	At           time.Time `json:"at"`
}

// Affinity 是一位偶像对用户的好感。Score 由 clampScore 限制在 0 到 100，Tier 由 TierOf 填写。
type Affinity struct {
	IdolID        string    `json:"idol_id"`
	Score         int       `json:"score"`
	Tier          string    `json:"tier"`
	CooldownUntil time.Time `json:"cooldown_until,omitempty"`
}

// Stock 是背包里一种礼物的数量。
type Stock struct {
	GiftID string `json:"gift_id"`
	Count  int    `json:"count"`
}

// Lore 是挂在某位偶像上的材料，retrieve 按用户这句话挑一段放进提示。
type Lore struct {
	ID     string `json:"id"`
	IdolID string `json:"idol_id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// CallLog 是一次通话记录。Connected 为真才计入 Stats 的通话次数和秒数。
type CallLog struct {
	IdolID    string    `json:"idol_id"`
	At        time.Time `json:"at"`
	Seconds   int       `json:"seconds"`
	Connected bool      `json:"connected"`
}

// DayMark 是用户时区里某一天的三项每日任务：说话、通话、送礼，以及是否已经发过奖励抽卡。
type DayMark struct {
	Day          string `json:"day"`
	Talk         bool   `json:"talk"`
	Call         bool   `json:"call"`
	Gift         bool   `json:"gift"`
	TalkRewarded bool   `json:"talk_rewarded"`
	CallRewarded bool   `json:"call_rewarded"`
	GiftRewarded bool   `json:"gift_rewarded"`
}

// ReachMark 是偶像主动开口的一次。Answered 为假时，reachDue 不再为这位偶像安排下一次。
type ReachMark struct {
	IdolID   string    `json:"idol_id"`
	At       time.Time `json:"at"`
	Answered bool      `json:"answered"`
}

// State 是 state.json 的整份偶像桌面。PresetRev 达到 presetRevision 后，Open 不再重灌目录。
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
