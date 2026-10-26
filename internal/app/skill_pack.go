// 方法论包和领域包的安装与开关。skillpack.Enabled 在没有工作区覆盖、也没有 config.yaml 的 packs 项时返回 false。InstallPack 在工作区可用时把该包的工作区开关写成 true，否则把 Config.Packs 的 Enabled 写成 true。本文件不改 refs。

package app

import (
	"fmt"
	"runtime"
	"strings"

	rt "github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/skillpack"
)

// PackPref 是 config.yaml 里 packs.<id> 的偏好，类型同 skillpack.Pref。
type PackPref = skillpack.Pref

// skillRoots 用 packPrefs 取出已启用包的技能目录，再交给 runtime.SkillRoots，并带上捆绑技能目录。
func (a *App) skillRoots(workspace string) []string {
	home := ""
	if a != nil && a.Home != nil {
		home = a.Home.Root
	}
	packs := skillpack.EnabledSkillDirs(home, workspace, a.packPrefs())
	return rt.SkillRoots(home, workspace, bundledSkillsDir(a.BundledEvals), packs...)
}

// packPrefs 返回 Config.Packs。App 为空时返回 nil。
func (a *App) packPrefs() map[string]skillpack.Pref {
	if a == nil {
		return nil
	}
	return a.Config.Packs
}

// packSessions 把 skillpack.RuntimeFor 的结果抄成 runtime.PackSession，带上 BootstrapSkill、Mapping 和 Methodology。
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

// ListPacks 列出家目录和工作区里的包状态。传入的工作区不可用时改用 App.Workspace。
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

// InstallPack 按 id 安装包。localPath 非空时来源是本地目录。工作区可用则 SetWorkspaceEnabled true，否则把 Config.Packs 该项 Enabled 写成 true 并 SaveConfig。id 空白返回 invalid pack id。
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

// UninstallPack 从家目录卸下该包。Home 为空返回 no home。
func (a *App) UninstallPack(id string) error {
	if a == nil || a.Home == nil {
		return fmt.Errorf("no home")
	}
	return skillpack.Uninstall(a.Home.Root, id)
}

// EnablePack 按 scope 改开关。空 scope 当作工作区。inherit 清除工作区覆盖，workspace 写入工作区开关，global 写入 Config.Packs 并 SaveConfig。工作区不可用时 inherit 和 workspace 返回 workspace not ready。
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

// UpdatePack 用已安装清单里的 Origin 再安装一次。未安装返回 pack is not installed。
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
