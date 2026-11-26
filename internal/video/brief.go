// brief.go 拥有导演的制作简报：把一句话的意图归一成参数，对照设置里已启用且有钥匙的账号，算出用到什么、缺什么、本机能兜底什么，并给出花额度前的确认卡。
// 不拥有执行。确认卡由宿主交给 ask_user，花额度的工具仍由各工坊挂载。

package video

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connection"
)

// briefIntents 是简报认得的意图：剪辑、配音、翻拍、成片、出图、短片、短剧、画布。
var briefIntents = []string{"cut", "dub", "remake", "film", "still", "short", "drama", "canvas"}

// briefFormats 是成片的七种片型，和创作桌、hypit 技能一致。
var briefFormats = []string{"talking", "drama", "podcast", "interview", "ranking", "explainer", "narration"}

// briefNeeds 是每种意图会用到的生成能力。剪辑、翻拍和路由只用本机 ffmpeg 或打开工坊，不在这里。
var briefNeeds = map[string][]string{
	"dub":   {connection.CapSpeech},
	"film":  {connection.CapImage, connection.CapSpeech},
	"still": {connection.CapImage},
}

// briefLocal 是本机能兜底的能力：语音没有账号时走 edge-tts，不花额度。
var briefLocal = map[string]bool{connection.CapSpeech: true}

// briefSeconds 是没写时长或越界时各意图的默认秒数。不在表里的意图不带时长。
var briefSeconds = map[string]int{"cut": 30, "film": 30, "short": 45, "drama": 60}

// briefAspects 是简报接受的画幅。
var briefAspects = []string{"9:16", "16:9", "1:1"}

// Brief 是归一后的制作简报。Intent 必填；Format 只对成片有意义。SourceHash 只能是工具返回过的已入库哈希。
type Brief struct {
	Intent     string `json:"intent"`
	Format     string `json:"format,omitempty"`
	Subject    string `json:"subject,omitempty"`
	Seconds    int    `json:"seconds,omitempty"`
	Aspect     string `json:"aspect"`
	Voice      string `json:"voice,omitempty"`
	Style      string `json:"style,omitempty"`
	SourceHash string `json:"source_hash,omitempty"`
}

// BriefPlan 是 video_brief 的结果。Have 按能力列出可用账号名；Local 是本机兜底的能力；Missing 是既没有账号也不能兜底的能力。
// Spend 为真表示这份方案会用到付费账号，宿主要先出确认卡。Ready 为假时缺账号，不出确认卡。
type BriefPlan struct {
	Brief   Brief               `json:"brief"`
	Needs   []string            `json:"needs"`
	Have    map[string][]string `json:"have"`
	Local   []string            `json:"local"`
	Missing []string            `json:"missing"`
	Spend   bool                `json:"spend"`
	Ready   bool                `json:"ready"`
}

// NormalizeBrief 把参数收成 Brief。意图或片型对不上返回错误，不猜。时长空着或不在 5 到 180 秒之间时用意图的默认值。画幅对不上时用 9:16。
// args 读取 intent、format、subject、seconds、aspect、voice、style、source_hash。
func NormalizeBrief(args map[string]any) (Brief, error) {
	b := Brief{
		Intent:     strings.ToLower(strings.TrimSpace(strAnyMap(args, "intent"))),
		Format:     strings.ToLower(strings.TrimSpace(strAnyMap(args, "format"))),
		Subject:    strings.TrimSpace(strAnyMap(args, "subject", "title")),
		Voice:      strings.TrimSpace(strAnyMap(args, "voice")),
		Style:      strings.TrimSpace(strAnyMap(args, "style")),
		SourceHash: strings.TrimSpace(strAnyMap(args, "source_hash", "hash")),
	}
	if !briefHas(briefIntents, b.Intent) {
		return Brief{}, fmt.Errorf("video_brief: intent must be one of %s", strings.Join(briefIntents, ", "))
	}
	if b.Intent == "film" {
		if b.Format == "" {
			b.Format = "explainer"
		}
		if !briefHas(briefFormats, b.Format) {
			return Brief{}, fmt.Errorf("video_brief: format must be one of %s", strings.Join(briefFormats, ", "))
		}
	} else {
		b.Format = ""
	}
	// 时长只对有默认值的意图有意义；越界时回到默认值，不截到边界。
	b.Seconds = intArg(args["seconds"])
	if def, ok := briefSeconds[b.Intent]; ok {
		if b.Seconds < 5 || b.Seconds > 180 {
			b.Seconds = def
		}
	} else {
		b.Seconds = 0
	}
	aspect := strings.NewReplacer("x", ":", "X", ":", "：", ":", " ", "").Replace(strAnyMap(args, "aspect"))
	if !briefHas(briefAspects, aspect) {
		aspect = "9:16"
	}
	b.Aspect = aspect
	return b, nil
}

