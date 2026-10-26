// Package skillpack 安装并开关技能包。Enabled 在工作区偏好和全局偏好都没有该 id 时返回 false，默认关闭。安装根目录是 home/packs/<id>，技能文件在其中的 skills/，清单是同级的 pack.json。工作区开关写在 <workspace>/.yoyo/packs.json。
package skillpack

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// scanFile 拒绝空文件。png、jpg、jpeg、gif、webp、ico 不再检查正文。含 NUL 返回 "binary content"，非法 UTF-8 返回 "invalid UTF-8"。正文同时出现 "curl " 和 "| sh" 或 "| bash" 时返回 "install script pipe-to-shell"。
// rel 的扩展名转成小写后，是 png、jpg、jpeg、gif、webp、ico 则不再检查正文。
// raw 长度为 0 时返回错误；不是上述图片时检查 NUL、UTF-8，以及 curl 和管道到 sh 或 bash。
func scanFile(rel string, raw []byte) error {
	if len(raw) == 0 {
		return fmt.Errorf("empty file")
	}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico":
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

// InstallOptions 选择安装来源。Origin 为空时改用内置目录里的仓库。
type InstallOptions struct {
	Origin Origin
	GOOS   string
}

// Install 把技能文件写到 home/packs/<id>/skills，并写同级 pack.json。未知 id 且 Origin 没有 Kind、Path、Repo 时返回 "unknown pack"。Kind 为空时，有 Path 视为 local，否则视为 github。local 读本地目录，github 拉远程树。成功后按 GOOS 生成 references/yoyo-tools.md。
// opts 的 Origin 决定本地还是 GitHub；GOOS 交给 writeGeneratedMapping。
func Install(home, id string, opts InstallOptions) (Manifest, error) {
	id = SanitizeID(id)
	known, ok := Lookup(id)
	if !ok && opts.Origin.Kind == "" && opts.Origin.Path == "" && opts.Origin.Repo == "" {
		return Manifest{}, fmt.Errorf("unknown pack %s", id)
	}
	origin := opts.Origin
	if origin.Kind == "" {
		if strings.TrimSpace(origin.Path) != "" {
			origin.Kind = KindLocal
		} else {
			origin.Kind = KindGitHub
		}
	}
	if origin.Kind == KindGitHub {
		if origin.Repo == "" && ok {
			origin.Repo = known.DefaultRepo
		}
		if origin.Ref == "" && ok {
			origin.Ref = known.DefaultRef
		}
		if origin.SkillsRel == "" && ok {
			origin.SkillsRel = known.SkillsRel
		}
	}
	var files []PackFile
	commit := ""
	var err error
	switch origin.Kind {
	case KindLocal:
		files, err = readLocal(origin.Path, firstNonEmpty(origin.SkillsRel, known.SkillsRel, "skills"))
	case KindGitHub:
		files, commit, err = fetchGitHub(origin.Repo, origin.Ref, firstNonEmpty(origin.SkillsRel, "skills"))
	default:
		return Manifest{}, fmt.Errorf("unsupported origin %s", origin.Kind)
	}
	if err != nil {
		return Manifest{}, err
	}
	if err := writePackTree(home, id, files); err != nil {
		return Manifest{}, err
	}
	if err := writeGeneratedMapping(home, id, opts.GOOS); err != nil {
		return Manifest{}, err
	}
	count := 0
	for _, f := range files {
		if strings.EqualFold(pathBase(f.Rel), "skill.md") {
			count++
		}
	}
	m := Manifest{
		ID:             id,
		Name:           firstNonEmpty(known.Name, id),
		Description:    known.Description,
		License:        known.License,
		BootstrapSkill: known.BootstrapSkill,
		Methodology:    known.Methodology,
		Origin:         origin,
		Commit:         commit,
		SkillCount:     count,
		InstalledAt:    time.Now().UTC(),
	}
	if m.BootstrapSkill == "" && id == SuperpowersID {
		m.BootstrapSkill = "using-superpowers"
	}
	if err := WriteManifest(home, m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// Uninstall 删掉 home/packs/<id>。id 净化后为空返回 invalid pack id。没有 pack.json 时返回 "pack <id> is not installed"。
func Uninstall(home, id string) error {
	id = SanitizeID(id)
	if id == "" {
		return errInvalidID
	}
	dir := Dir(home, id)
	if _, err := os.Stat(ManifestPath(home, id)); err != nil {
		return fmt.Errorf("pack %s is not installed", id)
	}
	return os.RemoveAll(dir)
}

// writePackTree 先把文件写到 skills 目录加 .install-tmp 的临时目录，再删掉旧的 skills 目录并改名换上。脚本路径用 0o755，其余用 0o644。路径不合法或逃出临时目录则返回错误。
// home 和 id 一起交给 SkillsDir。
// id 和 home 一起交给 SkillsDir。
// files 逐个检查 validRel 后写入临时目录。
func writePackTree(home, id string, files []PackFile) error {
	dst := SkillsDir(home, id)
	tmp := dst + ".install-tmp"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	for _, f := range files {
		if !validRel(f.Rel) {
			return fmt.Errorf("invalid pack path %s", f.Rel)
		}
		p := filepath.Join(tmp, filepath.FromSlash(f.Rel))
		if !withinDir(tmp, p) {
			return fmt.Errorf("pack path escapes %s", f.Rel)
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if isScriptRel(f.Rel) {
			mode = 0o755
		}
		if err := os.WriteFile(p, f.Data, mode); err != nil {
			return err
		}
	}
	parent := filepath.Dir(dst)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	_ = os.RemoveAll(dst)
	return os.Rename(tmp, dst)
}

// writeGeneratedMapping 在选中的技能目录 references/yoyo-tools.md 写入 MappingFile。没有目标技能目录时什么也不写。
// home 和 id 交给 mappingTarget，并用来拼 references 目录。
// id 和 goos 一起交给 MappingFile。
// goos 和 id 一起交给 MappingFile。
func writeGeneratedMapping(home, id, goos string) error {
	name := mappingTarget(home, id)
	if name == "" {
		return nil
	}
	dir := filepath.Join(SkillsDir(home, id), name, "references")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "yoyo-tools.md"), []byte(MappingFile(id, goos)), 0o644)
}

// mappingTarget 选择接收 yoyo-tools.md 的技能目录。先看内置目录的 BootstrapSkill，再看与包 id 同名的目录，以存在 SKILL.md 为准。都没有则返回空串。
// home 和 id 一起交给 SkillsDir，用来确认 SKILL.md 是否存在。
func mappingTarget(home, id string) string {
	id = SanitizeID(id)
	known, _ := Lookup(id)
	for _, name := range []string{known.BootstrapSkill, id} {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(SkillsDir(home, id), name, "SKILL.md")); err == nil {
			return name
		}
	}
	return ""
}

// readLocal 从本地目录收集文件。root 下若有 skillsRel 子目录就从那里走。跳过点开头的目录和不允许的相对路径。单文件超过 1 MiB、总字节超过 16 MiB 或文件数达到 400 即失败。没有 SKILL.md 返回 "local pack has no SKILL.md"。每个 SKILL.md 都要能被 ParseSkillMD 解析。
func readLocal(root, skillsRel string) ([]PackFile, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("empty local path")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("local pack path is not a directory")
	}
	base := abs
	skillsRel = strings.Trim(strings.ReplaceAll(skillsRel, "\\", "/"), "/")
	cand := filepath.Join(abs, filepath.FromSlash(skillsRel))
	if st, err := os.Stat(cand); err == nil && st.IsDir() {
		base = cand
	}
	var files []PackFile
	total := 0
	err = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !validRel(rel) {
			return nil
		}
		if info.Size() > maxFileBytes {
			return fmt.Errorf("%s larger than 1MiB", rel)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := scanFile(rel, data); err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		total += len(data)
		if total > maxPackBytes {
			return fmt.Errorf("pack larger than 16MiB")
		}
		if len(files) >= maxPackFiles {
			return fmt.Errorf("pack has more than %d files", maxPackFiles)
		}
		files = append(files, PackFile{Rel: rel, Data: data})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !packHasSkillMD(files) {
		return nil, fmt.Errorf("local pack has no SKILL.md under %s", base)
	}
	for _, f := range files {
		if strings.EqualFold(pathBase(f.Rel), "skill.md") {
			if _, err := artifact.ParseSkillMD(string(f.Data), f.Rel); err != nil {
				return nil, fmt.Errorf("%s: %w", f.Rel, err)
			}
		}
	}
	return files, nil
}

// withinDir 判断 p 是否仍在 root 之内。相对路径是 .. 或以 .. 加分隔符开头时返回 false。
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
