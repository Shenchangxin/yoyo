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
	raw, err := skillmarket.Get(skillmarket.SkillURL(slug))
	if err != nil {
		return nil, err
	}
	_, scan := skillmarket.ScanSkillMD(raw, slug)
	if !scan.OK {
		return map[string]any{"scan": scan, "ok": false}, fmt.Errorf("scan blocked: %s", strings.Join(scan.Reasons, "; "))
	}
	dir, err := skillmarket.Install(a.Home.Skills(), slug, scan, raw)
	if err != nil {
		return nil, err
	}
	sk := a.GetSkill("", slug)
	return map[string]any{
		"ok":   true,
		"dir":  dir,
		"scan": scan,
		"skill": sk,
	}, nil
}

func (a *App) UninstallMarketSkill(slug string) error {
	return skillmarket.Uninstall(a.Home.Skills(), slug)
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
