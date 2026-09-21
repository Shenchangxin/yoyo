package artifact

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type skillFrontmatter struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	DescriptionZH string `yaml:"description_zh"`
	DisplayName   string `yaml:"display_name"`
	Icon          string `yaml:"icon"`
	License       string `yaml:"license"`
	Compatibility string `yaml:"compatibility"`
	Metadata      any    `yaml:"metadata"`
	AllowedTools  any    `yaml:"allowed-tools"`
}

// ParseSkillMD parses an Agent Skills SKILL.md document.
func ParseSkillMD(raw, source string) (Skill, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(raw, "---\n") {
		return Skill{}, fmt.Errorf("skill: missing YAML frontmatter")
	}
	rest := strings.TrimPrefix(raw, "---\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return Skill{}, fmt.Errorf("skill: unterminated frontmatter")
	}
	fm := rest[:end]
	body := strings.TrimSpace(rest[end+4:])
	var meta skillFrontmatter
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return Skill{}, fmt.Errorf("skill: frontmatter: %w", err)
	}
	if meta.Name == "" {
		return Skill{}, fmt.Errorf("skill: name and description are required")
	}
	desc := strings.TrimSpace(meta.Description)
	if desc == "" {
		desc = strings.TrimSpace(meta.DescriptionZH)
	}
	if desc == "" {
		return Skill{}, fmt.Errorf("skill: name and description are required")
	}
	if len(meta.Name) > 64 {
		return Skill{}, fmt.Errorf("skill: name too long")
	}
	if len(desc) > 4096 {
		return Skill{}, fmt.Errorf("skill: description too long")
	}
	return Skill{
		Name:          meta.Name,
		Description:   desc,
		DisplayName:   strings.TrimSpace(meta.DisplayName),
		Icon:          SafeIconURL(meta.Icon),
		License:       meta.License,
		Compatibility: meta.Compatibility,
		Metadata:      flattenMeta(meta.Metadata),
		AllowedTools:  anyString(meta.AllowedTools),
		Body:          body,
		Source:        source,
		Dir:           skillDir(source),
	}, nil
}

func skillDir(source string) string {
	if source == "" || source == "builtin" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(source), "skill.md") {
		return filepath.Dir(source)
	}
	return ""
}

func LoadSkillFile(path string) (Skill, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, err
	}
	return ParseSkillMD(string(b), path)
}

func (s Skill) CatalogLine() string {
	return s.Name + ": " + s.Description
}

// SafeIconURL accepts https icons from skill frontmatter. Relative paths and
// non-http schemes are ignored so the GUI never loads javascript: or file: URLs.
func SafeIconURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".localhost") {
		return ""
	}
	return u.String()
}

func anyString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			if s := anyString(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func flattenMeta(v any) map[string]string {
	out := map[string]string{}
	walkMeta("", v, out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func walkMeta(prefix string, v any, out map[string]string) {
	switch t := v.(type) {
	case nil:
		return
	case string:
		if prefix != "" && strings.TrimSpace(t) != "" {
			out[prefix] = strings.TrimSpace(t)
		}
	case map[string]any:
		for k, child := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			walkMeta(key, child, out)
		}
	case map[any]any:
		for k, child := range t {
			key := fmt.Sprint(k)
			if prefix != "" {
				key = prefix + "." + key
			}
			walkMeta(key, child, out)
		}
	case []any:
		if prefix != "" {
			out[prefix] = anyString(t)
		}
	default:
		if prefix != "" {
			s := strings.TrimSpace(fmt.Sprint(t))
			if s != "" {
				out[prefix] = s
			}
		}
	}
}
