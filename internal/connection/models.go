package connection

import (
	"encoding/json"
	"strings"
)

var videoModelNeedles = []string{
	"seedance", "sora", "veo", "kling", "hailuo", "pika", "runway",
	"gen-3", "gen3", "hunyuan-video", "hunyuanvideo", "cogvideo", "mochi",
	"latte", "stable-video", "animatediff", "ltx-video", "ltxvideo",
	"minimax-video", "abab-video", "wanx", "wan2", "wan3", "wan-video",
}

var imageModelNeedles = []string{
	"seedream", "gpt-image", "dall-e", "dalle", "imagen", "flux", "sdxl",
	"stable-diffusion", "midjourney", "nano-banana", "nanobanana", "ideogram",
	"recraft", "playground", "leonardo",
}

var speechModelNeedles = []string{
	"tts", "speech", "voice", "audio", "music", "sound",
}

// CapOfModel infers a generation capability from a model id.
func CapOfModel(model string) string {
	s := strings.ToLower(strings.TrimSpace(model))
	if s == "" {
		return CapChat
	}
	if isSpeechModelName(s) {
		return CapSpeech
	}
	if isVideoModelName(s) {
		return CapVideo
	}
	if isImageModelName(s) {
		return CapImage
	}
	return CapChat
}

func isVideoModelName(s string) bool {
	if strings.Contains(s, "qwen") && !strings.Contains(s, "video") {
		return false
	}
	if strings.Contains(s, "video") || strings.Contains(s, "svd") {
		return true
	}
	for _, n := range videoModelNeedles {
		if strings.Contains(s, n) {
			return true
		}
	}
	if strings.Contains(s, "wan") && !strings.Contains(s, "qwen") {
		return true
	}
	return false
}

func isImageModelName(s string) bool {
	if isVideoModelName(s) || isSpeechModelName(s) {
		return false
	}
	if strings.Contains(s, "image") {
		return true
	}
	for _, n := range imageModelNeedles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func isSpeechModelName(s string) bool {
	for _, n := range speechModelNeedles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func generationCap(cap string) bool {
	switch NormalizeCap(cap) {
	case CapChat, CapImage, CapVideo, CapSpeech:
		return true
	default:
		return false
	}
}

// CapsOf is declared capabilities plus those inferred from model names.
func CapsOf(c Connection) []string {
	seen := map[string]bool{}
	var out []string
	add := func(cap string) {
		cap = NormalizeCap(cap)
		if cap == "" || cap == CapProtocol || seen[cap] {
			return
		}
		seen[cap] = true
		out = append(out, cap)
	}
	declared := declaredGenerationCaps(c)
	for _, x := range c.Capabilities {
		add(x)
	}
	for _, m := range c.Models {
		got := CapOfModel(m)
		if got == CapChat && !HasCap(c, CapChat) && len(declared) > 0 {
			continue
		}
		add(got)
	}
	for cap, model := range c.DefaultModel {
		add(cap)
		add(CapOfModel(model))
	}
	if out == nil {
		out = []string{}
	}
	return out
}

func MatchesCap(c Connection, cap string) bool {
	want := NormalizeCap(cap)
	if want == "" {
		return true
	}
	for _, x := range CapsOf(c) {
		if x == want {
			return true
		}
	}
	return false
}

func declaredGenerationCaps(c Connection) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range c.Capabilities {
		x = NormalizeCap(x)
		if !generationCap(x) || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	return out
}

// ModelsFor returns the models on c that belong to cap.
func ModelsFor(c Connection, cap string) []string {
	cap = NormalizeCap(cap)
	if cap == "" {
		return compact(append([]string{ModelFor(c, "")}, c.Models...))
	}
	only := declaredGenerationCaps(c)
	single := len(only) == 1 && only[0] == cap
	var matched, fallback []string
	for _, m := range c.Models {
		got := CapOfModel(m)
		if got == cap {
			if cap == CapChat && !HasCap(c, CapChat) && len(only) > 0 {
				continue
			}
			matched = append(matched, m)
			continue
		}
		if got == CapChat && (cap == CapChat || (single && !HasCap(c, CapChat))) {
			fallback = append(fallback, m)
		}
	}
	out := matched
	if len(out) == 0 || (single && cap != CapChat) {
		out = append(out, fallback...)
	}
	if c.DefaultModel != nil {
		if d := strings.TrimSpace(c.DefaultModel[cap]); d != "" {
			out = append([]string{d}, out...)
		}
	}
	return compact(out)
}

func ModelsExcept(c Connection, cap string) []string {
	cap = NormalizeCap(cap)
	take := map[string]bool{}
	for _, m := range ModelsFor(c, cap) {
		take[m] = true
	}
	var out []string
	for _, m := range c.Models {
		if !take[m] {
			out = append(out, m)
		}
	}
	return compact(out)
}

func ModelForCap(c Connection, cap string) string {
	cap = NormalizeCap(cap)
	if c.DefaultModel != nil {
		if v := strings.TrimSpace(c.DefaultModel[cap]); v != "" {
			return v
		}
	}
	if ms := ModelsFor(c, cap); len(ms) > 0 {
		return ms[0]
	}
	return ""
}

func ModelsJSONFor(c Connection, cap string) string {
	ms := ModelsFor(c, cap)
	if len(ms) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ms)
	return string(b)
}
