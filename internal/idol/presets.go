// 本文件拥有偶像桌面可直接选用的语音、转写和图像供应商目录。端点、模型 id 和音色 id 是数据，不在注释里改写。
// 本客户端说不了的协议不在这份目录里。MatchPreset 和 ApplyServicePresets 只按目录收口，不调用模型。

package idol

import "strings"

// ServiceVoice 是一套预设里的一种音色。Models 非空时只属于那些模型 id；空表示该预设的每个模型都能用。
type ServiceVoice struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Models []string `json:"models,omitempty"`
}

// ServicePreset 是一个不用手填地址就能调用的 OpenAI 兼容端点。Kind 取 speech、asr、image。
type ServicePreset struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Kind     string         `json:"kind"` // speech、asr、image
	Vendor   string         `json:"vendor"`
	Endpoint string         `json:"endpoint"`
	Models   []string       `json:"models"`
	Voices   []ServiceVoice `json:"voices,omitempty"`
}

// voice 组装一条音色。后面的模型 id 写入 Models。
// label 写入 Label。
// models 写入 Models。
func voice(id, label string, models ...string) ServiceVoice {
	return ServiceVoice{ID: id, Label: label, Models: models}
}

// buildPresets 返回内置目录。下面的名字、端点和 id 是数据。
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

// servicePresets 是进程启动时建好的目录，ServicePresets 返回它的副本。
var servicePresets = buildPresets()

// ServicePresets 返回内置供应商目录的副本。音色和模型列表与目录共享，调用方不要改它们。
func ServicePresets() []ServicePreset {
	out := make([]ServicePreset, len(servicePresets))
	copy(out, servicePresets)
	return out
}

// Preset 按 id 返回一条目录。没有则 ok 为 false。
func Preset(id string) (ServicePreset, bool) {
	id = strings.TrimSpace(id)
	for _, p := range servicePresets {
		if p.ID == id {
			return p, true
		}
	}
	return ServicePreset{}, false
}

// VoicesFor 列出属于该模型的音色。model 为空时返回这套预设的全部音色。
// p 的 Voices 逐项检查。Models 长度为 0、model 为空，或 Models 含有 model 时留下。
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

// MatchPreset 为一条已有的生成连接找目录 id。同一 Kind 里端点对上优先，对不上再用供应商名。
// kind 去掉空白后与目录的 Kind 比较。
// vendor 去掉空白。非空时，在端点没对上的情况下与目录 Vendor 不区分大小写比较。
// endpoint 去掉空白。非空时先用 SameEndpoint 和目录端点比较。
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

// InferProviders 在供应商 id 为空、端点对上目录、且已存的模型和音色都属于该目录时，填上目录 id。已知主机上的私有模型或克隆音色保持为自定义端点，id 留空。
// svc 的 Speech、ASR、Image 三组 Provider、BaseURL、Model，以及 SpeechVoice，分别传给 inferOne 后写回。
func InferProviders(svc Services) Services {
	svc.SpeechProvider = inferOne(svc.SpeechProvider, svc.SpeechBaseURL, svc.SpeechModel, svc.SpeechVoice, "speech")
	svc.ASRProvider = inferOne(svc.ASRProvider, svc.ASRBaseURL, svc.ASRModel, "", "asr")
	svc.ImageProvider = inferOne(svc.ImageProvider, svc.ImageBaseURL, svc.ImageModel, "", "image")
	return svc
}

// inferOne 解析一个通道的供应商 id。id 已经是同种类的目录就用它。端点对上但模型或音色不在目录里时返回空字符串。
// base 去掉空白，为空时返回空字符串，否则和目录端点做 SameEndpoint。
// model 去掉空白。非空且不在目录 Models 里时返回空字符串。
// voiceID 去掉空白。非空、目录有 Voices，且 voiceKnown 为假时返回空字符串。
// kind 与 Preset 的 Kind、目录的 Kind 比较。
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

// voiceKnown 看这个音色 id 是否出现在该模型的 VoicesFor 结果里。
// p 传给 VoicesFor。
// model 传给 VoicesFor。
func voiceKnown(p ServicePreset, model, id string) bool {
	for _, v := range VoicesFor(p, model) {
		if v.ID == id {
			return true
		}
	}
	return false
}

