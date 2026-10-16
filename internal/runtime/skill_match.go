package runtime

import (
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

const (
	SkillActionHint     = "hint"
	SkillActionAutoload = "autoload"

	skillHintMinScore     = 28
	skillAutoloadMinScore = 55
	skillAutoloadMargin   = 12
	skillMatchTopK        = 3
)

type SkillSuggestion struct {
	Name   string `json:"name"`
	Score  int    `json:"score"`
	Reason string `json:"reason"`
	Action string `json:"action"`
}

type SkillMatchReq struct {
	Voice     string
	Skills    []artifact.Skill
	Packs     []PackSession
	Loaded    []string
	Mentioned []string
	Depth     int
	Mode      string
}

func NormalizeAutoInvoke(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "off", "hint", "on":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "on"
	}
}

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

// MatchSkills scores the operator voice against the catalog. It never
// uses embeddings. Autoload is reserved for a unique high-score domain
// pack orchestrator when mode is "on".
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
			if best.Score-rival >= skillAutoloadMargin {
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
	if score <= 0 {
		return 0, ""
	}
	return score, strings.Join(reasons, ", ")
}

func skillNameParts(sk artifact.Skill) []string {
	raw := sk.Name + " " + sk.DisplayName
	if sk.Dir != "" {
		raw += " " + filepath.Base(sk.Dir)
	}
	return tokenList(raw)
}

var routeLex = [][2]string{
	{"novel", "小说"},
	{"book", "小说"},
	{"game", "游戏"},
	{"playable", "可玩"},
	{"weekly", "周报"},
	{"report", "周报"},
	{"minutes", "纪要"},
	{"meeting", "会议"},
	{"clean", "清洗"},
	{"data", "数据"},
	{"competitor", "竞品"},
	{"review", "评审"},
	{"diff", "diff"},
	{"plan", "计划"},
	{"script", "剧本"},
	{"storyboard", "分镜"},
	{"drama", "短剧"},
}

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

func isCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
