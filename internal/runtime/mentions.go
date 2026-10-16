package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// Mentions are user-designated pins. They are working memory, not harness
// pins: expand into a bounded inject block instead of dumping the repo.
type Mention struct {
	Kind string `json:"kind"` // file | folder | harness | skill
	Ref  string `json:"ref"`
}

var (
	reFile    = regexp.MustCompile(`@file:(\S+)`)
	reFolder  = regexp.MustCompile(`@folder:(\S+)`)
	reSkill   = regexp.MustCompile(`@skill:(\S+)`)
	reHarness = regexp.MustCompile(`@harness\b`)
)

func ParseMentions(text string) []Mention {
	var out []Mention
	seen := map[string]bool{}
	add := func(m Mention) {
		key := m.Kind + ":" + m.Ref
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, m)
	}
	for _, m := range reFile.FindAllStringSubmatch(text, 8) {
		add(Mention{Kind: "file", Ref: m[1]})
	}
	for _, m := range reFolder.FindAllStringSubmatch(text, 4) {
		add(Mention{Kind: "folder", Ref: m[1]})
	}
	for _, m := range reSkill.FindAllStringSubmatch(text, 8) {
		add(Mention{Kind: "skill", Ref: strings.Trim(m[1], `'"`)})
	}
	if reHarness.MatchString(text) {
		add(Mention{Kind: "harness", Ref: "active"})
	}
	return out
}

func ExpandMentions(workspace, text, harnessNote string, budget int) (inject string, refs []Mention) {
	return ExpandMentionsSkills(workspace, text, harnessNote, nil, budget)
}

func ExpandMentionsSkills(workspace, text, harnessNote string, skills map[string]string, budget int) (inject string, refs []Mention) {
	return ExpandMentionsPacks(workspace, text, harnessNote, skills, nil, budget)
}

const (
	mentionFileRunes    = 800
	mentionPrimaryRunes = 12_000
	mentionPrimaryBytes = 8 * 1024
)

func ExpandMentionsPacks(workspace, text, harnessNote string, skills map[string]string, skillDirs map[string]string, budget int) (inject string, refs []Mention) {
	refs = ParseMentions(text)
	if budget <= 0 {
		budget = 2400
	}
	var b strings.Builder
	used := 0
	for _, m := range refs {
		if m.Kind == "file" && fileLooksPrimary(workspace, m.Ref) {
			chunk := mentionPrimaryFile(workspace, m.Ref)
			if chunk != "" {
				b.WriteString(chunk)
			}
			continue
		}
		remain := budget - used
		chunk := mentionChunk(workspace, harnessNote, skills, skillDirs, m, remain)
		if chunk == "" {
			continue
		}
		if used+len(chunk) > budget && used > 0 {
			b.WriteString("…[mentions truncated]\n")
			break
		}
		b.WriteString(chunk)
		used += len(chunk)
	}
	return strings.TrimSpace(b.String()), refs
}

func ExpandNamedSkills(skills, skillDirs map[string]string, names []string, budget int) string {
	if budget <= 0 {
		budget = 2400
	}
	var b strings.Builder
	used := 0
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		remain := budget - used
		chunk := mentionChunk("", "", skills, skillDirs, Mention{Kind: "skill", Ref: name}, remain)
		if chunk == "" {
			continue
		}
		if used+len(chunk) > budget && used > 0 {
			b.WriteString("…[mentions truncated]\n")
			break
		}
		b.WriteString(chunk)
		used += len(chunk)
	}
	return strings.TrimSpace(b.String())
}

func MentionSkillNames(refs []Mention) []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range refs {
		if r.Kind != "skill" || r.Ref == "" || seen[r.Ref] {
			continue
		}
		seen[r.Ref] = true
		out = append(out, r.Ref)
	}
	return out
}

func lookupSkillMention(skills, skillDirs map[string]string, ref string) (body, dir string, ok bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" || skills == nil {
		return "", "", false
	}
	if body, ok = skills[ref]; ok {
		if skillDirs != nil {
			dir = skillDirs[ref]
		}
		return body, dir, true
	}
	for k, v := range skills {
		if strings.EqualFold(k, ref) {
			if skillDirs != nil {
				dir = skillDirs[k]
			}
			return v, dir, true
		}
	}
	if skillDirs == nil {
		return "", "", false
	}
	for k, d := range skillDirs {
		if strings.EqualFold(k, ref) || (d != "" && strings.EqualFold(filepath.Base(d), ref)) {
			if body, ok = skills[k]; ok {
				return body, d, true
			}
		}
	}
	return "", "", false
}

