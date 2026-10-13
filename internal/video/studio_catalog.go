package video

import "strings"

type StudioCharacter struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Hint        string `json:"hint"`
	Preserve    string `json:"preserve"`
	Prohibit    string `json:"prohibit"`
	Tags        []string `json:"tags,omitempty"`
}

type StudioStyle struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Hint     string `json:"hint"`
	Contract string `json:"contract"`
	Forbid   string `json:"forbid"`
}

type StudioPreset struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Hint     string `json:"hint"`
	Aspect   string `json:"aspect"`
	Platform string `json:"platform"`
	Duration int    `json:"duration"`
}

type StudioTemplate struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Hint        string   `json:"hint"`
	PresetID    string   `json:"preset_id"`
	Style       string   `json:"style"`
	CharacterID string   `json:"character_id"`
	Platform    string   `json:"platform"`
	Tone        string   `json:"tone"`
	Tags        []string `json:"tags,omitempty"`
}

func StudioCharacters() []StudioCharacter {
	return []StudioCharacter{
		{ID: "presenter", Name: "口播主持", Hint: "干净出镜、对镜头说话", Preserve: "one adult presenter facing camera, clear face, simple clothing, even lighting", Prohibit: "crowd, extra faces, heavy makeup, costume, text overlay", Tags: []string{"ugc", "talk"}},
		{ID: "student", Name: "学生", Hint: "校园/学习向", Preserve: "young student, school-casual clothes, simple background", Prohibit: "uniform fetish, extra people, brand logos", Tags: []string{"edu"}},
		{ID: "maker", Name: "创作者", Hint: "桌面/手作特写", Preserve: "hands and desk, tools in frame, practical lighting", Prohibit: "unrelated faces, cluttered ads", Tags: []string{"howto"}},
		{ID: "reviewer", Name: "测评", Hint: "产品特写+人", Preserve: "one reviewer with a single product hero, clean table", Prohibit: "fake luxury set, extra products", Tags: []string{"product"}},
		{ID: "narrator", Name: "画外音", Hint: "不出镜，只出画面", Preserve: "no presenter face, scenes and objects carry the story", Prohibit: "talking head, lip sync", Tags: []string{"explainer"}},
		{ID: "cast", Name: "双人对话", Hint: "两个人物保持外形", Preserve: "exactly two characters, consistent hair and clothes across shots", Prohibit: "extra people, costume changes", Tags: []string{"story"}},
	}
}

func StudioStyles() []StudioStyle {
	return []StudioStyle{
		{ID: "clean-ugc", Name: "干净 UGC", Hint: "手机纪实", Contract: "natural phone video still, daylight, slight grain, 9:16 framing", Forbid: "cinematic anamorphic, heavy CGI, watermarks"},
		{ID: "paper", Name: "纸上讲解", Hint: "静帧+运镜讲解", Contract: "flat illustration, limited palette, clear silhouettes, readable at 9:16", Forbid: "photoreal faces, busy textures"},
		{ID: "cinematic", Name: "电影感", Hint: "浅景深短片", Contract: "cinematic still, shallow depth, motivated light, 16:9 or 9:16", Forbid: "stock-photo smiles, neon UI"},
		{ID: "anime", Name: "动漫", Hint: "稳定角色线", Contract: "consistent anime character, clean line, limited shading", Forbid: "photoreal skin, extra fingers"},
		{ID: "3d", Name: "三维", Hint: "体积光产品/角色", Contract: "stylized 3d render, soft studio light, consistent materials", Forbid: "live-action plates"},
	}
}

func StudioPresets() []StudioPreset {
	return []StudioPreset{
		{ID: "douyin-shorts", Name: "抖音竖屏", Hint: "9:16 · 约 60 秒", Aspect: "9:16", Platform: "douyin", Duration: 60},
		{ID: "xhs", Name: "小红书竖屏", Hint: "9:16 · 约 60 秒", Aspect: "9:16", Platform: "xhs", Duration: 60},
		{ID: "channels", Name: "视频号", Hint: "9:16 · 约 60 秒", Aspect: "9:16", Platform: "channels", Duration: 60},
		{ID: "bilibili", Name: "B 站横屏", Hint: "16:9 · 约 3 分钟", Aspect: "16:9", Platform: "bilibili", Duration: 180},
		{ID: "landscape", Name: "横屏讲解", Hint: "16:9 · 约 60 秒", Aspect: "16:9", Platform: "generic", Duration: 60},
		{ID: "square", Name: "方形", Hint: "1:1 · 约 30 秒", Aspect: "1:1", Platform: "xhs", Duration: 30},
	}
}

