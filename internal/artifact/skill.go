package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type skillFrontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license"`
	Compatibility string            `yaml:"compatibility"`
	Metadata      map[string]string `yaml:"metadata"`
	AllowedTools  string            `yaml:"allowed-tools"`
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
	if meta.Name == "" || meta.Description == "" {
		return Skill{}, fmt.Errorf("skill: name and description are required")
	}
	if len(meta.Name) > 64 {
		return Skill{}, fmt.Errorf("skill: name too long")
	}
	if len(meta.Description) > 1024 {
		return Skill{}, fmt.Errorf("skill: description too long")
	}
	return Skill{
		Name:          meta.Name,
		Description:   meta.Description,
		License:       meta.License,
		Compatibility: meta.Compatibility,
		Metadata:      meta.Metadata,
		AllowedTools:  meta.AllowedTools,
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