func mentionChunk(workspace, harnessNote string, skills, skillDirs map[string]string, m Mention, remain int) string {
	if remain < 32 {
		return ""
	}
	switch m.Kind {
	case "skill":
		body, dir, ok := lookupSkillMention(skills, skillDirs, m.Ref)
		if skills == nil {
			return fmt.Sprintf("## @skill:%s\nERROR: no skill catalog\n", m.Ref)
		}
		if !ok {
			return fmt.Sprintf("## @skill:%s\nERROR: unknown skill\n", m.Ref)
		}
		body = DecorateSkillBody(m.Ref, body, dir)
		capped, trunc := capText(body, remain)
		out := "## @skill:" + m.Ref + "\n" + capped
		if trunc {
			out += "\n…[truncated]"
		}
		return out + "\n"
	case "harness":
		note := harnessNote
		if note == "" {
			note = "refs/active"
		}
		return "## @harness\n" + note + "\n"
	case "folder":
		p, err := jailPath(workspace, m.Ref)
		if err != nil {
			return fmt.Sprintf("## @folder:%s\nERROR: %s\n", m.Ref, err)
		}
		ents, err := os.ReadDir(p)
		if err != nil {
			return fmt.Sprintf("## @folder:%s\nERROR: %s\n", m.Ref, err)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "## @folder:%s\n", m.Ref)
		n := 0
		for _, e := range ents {
			if n >= 40 {
				b.WriteString("…\n")
				break
			}
			if e.IsDir() {
				b.WriteString("d ")
			} else {
				b.WriteString("f ")
			}
			b.WriteString(e.Name())
			b.WriteByte('\n')
			n++
		}
		return b.String()
	default:
		p, err := jailPath(workspace, m.Ref)
		if err != nil {
			return fmt.Sprintf("## @file:%s\nERROR: %s\n", m.Ref, err)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return fmt.Sprintf("## @file:%s\nERROR: %s\n", m.Ref, err)
		}
		if looksPrimaryBytes(raw) {
			return mentionPrimaryFile(workspace, m.Ref)
		}
		body := string(raw)
		capped, trunc := capText(body, mentionFileRunes)
		var b strings.Builder
		fmt.Fprintf(&b, "## @file:%s\n", m.Ref)
		b.WriteString(capped)
		if trunc {
			b.WriteString("\n…[truncated; use read_file for more]")
		}
		b.WriteByte('\n')
		return b.String()
	}
}

func fileLooksPrimary(workspace, rel string) bool {
	p, err := jailPath(workspace, rel)
	if err != nil {
		return false
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	return looksPrimaryBytes(raw)
}

func looksPrimaryBytes(raw []byte) bool {
	if len(raw) > mentionPrimaryBytes {
		return true
	}
	return utf8.RuneCount(raw) > 4000
}

func mentionPrimaryFile(workspace, rel string) string {
	p, err := jailPath(workspace, rel)
	if err != nil {
		return fmt.Sprintf("## @file:%s\nERROR: %s\n", rel, err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return fmt.Sprintf("## @file:%s\nERROR: %s\n", rel, err)
	}
	runes := utf8.RuneCount(raw)
	var b strings.Builder
	fmt.Fprintf(&b, "## @file:%s\n%d bytes, %d runes. This is the task source of truth — call read_file on this path (or task profile=explore for chapter batches). Do not invent chapters from this outline.\n", rel, len(raw), runes)
	if !looksTextBytes(raw) {
		b.WriteString("(binary or non-text; use read_file / view_image)\n")
		return b.String()
	}
	outline := fileOutline(string(raw), mentionPrimaryRunes)
	b.WriteString(outline)
	if !strings.HasSuffix(outline, "\n") {
		b.WriteByte('\n')
	}
	return b.String()
}

func looksTextBytes(raw []byte) bool {
	n := len(raw)
	if n > 1024 {
		n = 1024
	}
	for _, c := range raw[:n] {
		if c == 0 {
			return false
		}
	}
	return true
}

var reOutlineLine = regexp.MustCompile(`(?m)^(#{1,6}\s+\S+|第.{0,12}[回章节卷篇].+)$`)

func fileOutline(body string, maxRunes int) string {
	var lines []string
	seen := map[string]bool{}
	for _, m := range reOutlineLine.FindAllString(body, 80) {
		s := strings.TrimSpace(m)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		lines = append(lines, s)
	}
	var b strings.Builder
	if len(lines) > 0 {
		b.WriteString("Outline:\n")
		for _, s := range lines {
			b.WriteString("- ")
			b.WriteString(s)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	b.WriteString("Head:\n")
	n := 0
	for _, line := range strings.Split(body, "\n") {
		if n >= 80 {
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
		n++
	}
	out := b.String()
	capped, trunc := capText(out, maxRunes)
	if trunc {
		capped += "\n…[outline truncated; use read_file for the rest]"
	}
	return capped
}

func jailPath(workspace, rel string) (string, error) {
	rel = strings.Trim(rel, `"'`)
	rel = capability.CanonicalizeToolPath(rel)
	workspace = capability.CanonicalizeToolPath(workspace)
	p := rel
	if !filepath.IsAbs(p) {
		p = filepath.Join(workspace, rel)
	}
	p = filepath.Clean(p)
	if !capability.WithinWorkspace(workspace, p) {
		return "", fmt.Errorf("path escapes workspace")
	}
	return p, nil
}
