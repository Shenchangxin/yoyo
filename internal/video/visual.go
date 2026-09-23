package video

import "strings"

// NormalizeAspect keeps the four ratios the video adapters actually honor.
func NormalizeAspect(s string) string {
	switch strings.TrimSpace(s) {
	case "9:16", "16:9", "1:1", "adaptive":
		return s
	default:
		return "16:9"
	}
}

// ImageSizeFor is the still intent, not a vendor enum.
// Character sheets are landscape turnarounds regardless of episode frame.
// Scenes follow the episode aspect. Props are square product shots.
func ImageSizeFor(kind, aspect string) string {
	switch kind {
	case "prop":
		return "1024x1024"
	case "character":
		return "1920x1080"
	default:
		switch NormalizeAspect(aspect) {
		case "9:16":
			return "1080x1920"
		case "1:1":
			return "1024x1024"
		default:
			return "1920x1080"
		}
	}
}

// OpenAIImageSize maps an arbitrary WxH onto gpt-image allowed sizes.
func OpenAIImageSize(size string) string {
	w, h, ok := splitSize(size)
	if !ok {
		return "1024x1024"
	}
	if w == h {
		return "1024x1024"
	}
	if w > h {
		return "1536x1024"
	}
	return "1024x1536"
}

// GeminiAspectFromSize reads WxH, else falls back to the episode ratio.
func GeminiAspectFromSize(size, fallback string) string {
	w, h, ok := splitSize(size)
	if !ok {
		ar := NormalizeAspect(fallback)
		if ar == "adaptive" {
			return "16:9"
		}
		return ar
	}
	if w == h {
		return "1:1"
	}
	if w > h {
		return "16:9"
	}
	return "9:16"
}

// IsNarrator is a visual predicate: voice-only rows do not get a face still.
func IsNarrator(name, role string) bool {
	text := strings.ToLower(strings.TrimSpace(name + " " + role))
	if text == "" {
		return false
	}
	keys := []string{"旁白", "画外音", "narrator", "voice-over", "voiceover", "ナレーター", "ナレーション", "내레이션", "해설"}
	for _, k := range keys {
		if strings.Contains(text, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

func chineseContent(lang string) bool {
	s := strings.ToLower(strings.TrimSpace(lang))
	return s == "" || s == "zh" || s == "zh-cn" || s == "zh-hans"
}
