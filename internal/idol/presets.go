package idol

import "strings"

// ServiceVoice is one timbre a preset can speak with. Models, when set, limits
// the voice to those model ids; empty means every model of the preset.
type ServiceVoice struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Models []string `json:"models,omitempty"`
}

// ServicePreset is an OpenAI-compatible endpoint the idol desk can call
// without the operator pasting a URL. Protocols this client cannot speak
// (vendor-specific ASR, Seedream, and similar) are left out on purpose.
type ServicePreset struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Kind     string         `json:"kind"` // speech, asr, image
	Vendor   string         `json:"vendor"`
	Endpoint string         `json:"endpoint"`
	Models   []string       `json:"models"`
	Voices   []ServiceVoice `json:"voices,omitempty"`
}

func voice(id, label string, models ...string) ServiceVoice {
	return ServiceVoice{ID: id, Label: label, Models: models}
}

func buildPresets() []ServicePreset {
	mini := []string{"gpt-4o-mini-tts"}
	cosy := []string{"cosyvoice-v2"}
	return []ServicePreset{
		{
			ID: "openai-speech", Name: "OpenAI", Kind: "speech", Vendor: "openai",
			Endpoint: "https://api.openai.com/v1",
			Models:   []string{"gpt-4o-mini-tts", "tts-1", "tts-1-hd"},
			Voices: []ServiceVoice{
				voice("alloy", "Alloy"),
				voice("echo", "Echo"),
				voice("fable", "Fable"),
				voice("onyx", "Onyx"),
				voice("nova", "Nova"),
				voice("shimmer", "Shimmer"),
				voice("ash", "Ash", mini...),
				voice("ballad", "Ballad", mini...),
				voice("coral", "Coral", mini...),
				voice("sage", "Sage", mini...),
				voice("verse", "Verse", mini...),
			},
		},
		{
			ID: "minimax-speech", Name: "MiniMax", Kind: "speech", Vendor: "minimax",
			Endpoint: "https://api.minimax.io/v1",
			Models:   []string{"speech-2.6-hd", "speech-2.6-turbo"},
			Voices: []ServiceVoice{
				voice("male-qn-qingse", "青涩青年"),
				voice("male-qn-jingying", "精英青年"),
				voice("male-qn-badao", "霸道青年"),
				voice("male-qn-daxuesheng", "青年大学生"),
				voice("female-shaonv", "少女"),
				voice("female-yujie", "御姐"),
				voice("female-chengshu", "成熟女性"),
				voice("female-tianmei", "甜美女性"),
				voice("presenter_male", "男主持人"),
				voice("presenter_female", "女主持人"),
				voice("audiobook_male_1", "男声有声书"),
				voice("audiobook_female_1", "女声有声书"),
			},
		},
		{
			ID: "qwen-speech", Name: "通义千问 Qwen", Kind: "speech", Vendor: "aliyun",
			Endpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
			Models:   []string{"qwen3-tts-flash", "cosyvoice-v2"},
			Voices: []ServiceVoice{
				voice("Cherry", "Cherry · 温柔", "qwen3-tts-flash"),
				voice("Serena", "Serena · 活泼", "qwen3-tts-flash"),
				voice("Ethan", "Ethan · 沉稳", "qwen3-tts-flash"),
				voice("Chelsie", "Chelsie · 二次元", "qwen3-tts-flash"),
				voice("longxiaochun_v2", "龙小淳", cosy...),
				voice("longxiaoxia_v2", "龙小夏", cosy...),
				voice("longyue_v2", "龙悦", cosy...),
				voice("longwan_v2", "龙婉", cosy...),
				voice("longcheng_v2", "龙橙", cosy...),
				voice("longhua_v2", "龙华", cosy...),
				voice("longshu_v2", "龙书", cosy...),
			},
		},
		{
			ID: "openai-asr", Name: "OpenAI", Kind: "asr", Vendor: "openai",
			Endpoint: "https://api.openai.com/v1",
			Models:   []string{"gpt-4o-mini-transcribe", "gpt-4o-transcribe", "whisper-1"},
		},
		{
			ID: "groq-asr", Name: "Groq", Kind: "asr", Vendor: "groq",
			Endpoint: "https://api.groq.com/openai/v1",
			Models:   []string{"whisper-large-v3-turbo", "whisper-large-v3"},
		},
		{
			ID: "qwen-asr", Name: "通义千问 Qwen", Kind: "asr", Vendor: "aliyun",
			Endpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
			Models:   []string{"qwen3-asr-flash"},
		},
		{
			ID: "openai-image", Name: "OpenAI", Kind: "image", Vendor: "openai",
			Endpoint: "https://api.openai.com/v1",
			Models:   []string{"gpt-image-2", "gpt-image-1", "dall-e-3"},
		},
		{
			ID: "qwen-image", Name: "通义千问 Qwen", Kind: "image", Vendor: "aliyun",
			Endpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1",
			Models:   []string{"qwen-image-plus", "qwen-image"},
		},
		{
			ID: "zhipu-image", Name: "智谱 CogView", Kind: "image", Vendor: "zhipu",
			Endpoint: "https://open.bigmodel.cn/api/paas/v4",
			Models:   []string{"cogview-4", "cogview-3-flash"},
		},
	}
}

