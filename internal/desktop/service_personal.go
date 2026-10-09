package desktop

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/expert"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/osutil"
	"github.com/Shenchangxin/yoyo/internal/project"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

func (s *Service) InboxList() []inbox.Item {
	if s.RPC != nil {
		v, _ := s.call("inbox.list", nil)
		out, _ := decode[[]inbox.Item](v, nil)
		return out
	}
	return s.App.InboxList()
}

func (s *Service) InboxMarkRead(id string) {
	if s.RPC != nil {
		_, _ = s.call("inbox.read", map[string]any{"id": id})
		return
	}
	s.App.InboxMarkRead(id)
}

func (s *Service) InboxDismiss(id string) bool {
	if s.RPC != nil {
		v, _ := s.call("inbox.dismiss", map[string]any{"id": id})
		m, _ := decode[map[string]any](v, nil)
		b, _ := m["ok"].(bool)
		return b
	}
	return s.App.InboxDismiss(id)
}

func (s *Service) ProjectsList() []project.Project {
	if s.RPC != nil {
		v, _ := s.call("projects.list", nil)
		out, _ := decode[[]project.Project](v, nil)
		return out
	}
	return s.App.ProjectsList()
}

func (s *Service) ProjectCreate(p project.Project) project.Project {
	if s.RPC != nil {
		v, _ := s.call("projects.create", p)
		out, _ := decode[project.Project](v, nil)
		return out
	}
	return s.App.ProjectCreate(p)
}

func (s *Service) MemoryList(q, kind string) []memory.Item {
	if s.RPC != nil {
		v, _ := s.call("memory.list", map[string]any{"q": q, "kind": kind})
		out, _ := decode[[]memory.Item](v, nil)
		return out
	}
	return s.App.MemoryList(q, kind)
}

func (s *Service) MemoryPromote(id string) bool {
	if s.RPC != nil {
		v, _ := s.call("memory.promote", map[string]any{"id": id})
		ok, _ := decode[bool](v, nil)
		return ok
	}
	return s.App.MemoryPromote(id)
}

func (s *Service) MemoryForget(id string) bool {
	if s.RPC != nil {
		v, _ := s.call("memory.forget", map[string]any{"id": id})
		ok, _ := decode[bool](v, nil)
		return ok
	}
	return s.App.MemoryForget(id)
}

func (s *Service) ScheduleList() []schedule.Job {
	if s.RPC != nil {
		v, _ := s.call("schedule.list", nil)
		out, _ := decode[[]schedule.Job](v, nil)
		return out
	}
	return s.App.ScheduleList()
}

func (s *Service) ScheduleCreate(j schedule.Job) schedule.Job {
	if s.RPC != nil {
		v, _ := s.call("schedule.create", j)
		out, _ := decode[schedule.Job](v, nil)
		return out
	}
	return s.App.ScheduleCreate(j)
}

func (s *Service) ScheduleCancel(id string) bool {
	if s.RPC != nil {
		v, _ := s.call("schedule.cancel", map[string]any{"id": id})
		ok, _ := decode[bool](v, nil)
		return ok
	}
	return s.App.ScheduleCancel(id)
}

func (s *Service) TriggerWebhook(id string) error {
	if s.RPC != nil {
		_, err := s.call("schedule.trigger", map[string]any{"id": id})
		return err
	}
	return s.App.TriggerWebhook(id)
}

func (s *Service) SetSessionConnectors(id string, accounts []string) (app.SessionMeta, error) {
	if s.RPC != nil {
		v, err := s.call("thread.connectors.set", map[string]any{"session": id, "accounts": accounts})
		return decode[app.SessionMeta](v, err)
	}
	return s.App.SetSessionConnectors(id, accounts)
}

func (s *Service) ConnectorsList() []connector.Account {
	if s.RPC != nil {
		v, _ := s.call("connectors.list", nil)
		out, _ := decode[[]connector.Account](v, nil)
		return out
	}
	return s.App.ConnectorsList()
}

