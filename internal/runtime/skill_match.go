// 按操作者原话给技能目录打分。不用嵌入向量。自动加载只留给 mode 为 on 时分数够高且领先的领域包编排技能。Depth 大于 0 直接返回空。

package runtime

import (
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// SkillActionHint 是 hint，SkillActionAutoload 是 autoload。分数门槛和条数上限也在这里。
const (
	// SkillActionHint 是 hint。这种建议只提示，不把正文注入上下文。
	SkillActionHint = "hint"
	// SkillActionAutoload 是 autoload。命中后 ApplySkillMatch 会把正文注入上下文。
	SkillActionAutoload = "autoload"

	// skillHintMinScore 是 28。低于它的技能不进入建议。
	skillHintMinScore = 28
	// skillAutoloadMinScore 是 55。领域编排技能至少要达到它才可能自动加载。
	skillAutoloadMinScore = 55
	// skillAutoloadMargin 是 12。最高编排技能必须比其他编排技能至少高出这个分差。
	skillAutoloadMargin = 12
	// skillMatchTopK 是 3。排序后只留前 3 条。
	skillMatchTopK = 3
)

// SkillSuggestion 是一条匹配结果，含名字、分数、原因和动作。
type SkillSuggestion struct {
	Name   string `json:"name"`
	Score  int    `json:"score"`
	Reason string `json:"reason"`
	Action string `json:"action"`
}

// SkillMatchReq 是一次匹配的输入：原话、目录、已启用包、已加载名、已提及名、深度、模式和车道。
type SkillMatchReq struct {
	Voice     string
	Skills    []artifact.Skill
	Packs     []PackSession
	Loaded    []string
	Mentioned []string
	Depth     int
	Mode      string
	Lane      string
}

// NormalizeAutoInvoke 只接受 off、hint、on。其余值，包括空串，返回 on。
// mode 去掉空白并转成小写后，是 off、hint 或 on 就返回该值，其余返回 on。
func NormalizeAutoInvoke(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "off", "hint", "on":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "on"
	}
}