var servicePresets = buildPresets()

// ServicePresets returns the built-in provider catalog. The slice is a copy;
// the voice and model lists are shared and must not be mutated.
func ServicePresets() []ServicePreset {
	out := make([]ServicePreset, len(servicePresets))
	copy(out, servicePresets)
	return out
}

// Preset returns one catalog entry by id.
func Preset(id string) (ServicePreset, bool) {
	id = strings.TrimSpace(id)
	for _, p := range servicePresets {
		if p.ID == id {
			return p, true
		}
	}
	return ServicePreset{}, false
}

// VoicesFor lists timbres that belong to model. An empty model returns every
// voice of the preset.
func VoicesFor(p ServicePreset, model string) []ServiceVoice {
	model = strings.TrimSpace(model)
	out := make([]ServiceVoice, 0, len(p.Voices))
	for _, v := range p.Voices {
		if len(v.Models) == 0 || model == "" || contains(v.Models, model) {
			out = append(out, v)
		}
	}
	return out
}

// MatchPreset finds a catalog id for an existing generation connection.
// Endpoint match wins; vendor is the fallback inside the same kind.
func MatchPreset(kind, vendor, endpoint string) string {
	kind = strings.TrimSpace(kind)
	endpoint = strings.TrimSpace(endpoint)
	vendor = strings.TrimSpace(vendor)
	if endpoint != "" {
		for _, p := range servicePresets {
			if p.Kind == kind && SameEndpoint(p.Endpoint, endpoint) {
				return p.ID
			}
		}
	}
	if vendor != "" {
		for _, p := range servicePresets {
			if p.Kind == kind && strings.EqualFold(p.Vendor, vendor) {
				return p.ID
			}
		}
	}
	return ""
}

// InferProviders fills an empty provider id when the saved endpoint is one of
// the presets and the saved model and voice already belong to it. A private
// model or a cloned voice on a known host stays a custom endpoint.
func InferProviders(svc Services) Services {
	svc.SpeechProvider = inferOne(svc.SpeechProvider, svc.SpeechBaseURL, svc.SpeechModel, svc.SpeechVoice, "speech")
	svc.ASRProvider = inferOne(svc.ASRProvider, svc.ASRBaseURL, svc.ASRModel, "", "asr")
	svc.ImageProvider = inferOne(svc.ImageProvider, svc.ImageBaseURL, svc.ImageModel, "", "image")
	return svc
}

