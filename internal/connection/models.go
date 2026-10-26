// 从模型 id 的子串推断聊天、图像、视频、语音、音乐、抠图和语音识别能力。

package connection

import (
	"encoding/json"
	"strings"
)

// videoModelNeedles 是视频模型 id 里要匹配的子串。
var videoModelNeedles = []string{
	"seedance", "sora", "veo", "kling", "hailuo", "pika", "runway",
	"gen-3", "gen3", "hunyuan-video", "hunyuanvideo", "cogvideo", "mochi",
	"latte", "stable-video", "animatediff", "ltx-video", "ltxvideo",
	"minimax-video", "abab-video", "wanx", "wan2", "wan3", "wan-video",
	"jimeng_v", "vgfm", "vidu",
}

// imageModelNeedles 是图像模型 id 里要匹配的子串。
var imageModelNeedles = []string{
	"seedream", "gpt-image", "dall-e", "dalle", "imagen", "flux", "sdxl",
	"stable-diffusion", "midjourney", "nano-banana", "nanobanana", "ideogram",
	"recraft", "playground", "leonardo",
	"qwen-image", "jimeng_t2i", "kolors", "cogview", "hunyuan-image",
}

// speechModelNeedles 是语音合成模型 id 里要匹配的子串。
var speechModelNeedles = []string{
	"tts", "speech", "voice", "seed-tts", "cosyvoice",
}

// musicModelNeedles 是音乐模型 id 里要匹配的子串。
var musicModelNeedles = []string{
	"lyria", "stable-audio", "suno", "udio", "chirp-v", "chirp-auk",
	"eleven_music", "text_to_sound", "music-1", "music-2", "musicgen", "riffusion",
	"mureka", "skymusic", "music-3",
}

// mattingModelNeedles 是抠图模型 id 里要匹配的子串。
var mattingModelNeedles = []string{
	"saliency", "removebg", "matting", "matte",
}

// asrModelNeedles 是语音识别模型 id 里要匹配的子串。
var asrModelNeedles = []string{
	"whisper", "transcribe", "stt", "paraformer", "sensevoice", "nova-2", "nova-3",
	"seed-asr", "spark-asr",
}

// CapOfModel 按模型名子串返回一种生成能力。空名字返回 CapChat。识别顺序是语音识别、抠图、音乐、语音、视频、图像，其余是聊天。
// model 去掉空白并转小写。空则返回 CapChat。否则按语音识别、抠图、音乐、语音、视频、图像的顺序判断，都不是则返回 CapChat。
func CapOfModel(model string) string {
	s := strings.ToLower(strings.TrimSpace(model))
	if s == "" {
		return CapChat
	}
	if isAsrModelName(s) {
		return CapASR
	}
	if isMattingModelName(s) {
		return CapMatting
	}
	if isMusicModelName(s) {
		return CapMusic
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

// isVideoModelName 在名字含 video、svd、视频针或单独的 wan 时返回真。含 qwen 且不含 video 时返回假。
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

// isImageModelName 在已被判成视频、语音、音乐、识别或抠图时返回假。名字含 image 或图像针时返回真。
func isImageModelName(s string) bool {
	if isVideoModelName(s) || isSpeechModelName(s) || isMusicModelName(s) || isAsrModelName(s) || isMattingModelName(s) {
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

// isSpeechModelName 在名字含语音针时返回真。
func isSpeechModelName(s string) bool {
	for _, n := range speechModelNeedles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// isMattingModelName 在名字含抠图针时返回真。
func isMattingModelName(s string) bool {
	for _, n := range mattingModelNeedles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// isMusicModelName 在名字含音乐针时返回真。
func isMusicModelName(s string) bool {
	for _, n := range musicModelNeedles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// isAsrModelName 在名字含语音识别针时返回真。
func isAsrModelName(s string) bool {
	for _, n := range asrModelNeedles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// generationCap 报告该能力是不是聊天、图像、视频、语音、音乐、抠图、素材或语音识别。
// cap 传给 NormalizeCap。结果是 CapChat、CapImage、CapVideo、CapSpeech、CapMusic、CapMatting、CapStock 或 CapASR 时返回真。
func generationCap(cap string) bool {
	switch NormalizeCap(cap) {
	case CapChat, CapImage, CapVideo, CapSpeech, CapMusic, CapMatting, CapStock, CapASR:
		return true
	default:
		return false
	}
}

// CapsOf 合并声明的能力，以及从模型名和 DefaultModel 推断出的能力。已声明生成能力时，不把仅被当成聊天的模型名再加成聊天。
// c 的 Capabilities、Models 和 DefaultModel 用来收集能力。Models 里的名字传给 CapOfModel。
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

// MatchesCap 在 cap 为空时返回真，否则看 CapsOf 里有没有它。
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

// declaredGenerationCaps 返回连接声明过的生成能力，去掉重复。
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

// ModelsFor 返回属于该能力的模型名。cap 为空时返回默认模型和全部 Models。单一生成能力且不是聊天时，聊天名会当作回落。
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

// ModelsExcept 返回 Models 里不属于该能力的名字。
// cap 传给 NormalizeCap，再传给 ModelsFor，用来从 c.Models 里排除这些名字。
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

// ModelForCap 先取 DefaultModel 里的该能力，否则取 ModelsFor 的第一项。
// c 的 DefaultModel 按规范化后的 cap 取值，去掉空白后非空就返回。否则用 ModelsFor 的第一项。
// cap 传给 NormalizeCap，作为 DefaultModel 的键，并传给 ModelsFor。
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

// ModelsJSONFor 把 ModelsFor 的结果编成 JSON 数组。没有模型时返回 []。
// c 传给 ModelsFor。
// cap 传给 ModelsFor。得到的列表为空时返回 []。
func ModelsJSONFor(c Connection, cap string) string {
	ms := ModelsFor(c, cap)
	if len(ms) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ms)
	return string(b)
}
