package desktop

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// Attach 记下 Wails 应用和主窗口，供菜单、托盘和对话框使用。
// gui 写入 gui。
// win 写入 win。
func (s *Service) Attach(gui *application.App, win application.Window) {
	s.gui = gui
	s.win = win
}

// CloseToTray 返回配置里的 CloseToTray。为真时关窗口会藏进托盘。
func (s *Service) CloseToTray() bool {
	return s.GetConfig().CloseToTray
}

// CompanionEnabled 返回配置里的 CompanionEnabled。
func (s *Service) CompanionEnabled() bool {
	return s.GetConfig().CompanionEnabled
}

// emitSessions 在 gui 非空时发出 yoyo:sessions，载荷原样送出。
// payload 在 gui 非 nil 时作为 yoyo:sessions 的载荷。
func (s *Service) emitSessions(payload any) {
	if s.gui != nil {
		s.gui.Event.Emit("yoyo:sessions", payload)
	}
}

// DeleteSession 在 RPC 非空时调用 thread.delete，否则 App.DeleteSession。成功后发出 yoyo:sessions，载荷是 id。返回值里 ok 表示没有错误。
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

// ArchiveSession 在 RPC 非空时调用 thread.archive，否则 App.ArchiveSession。成功后发出 yoyo:sessions。
// id 在 RPC 非空时作为 thread.archive 的 session。否则传给 App.ArchiveSession。
// archived 在 RPC 非空时放进 thread.archive。否则传给 App.ArchiveSession。
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

// PinSession 在 RPC 非空时调用 thread.pin，否则 App.PinSession。成功后发出 yoyo:sessions。
// id 在 RPC 非空时作为 thread.pin 的 session。否则传给 App.PinSession。
// pinned 在 RPC 非空时放进 thread.pin。否则传给 App.PinSession。
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

// SearchSessions 在 RPC 非空时调用 thread.search，否则 App.SearchSessions。
// query 在 RPC 非空时放进 thread.search。否则传给 App.SearchSessions。
// includeArchived 在 RPC 非空时作为 thread.search 的 include_archived。否则传给 App.SearchSessions。
func (s *Service) SearchSessions(query string, includeArchived bool) ([]app.SessionMeta, error) {
	if s.RPC != nil {
		return decode[[]app.SessionMeta](s.call("thread.search", map[string]any{"query": query, "include_archived": includeArchived}))
	}
	return s.App.SearchSessions(query, includeArchived)
}

// ExportSession 在 RPC 非空时调用 thread.export，并返回 markdown 字段。否则走 App.ExportSession。
// id 在 RPC 非空时作为 thread.export 的 session。否则传给 App.ExportSession。
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

// SetSessionModel 在 RPC 非空时调用 thread.model.set，否则走 App.SetSessionModel。
// id 在 RPC 非空时作为 thread.model.set 的 session。否则传给 App.SetSessionModel。
func (s *Service) SetSessionModel(id, model string) (app.SessionMeta, error) {
	if s.RPC != nil {
		return decode[app.SessionMeta](s.call("thread.model.set", map[string]any{"session": id, "model": model}))
	}
	return s.App.SetSessionModel(id, model)
}

// SetSessionAuthMode 在 RPC 非空时调用 thread.auth.set，否则走 App.SetSessionAuthMode。
// id 在 RPC 非空时作为 thread.auth.set 的 session。否则传给 App.SetSessionAuthMode。
// mode 在 RPC 非空时放进 thread.auth.set。否则传给 App.SetSessionAuthMode。
func (s *Service) SetSessionAuthMode(id, mode string) (app.SessionMeta, error) {
	if s.RPC != nil {
		return decode[app.SessionMeta](s.call("thread.auth.set", map[string]any{"session": id, "mode": mode}))
	}
	return s.App.SetSessionAuthMode(id, mode)
}

