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

// InstallOptions selects the source. Empty Origin uses the known catalog.
type InstallOptions struct {
	Origin Origin
	GOOS   string
}

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

// mappingTarget is the skill folder that receives install-time yoyo-tools.md.
// Methodology packs prefer the bootstrap skill; domain packs use a skill
// whose name matches the pack id. Missing folder → no adapter file.
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