// VideoBrief 归一简报，再对照 ProviderMenu 给每种能力找账号。有账号算 Have 并记为花额度；没有账号但本机能兜底算 Local；都不行算 Missing。
// args 交给 NormalizeBrief。
func (e *Engine) VideoBrief(args map[string]any) (BriefPlan, error) {
	b, err := NormalizeBrief(args)
	if err != nil {
		return BriefPlan{}, err
	}
	plan := BriefPlan{Brief: b, Needs: []string{}, Have: map[string][]string{}, Local: []string{}, Missing: []string{}}
	for _, cap := range briefNeeds[b.Intent] {
		plan.Needs = append(plan.Needs, cap)
		var names []string
		for _, row := range e.ProviderMenu(cap) {
			name, _ := row["name"].(string)
			if strings.TrimSpace(name) == "" {
				name, _ = row["id"].(string)
			}
			if strings.TrimSpace(name) != "" {
				names = append(names, strings.TrimSpace(name))
			}
		}
		switch {
		case len(names) > 0:
			plan.Have[cap] = names
			plan.Spend = true
		case briefLocal[cap]:
			plan.Local = append(plan.Local, cap)
		default:
			plan.Missing = append(plan.Missing, cap)
		}
	}
	plan.Ready = len(plan.Missing) == 0
	return plan, nil
}

// BriefConfirmArgs 是交给 ask_user 的确认卡参数：一道题，选项是开拍、改一下、先不做，hold 为真，所以不会空闲通过。按内容语言写中文或英文。
// plan 写进题干，列出片型、时长、画幅、主题和每种能力用的账号。
func (e *Engine) BriefConfirmArgs(plan BriefPlan) map[string]any {
	en := strings.HasPrefix(strings.ToLower(strings.TrimSpace(e.ContentLanguage())), "en")
	title, ask, goLabel, editLabel, stopLabel := "制作简报", "按这个方案开拍？", "开拍", "改一下", "先不做"
	if en {
		title, ask, goLabel, editLabel, stopLabel = "Production brief", "Start with this plan?", "Start", "Change it", "Not now"
	}
	return map[string]any{
		"title": title,
		"hold":  true,
		"questions": []any{map[string]any{
			"id":          "go",
			"prompt":      BriefSummary(plan, en) + "\n" + ask,
			"recommended": "go",
			"options": []any{
				map[string]any{"id": "go", "label": goLabel},
				map[string]any{"id": "edit", "label": editLabel},
				map[string]any{"id": "stop", "label": stopLabel},
			},
		}},
	}
}

// BriefSummary 把简报写成一行给人看的字，再写一行每种能力用哪个账号或本机。en 为真时写英文。
func BriefSummary(plan BriefPlan, en bool) string {
	b := plan.Brief
	sep, list := " · ", "、"
	if en {
		list = ", "
	}
	parts := []string{briefLabel(b.Intent, b.Format, en)}
	if b.Seconds > 0 {
		if en {
			parts = append(parts, fmt.Sprintf("%ds", b.Seconds))
		} else {
			parts = append(parts, fmt.Sprintf("%d 秒", b.Seconds))
		}
	}
	parts = append(parts, b.Aspect)
	if b.Subject != "" {
		parts = append(parts, b.Subject)
	}
	line := strings.Join(parts, sep)
	var uses []string
	for _, cap := range plan.Needs {
		switch {
		case len(plan.Have[cap]) > 0:
			uses = append(uses, capLabel(cap, en)+" "+plan.Have[cap][0])
		case briefHas(plan.Local, cap):
			if en {
				uses = append(uses, capLabel(cap, en)+" local")
			} else {
				uses = append(uses, capLabel(cap, en)+" 本机")
			}
		}
	}
	if len(uses) > 0 {
		if en {
			line += "\nUses: " + strings.Join(uses, list)
		} else {
			line += "\n用到：" + strings.Join(uses, list)
		}
	}
	return line
}

// BriefDecision 从确认卡的答复里读出 go、edit 或 stop。跳过 note: 行。读不到，或答复是未确认、拒绝时返回 unconfirmed。
// answer 是 ask_user 的工具结果，每行形如 “go: edit — 改一下”。
func BriefDecision(answer string) string {
	for _, line := range strings.Split(answer, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "note:") {
			continue
		}
		_, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		id, _, _ := strings.Cut(strings.TrimSpace(rest), " — ")
		switch id = strings.TrimSpace(id); id {
		case "go", "edit", "stop":
			return id
		}
	}
	return "unconfirmed"
}

// briefLabel 是意图给人看的名字；成片时换成片型名。en 为真时写英文。
func briefLabel(intent, format string, en bool) string {
	zh := map[string]string{
		"cut": "剪辑", "dub": "配音", "remake": "翻拍", "still": "出图", "short": "短片", "drama": "短剧", "canvas": "画布",
		"talking": "口播", "podcast": "对谈", "interview": "街采", "ranking": "排行榜", "explainer": "讲解", "narration": "旁白",
	}
	enl := map[string]string{
		"cut": "Cut", "dub": "Dub", "remake": "Remake", "still": "Still", "short": "Short", "drama": "Drama", "canvas": "Canvas",
		"talking": "Talking head", "podcast": "Podcast", "interview": "Street interview", "ranking": "Ranking", "explainer": "Explainer", "narration": "Narration",
	}
	key := intent
	if intent == "film" {
		key = format
		// 成片的两人戏和短剧意图同名，单独起名。
		if format == "drama" {
			if en {
				return "Two-hander"
			}
			return "两人戏"
		}
	}
	if en {
		return enl[key]
	}
	return zh[key]
}

// capLabel 是能力给人看的名字。en 为真时写英文。
func capLabel(cap string, en bool) string {
	switch cap {
	case connection.CapImage:
		if en {
			return "Image"
		}
		return "图像"
	case connection.CapSpeech:
		if en {
			return "Voice"
		}
		return "语音"
	}
	return cap
}

// briefHas 判断 list 里有没有 s。
func briefHas(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
