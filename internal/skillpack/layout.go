// 技能包在磁盘上的位置：home/packs/<id>，以及工作区 .yoyo/packs.json。

package skillpack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Root 返回 home/packs。
func Root(home string) string {
	return filepath.Join(home, "packs")
}

// Dir 返回 home/packs/<净化后的 id>。
func Dir(home, id string) string {
	return filepath.Join(Root(home), SanitizeID(id))
}

// SkillsDir 返回该包目录下的 skills。
// home 和 id 一起交给 Dir。
// id 和 home 一起交给 Dir。
func SkillsDir(home, id string) string {
	return filepath.Join(Dir(home, id), "skills")
}

// ManifestPath 返回该包目录下的 pack.json。
// home 和 id 一起交给 Dir。
// id 和 home 一起交给 Dir。
func ManifestPath(home, id string) string {
	return filepath.Join(Dir(home, id), "pack.json")
}

// WorkspaceFile 返回 <workspace>/.yoyo/packs.json。
func WorkspaceFile(workspace string) string {
	return filepath.Join(workspace, ".yoyo", "packs.json")
}

// SanitizeID 转成小写，只留 a-z、0-9 和连字符，去掉首尾连字符，最长 64 个字符。
// id 转成小写并去掉两端空白，只留 a-z、0-9 和连字符，去掉首尾连字符，长于 64 时截到 64。
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

// ReadManifest 读取 pack.json。读失败、JSON 无效或 id 为空白时返回 false。
// home 和 id 一起交给 ManifestPath。
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

// WriteManifest 把清单写到 pack.json，权限 0o644。id 净化后为空则返回 invalid pack id。
// home 用来创建目录，并和净化后的 id 一起交给 ManifestPath。
// m 的 ID 经 SanitizeID，空则返回错误，否则写回 ID 再整份清单落盘。
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

// InstalledIDs 返回 home/packs 下能读到 pack.json 的子目录名。目录读失败时返回 nil。
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
