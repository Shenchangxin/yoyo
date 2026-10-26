// 读取工作区和全局开关，汇总已安装技能包是否启用。

package skillpack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

var errInvalidID = fmt.Errorf("invalid pack id")

// ReadWorkspace 读取 <workspace>/.yoyo/packs.json。文件不存在或 JSON 无效时返回空表。
func ReadWorkspace(workspace string) map[string]WorkspacePref {
	out := map[string]WorkspacePref{}
	b, err := os.ReadFile(WorkspaceFile(workspace))
	if err != nil {
		return out
	}
	_ = json.Unmarshal(b, &out)
	return out
}

// WriteWorkspace 把偏好写回 <workspace>/.yoyo/packs.json。workspace 为空白时返回 "empty workspace"。prefs 为 nil 时写成空对象。
func WriteWorkspace(workspace string, prefs map[string]WorkspacePref) error {
	if strings.TrimSpace(workspace) == "" {
		return fmt.Errorf("empty workspace")
	}
	dir := filepath.Join(workspace, ".yoyo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if prefs == nil {
		prefs = map[string]WorkspacePref{}
	}
	b, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(WorkspaceFile(workspace), b, 0o644)
}

// Enabled 先看工作区偏好：该 id 有条目就用它的 Enabled，假也算数。否则看全局偏好。两处都没有则返回 false。
// global 在工作区没有该 id 时再查；为 nil 则不再查。
// workspace 有该 id 时直接返回它的 Enabled。
func Enabled(id string, global map[string]Pref, workspace map[string]WorkspacePref) bool {
	id = SanitizeID(id)
	if wp, ok := workspace[id]; ok {
		return wp.Enabled
	}
	if global != nil {
		if p, ok := global[id]; ok {
			return p.Enabled
		}
	}
	return false
}

// SetWorkspaceEnabled 把该 id 的工作区开关写成 enabled。id 净化后为空则返回 invalid pack id。
// workspace 交给 ReadWorkspace，写回时再交给 WriteWorkspace。
func SetWorkspaceEnabled(workspace, id string, enabled bool) error {
	id = SanitizeID(id)
	if id == "" {
		return errInvalidID
	}
	prefs := ReadWorkspace(workspace)
	prefs[id] = WorkspacePref{Enabled: enabled}
	return WriteWorkspace(workspace, prefs)
}

// ClearWorkspaceEnabled 删掉该 id 的工作区条目，之后 Enabled 回落到全局偏好或 false。
// workspace 交给 ReadWorkspace，删掉后再交给 WriteWorkspace。
func ClearWorkspaceEnabled(workspace, id string) error {
	id = SanitizeID(id)
	if id == "" {
		return errInvalidID
	}
	prefs := ReadWorkspace(workspace)
	delete(prefs, id)
	return WriteWorkspace(workspace, prefs)
}

// List 先按 Catalog 的顺序给出已知包，再追加磁盘上已安装但不在目录里的 id，后者按 id 排序。
// home 交给 InstalledIDs 和 statusOf。
// workspace 交给 ReadWorkspace，得到的表再交给 statusOf。
// global 交给 statusOf。
func List(home, workspace string, global map[string]Pref) []Status {
	wsPrefs := ReadWorkspace(workspace)
	seen := map[string]bool{}
	var out []Status
	for _, k := range Catalog() {
		st := statusOf(home, workspace, global, wsPrefs, k.ID, k)
		seen[st.ID] = true
		out = append(out, st)
	}
	var extra []Status
	for _, id := range InstalledIDs(home) {
		if seen[id] {
			continue
		}
		seen[id] = true
		extra = append(extra, statusOf(home, workspace, global, wsPrefs, id, Known{}))
	}
	sort.Slice(extra, func(i, j int) bool { return extra[i].ID < extra[j].ID })
	return append(out, extra...)
}