func StudioTemplates() []StudioTemplate {
	return []StudioTemplate{
		{ID: "explainer", Title: "知识讲解", Hint: "钩子→三点→收束", PresetID: "douyin-shorts", Style: "paper", CharacterID: "narrator", Platform: "douyin", Tone: "professional", Tags: []string{"explainer", "tutorial"}},
		{ID: "ugc-hook", Title: "UGC 钩子", Hint: "前 3 秒画面", PresetID: "douyin-shorts", Style: "clean-ugc", CharacterID: "presenter", Platform: "douyin", Tone: "energetic", Tags: []string{"ugc"}},
		{ID: "product", Title: "产品特写", Hint: "痛点→演示→CTA", PresetID: "xhs", Style: "3d", CharacterID: "reviewer", Platform: "xhs", Tone: "natural", Tags: []string{"product", "ad"}},
		{ID: "news-take", Title: "热点表态", Hint: "热榜一事一观点", PresetID: "douyin-shorts", Style: "clean-ugc", CharacterID: "presenter", Platform: "douyin", Tone: "storytelling", Tags: []string{"news"}},
		{ID: "story", Title: "微故事", Hint: "人物连续外形", PresetID: "landscape", Style: "cinematic", CharacterID: "cast", Platform: "bilibili", Tone: "storytelling", Tags: []string{"cinematic", "story"}},
		{ID: "howto", Title: "三步教程", Hint: "桌面手作", PresetID: "xhs", Style: "clean-ugc", CharacterID: "maker", Platform: "xhs", Tone: "professional", Tags: []string{"tutorial"}},
	}
}

func LookupStudioCharacter(id string) (StudioCharacter, bool) {
	id = strings.TrimSpace(id)
	for _, c := range StudioCharacters() {
		if c.ID == id {
			return c, true
		}
	}
	return StudioCharacter{}, false
}

func LookupStudioStyle(id string) (StudioStyle, bool) {
	id = strings.TrimSpace(id)
	for _, s := range StudioStyles() {
		if s.ID == id {
			return s, true
		}
	}
	return StudioStyle{}, false
}

func LookupStudioPreset(id string) (StudioPreset, bool) {
	id = strings.TrimSpace(id)
	for _, p := range StudioPresets() {
		if p.ID == id {
			return p, true
		}
	}
	return StudioPreset{}, false
}

func LookupStudioTemplate(id string) (StudioTemplate, bool) {
	id = strings.TrimSpace(id)
	for _, t := range StudioTemplates() {
		if t.ID == id {
			return t, true
		}
	}
	return StudioTemplate{}, false
}

func DefaultStudioIdentity() StudioIdentity {
	c, _ := LookupStudioCharacter("presenter")
	s, _ := LookupStudioStyle("clean-ugc")
	return StudioIdentity{
		CharacterID: c.ID, StyleID: s.ID, Name: c.Name,
		Preserve: c.Preserve + "; " + s.Contract,
		Prohibit: c.Prohibit + "; " + s.Forbid,
	}
}

func presetIDForAspect(aspect string) string {
	switch normalizeAspect(aspect) {
	case "16:9":
		return "landscape"
	case "1:1":
		return "square"
	default:
		return "douyin-shorts"
	}
}

func StudioCatalog() map[string]any {
	return map[string]any{
		"characters": StudioCharacters(),
		"styles":     StudioStyles(),
		"presets":    StudioPresets(),
		"templates":  StudioTemplates(),
		"voices":     StudioVoices(),
		"platforms":  []string{"douyin", "xhs", "channels", "bilibili", "kuaishou", "zhihu", "generic"},
		"tones":      []string{"natural", "professional", "energetic", "storytelling"},
	}
}

func StudioVoices() []map[string]string {
	return []map[string]string{
		{"id": "alloy", "name": "Alloy"},
		{"id": "echo", "name": "Echo"},
		{"id": "fable", "name": "Fable"},
		{"id": "onyx", "name": "Onyx"},
		{"id": "nova", "name": "Nova"},
		{"id": "shimmer", "name": "Shimmer"},
	}
}