func (s *Service) ConnectorCatalog() []connector.CatalogEntry {
	if s.RPC != nil {
		v, _ := s.call("connectors.catalog", nil)
		out, _ := decode[[]connector.CatalogEntry](v, nil)
		return out
	}
	return s.App.ConnectorCatalog()
}

func (s *Service) ConnectorConnect(acct connector.Account) connector.Account {
	if s.RPC != nil {
		v, _ := s.call("connectors.connect", acct)
		out, _ := decode[connector.Account](v, nil)
		return out
	}
	return s.App.ConnectorConnect(acct)
}

func (s *Service) IsolationReport() map[string]any {
	if s.RPC != nil {
		v, err := s.call("isolation.report", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.IsolationReport()
}

func (s *Service) ReviewQueue() map[string]any {
	if s.RPC != nil {
		v, err := s.call("review.queue", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.ReviewQueue()
}

func (s *Service) BrowserView() map[string]any {
	if s.RPC != nil {
		v, err := s.call("browser.view", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	if s.App == nil {
		return map[string]any{"lane": "isolated", "live": false, "log": []any{}}
	}
	return s.App.BrowserView()
}

func (s *Service) BrowserTakeover() error {
	return s.BrowserTakeoverAt("")
}

func (s *Service) BrowserTakeoverAt(path string) error {
	if s.RPC != nil {
		_, err := s.call("browser.takeover", map[string]any{"path": path})
		return err
	}
	if s.App == nil {
		return nil
	}
	return s.App.BrowserTakeoverAt(path)
}

func (s *Service) PhoneStatus() map[string]any {
	if s.RPC != nil {
		v, err := s.call("phone.status", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.PhoneStatus()
}

func (s *Service) StartMCPHTTP(name, endpoint string) error {
	if s.RPC != nil {
		_, err := s.call("mcp.start_http", map[string]any{"name": name, "endpoint": endpoint})
		return err
	}
	return s.App.StartMCPHTTP(name, endpoint)
}

func (s *Service) AdmitExpert(dir string) (expert.Pack, error) {
	if s.RPC != nil {
		return decode[expert.Pack](s.call("expert.admit", map[string]any{"dir": dir}))
	}
	return s.App.AdmitExpert(dir)
}

func (s *Service) DistillExpert(name, description, body string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("expert.distill", map[string]any{"name": name, "description": description, "body": body})
		m, _ := decode[map[string]any](v, err)
		p, _ := m["path"].(string)
		return p, err
	}
	return s.App.DistillExpert(name, description, body)
}

func (s *Service) ClipboardRead() (string, error) {
	return osutil.ClipboardRead()
}

func (s *Service) Screenshot(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("screenshot path required")
	}
	return osutil.Screenshot(filepath.Clean(path))
}

func (s *Service) RaiseWindow() {
	Raise(s.win)
}

func (s *Service) RaiseSession(id string) {
	s.RaiseWindow()
	if s.gui != nil && strings.TrimSpace(id) != "" {
		s.gui.Event.Emit("yoyo:focus", strings.TrimSpace(id))
	}
}

func (s *Service) ComputerAllow(name string) {
	if s.RPC != nil {
		_, _ = s.call("computer.allow", map[string]any{"app": name})
		return
	}
	s.App.ComputerAllow(name)
}

func (s *Service) ConnectorAuthURL(provider, clientID, redirect string) map[string]any {
	if s.RPC != nil {
		v, err := s.call("connectors.auth_url", map[string]any{"provider": provider, "client_id": clientID, "redirect": redirect})
		m, _ := decode[map[string]any](v, err)
		return m
	}
	m, _ := s.App.ConnectorAuthURL(provider, clientID, redirect)
	return m
}

func (s *Service) ConnectorStoreToken(id, token string) error {
	if s.RPC != nil {
		_, err := s.call("connectors.token", map[string]any{"id": id, "token": token})
		return err
	}
	return s.App.ConnectorStoreToken(id, token)
}

func (s *Service) ConnectorComplete(provider, code, clientID, secret, redirect string) (connector.Account, error) {
	if s.RPC != nil {
		v, err := s.call("connectors.complete", map[string]any{
			"provider": provider, "code": code, "client_id": clientID, "secret": secret, "redirect": redirect,
		})
		return decode[connector.Account](v, err)
	}
	return s.App.ConnectorComplete(provider, code, clientID, secret, redirect)
}

func (s *Service) ConnectorDisconnect(id string) bool {
	if s.RPC != nil {
		v, _ := s.call("connectors.disconnect", map[string]any{"id": id})
		m, _ := decode[map[string]any](v, nil)
		ok, _ := m["ok"].(bool)
		return ok
	}
	return s.App.ConnectorDisconnect(id)
}

func (s *Service) PersonalSnapshot() map[string]any {
	if s.RPC != nil {
		v, err := s.call("personal.snapshot", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.PersonalSnapshot()
}

func (s *Service) PersonalAnswer(id, text string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.answer", map[string]any{"id": id, "text": text})
	}
	return s.App.PersonalAnswer(id, text)
}

func (s *Service) PersonalDecide(id, hash string, approve bool) (any, error) {
	if s.RPC != nil {
		return s.call("personal.decide", map[string]any{"id": id, "hash": hash, "approve": approve})
	}
	return s.App.PersonalDecide(id, hash, approve)
}

func (s *Service) PersonalIdea(id, action string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.idea", map[string]any{"id": id, "action": action})
	}
	return s.App.PersonalIdea(id, action)
}

func (s *Service) PersonalGoal(title, description string, milestones []string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.goal", map[string]any{"title": title, "description": description, "milestones": milestones})
	}
	return s.App.PersonalGoal(title, description, milestones)
}

func (s *Service) PersonalWatch(title, url, condition, value string, interval int) (any, error) {
	if s.RPC != nil {
		return s.call("personal.watch", map[string]any{"title": title, "url": url, "condition": condition, "value": value, "interval_minutes": interval})
	}
	return s.App.PersonalWatch(title, url, condition, value, interval)
}

func (s *Service) PersonalCancel(id string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.cancel", map[string]any{"id": id})
	}
	return s.App.PersonalCancel(id)
}

func (s *Service) PersonalChoice(id, option string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.choice", map[string]any{"id": id, "option": option})
	}
	return s.App.PersonalChoice(id, option)
}

func (s *Service) PersonalPause(id string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.pause", map[string]any{"id": id})
	}
	return s.App.PersonalPause(id)
}

func (s *Service) PersonalResume(id string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.resume", map[string]any{"id": id})
	}
	return s.App.PersonalResume(id)
}

func (s *Service) PersonalRetry(id string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.retry", map[string]any{"id": id})
	}
	return s.App.PersonalRetry(id)
}

func (s *Service) PersonalGoalStatus(id, status string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.goal.status", map[string]any{"id": id, "status": status})
	}
	return s.App.PersonalGoalStatus(id, status)
}

func (s *Service) PersonalMilestone(goalID, milestoneID string, done bool) (any, error) {
	if s.RPC != nil {
		return s.call("personal.milestone", map[string]any{"goal_id": goalID, "milestone_id": milestoneID, "done": done})
	}
	return s.App.PersonalMilestone(goalID, milestoneID, done)
}

func (s *Service) PersonalMonitorStatus(id, status string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.monitor.status", map[string]any{"id": id, "status": status})
	}
	return s.App.PersonalMonitorStatus(id, status)
}

func (s *Service) MemoryWrite(kind, text, project string) memory.Item {
	if s.RPC != nil {
		v, _ := s.call("memory.write", map[string]any{"kind": kind, "text": text, "project": project})
		out, _ := decode[memory.Item](v, nil)
		return out
	}
	return s.App.MemoryWrite(kind, text, project)
}
