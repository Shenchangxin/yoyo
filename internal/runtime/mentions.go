package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
	refs = ParseMentions(text)
	if budget <= 0 {
		budget = 2400
	}
	var b strings.Builder
	used := 0
	for _, m := range refs {
		chunk := mentionChunk(workspace, harnessNote, skills, m, budget-used)
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

func mentionChunk(workspace, harnessNote string, skills map[string]string, m Mention, remain int) string {
	if remain < 32 {
		return ""
	}
	switch m.Kind {
	case "skill":
		if skills == nil {
			return fmt.Sprintf("## @skill:%s\nERROR: no skill catalog\n", m.Ref)
		}
		body, ok := skills[m.Ref]
		if !ok {
			return fmt.Sprintf("## @skill:%s\nERROR: unknown skill\n", m.Ref)
		}
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
		body := string(raw)
		capped, trunc := capText(body, 800)
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

func jailPath(workspace, rel string) (string, error) {
	rel = strings.Trim(rel, `"'`)
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
