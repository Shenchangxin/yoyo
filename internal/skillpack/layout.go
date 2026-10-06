package skillpack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func Root(home string) string {
	return filepath.Join(home, "packs")
}

func Dir(home, id string) string {
	return filepath.Join(Root(home), SanitizeID(id))
}

func SkillsDir(home, id string) string {
	return filepath.Join(Dir(home, id), "skills")
}

func ManifestPath(home, id string) string {
	return filepath.Join(Dir(home, id), "pack.json")
}

func WorkspaceFile(workspace string) string {
	return filepath.Join(workspace, ".yoyo", "packs.json")
}

func SanitizeID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	out = strings.Trim(out, "-")
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

func ReadManifest(home, id string) (Manifest, bool) {
	b, err := os.ReadFile(ManifestPath(home, id))
	if err != nil {
		return Manifest{}, false
	}
	var m Manifest
	if json.Unmarshal(b, &m) != nil || strings.TrimSpace(m.ID) == "" {
		return Manifest{}, false
	}
	return m, true
}

func WriteManifest(home string, m Manifest) error {
	id := SanitizeID(m.ID)
	if id == "" {
		return errInvalidID
	}
	m.ID = id
	if err := os.MkdirAll(Dir(home, id), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ManifestPath(home, id), b, 0o644)
}

func InstalledIDs(home string) []string {
	ents, err := os.ReadDir(Root(home))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if _, ok := ReadManifest(home, id); ok {
			out = append(out, id)
		}
	}
	return out
}
