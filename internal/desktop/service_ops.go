package desktop

import (
	"errors"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

func (s *Service) Attach(gui *application.App, win application.Window) {
	s.gui = gui
	s.win = win
}

func (s *Service) CloseToTray() bool {
	return s.GetConfig().CloseToTray
}

func (s *Service) CompanionEnabled() bool {
	return s.GetConfig().CompanionEnabled
}

func (s *Service) emitSessions(payload any) {
	if s.gui != nil {
		s.gui.Event.Emit("yoyo:sessions", payload)
	}
}

func (s *Service) DeleteSession(id string) (map[string]any, error) {
	var err error
	if s.RPC != nil {
		_, err = s.call("thread.delete", map[string]any{"session": id})
	} else {
		err = s.App.DeleteSession(id)
	}
	if err == nil {
		s.emitSessions(id)
	}
	return map[string]any{"ok": err == nil, "id": id}, err
}

func (s *Service) ArchiveSession(id string, archived bool) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("thread.archive", map[string]any{"session": id, "archived": archived}))
	} else {
		m, err = s.App.ArchiveSession(id, archived)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

func (s *Service) PinSession(id string, pinned bool) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("thread.pin", map[string]any{"session": id, "pinned": pinned}))
	} else {
		m, err = s.App.PinSession(id, pinned)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

func (s *Service) SearchSessions(query string, includeArchived bool) ([]app.SessionMeta, error) {
	if s.RPC != nil {
		return decode[[]app.SessionMeta](s.call("thread.search", map[string]any{"query": query, "include_archived": includeArchived}))
	}
	return s.App.SearchSessions(query, includeArchived)
}

func (s *Service) ExportSession(id string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("thread.export", map[string]any{"session": id})
		m, err2 := decode[map[string]any](v, err)
		if err2 != nil {
			return "", err2
		}
		t, _ := m["markdown"].(string)
		return t, nil
	}
	return s.App.ExportSession(id)
}

func (s *Service) SetSessionModel(id, model string) (app.SessionMeta, error) {
	if s.RPC != nil {
		return decode[app.SessionMeta](s.call("thread.model.set", map[string]any{"session": id, "model": model}))
	}
	return s.App.SetSessionModel(id, model)
}

func (s *Service) SetSessionAuthMode(id, mode string) (app.SessionMeta, error) {
	if s.RPC != nil {
		return decode[app.SessionMeta](s.call("thread.auth.set", map[string]any{"session": id, "mode": mode}))
	}
	return s.App.SetSessionAuthMode(id, mode)
}

func (s *Service) SetSessionWorkspace(id, workspace string) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("thread.workspace.set", map[string]any{"session": id, "workspace": workspace}))
	} else {
		m, err = s.App.SetSessionWorkspace(id, workspace)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

func (s *Service) SetSessionIsolate(id string, isolate bool) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("thread.isolate.set", map[string]any{"session": id, "isolate": isolate}))
	} else {
		m, err = s.App.SetSessionIsolate(id, isolate)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

func (s *Service) SetSessionPinnedSkills(id string, names []string) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("thread.skills.pin", map[string]any{"session": id, "names": names}))
	} else {
		m, err = s.App.SetSessionPinnedSkills(id, names)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

func (s *Service) CompactSession(id string) (string, error) {
	return s.CompactSessionFocus(id, "")
}

func (s *Service) CompactSessionFocus(id, focus string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("thread.compact", map[string]any{"session": id, "focus": focus})
		m, err2 := decode[map[string]any](v, err)
		if err2 != nil {
			return "", err2
		}
		t, _ := m["note"].(string)
		return t, nil
	}
	return s.App.CompactSessionFocus(id, focus)
}

func (s *Service) QueueList(id string) []app.QueuedTurn {
	if s.RPC != nil {
		v, err := s.call("thread.queue.list", map[string]any{"session": id})
		q, _ := decode[[]app.QueuedTurn](v, err)
		if q == nil {
			return []app.QueuedTurn{}
		}
		return q
	}
	return s.App.QueueList(id)
}

func (s *Service) QueueCancel(id, itemID string) bool {
	if s.RPC != nil {
		v, _ := s.call("thread.queue.cancel", map[string]any{"session": id, "id": itemID})
		m, _ := decode[map[string]any](v, nil)
		ok, _ := m["ok"].(bool)
		return ok
	}
	return s.App.QueueCancel(id, itemID)
}

