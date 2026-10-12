package video

// MediaRef is the shared byte ticket across Studio, Drama, Canvas, and uploads.
type MediaRef struct {
	Kind       string `json:"kind"`
	CASHash    string `json:"cas_hash"`
	MIME       string `json:"mime,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	DurationMs int    `json:"duration_ms,omitempty"`
	PosterHash string `json:"poster_hash,omitempty"`
	Source     string `json:"source,omitempty"`
	OriginID   string `json:"origin_id,omitempty"`
}

func (m MediaRef) Valid() bool {
	switch m.Kind {
	case "image", "video", "audio", "script", "subtitle", "storyboard":
		return m.CASHash != ""
	default:
		return false
	}
}
