package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/hostopen"
	"github.com/Shenchangxin/yoyo/internal/skillmarket"
)

func (a *App) SkillMarket(refresh bool) (skillmarket.Catalog, error) {
	dir := filepath.Join(a.Home.Root, "skill-market")
	return skillmarket.Load(dir, refresh)
}

func (a *App) InstallMarketSkill(slug string) (map[string]any, error) {
	slug = skillmarket.SanitizeSlug(slug)
	if slug == "" {
		return nil, fmt.Errorf("invalid skill slug")
	}
	files, warn, err := skillmarket.FetchPack(slug)
	if err != nil {
		return nil, err
	}
	raw := packSkillMD(files)
	if len(raw) == 0 {
		return nil, fmt.Errorf("pack missing SKILL.md")
	}
	skParsed, scan := skillmarket.ScanSkillMD(raw, slug)
	if !scan.OK {
		return map[string]any{"scan": scan, "ok": false}, fmt.Errorf("scan blocked: %s", strings.Join(scan.Reasons, "; "))
	}
	scan.Files = len(files)
	if skillmarket.MentionsScripts(skParsed.Body) && !packHasScripts(files) {
		if warn == "" {
			warn = "skill references scripts/ but the downloaded pack has none"
		}
	}
	dir, err := skillmarket.Install(a.Home.Skills(), slug, scan, files)
	if err != nil {
		return nil, err
	}
	sk := a.GetSkill("", scan.Name)
	if sk == nil {
		sk = a.GetSkill("", slug)
	}
	return map[string]any{
		"ok":      true,
		"dir":     dir,
		"scan":    scan,
		"skill":   sk,
		"files":   skillmarketFiles(files),
		"warning": warn,
	}, nil
}

func (a *App) UninstallMarketSkill(slug string) error {
	return skillmarket.Uninstall(a.Home.Skills(), slug)
}

func packSkillMD(files []skillmarket.PackFile) []byte {
	for _, f := range files {
		if strings.EqualFold(f.Rel, "SKILL.md") {
			return f.Data
		}
	}
	return nil
}

func packHasScripts(files []skillmarket.PackFile) bool {
	for _, f := range files {
		rel := strings.ToLower(strings.ReplaceAll(f.Rel, "\\", "/"))
		if strings.HasPrefix(rel, "scripts/") {
			return true
		}
	}
	return false
}

func skillmarketFiles(files []skillmarket.PackFile) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Rel)
	}
	return out
}

func skillOrigin(dir, home, bundled string) string {
	dir = filepath.Clean(dir)
	if bundled != "" && pathUnder(bundled, dir) {
		return "bundled"
	}
	if home != "" && pathUnder(filepath.Join(home, "skills"), dir) {
		if _, err := os.Stat(filepath.Join(dir, ".market.json")); err == nil {
			return "market"
		}
		return "home"
	}
	if dir != "" && dir != "." {
		return "workspace"
	}
	return "cas"
}

func pathUnder(parent, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && !strings.HasPrefix(rel, ".."))
}

func (a *App) OpenPath(path string) error {
	return hostopen.Path(path)
}

func (a *App) OpenInEditor(path string) error {
	return hostopen.Editor(path)
}

func (a *App) OpenWorkspaceTerminal(dir string) error {
	return hostopen.Terminal(dir)
}
