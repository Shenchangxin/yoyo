// mediaref.go 定义工作室、短剧、画布和上传共用的媒体字节票。

package video

// MediaRef 指向内容寻址存储里的一份媒体，并带上种类、尺寸、时长和来源。
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

// Valid 只接受 image、video、audio、script、subtitle、storyboard，且 CASHash 非空。
func (m MediaRef) Valid() bool {
	switch m.Kind {
	case "image", "video", "audio", "script", "subtitle", "storyboard":
		return m.CASHash != ""
	default:
		return false
	}
}