// ApplyServicePresets 把已命名供应商的端点换成目录里的地址，并把模型和音色收进该供应商的列表。未知的供应商 id 清掉；没有目录 id 的自定义端点保持原样。
// svc 的 ChatBaseURL、ChatModel、JudgeModel 去掉空白。Speech、ASR、Image 的 Provider、BaseURL、Model，以及 SpeechVoice，传给 applyKind。
func ApplyServicePresets(svc Services) Services {
	svc.ChatBaseURL = strings.TrimSpace(svc.ChatBaseURL)
	svc.ChatModel = strings.TrimSpace(svc.ChatModel)
	svc.JudgeModel = strings.TrimSpace(svc.JudgeModel)
	applyKind(&svc.SpeechProvider, &svc.SpeechBaseURL, &svc.SpeechModel, &svc.SpeechVoice, "speech")
	applyKind(&svc.ASRProvider, &svc.ASRBaseURL, &svc.ASRModel, nil, "asr")
	applyKind(&svc.ImageProvider, &svc.ImageBaseURL, &svc.ImageModel, nil, "image")
	return svc
}

// NormalizeServices 先按已知端点推断预设，再把模型和音色收进该预设。
// svc 先传给 InferProviders，再传给 ApplyServicePresets。
func NormalizeServices(svc Services) Services {
	return ApplyServicePresets(InferProviders(svc))
}

// applyKind 按供应商 id 改写一个通道。id 不是该种类的目录时，非空 id 被清掉，端点、模型和音色只做 TrimSpace。对上目录时端点换成目录地址，模型用 pickListed，音色用 pickVoice。
// base 在对不上该种类时去掉空白，对上时改成目录的 Endpoint。
// model 在对不上时去掉空白，对上时传给 pickListed。
// voiceID 非 nil 时，对不上则去掉空白，对上则传给 pickVoice。
// kind 与目录 Kind 比较。Preset 失败或 Kind 不相等时走清空分支。
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

// pickListed 在 options 里能找到 current 就留下它，否则用第一项。options 为空时留下 current。
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

// pickVoice 在该模型的音色里能找到 current 就留下它，否则用第一项。没有音色时留下 current。
// model 和 p 传给 VoicesFor。
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

// contains 看 want 是否与 list 里某一项完全相同。
func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// SameEndpoint 判断两个 API 根是不是同一主机和路径。末尾的 /v1 忽略，所以 https://api.openai.com 对得上存成 https://api.openai.com/v1 的预设。/v4 这类其他版本留在路径里。
// b 交给 endpointKey。与 a 的键都非空且相等时返回 true。
func SameEndpoint(a, b string) bool {
	ka, kb := endpointKey(a), endpointKey(b)
	return ka != "" && ka == kb
}

// endpointKey 去掉协议、大小写和末尾斜杠，并剥掉结尾的 /v1，得到用来比较的键。空字符串表示对不上。
// raw 去掉空白并转成小写，去掉 https:// 或 http:// 前缀和末尾斜杠，再去掉结尾的 /v1。
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

// MediaURL 把 API 根和一段 OpenAI 风格路径接起来。根上已经有版本（…/v1 或 …/v4）时不再加第二个 /v1。后缀已经在根末尾时直接返回根。
// endpoint 去掉空白和末尾斜杠。它为空，或 suffix 清理后为空时，返回这个结果。
// suffix 去掉空白和两端斜杠。已出现在 endpoint 末尾时不再拼接。有版本时直接接上，否则先接 /v1/。
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

// hasAPIVersion 看地址里有没有 /v1 到 /v5，在末尾或后面还跟着路径。
// u 的末尾是 /v1、/v2、/v3、/v4 或 /v5，或包含这些标记再加斜杠时返回 true。
func hasAPIVersion(u string) bool {
	for _, marker := range []string{"/v1", "/v2", "/v3", "/v4", "/v5"} {
		if strings.HasSuffix(u, marker) || strings.Contains(u, marker+"/") {
			return true
		}
	}
	return false
}
