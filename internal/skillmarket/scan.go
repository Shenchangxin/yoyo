// 扫描 SKILL.md：体积、UTF-8，以及 curl 管道到 shell。

package skillmarket

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// Scan 是一次 SKILL.md 扫描的结果：能否安装、原因、是否要 shell、名字和图标。
type Scan struct {
	OK         bool     `json:"ok"`
	Reasons    []string `json:"reasons,omitempty"`
	NeedsShell bool     `json:"needs_shell,omitempty"`
	Name       string   `json:"name,omitempty"`
	Icon       string   `json:"icon,omitempty"`
	BodyBytes  int      `json:"body_bytes"`
	Files      int      `json:"files,omitempty"`
}

// scanPackFile 拒绝空文件。png、jpg、jpeg、gif、webp、ico 不再检查正文。含 NUL 返回 "binary content"，非法 UTF-8 返回 "invalid UTF-8"。同时出现 "curl " 和 "| sh" 或 "| bash" 时返回 "install script pipe-to-shell"。
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

// isImageRel 在扩展名是 png、jpg、jpeg、gif、webp、ico 时返回 true。
func isImageRel(rel string) bool {
	switch strings.ToLower(path.Ext(rel)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico":
		return true
	default:
		return false
	}
}

// ScanSkillMD 解析 SKILL.md。空文件、超过 256 KiB、含 NUL、非法 UTF-8、解析失败，或正文里有 curl 管道到 sh/bash，都会把 OK 设为假。allowed-tools 含 bash、shell、powershell、cmd，或正文提到 scripts/ 时 NeedsShell 为真。
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

// fail 把 OK 设为假并追加一条原因。
func fail(s Scan, reason string) Scan {
	s.OK = false
	s.Reasons = append(s.Reasons, reason)
	return s
}

// MentionsScripts 在正文含 scripts/ 或 scripts\\ 时返回 true。
func MentionsScripts(body string) bool {
	return strings.Contains(body, "scripts/") || strings.Contains(strings.ToLower(body), "scripts\\")
}

// Peek 是从前言里看到的图标、显示名和版本。
type Peek struct {
	Icon        string
	DisplayName string
	Version     string
}

// PeekMeta 读 YAML 前言里的 icon、display_name 和 version。不是以 --- 开头时返回零值。icon 还要过 SafeIconURL。
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