func (s *Service) QueueReorder(id, itemID string, delta int) bool {
	if s.RPC != nil {
		v, _ := s.call("thread.queue.reorder", map[string]any{"session": id, "id": itemID, "delta": delta})
		m, _ := decode[map[string]any](v, nil)
		ok, _ := m["ok"].(bool)
		return ok
	}
	return s.App.QueueReorder(id, itemID, delta)
}

func (s *Service) ApplySessionWorktree(id string) (app.SessionMeta, error) {
	if s.RPC != nil {
		v, err := s.call("thread.worktree.apply", map[string]any{"session": id})
		return decode[app.SessionMeta](v, err)
	}
	return s.App.ApplySessionWorktree(id)
}

func (s *Service) Steer(id, text string) error {
	if s.RPC != nil {
		_, err := s.call("turn.steer", map[string]any{"session": id, "text": text})
		return err
	}
	return s.App.Steer(id, text)
}

func (s *Service) SearchFiles(workspace, query string) []runtime.FileHit {
	if s.RPC != nil {
		v, err := s.call("fs.search", map[string]any{"workspace": workspace, "query": query, "limit": 40})
		hits, _ := decode[[]runtime.FileHit](v, err)
		if hits == nil {
			return []runtime.FileHit{}
		}
		return hits
	}
	if workspace == "" && s.App != nil {
		workspace = s.App.Workspace()
	}
	return runtime.FuzzySearch(workspace, query, 40)
}

func (s *Service) WorkspaceTree(workspace string) []runtime.FileHit {
	if s.RPC != nil {
		v, err := s.call("fs.tree", map[string]any{"workspace": workspace})
		hits, _ := decode[[]runtime.FileHit](v, err)
		if hits == nil {
			return []runtime.FileHit{}
		}
		return hits
	}
	if workspace == "" && s.App != nil {
		workspace = s.App.Workspace()
	}
	return runtime.ListTree(workspace, 4000)
}

func (s *Service) ListSkills() []map[string]string {
	return s.ListSkillsFor("")
}

func (s *Service) ListSkillsFor(workspace string) []map[string]string {
	if s.RPC != nil {
		v, err := s.call("skills.list", map[string]any{"workspace": workspace})
		out, _ := decode[[]map[string]string](v, err)
		if out == nil {
			return []map[string]string{}
		}
		return out
	}
	if s.App == nil {
		return nil
	}
	return s.App.ListSkills(workspace)
}

func (s *Service) GetSkill(workspace, name string) map[string]string {
	if s.RPC != nil {
		v, err := s.call("skills.get", map[string]any{"workspace": workspace, "name": name})
		out, _ := decode[map[string]string](v, err)
		if out == nil {
			return map[string]string{}
		}
		return out
	}
	if s.App == nil {
		return nil
	}
	sk := s.App.GetSkill(workspace, name)
	if sk == nil {
		return map[string]string{}
	}
	return sk
}

func (s *Service) SkillMarket(refresh bool) (any, error) {
	if s.RPC != nil {
		return s.call("skills.market", map[string]any{"refresh": refresh})
	}
	if s.App == nil {
		return nil, errors.New("no app")
	}
	return s.App.SkillMarket(refresh)
}

func (s *Service) InstallMarketSkill(slug string) (map[string]any, error) {
	if s.RPC != nil {
		v, err := s.call("skills.install", map[string]any{"slug": slug})
		return decode[map[string]any](v, err)
	}
	if s.App == nil {
		return nil, errors.New("no app")
	}
	return s.App.InstallMarketSkill(slug)
}

func (s *Service) UninstallMarketSkill(slug string) error {
	if s.RPC != nil {
		_, err := s.call("skills.uninstall", map[string]any{"slug": slug})
		return err
	}
	if s.App == nil {
		return errors.New("no app")
	}
	return s.App.UninstallMarketSkill(slug)
}

