package cut

// Breakdown is scene/silence analysis of one source.
type Breakdown struct {
	Duration float64   `json:"duration,omitempty"`
	Scenes   []float64 `json:"scenes,omitempty"`
	Silences []Silence `json:"silences,omitempty"`
	Clips    []Clip    `json:"clips,omitempty"`
}

// EditList is the operator-chosen (or agent-proposed) cut list.
type EditList struct {
	Cuts []Clip `json:"cuts,omitempty"`
}

// Stems are CAS hashes for mix inputs.
type Stems struct {
	Original string `json:"original,omitempty"`
	Voice    string `json:"voice,omitempty"`
	Dub      string `json:"dub,omitempty"`
	Music    string `json:"music,omitempty"`
	SFX      string `json:"sfx,omitempty"`
}

// Captions is timed text plus stored SRT/VTT blobs.
type Captions struct {
	Language string `json:"language,omitempty"`
	Cues     []Cue  `json:"cues,omitempty"`
	SRTHash  string `json:"srt_hash,omitempty"`
	VTTHash  string `json:"vtt_hash,omitempty"`
}

type Silence struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type Cue struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
	Omit  bool    `json:"omit,omitempty"`
}

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

func (c Clip) Dur() float64 {
	d := c.End - c.Start
	if d < 0 {
		return 0
	}
	return d
}