// SetSessionWorkspace 在 RPC 非空时调用 thread.workspace.set，否则 App.SetSessionWorkspace。成功后发出 yoyo:sessions。
// id 在 RPC 非空时作为 thread.workspace.set 的 session。否则传给 App.SetSessionWorkspace。
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

// SetSessionIsolate 在 RPC 非空时调用 thread.isolate.set，否则 App.SetSessionIsolate。成功后发出 yoyo:sessions。
// id 在 RPC 非空时作为 thread.isolate.set 的 session。否则传给 App.SetSessionIsolate。
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

// SetSessionPinnedSkills 在 RPC 非空时调用 thread.skills.pin，否则 App.SetSessionPinnedSkills。成功后发出 yoyo:sessions。
// id 在 RPC 非空时作为 thread.skills.pin 的 session。否则传给 App.SetSessionPinnedSkills。
// names 在 RPC 非空时放进 thread.skills.pin。否则传给 App.SetSessionPinnedSkills。
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

// CompactSession 调用 CompactSessionFocus，焦点为空。
// id 和空字符串传给 CompactSessionFocus。
func (s *Service) CompactSession(id string) (string, error) {
	return s.CompactSessionFocus(id, "")
}

// CompactSessionFocus 在 RPC 非空时调用 thread.compact，并返回 note。否则走 App.CompactSessionFocus。
// id 在 RPC 非空时作为 thread.compact 的 session。否则传给 App.CompactSessionFocus。
// focus 在 RPC 非空时放进 thread.compact。否则传给 App.CompactSessionFocus。
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

// QueueList 在 RPC 非空时调用 thread.queue.list。结果为 nil 时返回空切片。否则走 App.QueueList。
// id 在 RPC 非空时作为 thread.queue.list 的 session。否则传给 App.QueueList。
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

// QueueCancel 在 RPC 非空时调用 thread.queue.cancel，并读取 ok。否则走 App.QueueCancel。
// id 在 RPC 非空时作为 thread.queue.cancel 的 session。否则传给 App.QueueCancel。
// itemID 在 RPC 非空时作为 thread.queue.cancel 的 id。否则传给 App.QueueCancel。
func (s *Service) QueueCancel(id, itemID string) bool {
	if s.RPC != nil {
		v, _ := s.call("thread.queue.cancel", map[string]any{"session": id, "id": itemID})
		m, _ := decode[map[string]any](v, nil)
		ok, _ := m["ok"].(bool)
		return ok
	}
	return s.App.QueueCancel(id, itemID)
}

// QueueReorder 在 RPC 非空时调用 thread.queue.reorder，并读取 ok。否则走 App.QueueReorder。
// id 在 RPC 非空时作为 thread.queue.reorder 的 session。否则传给 App.QueueReorder。
// itemID 在 RPC 非空时作为 thread.queue.reorder 的 id。否则传给 App.QueueReorder。
// delta 在 RPC 非空时放进 thread.queue.reorder。否则传给 App.QueueReorder。
func (s *Service) QueueReorder(id, itemID string, delta int) bool {
	if s.RPC != nil {
		v, _ := s.call("thread.queue.reorder", map[string]any{"session": id, "id": itemID, "delta": delta})
		m, _ := decode[map[string]any](v, nil)
		ok, _ := m["ok"].(bool)
		return ok
	}
	return s.App.QueueReorder(id, itemID, delta)
}

// ApplySessionWorktree 在 RPC 非空时调用 thread.worktree.apply，否则走 App.ApplySessionWorktree。
// id 在 RPC 非空时作为 thread.worktree.apply 的 session。否则传给 App.ApplySessionWorktree。
func (s *Service) ApplySessionWorktree(id string) (app.SessionMeta, error) {
	if s.RPC != nil {
		v, err := s.call("thread.worktree.apply", map[string]any{"session": id})
		return decode[app.SessionMeta](v, err)
	}
	return s.App.ApplySessionWorktree(id)
}