func (s *Service) KeyStatus() map[string]any {
	if s.RPC != nil {
		v, err := s.call("key.status", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	if s.App == nil {
		return map[string]any{}
	}
	return s.App.Vault.Status()
}

func (s *Service) StopMCP(name string) error {
	if s.RPC != nil {
		_, err := s.call("mcp.stop", map[string]any{"name": name})
		return err
	}
	return s.App.StopMCP(name)
}

func (s *Service) MCPList() map[string]any {
	if s.RPC != nil {
		v, err := s.call("mcp.list", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return map[string]any{"servers": s.App.MCP.Info(), "tools": s.App.MCP.Tools()}
}

func (s *Service) Logs(limit int) map[string]any {
	if s.RPC != nil {
		v, err := s.call("logs.tail", map[string]any{"limit": limit})
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.Logs(limit)
}

func (s *Service) Journal(limit int) map[string]any {
	if s.RPC != nil {
		v, err := s.call("journal.tail", map[string]any{"limit": limit})
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.JournalTail(limit)
}

func (s *Service) ExportDiagnostics() (string, error) {
	if s.RPC != nil {
		v, err := s.call("logs.export", map[string]any{})
		m, _ := decode[map[string]any](v, err)
		path, _ := m["path"].(string)
		return path, err
	}
	return s.App.ExportDiagnostics("")
}

func (s *Service) DiagnoseSession(sessionID string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("logs.diagnose", map[string]any{"session": sessionID})
		m, _ := decode[map[string]any](v, err)
		text, _ := m["text"].(string)
		return text, err
	}
	return s.App.DiagnoseSession(sessionID, 80)
}

func (s *Service) FrontendLogs(events []diaglog.FrontendEvent) error {
	if s.RPC != nil {
		_, err := s.call("logs.frontend", map[string]any{"events": events})
		return err
	}
	if s.App != nil && s.App.Log != nil {
		return s.App.Log.AppendFrontend(events)
	}
	return nil
}

func (s *Service) Doctor() map[string]any {
	if s.RPC != nil {
		v, err := s.call("doctor", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.Doctor()
}

func (s *Service) About() map[string]any {
	if s.RPC != nil {
		v, err := s.call("about", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.Health()
}

func (s *Service) CheckUpdate() map[string]any {
	if s.RPC != nil {
		v, err := s.call("update.check", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.CheckUpdate()
}

func (s *Service) TestProvider() map[string]any {
	if s.RPC != nil {
		v, err := s.call("provider.test", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	if s.App == nil {
		return map[string]any{"ok": false, "error": "unavailable"}
	}
	return s.App.TestProvider()
}

func (s *Service) ReplaceMCP(servers []app.MCPServerConfig) error {
	if s.RPC != nil {
		_, err := s.call("mcp.replace", map[string]any{"servers": servers})
		return err
	}
	return s.App.ReplaceMCP(servers)
}

func (s *Service) RevealLogs() error {
	var dir string
	if s.RPC != nil {
		v, err := s.call("logs.tail", map[string]any{"limit": 1})
		m, _ := decode[map[string]any](v, err)
		dir, _ = m["dir"].(string)
	} else if s.App != nil && s.App.Log != nil {
		dir = s.App.Log.Dir()
	} else if s.App != nil && s.App.Home != nil {
		dir = s.App.Home.Logs()
	}
	if dir == "" {
		return errors.New("log directory unknown")
	}
	return RevealPath(dir)
}

func (s *Service) RevealJournal() error {
	var path string
	if s.RPC != nil {
		v, err := s.call("journal.tail", map[string]any{"limit": 1})
		m, _ := decode[map[string]any](v, err)
		path, _ = m["path"].(string)
	} else if s.App != nil && s.App.Journal != nil {
		path = s.App.Journal.Path
	}
	if path == "" {
		return errors.New("journal path unknown")
	}
	return RevealPath(path)
}

func (s *Service) NotifyAllowed() bool {
	cfg := s.GetConfig()
	if !cfg.NotificationsEnabled {
		return false
	}
	if cfg.NotifyWhenUnfocusedOnly && s.win != nil && s.win.IsFocused() {
		return false
	}
	return true
}

func (s *Service) PickFolder() (string, error) {
	if s.gui == nil {
		return "", errors.New("native dialog unavailable")
	}
	dlg := s.gui.Dialog.OpenFile().CanChooseFiles(false).CanChooseDirectories(true).SetTitle(nativeCopy(s.GetConfig().Locale).ChooseWorkspace)
	if s.win != nil {
		dlg = dlg.AttachToWindow(s.win)
	}
	return dlg.PromptForSingleSelection()
}

func (s *Service) PickFiles() ([]string, error) {
	if s.gui == nil {
		return nil, errors.New("native dialog unavailable")
	}
	dlg := s.gui.Dialog.OpenFile().CanChooseFiles(true).CanChooseDirectories(false).SetTitle(nativeCopy(s.GetConfig().Locale).AttachFiles)
	if s.win != nil {
		dlg = dlg.AttachToWindow(s.win)
	}
	return dlg.PromptForMultipleSelection()
}

func (s *Service) ShowAboutNative() {
	if s.gui == nil {
		return
	}
	about := s.About()
	n := nativeCopy(s.GetConfig().Locale)
	msg := fmt.Sprintf("Yoyo %v\nharness %v\nmodel %v", about["version"], about["harness"], about["model"])
	s.gui.Dialog.Info().SetTitle(n.About).SetMessage(msg).Show()
}
