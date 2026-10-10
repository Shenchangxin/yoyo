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

func ReadWorkspace(workspace string) map[string]WorkspacePref {
	out := map[string]WorkspacePref{}
	b, err := os.ReadFile(WorkspaceFile(workspace))
	if err != nil {
		return out
	}
	_ = json.Unmarshal(b, &out)
	return out
}

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

// Enabled is workspace override > global config > default false.
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

func SetWorkspaceEnabled(workspace, id string, enabled bool) error {
	id = SanitizeID(id)
	if id == "" {
		return errInvalidID
	}
	prefs := ReadWorkspace(workspace)
	prefs[id] = WorkspacePref{Enabled: enabled}
	return WriteWorkspace(workspace, prefs)
}

func ClearWorkspaceEnabled(workspace, id string) error {
	id = SanitizeID(id)
	if id == "" {
		return errInvalidID
	}
	prefs := ReadWorkspace(workspace)
	delete(prefs, id)
	return WriteWorkspace(workspace, prefs)
}

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

func EnabledSkillDirs(home, workspace string, global map[string]Pref) []string {
	var out []string
	for _, st := range List(home, workspace, global) {
		if st.Enabled && st.SkillsDir != "" {
			out = append(out, st.SkillsDir)
		}
	}
	return out
}

func RuntimeFor(home, workspace string, global map[string]Pref, goos string) []Session {
	var out []Session
	for _, st := range List(home, workspace, global) {
		if !st.Enabled || st.BootstrapSkill == "" {
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

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
