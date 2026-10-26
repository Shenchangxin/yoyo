// types.go 描述一条素材的场景、静音、剪辑表、音轨和字幕。

package cut

// Breakdown 是一条素材的时长、场景切点、静音区间和片段。
type Breakdown struct {
	Duration float64   `json:"duration,omitempty"`
	Scenes   []float64 `json:"scenes,omitempty"`
	Silences []Silence `json:"silences,omitempty"`
	Clips    []Clip    `json:"clips,omitempty"`
}

// EditList 是操作者选定或代理提议的剪辑表。
type EditList struct {
	Cuts []Clip `json:"cuts,omitempty"`
}

// Stems 是混音输入在内容寻址存储里的哈希：原声、人声、配音、音乐和音效。
type Stems struct {
	Original string `json:"original,omitempty"`
	Voice    string `json:"voice,omitempty"`
	Dub      string `json:"dub,omitempty"`
	Music    string `json:"music,omitempty"`
	SFX      string `json:"sfx,omitempty"`
}

// Captions 是带时间的文本，以及已存的 SRT、VTT 哈希。
type Captions struct {
	Language string `json:"language,omitempty"`
	Cues     []Cue  `json:"cues,omitempty"`
	SRTHash  string `json:"srt_hash,omitempty"`
	VTTHash  string `json:"vtt_hash,omitempty"`
}

// Silence 是一段静音的起止秒数。
type Silence struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Cue 是一条字幕的起止秒数、文本，以及是否省略。
type Cue struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
	Omit  bool    `json:"omit,omitempty"`
}

// Clip 是一段成片，带标识、媒体哈希、起止、标题、理由和分数。
type Clip struct {
	ID     string         `json:"id,omitempty"`
	Hash   string         `json:"hash,omitempty"`
	Start  float64        `json:"start"`
	End    float64        `json:"end"`
	Title  string         `json:"title,omitempty"`
	Reason string         `json:"reason,omitempty"`
	Score  float64        `json:"score,omitempty"`
	Scores map[string]int `json:"scores,omitempty"`
}

// Dur 返回片段秒数。结束早于开始时返回 0。
func (c Clip) Dur() float64 {
	d := c.End - c.Start
	if d < 0 {
		return 0
	}
	return d
}