// Steer 在 RPC 非空时调用 turn.steer，否则走 App.Steer。
// id 在 RPC 非空时作为 turn.steer 的 session。否则传给 App.Steer。
// text 在 RPC 非空时放进 turn.steer。否则传给 App.Steer。
func (s *Service) Steer(id, text string) error {
	if s.RPC != nil {
		_, err := s.call("turn.steer", map[string]any{"session": id, "text": text})
		return err
	}
	return s.App.Steer(id, text)
}

// SearchFiles 在 RPC 非空时调用 fs.search，limit 为 40。结果为 nil 时返回空切片。同进程在 workspace 为空时用 App.Workspace，再 FuzzySearch，上限 40。
// query 在 RPC 非空时放进 fs.search。否则传给 runtime.FuzzySearch。
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

// WorkspaceTree 在 RPC 非空时调用 fs.tree。结果为 nil 时返回空切片。同进程在 workspace 为空时用 App.Workspace，再 ListTree，上限 4000。
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

// ListSkills 调用 ListSkillsFor，工作区为空。
func (s *Service) ListSkills() []map[string]string {
	return s.ListSkillsFor("")
}

// ListSkillsFor 在 RPC 非空时调用 skills.list。结果为 nil 时返回空切片。没有 App 时返回 nil。否则走 App.ListSkills。
// workspace 在 RPC 非空时放进 skills.list。否则传给 App.ListSkills。
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

// GetSkill 在 RPC 非空时调用 skills.get。结果为 nil 时返回空 map。没有 App 或技能不存在时，同进程分别返回 nil 和空 map。
// workspace 在 RPC 非空时放进 skills.get。否则传给 App.GetSkill。
// name 在 RPC 非空时放进 skills.get。否则传给 App.GetSkill。
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

// SkillMarket 在 RPC 非空时调用 skills.market。没有 App 时返回 no app。否则走 App.SkillMarket。
// refresh 在 RPC 非空时放进 skills.market。否则传给 App.SkillMarket。
// hub 在 RPC 非空时放进 skills.market。否则传给 App.SkillMarket。
// query 在 RPC 非空时放进 skills.market。否则传给 App.SkillMarket。
func (s *Service) SkillMarket(refresh bool, hub string, query string) (any, error) {
	if s.RPC != nil {
		return s.call("skills.market", map[string]any{"refresh": refresh, "hub": hub, "query": query})
	}
	if s.App == nil {
		return nil, errors.New("no app")
	}
	return s.App.SkillMarket(refresh, hub, query)
}

// ModelCatalog 在 RPC 非空时调用 models.catalog。没有 App 时返回 no app。否则走 App.ModelCatalog。
// refresh 在 RPC 非空时放进 models.catalog。否则传给 App.ModelCatalog。
func (s *Service) ModelCatalog(refresh bool) (any, error) {
	if s.RPC != nil {
		return s.call("models.catalog", map[string]any{"refresh": refresh})
	}
	if s.App == nil {
		return nil, errors.New("no app")
	}
	return s.App.ModelCatalog(refresh)
}

// InstallMarketSkill 在 RPC 非空时调用 skills.install。没有 App 时返回 no app。否则走 App.InstallMarketSkill。
// slug 在 RPC 非空时放进 skills.install。否则传给 App.InstallMarketSkill。
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

// UninstallMarketSkill 在 RPC 非空时调用 skills.uninstall。没有 App 时返回 no app。否则走 App.UninstallMarketSkill。
// slug 在 RPC 非空时放进 skills.uninstall。否则传给 App.UninstallMarketSkill。
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

