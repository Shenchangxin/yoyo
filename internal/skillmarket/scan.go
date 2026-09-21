package skillmarket

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

type Scan struct {
	OK         bool     `json:"ok"`
	Reasons    []string `json:"reasons,omitempty"`
	NeedsShell bool     `json:"needs_shell,omitempty"`
	Name       string   `json:"name,omitempty"`
	Icon       string   `json:"icon,omitempty"`
	BodyBytes  int      `json:"body_bytes"`
	Files      int      `json:"files,omitempty"`
}

func scanPackFile(rel string, raw []byte) error {
	if len(raw) == 0 {
		return fmt.Errorf("empty file")
	}
	if isImageRel(rel) {
		return nil
	}
	if bytes.IndexByte(raw, 0) >= 0 {
		return fmt.Errorf("binary content")
	}
	if !utf8.Valid(raw) {
		return fmt.Errorf("invalid UTF-8")
	}
	lower := strings.ToLower(string(raw))
	if strings.Contains(lower, "curl ") && (strings.Contains(lower, "| sh") || strings.Contains(lower, "| bash")) {
		return fmt.Errorf("install script pipe-to-shell")
	}
	return nil
}

func isImageRel(rel string) bool {
	switch strings.ToLower(path.Ext(rel)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico":
		return true
	default:
		return false
	}
}

func ScanSkillMD(raw []byte, slug string) (artifact.Skill, Scan) {
	s := Scan{OK: true, BodyBytes: len(raw)}
	if len(raw) == 0 {
		return artifact.Skill{}, fail(s, "empty SKILL.md")
	}
	if len(raw) > 256<<10 {
		return artifact.Skill{}, fail(s, "SKILL.md larger than 256KiB")
	}
	if bytes.IndexByte(raw, 0) >= 0 {
		return artifact.Skill{}, fail(s, "binary content")
	}
	if !utf8.Valid(raw) {
		return artifact.Skill{}, fail(s, "invalid UTF-8")
	}
	sk, err := artifact.ParseSkillMD(string(raw), slug)
	if err != nil {
		return artifact.Skill{}, fail(s, err.Error())
	}
	s.Name = sk.Name
	s.Icon = sk.Icon
	tools := strings.ToLower(sk.AllowedTools)
	if strings.Contains(tools, "bash") || strings.Contains(tools, "shell") || strings.Contains(tools, "powershell") || strings.Contains(tools, "cmd") {
		s.NeedsShell = true
	}
	body := strings.ToLower(sk.Body)
	if strings.Contains(body, "curl ") && (strings.Contains(body, "| sh") || strings.Contains(body, "| bash")) {
		return sk, fail(s, "install script pipe-to-shell")
	}
	if MentionsScripts(sk.Body) {
		s.NeedsShell = true
	}
	return sk, s
}

func fail(s Scan, reason string) Scan {
	s.OK = false
	s.Reasons = append(s.Reasons, reason)
	return s
}

// MentionsScripts reports whether the skill body tells the model to run helpers
// from scripts/.
func MentionsScripts(body string) bool {
	return strings.Contains(body, "scripts/") || strings.Contains(strings.ToLower(body), "scripts\\")
}

type Peek struct {
	Icon        string
	DisplayName string
	Version     string
}

func PeekMeta(raw []byte) Peek {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return Peek{}
	}
	rest := strings.TrimPrefix(text, "---\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return Peek{}
	}
	var p Peek
	if sk, err := artifact.ParseSkillMD(text, "peek"); err == nil {
		p.Icon = sk.Icon
		p.DisplayName = sk.DisplayName
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "icon":
			if u := artifact.SafeIconURL(val); u != "" {
				p.Icon = u
			}
		case "display_name":
			if val != "" {
				p.DisplayName = val
			}
		case "version":
			p.Version = val
		}
	}
	return p
}