// skipSkillRequest 在原话里出现跳过技能、不用技能、不要用技能、不要加载技能、skip the skill、skip skills 或 skip skill 时返回真。空串返回假。
// voice 去掉空白并转成小写，空串返回假；否则看是否包含上述片段。
func skipSkillRequest(voice string) bool {
	s := strings.ToLower(strings.TrimSpace(voice))
	if s == "" {
		return false
	}
	for _, p := range []string{
		"skip the skill", "skip skills", "skip skill",
		"跳过技能", "不用技能", "不要用技能", "不要加载技能",
		"不用 skill", "不用skill",
	} {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// hasSpatialGameLoaded 在已加载名里出现 yoyo-game 或 game-world-analyze、game-world-rules、game-world-place、game-world-build、game-world-qa 时返回真。
// loaded 逐项去掉空白并转成小写，命中上述名字就返回真。
func hasSpatialGameLoaded(loaded []string) bool {
	for _, n := range loaded {
		switch strings.ToLower(strings.TrimSpace(n)) {
		case "yoyo-game", "game-world-analyze", "game-world-rules", "game-world-place", "game-world-build", "game-world-qa":
			return true
		}
	}
	return false
}

// novelAdaptVoice 在原话含小说、改编小说、这本书、novel，或 adapt this book、adapt the novel、adapt a novel、adapt this novel 时返回真。
// voice 去掉空白并转成小写，空串返回假；否则看是否包含上述片段。
func novelAdaptVoice(voice string) bool {
	s := strings.ToLower(strings.TrimSpace(voice))
	if s == "" {
		return false
	}
	for _, p := range []string{"小说", "改编小说", "这本书", "novel", "adapt this book", "adapt the novel", "adapt a novel", "adapt this novel"} {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// novelPackAllowed 对非小说改游戏技能恒为真。小说改游戏技能只有 novelAdaptVoice 为真才允许自动加载。
// name 交给 IsNovelGameSkill，不是小说改游戏技能时直接返回真。
// voice 只在 name 属于小说改游戏技能时交给 novelAdaptVoice。
func novelPackAllowed(name, voice string) bool {
	if !IsNovelGameSkill(name) {
		return true
	}
	return novelAdaptVoice(voice)
}

// isProcessSkill 只认 brainstorming、writing-plans、systematic-debugging、using-superpowers、test-driven-development、requesting-code-review、receiving-code-review、finishing-a-development-branch、executing-plans、subagent-driven-development、using-git-worktrees。
// name 去掉空白并转成小写后再和上述名单比较。
func isProcessSkill(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "brainstorming", "writing-plans", "systematic-debugging",
		"using-superpowers", "test-driven-development",
		"requesting-code-review", "receiving-code-review",
		"finishing-a-development-branch", "executing-plans",
		"subagent-driven-development", "using-git-worktrees":
		return true
	default:
		return false
	}
}

// MatchSkills 用原话给目录打分，不用嵌入向量。mode 为 off、Depth 大于 0、原话要求跳过技能，或原话不足 2 个 rune 时返回 nil。已加载和已提及的名字跳过。AllowsImplicit 为假的跳过。空间游戏已加载或 Lane 为 game 时跳过小说改游戏技能。流程技能在没有方法论包时跳过。编排技能加 20 分。低于 skillHintMinScore 的丢掉。流程技能动作保持 hint。mode 为 on 时，唯一达到 skillAutoloadMinScore 且领先其他编排技能至少 skillAutoloadMargin、并且 novelPackAllowed 为真的编排技能才标成 autoload。结果最多 skillMatchTopK 条。
// req 的 Mode、Depth、Voice、Loaded、Mentioned、Packs、Lane 和 Skills 分别参与提前返回、跳过名单、领域包和打分。
func MatchSkills(req SkillMatchReq) []SkillSuggestion {
	mode := NormalizeAutoInvoke(req.Mode)
	if mode == "off" || req.Depth > 0 || skipSkillRequest(req.Voice) {
		return nil
	}
	voice := strings.TrimSpace(req.Voice)
	if utf8.RuneCountInString(voice) < 2 {
		return nil
	}
	skip := map[string]bool{}
	for _, n := range req.Loaded {
		skip[strings.ToLower(strings.TrimSpace(n))] = true
	}
	for _, n := range req.Mentioned {
		skip[strings.ToLower(strings.TrimSpace(n))] = true
	}
	orch := DomainOrchestrators(req.Packs)
	method := HasMethodologyPack(req.Packs)
	spatialLoaded := hasSpatialGameLoaded(req.Loaded) || strings.EqualFold(strings.TrimSpace(req.Lane), "game")
	voiceToks := matchTokens(voice)
	type scored struct {
		SkillSuggestion
		orch bool
	}
	var ranked []scored
	for _, sk := range uniqueCatalogSkills(req.Skills) {
		if sk.Name == "" || skip[strings.ToLower(sk.Name)] {
			continue
		}
		if !sk.AllowsImplicit() {
			continue
		}
		if spatialLoaded && IsNovelGameSkill(sk.Name) {
			continue
		}
		proc := isProcessSkill(sk.Name)
		if proc && !method {
			continue
		}
		score, reason := scoreSkill(sk, voice, voiceToks)
		if orch[sk.Name] {
			score += 20
			if reason == "" {
				reason = "enabled domain pack orchestrator"
			} else {
				reason += "; domain orchestrator"
			}
		}
		if score < skillHintMinScore {
			continue
		}
		action := SkillActionHint
		if proc {
			action = SkillActionHint
		}
		ranked = append(ranked, scored{
			SkillSuggestion: SkillSuggestion{Name: sk.Name, Score: score, Reason: reason, Action: action},
			orch:            orch[sk.Name],
		})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score != ranked[j].Score {
			return ranked[i].Score > ranked[j].Score
		}
		if ranked[i].orch != ranked[j].orch {
			return ranked[i].orch
		}
		return ranked[i].Name < ranked[j].Name
	})
	if len(ranked) > skillMatchTopK {
		ranked = ranked[:skillMatchTopK]
	}
	if mode == "on" {
		var best *scored
		for i := range ranked {
			if ranked[i].orch && ranked[i].Score >= skillAutoloadMinScore {
				best = &ranked[i]
				break
			}
		}
		if best != nil {
			rival := 0
			for _, r := range ranked {
				if r.Name == best.Name || !r.orch {
					continue
				}
				if r.Score > rival {
					rival = r.Score
				}
			}
			if best.Score-rival >= skillAutoloadMargin && novelPackAllowed(best.Name, voice) {
				best.Action = SkillActionAutoload
			}
		}
	}
	out := make([]SkillSuggestion, 0, len(ranked))
	for _, r := range ranked {
		out = append(out, r.SkillSuggestion)
	}
	return out
}

// ApplySkillMatch 调用 MatchSkills。动作为 autoload 的名字交给 ExpandNamedSkills。没有自动加载时注入为空。
// req 交给 MatchSkills。
// bodies 在有 autoload 名字时连同 dirs、这些名字和 budget 交给 ExpandNamedSkills。
// dirs 在有 autoload 名字时交给 ExpandNamedSkills。
// budget 在有 autoload 名字时交给 ExpandNamedSkills。
func ApplySkillMatch(req SkillMatchReq, bodies, dirs map[string]string, budget int) (inject string, hints []SkillSuggestion, auto []string) {
	hints = MatchSkills(req)
	for _, s := range hints {
		if s.Action == SkillActionAutoload {
			auto = append(auto, s.Name)
		}
	}
	if len(auto) == 0 {
		return "", hints, nil
	}
	return ExpandNamedSkills(bodies, dirs, auto, budget), hints, auto
}

// FormatSkillHints 只列出动作为 hint 的建议。自动加载的不出现。没有可列项时返回空串。
// hints 里只留下 Action 为 hint 且 Name 去掉空白后非空的项。
func FormatSkillHints(hints []SkillSuggestion) string {
	var lines []string
	for _, s := range hints {
		if s.Action != SkillActionHint || strings.TrimSpace(s.Name) == "" {
			continue
		}
		line := "- " + s.Name
		if s.Reason != "" {
			line += " — " + s.Reason
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n## Suggested skills this turn\nHost hint — not loaded. Call load_skill if it fits. Do not treat this as already loaded.\n" + strings.Join(lines, "\n") + "\n"
}

// uniqueCatalogSkills 按 Name 去重，保留第一次出现。空名丢掉。
// in 按 Name 去重，空名丢掉，保留第一次出现。
func uniqueCatalogSkills(in []artifact.Skill) []artifact.Skill {
	seen := map[string]bool{}
	var out []artifact.Skill
	for _, s := range in {
		if s.Name == "" || seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		out = append(out, s)
	}
	return out
}

// scoreSkill 原话含规范名加 80，含显示名加 50，含不同的目录基名加 40，否则命中的名字片段每个加 36。描述词重叠中文加 6、其余加 4，重叠分最多 40。名字是 internet-reach 且 researchVoice 为真时再加 36。分数小于等于 0 返回 0。
// sk 提供 Name、DisplayName、Dir 和 Description。
// voice 转成小写后看是否包含这些名字，并交给 researchVoice。
// voiceToks 用来看规范名、名字片段和描述词是否命中。
func scoreSkill(sk artifact.Skill, voice string, voiceToks map[string]bool) (int, string) {
	name := strings.ToLower(strings.TrimSpace(sk.Name))
	display := strings.ToLower(strings.TrimSpace(sk.DisplayName))
	slug := ""
	if sk.Dir != "" {
		slug = strings.ToLower(filepath.Base(sk.Dir))
	}
	voiceFold := strings.ToLower(voice)
	score := 0
	var reasons []string
	if name != "" && (strings.Contains(voiceFold, name) || voiceToks[name]) {
		score += 80
		reasons = append(reasons, "named "+sk.Name)
	} else if display != "" && strings.Contains(voiceFold, display) {
		score += 50
		reasons = append(reasons, "named "+sk.DisplayName)
	} else if slug != "" && slug != name && strings.Contains(voiceFold, slug) {
		score += 40
		reasons = append(reasons, "named "+slug)
	} else {
		for _, part := range skillNameParts(sk) {
			if len(part) < 3 || !voiceToks[part] {
				continue
			}
			score += 36
			reasons = append(reasons, "token "+part)
		}
	}
	descToks := matchTokens(sk.Description + " " + sk.Name + " " + sk.DisplayName + " " + slug)
	overlap := 0
	for tok := range voiceToks {
		if utf8.RuneCountInString(tok) < 2 || !descToks[tok] {
			continue
		}
		if utf8.RuneCountInString(tok) >= 2 && isCJK(tok) {
			overlap += 6
		} else {
			overlap += 4
		}
	}
	if overlap > 40 {
		overlap = 40
	}
	score += overlap
	if overlap > 0 && len(reasons) == 0 {
		reasons = append(reasons, "description overlap")
	}
	if strings.EqualFold(name, "internet-reach") && researchVoice(voice) {
		score += 36
		reasons = append(reasons, "internet research")
	}
	if score <= 0 {
		return 0, ""
	}
	return score, strings.Join(reasons, ", ")
}

// researchVoice 在原话含调研、全网搜、搜一下网上、网上搜、查一下网上、联网、research this、search the web、web search、youtube、bilibili、b站、小红书或 xiaohongshu 时返回真。
func researchVoice(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return false
	}
	for _, p := range []string{
		"调研", "全网搜", "搜一下网上", "网上搜", "查一下网上", "联网",
		"research this", "search the web", "web search",
		"youtube", "bilibili", "b站", "小红书", "xiaohongshu",
	} {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// skillNameParts 把规范名、显示名和目录基名交给 tokenList。
func skillNameParts(sk artifact.Skill) []string {
	raw := sk.Name + " " + sk.DisplayName
	if sk.Dir != "" {
		raw += " " + filepath.Base(sk.Dir)
	}
	return tokenList(raw)
}

// routeLex 是中英对照词对。matchTokens 在原话命中任一侧时把两侧都放进词集合。
var routeLex = [][2]string{
	{"novel", "小说"},
	{"book", "小说"},
	{"game", "游戏"},
	{"game-world", "游戏世界"},
	{"playable", "可玩"},
	{"3d", "3d"},
	{"weekly", "周报"},
	{"report", "周报"},
	{"minutes", "纪要"},
	{"meeting", "会议"},
	{"clean", "清洗"},
	{"data", "数据"},
	{"competitor", "竞品"},
	{"research", "调研"},
	{"youtube", "油管"},
	{"bilibili", "哔哩"},
	{"xiaohongshu", "小红书"},
	{"twitter", "推特"},
	{"review", "评审"},
	{"diff", "diff"},
	{"plan", "计划"},
	{"script", "剧本"},
	{"storyboard", "分镜"},
	{"drama", "短剧"},
}

// matchTokens 先收 tokenList 的结果，再按 routeLex 把命中的英文和至少两个 rune 的中文放进集合。
func matchTokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, tok := range tokenList(s) {
		out[tok] = true
	}
	fold := strings.ToLower(s)
	for _, pair := range routeLex {
		en, zh := pair[0], pair[1]
		if strings.Contains(fold, en) || strings.Contains(s, zh) {
			out[en] = true
			if utf8.RuneCountInString(zh) >= 2 {
				out[zh] = true
			}
		}
	}
	return out
}

// tokenList 收至少 3 个字符的拉丁词、单个汉字、相邻两字，以及用 -、_、/、.、: 切开后长度至少 3 的片段。
func tokenList(s string) []string {
	s = strings.ToLower(s)
	var out []string
	seen := map[string]bool{}
	add := func(t string) {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			return
		}
		seen[t] = true
		out = append(out, t)
	}
	var latin strings.Builder
	flushLatin := func() {
		if latin.Len() >= 3 {
			add(latin.String())
		}
		latin.Reset()
	}
	runes := []rune(s)
	for i, r := range runes {
		if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			latin.WriteRune(r)
			continue
		}
		flushLatin()
		if unicode.Is(unicode.Han, r) {
			add(string(r))
			if i+1 < len(runes) && unicode.Is(unicode.Han, runes[i+1]) {
				add(string(runes[i : i+2]))
			}
		}
	}
	flushLatin()
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == '/' || r == '.' || r == ':'
	}) {
		part = strings.TrimSpace(part)
		if len(part) >= 3 {
			add(part)
		}
	}
	return out
}

// isCJK 在字符串里有汉字时返回真。
func isCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