// ListPacks 在 RPC 非空时调用 packs.list。结果为 nil 时返回空切片。没有 App 时返回 nil。同进程把 ListPacks 的每项编成 map。
// workspace 在 RPC 非空时放进 packs.list。否则传给 App.ListPacks。
func (s *Service) ListPacks(workspace string) []map[string]any {
	if s.RPC != nil {
		v, err := s.call("packs.list", map[string]any{"workspace": workspace})
		out, _ := decode[[]map[string]any](v, err)
		if out == nil {
			return []map[string]any{}
		}
		return out
	}
	if s.App == nil {
		return nil
	}
	list := s.App.ListPacks(workspace)
	out := make([]map[string]any, 0, len(list))
	for _, st := range list {
		b, _ := json.Marshal(st)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		out = append(out, m)
	}
	return out
}

// InstallPack 在 RPC 非空时调用 packs.install。没有 App 时返回 no app。同进程把 InstallPack 的结果编成 map。
// id 在 RPC 非空时放进 packs.install。否则传给 App.InstallPack。
// path 在 RPC 非空时放进 packs.install。否则传给 App.InstallPack。
func (s *Service) InstallPack(id, path string) (map[string]any, error) {
	if s.RPC != nil {
		v, err := s.call("packs.install", map[string]any{"id": id, "path": path})
		return decode[map[string]any](v, err)
	}
	if s.App == nil {
		return nil, errors.New("no app")
	}
	m, err := s.App.InstallPack(id, path)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out, nil
}

// UninstallPack 在 RPC 非空时调用 packs.uninstall。没有 App 时返回 no app。否则走 App.UninstallPack。
// id 在 RPC 非空时放进 packs.uninstall。否则传给 App.UninstallPack。
func (s *Service) UninstallPack(id string) error {
	if s.RPC != nil {
		_, err := s.call("packs.uninstall", map[string]any{"id": id})
		return err
	}
	if s.App == nil {
		return errors.New("no app")
	}
	return s.App.UninstallPack(id)
}

// EnablePack 在 RPC 非空时调用 packs.enable。没有 App 时返回 no app。否则走 App.EnablePack。
// id 在 RPC 非空时放进 packs.enable。否则传给 App.EnablePack。
// scope 在 RPC 非空时放进 packs.enable。否则传给 App.EnablePack。
// enabled 在 RPC 非空时放进 packs.enable。否则传给 App.EnablePack。
func (s *Service) EnablePack(id, scope string, enabled bool) error {
	if s.RPC != nil {
		_, err := s.call("packs.enable", map[string]any{"id": id, "scope": scope, "enabled": enabled})
		return err
	}
	if s.App == nil {
		return errors.New("no app")
	}
	return s.App.EnablePack(id, scope, enabled)
}

// UpdatePack 在 RPC 非空时调用 packs.update。没有 App 时返回 no app。同进程把 UpdatePack 的结果编成 map。
// id 在 RPC 非空时放进 packs.update。否则传给 App.UpdatePack。
func (s *Service) UpdatePack(id string) (map[string]any, error) {
	if s.RPC != nil {
		v, err := s.call("packs.update", map[string]any{"id": id})
		return decode[map[string]any](v, err)
	}
	if s.App == nil {
		return nil, errors.New("no app")
	}
	m, err := s.App.UpdatePack(id)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out, nil
}

// KeyStatus 在 RPC 非空时调用 key.status。没有 App 时返回空 map。否则返回 Vault.Status。
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

// StopMCP 在 RPC 非空时调用 mcp.stop，否则走 App.StopMCP。
// name 在 RPC 非空时放进 mcp.stop。否则传给 App.StopMCP。
func (s *Service) StopMCP(name string) error {
	if s.RPC != nil {
		_, err := s.call("mcp.stop", map[string]any{"name": name})
		return err
	}
	return s.App.StopMCP(name)
}