func inferOne(id, base, model, voiceID, kind string) string {
	id = strings.TrimSpace(id)
	if p, ok := Preset(id); ok && p.Kind == kind {
		return p.ID
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	model = strings.TrimSpace(model)
	voiceID = strings.TrimSpace(voiceID)
	for _, p := range servicePresets {
		if p.Kind != kind || !SameEndpoint(p.Endpoint, base) {
			continue
		}
		if model != "" && !contains(p.Models, model) {
			return ""
		}
		if voiceID != "" && len(p.Voices) > 0 && !voiceKnown(p, model, voiceID) {
			return ""
		}
		return p.ID
	}
	return ""
}

func voiceKnown(p ServicePreset, model, id string) bool {
	for _, v := range VoicesFor(p, model) {
		if v.ID == id {
			return true
		}
	}
	return false
}

// ApplyServicePresets copies the catalog endpoint onto a named provider and
// keeps the model and voice inside that provider's lists. An unknown provider
// id is cleared; a custom endpoint is left alone.
func ApplyServicePresets(svc Services) Services {
	svc.ChatBaseURL = strings.TrimSpace(svc.ChatBaseURL)
	svc.ChatModel = strings.TrimSpace(svc.ChatModel)
	svc.JudgeModel = strings.TrimSpace(svc.JudgeModel)
	applyKind(&svc.SpeechProvider, &svc.SpeechBaseURL, &svc.SpeechModel, &svc.SpeechVoice, "speech")
	applyKind(&svc.ASRProvider, &svc.ASRBaseURL, &svc.ASRModel, nil, "asr")
	applyKind(&svc.ImageProvider, &svc.ImageBaseURL, &svc.ImageModel, nil, "image")
	return svc
}

// NormalizeServices infers a preset from a known endpoint, then clamps the
// model and voice to that preset.
func NormalizeServices(svc Services) Services {
	return ApplyServicePresets(InferProviders(svc))
}

func applyKind(id, base, model, voiceID *string, kind string) {
	p, ok := Preset(*id)
	if !ok || p.Kind != kind {
		if strings.TrimSpace(*id) != "" {
			*id = ""
		}
		*base = strings.TrimSpace(*base)
		*model = strings.TrimSpace(*model)
		if voiceID != nil {
			*voiceID = strings.TrimSpace(*voiceID)
		}
		return
	}
	*id = p.ID
	*base = p.Endpoint
	*model = pickListed(*model, p.Models)
	if voiceID != nil {
		*voiceID = pickVoice(*voiceID, *model, p)
	}
}

func pickListed(current string, options []string) string {
	current = strings.TrimSpace(current)
	if contains(options, current) {
		return current
	}
	if len(options) == 0 {
		return current
	}
	return options[0]
}

func pickVoice(current, model string, p ServicePreset) string {
	voices := VoicesFor(p, model)
	current = strings.TrimSpace(current)
	for _, v := range voices {
		if v.ID == current {
			return current
		}
	}
	if len(voices) == 0 {
		return current
	}
	return voices[0].ID
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// SameEndpoint reports whether two API roots name the same host and path.
// A trailing /v1 is ignored so https://api.openai.com matches the preset that
// stores https://api.openai.com/v1. Other versions, such as /v4, stay put.
func SameEndpoint(a, b string) bool {
	ka, kb := endpointKey(a), endpointKey(b)
	return ka != "" && ka == kb
}

func endpointKey(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimRight(s, "/")
	if strings.HasSuffix(s, "/v1") {
		s = strings.TrimSuffix(s, "/v1")
	}
	return s
}

// MediaURL joins an API root with an OpenAI-style path. Roots that already
// carry a version (…/v1 or …/v4) are not given a second /v1.
func MediaURL(endpoint, suffix string) string {
	u := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	suffix = strings.Trim(strings.TrimSpace(suffix), "/")
	if u == "" || suffix == "" {
		return u
	}
	if strings.HasSuffix(u, "/"+suffix) {
		return u
	}
	if hasAPIVersion(u) {
		return u + "/" + suffix
	}
	return u + "/v1/" + suffix
}

func hasAPIVersion(u string) bool {
	for _, marker := range []string{"/v1", "/v2", "/v3", "/v4", "/v5"} {
		if strings.HasSuffix(u, marker) || strings.Contains(u, marker+"/") {
			return true
		}
	}
	return false
}
