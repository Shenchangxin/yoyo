package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/runtime"
)

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

func (a *App) InboxDismiss(id string) bool {
	if a.Inbox == nil {
		return false
	}
	return a.Inbox.Dismiss(id)
}
