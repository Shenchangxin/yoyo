package app

import (
	"fmt"
	"runtime"
	"strings"

	rt "github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/skillpack"
)

// PackPref is config.yaml packs.<id>.
type PackPref = skillpack.Pref

func (a *App) skillRoots(workspace string) []string {
	home := ""
	if a != nil && a.Home != nil {
		home = a.Home.Root
	}
	packs := skillpack.EnabledSkillDirs(home, workspace, a.packPrefs())
	return rt.SkillRoots(home, workspace, bundledSkillsDir(a.BundledEvals), packs...)
}

func (a *App) packPrefs() map[string]skillpack.Pref {
	if a == nil {
		return nil
	}
	return a.Config.Packs
}

func (a *App) packSessions(workspace string) []rt.PackSession {
	home := ""
	if a != nil && a.Home != nil {
		home = a.Home.Root
	}
	raw := skillpack.RuntimeFor(home, workspace, a.packPrefs(), runtime.GOOS)
	out := make([]rt.PackSession, 0, len(raw))
	for _, s := range raw {
		out = append(out, rt.PackSession{
			ID:             s.ID,
			BootstrapSkill: s.BootstrapSkill,
			Mapping:        s.Mapping,
			Methodology:    s.Methodology,
		})
	}
	return out
}

func (a *App) ListPacks(workspace string) []skillpack.Status {
	home := ""
	if a != nil && a.Home != nil {
		home = a.Home.Root
	}
	ws := strings.TrimSpace(workspace)
	if !WorkspaceReady(ws) {
		ws = a.Workspace()
	}
	return skillpack.List(home, ws, a.packPrefs())
}

func (a *App) InstallPack(id, localPath string) (skillpack.Manifest, error) {
	if a == nil || a.Home == nil {
		return skillpack.Manifest{}, fmt.Errorf("no home")
	}
	id = skillpack.SanitizeID(id)
	if id == "" {
		return skillpack.Manifest{}, fmt.Errorf("invalid pack id")
	}
	opts := skillpack.InstallOptions{GOOS: runtime.GOOS}
	if strings.TrimSpace(localPath) != "" {
		opts.Origin = skillpack.Origin{Kind: skillpack.KindLocal, Path: localPath}
	}
	m, err := skillpack.Install(a.Home.Root, id, opts)
	if err != nil {
		return m, err
	}
	ws := a.Workspace()
	if WorkspaceReady(ws) {
		if err := skillpack.SetWorkspaceEnabled(ws, id, true); err != nil {
			return m, err
		}
	} else {
		if a.Config.Packs == nil {
			a.Config.Packs = map[string]PackPref{}
		}
		a.Config.Packs[id] = PackPref{Enabled: true}
		if err := a.SaveConfig(); err != nil {
			return m, err
		}
	}
	return m, nil
}

func (a *App) UninstallPack(id string) error {
	if a == nil || a.Home == nil {
		return fmt.Errorf("no home")
	}
	return skillpack.Uninstall(a.Home.Root, id)
}

func (a *App) EnablePack(id, scope string, enabled bool) error {
	id = skillpack.SanitizeID(id)
	if id == "" {
		return fmt.Errorf("invalid pack id")
	}
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		scope = skillpack.ScopeWorkspace
	}
	switch scope {
	case skillpack.ScopeInherit:
		ws := a.Workspace()
		if !WorkspaceReady(ws) {
			return fmt.Errorf("workspace not ready")
		}
		return skillpack.ClearWorkspaceEnabled(ws, id)
	case skillpack.ScopeWorkspace:
		ws := a.Workspace()
		if !WorkspaceReady(ws) {
			return fmt.Errorf("workspace not ready")
		}
		return skillpack.SetWorkspaceEnabled(ws, id, enabled)
	case skillpack.ScopeGlobal:
		if a.Config.Packs == nil {
			a.Config.Packs = map[string]PackPref{}
		}
		a.Config.Packs[id] = PackPref{Enabled: enabled}
		return a.SaveConfig()
	default:
		return fmt.Errorf("unknown pack scope %s", scope)
	}
}

func (a *App) UpdatePack(id string) (skillpack.Manifest, error) {
	if a == nil || a.Home == nil {
		return skillpack.Manifest{}, fmt.Errorf("no home")
	}
	id = skillpack.SanitizeID(id)
	m, ok := skillpack.ReadManifest(a.Home.Root, id)
	if !ok {
		return skillpack.Manifest{}, fmt.Errorf("pack %s is not installed", id)
	}
	return skillpack.Install(a.Home.Root, id, skillpack.InstallOptions{Origin: m.Origin, GOOS: runtime.GOOS})
}
