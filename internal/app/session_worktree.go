// 会话的工作区副本和钉住的技能。Isolate 只表示这份副本，不是进程隔离。

package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// uniqueStrings 按出现顺序合并多组字符串，去掉空白和重复。
func uniqueStrings(groups ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, group := range groups {
		for _, s := range group {
			s = strings.TrimSpace(s)
			if s == "" || seen[s] {
				continue
			}
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// EnsureSessionWorktree 把项目根做成 Worktree 里的工作区副本，并把 Isolate 设为真。项目根不是目录则失败。
func (a *App) EnsureSessionWorktree(id string) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	origin := m.ProjectRoot()
	if !WorkspaceReady(origin) {
		return m, fmt.Errorf("workspace is not a directory")
	}
	dest := strings.TrimSpace(m.Worktree)
	if dest == "" {
		dest = filepath.Join(a.Home.Worktrees(), m.ID)
	}
	if err := runtime.PersistWorktree(origin, dest); err != nil {
		return m, err
	}
	m.Isolate = true
	m.OriginWorkspace = origin
	m.Workspace = origin
	m.Worktree = dest
	if err := a.writeSession(m); err != nil {
		return m, err
	}
	return m, nil
}

// ApplySessionWorktree 把工作区副本里的改动应用回项目根。没有副本则返回错误。
func (a *App) ApplySessionWorktree(id string) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	origin := m.ProjectRoot()
	dest := strings.TrimSpace(m.Worktree)
	if dest == "" {
		return m, fmt.Errorf("session has no worktree")
	}
	if err := runtime.ApplyWorktree(origin, dest); err != nil {
		return m, err
	}
	return m, nil
}

// SetSessionIsolate 在回合没在跑时切换 Isolate。打开则建立工作区副本；关闭则把 Isolate 写成假，工作区指回项目根。
func (a *App) SetSessionIsolate(id string, isolate bool) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	if a.Running(id) {
		return m, fmt.Errorf("cannot change isolate while a turn is running")
	}
	if isolate {
		return a.EnsureSessionWorktree(id)
	}
	m.Isolate = false
	if origin := m.ProjectRoot(); origin != "" {
		m.Workspace = origin
		m.OriginWorkspace = origin
	}
	if err := a.writeSession(m); err != nil {
		return m, err
	}
	return m, nil
}

// SetSessionPinnedSkills 只保留当前工作区里确实存在的技能名，并写回 meta。
func (a *App) SetSessionPinnedSkills(id string, names []string) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	known := map[string]bool{}
	for _, sk := range a.collectSkills(m.ProjectRoot()) {
		known[sk.Name] = true
	}
	var pinned []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" || !known[n] {
			continue
		}
		pinned = append(pinned, n)
	}
	m.PinnedSkills = uniqueStrings(pinned)
	if err := a.writeSession(m); err != nil {
		return m, err
	}
	return m, nil
}

// InboxDismiss 关闭一条收件箱项。没有收件箱时返回假。
func (a *App) InboxDismiss(id string) bool {
	if a.Inbox == nil {
		return false
	}
	return a.Inbox.Dismiss(id)
}
