package desktop

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

func (s *Service) Attach(gui *application.App, win application.Window) {
	s.gui = gui
	s.win = win
}

func (s *Service) CloseToTray() bool {
	if s.RPC != nil {
		v, err := s.call("config.get", nil)
		c, _ := decode[app.Config](v, err)
		return c.CloseToTray
	}
	if s.App == nil {
		return false
	}
	return s.App.Config.CloseToTray
}

func (s *Service) DeleteSession(id string) error {
	if s.RPC != nil {
		_, err := s.call("thread.delete", map[string]any{"session": id})
		return err
	}
	return s.App.DeleteSession(id)
}

func (s *Service) ArchiveSession(id string, archived bool) (app.SessionMeta, error) {
	if s.RPC != nil {
		return decode[app.SessionMeta](s.call("thread.archive", map[string]any{"session": id, "archived": archived}))
	}
	return s.App.ArchiveSession(id, archived)
}

func (s *Service) PinSession(id string, pinned bool) (app.SessionMeta, error) {
	if s.RPC != nil {
		return decode[app.SessionMeta](s.call("thread.pin", map[string]any{"session": id, "pinned": pinned}))
	}
	return s.App.PinSession(id, pinned)
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

func (s *Service) CompactSession(id string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("thread.compact", map[string]any{"session": id})
		m, err2 := decode[map[string]any](v, err)
		if err2 != nil {
			return "", err2
		}
		t, _ := m["note"].(string)
		return t, nil
	}
	return s.App.CompactSession(id)
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

func (s *Service) ListSkills() []map[string]string {
	if s.RPC != nil {
		v, err := s.call("skills.list", nil)
		out, _ := decode[[]map[string]string](v, err)
		if out == nil {
			return []map[string]string{}
		}
		return out
	}
	if s.App == nil {
		return nil
	}
	ws := s.App.Config.Workspace
	sk := runtime.LoadSkillDirs(runtime.SkillRoots(s.App.Home.Root, ws, filepath.Join(filepath.Dir(s.App.BundledEvals), "skills"))...)
	if _, _, _, cas, _, _, err := s.App.Materials(s.App.ActiveHash()); err == nil {
		sk = runtime.MergeSkills(cas, sk)
	}
	out := make([]map[string]string, 0, len(sk))
	for _, item := range sk {
		out = append(out, map[string]string{"name": item.Name, "description": item.Description})
	}
	return out
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

func (s *Service) PickFolder() (string, error) {
	if s.gui == nil {
		return "", errors.New("native dialog unavailable")
	}
	dlg := s.gui.Dialog.OpenFile().CanChooseFiles(false).CanChooseDirectories(true).SetTitle("Choose workspace")
	if s.win != nil {
		dlg = dlg.AttachToWindow(s.win)
	}
	return dlg.PromptForSingleSelection()
}

func (s *Service) PickFiles() ([]string, error) {
	if s.gui == nil {
		return nil, errors.New("native dialog unavailable")
	}
	dlg := s.gui.Dialog.OpenFile().CanChooseFiles(true).CanChooseDirectories(false).SetTitle("Attach files")
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
	msg := fmt.Sprintf("Yoyo %v\nharness %v\nmodel %v", about["version"], about["harness"], about["model"])
	s.gui.Dialog.Info().SetTitle("About Yoyo").SetMessage(msg).Show()
}