// MCPList 在 RPC 非空时调用 mcp.list。否则返回 servers 为 MCP.Info，tools 为 MCP.Tools。
func (s *Service) MCPList() map[string]any {
	if s.RPC != nil {
		v, err := s.call("mcp.list", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return map[string]any{"servers": s.App.MCP.Info(), "tools": s.App.MCP.Tools()}
}

// Logs 在 RPC 非空时调用 logs.tail，否则走 App.Logs。
// limit 在 RPC 非空时放进 logs.tail。否则传给 App.Logs。
func (s *Service) Logs(limit int) map[string]any {
	if s.RPC != nil {
		v, err := s.call("logs.tail", map[string]any{"limit": limit})
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.Logs(limit)
}

// Journal 在 RPC 非空时调用 journal.tail，否则走 App.JournalTail。
// limit 在 RPC 非空时放进 journal.tail。否则传给 App.JournalTail。
func (s *Service) Journal(limit int) map[string]any {
	if s.RPC != nil {
		v, err := s.call("journal.tail", map[string]any{"limit": limit})
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.JournalTail(limit)
}

// ExportDiagnostics 在 RPC 非空时调用 logs.export，并返回 path。否则走 App.ExportDiagnostics，目标为空。
func (s *Service) ExportDiagnostics() (string, error) {
	if s.RPC != nil {
		v, err := s.call("logs.export", map[string]any{})
		m, _ := decode[map[string]any](v, err)
		path, _ := m["path"].(string)
		return path, err
	}
	return s.App.ExportDiagnostics("")
}

// DiagnoseSession 在 RPC 非空时调用 logs.diagnose，并返回 text。同进程用 80 条调用 App.DiagnoseSession。
// sessionID 在 RPC 非空时作为 logs.diagnose 的 session。否则和 80 传给 App.DiagnoseSession。
func (s *Service) DiagnoseSession(sessionID string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("logs.diagnose", map[string]any{"session": sessionID})
		m, _ := decode[map[string]any](v, err)
		text, _ := m["text"].(string)
		return text, err
	}
	return s.App.DiagnoseSession(sessionID, 80)
}

// FrontendLogs 在 RPC 非空时调用 logs.frontend。同进程在 Log 非空时 AppendFrontend。没有日志器时返回 nil。
// events 在 RPC 非空时放进 logs.frontend。同进程在 App 和 Log 都非 nil 时传给 AppendFrontend。
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

// Doctor 在 RPC 非空时调用 doctor，否则走 App.Doctor。
func (s *Service) Doctor() map[string]any {
	if s.RPC != nil {
		v, err := s.call("doctor", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.Doctor()
}

// About 在 RPC 非空时调用 about，否则走 App.Health。
func (s *Service) About() map[string]any {
	if s.RPC != nil {
		v, err := s.call("about", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.Health()
}

// CheckUpdate 在 RPC 非空时调用 update.check，否则走 App.CheckUpdate。
func (s *Service) CheckUpdate() map[string]any {
	if s.RPC != nil {
		v, err := s.call("update.check", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.CheckUpdate()
}

// TestProvider 调用 TestChatProvider，id 为空。
func (s *Service) TestProvider() map[string]any {
	return s.TestChatProvider("")
}

// ListChatProviders 在 RPC 非空时调用 provider.list。结果为 nil 或没有 App 时返回空切片。否则走 App.ListChatProviders。
func (s *Service) ListChatProviders() []app.ChatProvider {
	if s.RPC != nil {
		v, err := s.call("provider.list", nil)
		list, _ := decode[[]app.ChatProvider](v, err)
		if list == nil {
			return []app.ChatProvider{}
		}
		return list
	}
	if s.App == nil {
		return []app.ChatProvider{}
	}
	return s.App.ListChatProviders()
}

// UpsertChatProvider 在 RPC 非空时调用 provider.upsert。没有 App 时返回 unavailable。否则走 App.UpsertChatProvider。
// in 在 RPC 非空时作为 provider.upsert 的 provider。否则传给 App.UpsertChatProvider。
// apiKey 在 RPC 非空时作为 provider.upsert 的 api_key。否则传给 App.UpsertChatProvider。
func (s *Service) UpsertChatProvider(in app.ChatProviderIn, apiKey string) (app.ChatProvider, error) {
	if s.RPC != nil {
		return decode[app.ChatProvider](s.call("provider.upsert", map[string]any{"provider": in, "api_key": apiKey}))
	}
	if s.App == nil {
		return app.ChatProvider{}, fmt.Errorf("unavailable")
	}
	return s.App.UpsertChatProvider(in, apiKey)
}

// DeleteChatProvider 在 RPC 非空时调用 provider.delete。没有 App 时返回 unavailable。否则走 App.DeleteChatProvider。
func (s *Service) DeleteChatProvider(id string) error {
	if s.RPC != nil {
		_, err := s.call("provider.delete", map[string]any{"id": id})
		return err
	}
	if s.App == nil {
		return fmt.Errorf("unavailable")
	}
	return s.App.DeleteChatProvider(id)
}

// SetDefaultChatProvider 在 RPC 非空时调用 provider.setDefault。没有 App 时返回 unavailable。否则走 App.SetDefaultChatProvider。
func (s *Service) SetDefaultChatProvider(id string) error {
	if s.RPC != nil {
		_, err := s.call("provider.setDefault", map[string]any{"id": id})
		return err
	}
	if s.App == nil {
		return fmt.Errorf("unavailable")
	}
	return s.App.SetDefaultChatProvider(id)
}

// TestChatProvider 在 RPC 非空时调用 provider.test。没有 App 时返回 ok 为假、error 为 unavailable。否则走 App.TestChatProvider。
func (s *Service) TestChatProvider(id string) map[string]any {
	if s.RPC != nil {
		v, err := s.call("provider.test", map[string]any{"id": id})
		m, _ := decode[map[string]any](v, err)
		return m
	}
	if s.App == nil {
		return map[string]any{"ok": false, "error": "unavailable"}
	}
	return s.App.TestChatProvider(id)
}

// ReplaceMCP 在 RPC 非空时调用 mcp.replace，否则走 App.ReplaceMCP。
// servers 在 RPC 非空时放进 mcp.replace。否则传给 App.ReplaceMCP。
func (s *Service) ReplaceMCP(servers []app.MCPServerConfig) error {
	if s.RPC != nil {
		_, err := s.call("mcp.replace", map[string]any{"servers": servers})
		return err
	}
	return s.App.ReplaceMCP(servers)
}

// RevealLogs 从 logs.tail 或本进程日志器取出目录，再用 RevealPath 打开。目录未知时返回 log directory unknown。
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

// RevealJournal 从 journal.tail 或本进程 Journal.Path 取出路径，再用 RevealPath 打开。路径未知时返回 journal path unknown。
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

// NotifyAllowed 在通知总开关关闭时返回假。NotifyWhenUnfocusedOnly 为真且主窗口正聚焦时也返回假。
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

// toastAllowed 先看 NotifyAllowed。kind 为 done 且主窗口正聚焦时返回假，因为人已经在应用里，铃铛还留着记录。审批和错误不受这条聚焦限制。
func (s *Service) toastAllowed(kind string) bool {
	if !s.NotifyAllowed() {
		return false
	}
	return !(kind == "done" && s.win != nil && s.win.IsFocused())
}

// PickFolder 打开只能选目录的对话框，标题用当前语言的选择工作区。有主窗口时附在窗口上。没有 gui 时返回 native dialog unavailable。
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

// PickFiles 打开只能选文件的多选对话框，标题用当前语言的附加文件。有主窗口时附在窗口上。没有 gui 时返回 native dialog unavailable。
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

// ShowAboutNative 用原生信息框显示 About 文案、版本、harness 和 model。没有 gui 时返回。
func (s *Service) ShowAboutNative() {
	if s.gui == nil {
		return
	}
	about := s.About()
	n := nativeCopy(s.GetConfig().Locale)
	msg := fmt.Sprintf("Yoyo %v\nharness %v\nmodel %v", about["version"], about["harness"], about["model"])
	s.gui.Dialog.Info().SetTitle(n.About).SetMessage(msg).Show()
}
