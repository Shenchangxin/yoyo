package skillmarket

import (
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

type Scan struct {
	OK         bool     `json:"ok"`
	Reasons    []string `json:"reasons,omitempty"`
	NeedsShell bool     `json:"needs_shell,omitempty"`
	Name       string   `json:"name,omitempty"`
	BodyBytes  int      `json:"body_bytes"`
}

func ScanSkillMD(raw []byte, slug string) (artifact.Skill, Scan) {
	s := Scan{OK: true, BodyBytes: len(raw)}
	if len(raw) == 0 {
		return artifact.Skill{}, fail(s, "empty SKILL.md")
	}
	if len(raw) > 256<<10 {
		return artifact.Skill{}, fail(s, "SKILL.md larger than 256KiB")
	}
	if strings.IndexByte(string(raw), 0) >= 0 {
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
	tools := strings.ToLower(sk.AllowedTools)
	if strings.Contains(tools, "bash") || strings.Contains(tools, "shell") || strings.Contains(tools, "powershell") || strings.Contains(tools, "cmd") {
		s.NeedsShell = true
	}
	body := strings.ToLower(sk.Body)
	if strings.Contains(body, "curl ") && (strings.Contains(body, "| sh") || strings.Contains(body, "| bash")) {
		return sk, fail(s, "install script pipe-to-shell")
	}
	return sk, s
}

func fail(s Scan, reason string) Scan {
	s.OK = false
	s.Reasons = append(s.Reasons, reason)
	return s
}
