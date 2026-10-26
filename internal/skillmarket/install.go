// 把市场技能写到 homeSkills/<slug>，并用 .market.json 标记这次安装。

package skillmarket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Record 是 .market.json 里的安装记录：来源、远程 id、文件列表和扫描结果。
type Record struct {
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Source    string    `json:"source"`
	Catalog   string    `json:"catalog"`
	URL       string    `json:"url"`
	Remote    string    `json:"remote,omitempty"`
	Icon      string    `json:"icon,omitempty"`
	Files     []string  `json:"files,omitempty"`
	Installed time.Time `json:"installed_at"`
	Scan      Scan      `json:"scan"`
}

// Install 以空记录调用 InstallRecord。
// homeSkills 交给 InstallRecord。
// slug 交给 InstallRecord。
// scan 交给 InstallRecord。
// files 交给 InstallRecord。
func Install(homeSkills, slug string, scan Scan, files []PackFile) (string, error) {
	return InstallRecord(homeSkills, slug, scan, files, Record{})
}

// InstallRecord 把文件写到 homeSkills/<slug>。先写到该目录加 .install-tmp，放上 .market.json，再改名换上。scan.OK 为假返回 "scan blocked"。没有根上的 SKILL.md 返回 "pack missing SKILL.md"。脚本用 0o755，其余用 0o644。来源为空时记成 HubWorkBuddy。
// files 交给 packHasSkillMD，再逐个写入临时目录。
// rec 的 Source 为空时改成 HubWorkBuddy，再写入 .market.json。
func InstallRecord(homeSkills, slug string, scan Scan, files []PackFile, rec Record) (string, error) {
	slug = SanitizeSlug(slug)
	if slug == "" {
		return "", fmt.Errorf("invalid skill slug")
	}
	if !scan.OK {
		return "", fmt.Errorf("scan blocked: %s", stringsJoin(scan.Reasons))
	}
	if !packHasSkillMD(files) {
		return "", fmt.Errorf("pack missing SKILL.md")
	}
	dir := filepath.Join(homeSkills, slug)
	tmp := dir + ".install-tmp"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	for _, f := range files {
		rel := filepath.FromSlash(f.Rel)
		if !validPackRel(rel) {
			return "", fmt.Errorf("invalid pack path %s", f.Rel)
		}
		dst := filepath.Join(tmp, rel)
		if !withinDir(tmp, dst) {
			return "", fmt.Errorf("pack path escapes skill directory")
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", err
		}
		mode := os.FileMode(0o644)
		if isScriptRel(f.Rel) {
			mode = 0o755
		}
		if err := os.WriteFile(dst, f.Data, mode); err != nil {
			return "", err
		}
	}
	if rec.Source == "" {
		rec.Source = HubWorkBuddy
	}
	if rec.Catalog == "" && rec.Source == HubWorkBuddy {
		rec.Catalog = "infometa/workbuddyskills"
	}
	if rec.URL == "" && rec.Source == HubWorkBuddy {
		rec.URL = SkillURL(slug)
	}
	rec.Slug = slug
	rec.Name = scan.Name
	rec.Icon = scan.Icon
	rec.Files = packRels(files)
	rec.Installed = time.Now().UTC()
	rec.Scan = scan
	b, _ := json.MarshalIndent(rec, "", "  ")
	if err := os.WriteFile(filepath.Join(tmp, ".market.json"), b, 0o644); err != nil {
		return "", err
	}
	_ = os.RemoveAll(dir)
	if err := os.Rename(tmp, dir); err != nil {
		return "", err
	}
	return dir, nil
}

// Uninstall 删掉 ResolveInstall 找到的目录。没有 .market.json 时返回 "not a market install"。
// homeSkills 和 name 一起交给 ResolveInstall。
// name 和 homeSkills 一起交给 ResolveInstall。
func Uninstall(homeSkills, name string) error {
	dir, err := ResolveInstall(homeSkills, name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, ".market.json")); err != nil {
		return fmt.Errorf("not a market install")
	}
	return os.RemoveAll(dir)
}

// ResolveInstall 先按净化后的目录名找带 .market.json 的安装。找不到再按记录里的 Name 或 Slug 扫 homeSkills。都没有返回 "not a market install"。
// name 先交给 SanitizeSlug 当目录名；对不上时再和记录里的 Name、Slug 比较，去掉两端空白后为空则返回错误。
func ResolveInstall(homeSkills, name string) (string, error) {
	slug := SanitizeSlug(name)
	if slug != "" {
		dir := filepath.Join(homeSkills, slug)
		if _, err := os.Stat(filepath.Join(dir, ".market.json")); err == nil {
			return dir, nil
		}
	}
	want := strings.TrimSpace(name)
	if want == "" {
		return "", fmt.Errorf("invalid skill slug")
	}
	ents, err := os.ReadDir(homeSkills)
	if err != nil {
		return "", fmt.Errorf("not a market install")
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(homeSkills, e.Name())
		b, err := os.ReadFile(filepath.Join(dir, ".market.json"))
		if err != nil {
			continue
		}
		var rec Record
		if json.Unmarshal(b, &rec) != nil {
			continue
		}
		if rec.Name == want || rec.Slug == want {
			return dir, nil
		}
	}
	return "", fmt.Errorf("not a market install")
}

// ReadRecord 读取目录下的 .market.json。读失败或 JSON 无效时返回 false。
// dir 拼上 .market.json 后交给 os.ReadFile。
func ReadRecord(dir string) (Record, bool) {
	var rec Record
	b, err := os.ReadFile(filepath.Join(dir, ".market.json"))
	if err != nil || json.Unmarshal(b, &rec) != nil {
		return Record{}, false
	}
	return rec, true
}

// PackIncomplete 在正文提到 scripts/，且 scripts 目录打不开或里面没有非点开头的文件时返回 true。
// dir 为空时返回 false；否则和 scripts 拼成目录去读。
// body 交给 MentionsScripts，没有提到脚本时返回 false。
func PackIncomplete(dir, body string) bool {
	if dir == "" || !MentionsScripts(body) {
		return false
	}
	ents, err := os.ReadDir(filepath.Join(dir, "scripts"))
	if err != nil {
		return true
	}
	for _, e := range ents {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		return false
	}
	return true
}

// validPackRel 拒绝空路径、绝对路径和逃出目录的路径，其余交给 packAllowed。
// rel 为空或是绝对路径时返回 false，Clean 之后是 . 或以 .. 开头也返回 false，否则交给 packAllowed。
func validPackRel(rel string) bool {
	if rel == "" || filepath.IsAbs(rel) {
		return false
	}
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, "..") {
		return false
	}
	return packAllowed(filepath.ToSlash(rel))
}

// withinDir 判断 p 是否仍在 root 之内。
func withinDir(root, p string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// isScriptRel 在扩展名是 sh、py、ps1、rb、js、mjs、ts，或无扩展名且位于 scripts/ 下时返回 true。
// rel 转成小写斜杠路径后看扩展名，或无扩展名且以 scripts/ 开头。
func isScriptRel(rel string) bool {
	rel = strings.ToLower(filepath.ToSlash(rel))
	switch filepath.Ext(rel) {
	case ".sh", ".py", ".ps1", ".rb", ".js", ".mjs", ".ts":
		return true
	}
	return filepath.Ext(rel) == "" && strings.HasPrefix(rel, "scripts/")
}

// stringsJoin 用 "; " 连接字符串。
func stringsJoin(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += "; "
		}
		out += v
	}
	return out
}