// statusOf 合并内置目录和 pack.json。已安装时 Root 是 home/packs/<id>，SkillsDir 是其中的 skills。Enabled 为真还要求已经安装。
// workspace 本函数不读取。
// global 非 nil 时按 id 取出 EnableGlobal，并交给 Enabled。
// wsPrefs 按 id 取出 EnableWorkspace，并交给 Enabled。
// known 的 ID 非空时写入名称、说明、许可证、引导技能、方法论和来源。
func statusOf(home, workspace string, global map[string]Pref, wsPrefs map[string]WorkspacePref, id string, known Known) Status {
	id = SanitizeID(id)
	st := Status{ID: id}
	if known.ID != "" {
		st.Name = known.Name
		st.Description = known.Description
		st.License = known.License
		st.BootstrapSkill = known.BootstrapSkill
		st.Methodology = known.Methodology
		st.Known = true
		st.Origin = Origin{Kind: KindGitHub, Repo: known.DefaultRepo, Ref: known.DefaultRef, SkillsRel: known.SkillsRel}
	}
	if m, ok := ReadManifest(home, id); ok {
		st.Installed = true
		st.Name = firstNonEmpty(m.Name, st.Name, id)
		st.Description = firstNonEmpty(m.Description, st.Description)
		st.License = firstNonEmpty(m.License, st.License)
		st.Version = m.Version
		st.Commit = m.Commit
		st.BootstrapSkill = firstNonEmpty(m.BootstrapSkill, st.BootstrapSkill)
		st.Methodology = st.Methodology || m.Methodology
		st.Origin = m.Origin
		st.InstalledAt = m.InstalledAt
		st.SkillCount = m.SkillCount
		st.Root = Dir(home, id)
		st.SkillsDir = SkillsDir(home, id)
	}
	if k, ok := Lookup(id); ok && !st.Known {
		st.Known = true
		st.Methodology = st.Methodology || k.Methodology
		st.BootstrapSkill = firstNonEmpty(st.BootstrapSkill, k.BootstrapSkill)
	}
	if st.Name == "" {
		st.Name = id
	}
	if st.Installed && st.SkillsDir != "" {
		names := skillNames(st.SkillsDir)
		st.Skills = names
		if st.SkillCount == 0 {
			st.SkillCount = len(names)
		}
	}
	if global != nil {
		if p, ok := global[id]; ok {
			st.EnableGlobal = p.Enabled
		}
	}
	if wp, ok := wsPrefs[id]; ok {
		v := wp.Enabled
		st.EnableWorkspace = &v
	}
	st.Enabled = Enabled(id, global, wsPrefs) && st.Installed
	return st
}

// skillNames 走目录收集每个 SKILL.md 的技能名，排序后返回。解析失败的文件跳过。
// skillsDir 交给 filepath.Walk，只收集名为 SKILL.md 的文件。
func skillNames(skillsDir string) []string {
	var names []string
	_ = filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || info.Name() != "SKILL.md" {
			return nil
		}
		sk, err := artifact.LoadSkillFile(path)
		if err != nil {
			return nil
		}
		names = append(names, sk.Name)
		return nil
	})
	sort.Strings(names)
	return names
}

// EnabledSkillDirs 返回已启用且技能目录非空的路径。
// home 和 workspace、global 一起交给 List。
// workspace 和 home、global 一起交给 List。
// global 和 home、workspace 一起交给 List。
func EnabledSkillDirs(home, workspace string, global map[string]Pref) []string {
	var out []string
	for _, st := range List(home, workspace, global) {
		if st.Enabled && st.SkillsDir != "" {
			out = append(out, st.SkillsDir)
		}
	}
	return out
}

// RuntimeFor 收集已启用的包，并带上 ToolMapping。方法论包若没有 BootstrapSkill 则跳过。领域包 Methodology 为假，没有 BootstrapSkill 也会进入结果。
// home 和 workspace、global 一起交给 List。
// workspace 和 home、global 一起交给 List。
// global 和 home、workspace 一起交给 List。
// goos 交给 ToolMapping。
func RuntimeFor(home, workspace string, global map[string]Pref, goos string) []Session {
	var out []Session
	for _, st := range List(home, workspace, global) {
		if !st.Enabled {
			continue
		}
		// 方法论包没有 BootstrapSkill 时跳过。领域包 Methodology 为假，没有 BootstrapSkill 也会进入会话。
		if st.Methodology && st.BootstrapSkill == "" {
			continue
		}
		out = append(out, Session{
			ID:             st.ID,
			BootstrapSkill: st.BootstrapSkill,
			Mapping:        ToolMapping(st.ID, goos),
			Methodology:    st.Methodology,
		})
	}
	return out
}

// firstNonEmpty 返回第一段去掉空白后仍非空的字符串。都空则返回空串。
// v 按顺序去掉空白，第一段仍非空的原样返回。
func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
