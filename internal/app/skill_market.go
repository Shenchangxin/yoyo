package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/hostopen"
	"github.com/Shenchangxin/yoyo/internal/skillmarket"
)

func (a *App) SkillMarket(refresh bool, hub, query string) (skillmarket.Catalog, error) {
	dir := filepath.Join(a.Home.Root, "skill-market")
	return skillmarket.LoadHub(dir, skillmarket.LoadOpts{Hub: hub, Query: query, Refresh: refresh})
}

func (a *App) InstallMarketSkill(slug string) (map[string]any, error) {
	hub, remote := skillmarket.ParseInstallID(slug)
	local := skillmarket.InstallSlug(remote)
	if local == "" {
		return nil, fmt.Errorf("invalid skill slug")
	}
	pack, err := skillmarket.FetchFrom(hub, remote)
	if err != nil {
		return nil, err
	}
	files, warn := pack.Files, pack.Warn
	raw := packSkillMD(files)
	if len(raw) == 0 {
		return nil, fmt.Errorf("pack missing SKILL.md")
	}
	skParsed, scan := skillmarket.ScanSkillMD(raw, local)
	if !scan.OK {
		return map[string]any{"scan": scan, "ok": false}, fmt.Errorf("scan blocked: %s", strings.Join(scan.Reasons, "; "))
	}
	scan.Files = len(files)
	if skillmarket.MentionsScripts(skParsed.Body) && !packHasScripts(files) {
		if warn == "" {
			warn = "skill references scripts/ but the downloaded pack has none"
		}
	}
	info, _ := skillmarket.LookupHub(hub)
	rec := skillmarket.Record{
		Source:  hub,
		Catalog: info.Name,
		URL:     pack.URL,
		Remote:  remote,
	}
	if pack.Catalog != "" {
		rec.Catalog = pack.Catalog
	}
	dir, err := skillmarket.InstallRecord(a.Home.Skills(), local, scan, files, rec)
	if err != nil {
		return nil, err
	}
	mcpURL := ""
	if skParsed.Metadata != nil {
		mcpURL = strings.TrimSpace(skParsed.Metadata["mcp"])
		if mcpURL == "" {
			mcpURL = strings.TrimSpace(skParsed.Metadata["mcp_url"])
		}
	}
	mcpNote := ""
	if mcpURL != "" {
		if err := a.StartMCPHTTP(local, mcpURL); err != nil {
			mcpNote = err.Error()
		} else {
			mcpNote = "mcp started"
		}
	}
	sk := a.GetSkill("", scan.Name)
	if sk == nil {
		sk = a.GetSkill("", local)
	}
	return map[string]any{
		"ok":      true,
		"dir":     dir,
		"scan":    scan,
		"skill":   sk,
		"files":   skillmarketFiles(files),
		"warning": warn,
		"mcp":     mcpNote,
		"bundle":  skillBundleNote(skParsed.Metadata),
	}, nil
}

func (a *App) UninstallMarketSkill(slug string) error {
	return skillmarket.Uninstall(a.Home.Skills(), slug)
}

func skillBundleNote(meta map[string]string) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(meta["connectors"])
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

func skillInstallID(dir string) string {
	rec, ok := skillmarket.ReadRecord(dir)
	if !ok {
		return ""
	}
	if rec.Remote != "" && rec.Source != "" {
		return skillmarket.ItemID(rec.Source, rec.Remote)
	}
	if rec.Slug != "" && rec.Source != "" && rec.Source != skillmarket.HubWorkBuddy {
		return skillmarket.ItemID(rec.Source, rec.Slug)
	}
	return rec.Slug
}

func skillOrigin(dir, home, bundled string) string {
	dir = filepath.Clean(dir)
	if bundled != "" && pathUnder(bundled, dir) {
		return "bundled"
	}
	if home != "" && pathUnder(filepath.Join(home, "packs"), dir) {
		return "pack"
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

func skillPackID(dir, home string) string {
	if home == "" || dir == "" {
		return ""
	}
	root := filepath.Join(home, "packs")
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(dir))
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || strings.HasPrefix(rel, "..") {
		return ""
	}
	id, _, _ := strings.Cut(rel, "/")
	return id
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

func (a *App) OpenURL(raw string) error {
	return hostopen.URL(raw)
}

func (a *App) OpenInEditor(path string) error {
	return hostopen.Editor(path)
}

func (a *App) OpenWorkspaceTerminal(dir string) error {
	return hostopen.Terminal(dir)
}
