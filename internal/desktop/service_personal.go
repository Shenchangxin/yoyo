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

// InboxList 在 RPC 非空时调用 inbox.list，否则走 App.InboxList。
func (s *Service) InboxList() []inbox.Item {
	if s.RPC != nil {
		v, _ := s.call("inbox.list", nil)
		out, _ := decode[[]inbox.Item](v, nil)
		return out
	}
	return s.App.InboxList()
}

// InboxMarkRead 在 RPC 非空时调用 inbox.read，否则走 App.InboxMarkRead。
// id 在 RPC 非空时放进 inbox.read。否则传给 App.InboxMarkRead。
func (s *Service) InboxMarkRead(id string) {
	if s.RPC != nil {
		_, _ = s.call("inbox.read", map[string]any{"id": id})
		return
	}
	s.App.InboxMarkRead(id)
}

// InboxDismiss 在 RPC 非空时调用 inbox.dismiss，并读取 ok。否则走 App.InboxDismiss。
// id 在 RPC 非空时放进 inbox.dismiss。否则传给 App.InboxDismiss。
func (s *Service) InboxDismiss(id string) bool {
	if s.RPC != nil {
		v, _ := s.call("inbox.dismiss", map[string]any{"id": id})
		m, _ := decode[map[string]any](v, nil)
		b, _ := m["ok"].(bool)
		return b
	}
	return s.App.InboxDismiss(id)
}

// ProjectsList 在 RPC 非空时调用 projects.list，否则走 App.ProjectsList。
func (s *Service) ProjectsList() []project.Project {
	if s.RPC != nil {
		v, _ := s.call("projects.list", nil)
		out, _ := decode[[]project.Project](v, nil)
		return out
	}
	return s.App.ProjectsList()
}

// ProjectCreate 在 RPC 非空时调用 projects.create，否则走 App.ProjectCreate。
func (s *Service) ProjectCreate(p project.Project) project.Project {
	if s.RPC != nil {
		v, _ := s.call("projects.create", p)
		out, _ := decode[project.Project](v, nil)
		return out
	}
	return s.App.ProjectCreate(p)
}

// MemoryList 在 RPC 非空时调用 memory.list，否则走 App.MemoryList。
// q 在 RPC 非空时放进 memory.list。否则传给 App.MemoryList。
// kind 在 RPC 非空时放进 memory.list。否则传给 App.MemoryList。
func (s *Service) MemoryList(q, kind string) []memory.Item {
	if s.RPC != nil {
		v, _ := s.call("memory.list", map[string]any{"q": q, "kind": kind})
		out, _ := decode[[]memory.Item](v, nil)
		return out
	}
	return s.App.MemoryList(q, kind)
}

// MemoryPromote 在 RPC 非空时调用 memory.promote。否则走 App.MemoryPromote。
// id 在 RPC 非空时放进 memory.promote。否则传给 App.MemoryPromote。
func (s *Service) MemoryPromote(id string) bool {
	if s.RPC != nil {
		v, _ := s.call("memory.promote", map[string]any{"id": id})
		ok, _ := decode[bool](v, nil)
		return ok
	}
	return s.App.MemoryPromote(id)
}

// MemoryForget 在 RPC 非空时调用 memory.forget。否则走 App.MemoryForget。
// id 在 RPC 非空时放进 memory.forget。否则传给 App.MemoryForget。
func (s *Service) MemoryForget(id string) bool {
	if s.RPC != nil {
		v, _ := s.call("memory.forget", map[string]any{"id": id})
		ok, _ := decode[bool](v, nil)
		return ok
	}
	return s.App.MemoryForget(id)
}

// ScheduleList 在 RPC 非空时调用 schedule.list，否则走 App.ScheduleList。
func (s *Service) ScheduleList() []schedule.Job {
	if s.RPC != nil {
		v, _ := s.call("schedule.list", nil)
		out, _ := decode[[]schedule.Job](v, nil)
		return out
	}
	return s.App.ScheduleList()
}

// ScheduleCreate 在 RPC 非空时调用 schedule.create，否则走 App.ScheduleCreate。
// j 在 RPC 非空时传给 schedule.create。否则传给 App.ScheduleCreate。
func (s *Service) ScheduleCreate(j schedule.Job) schedule.Job {
	if s.RPC != nil {
		v, _ := s.call("schedule.create", j)
		out, _ := decode[schedule.Job](v, nil)
		return out
	}
	return s.App.ScheduleCreate(j)
}

// ScheduleCancel 在 RPC 非空时调用 schedule.cancel。否则走 App.ScheduleCancel。
// id 在 RPC 非空时放进 schedule.cancel。否则传给 App.ScheduleCancel。
func (s *Service) ScheduleCancel(id string) bool {
	if s.RPC != nil {
		v, _ := s.call("schedule.cancel", map[string]any{"id": id})
		ok, _ := decode[bool](v, nil)
		return ok
	}
	return s.App.ScheduleCancel(id)
}

// TriggerWebhook 在 RPC 非空时调用 schedule.trigger，否则走 App.TriggerWebhook。
// id 在 RPC 非空时放进 schedule.trigger。否则传给 App.TriggerWebhook。
func (s *Service) TriggerWebhook(id string) error {
	if s.RPC != nil {
		_, err := s.call("schedule.trigger", map[string]any{"id": id})
		return err
	}
	return s.App.TriggerWebhook(id)
}

// SetSessionConnectors 在 RPC 非空时调用 thread.connectors.set，否则走 App.SetSessionConnectors。
// id 在 RPC 非空时作为 thread.connectors.set 的 session。否则传给 App.SetSessionConnectors。
// accounts 在 RPC 非空时放进 thread.connectors.set。否则传给 App.SetSessionConnectors。
func (s *Service) SetSessionConnectors(id string, accounts []string) (app.SessionMeta, error) {
	if s.RPC != nil {
		v, err := s.call("thread.connectors.set", map[string]any{"session": id, "accounts": accounts})
		return decode[app.SessionMeta](v, err)
	}
	return s.App.SetSessionConnectors(id, accounts)
}

// ConnectorsList 在 RPC 非空时调用 connectors.list，否则走 App.ConnectorsList。
func (s *Service) ConnectorsList() []connector.Account {
	if s.RPC != nil {
		v, _ := s.call("connectors.list", nil)
		out, _ := decode[[]connector.Account](v, nil)
		return out
	}
	return s.App.ConnectorsList()
}

// ConnectorCatalog 在 RPC 非空时调用 connectors.catalog，否则走 App.ConnectorCatalog。
func (s *Service) ConnectorCatalog() []connector.CatalogEntry {
	if s.RPC != nil {
		v, _ := s.call("connectors.catalog", nil)
		out, _ := decode[[]connector.CatalogEntry](v, nil)
		return out
	}
	return s.App.ConnectorCatalog()
}

// ConnectorConnect 在 RPC 非空时调用 connectors.connect，否则走 App.ConnectorConnect。
// acct 在 RPC 非空时传给 connectors.connect。否则传给 App.ConnectorConnect。
func (s *Service) ConnectorConnect(acct connector.Account) connector.Account {
	if s.RPC != nil {
		v, _ := s.call("connectors.connect", acct)
		out, _ := decode[connector.Account](v, nil)
		return out
	}
	return s.App.ConnectorConnect(acct)
}

// IsolationReport 在 RPC 非空时调用 isolation.report，否则走 App.IsolationReport。
func (s *Service) IsolationReport() map[string]any {
	if s.RPC != nil {
		v, err := s.call("isolation.report", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.IsolationReport()
}

// ReviewQueue 在 RPC 非空时调用 review.queue，否则走 App.ReviewQueue。
func (s *Service) ReviewQueue() map[string]any {
	if s.RPC != nil {
		v, err := s.call("review.queue", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.ReviewQueue()
}

// BrowserView 在 RPC 非空时调用 browser.view。没有 App 时返回 lane 为 isolated、live 为假、log 为空切片。否则走 App.BrowserView。
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

// BrowserTakeover 调用 BrowserTakeoverAt，路径为空。
func (s *Service) BrowserTakeover() error {
	return s.BrowserTakeoverAt("")
}

// BrowserTakeoverAt 在 RPC 非空时调用 browser.takeover。没有 App 时返回 nil。否则走 App.BrowserTakeoverAt。
// path 在 RPC 非空时放进 browser.takeover。App 非空时传给 App.BrowserTakeoverAt。
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

// PhoneStatus 在 RPC 非空时调用 phone.status，否则走 App.PhoneStatus。
func (s *Service) PhoneStatus() map[string]any {
	if s.RPC != nil {
		v, err := s.call("phone.status", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.PhoneStatus()
}

// StartMCPHTTP 在 RPC 非空时调用 mcp.start_http，否则走 App.StartMCPHTTP。
// name 在 RPC 非空时放进 mcp.start_http。否则传给 App.StartMCPHTTP。
// endpoint 在 RPC 非空时放进 mcp.start_http。否则传给 App.StartMCPHTTP。
func (s *Service) StartMCPHTTP(name, endpoint string) error {
	if s.RPC != nil {
		_, err := s.call("mcp.start_http", map[string]any{"name": name, "endpoint": endpoint})
		return err
	}
	return s.App.StartMCPHTTP(name, endpoint)
}

// AdmitExpert 在 RPC 非空时调用 expert.admit，否则走 App.AdmitExpert。
// dir 在 RPC 非空时放进 expert.admit。否则传给 App.AdmitExpert。
func (s *Service) AdmitExpert(dir string) (expert.Pack, error) {
	if s.RPC != nil {
		return decode[expert.Pack](s.call("expert.admit", map[string]any{"dir": dir}))
	}
	return s.App.AdmitExpert(dir)
}

// DistillExpert 在 RPC 非空时调用 expert.distill，并返回 path。否则走 App.DistillExpert。
// name 在 RPC 非空时放进 expert.distill。否则传给 App.DistillExpert。
// description 在 RPC 非空时放进 expert.distill。否则传给 App.DistillExpert。
// body 在 RPC 非空时放进 expert.distill。否则传给 App.DistillExpert。
func (s *Service) DistillExpert(name, description, body string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("expert.distill", map[string]any{"name": name, "description": description, "body": body})
		m, _ := decode[map[string]any](v, err)
		p, _ := m["path"].(string)
		return p, err
	}
	return s.App.DistillExpert(name, description, body)
}

// ClipboardRead 读取本机剪贴板，不经过 RPC。
func (s *Service) ClipboardRead() (string, error) {
	return osutil.ClipboardRead()
}

// Screenshot 把去掉空白并 Clean 过的路径交给本机截图。路径为空时返回 screenshot path required。不经过 RPC。
func (s *Service) Screenshot(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("screenshot path required")
	}
	return osutil.Screenshot(filepath.Clean(path))
}

// RaiseWindow 对主窗口调用 Raise。
func (s *Service) RaiseWindow() {
	Raise(s.win)
}

// RaiseSession 先 RaiseWindow。gui 非空且 id 去掉空白后非空时，发出 yoyo:focus，载荷是这个 id。
func (s *Service) RaiseSession(id string) {
	s.RaiseWindow()
	if s.gui != nil && strings.TrimSpace(id) != "" {
		s.gui.Event.Emit("yoyo:focus", strings.TrimSpace(id))
	}
}

// ComputerAllow 在 RPC 非空时调用 computer.allow，字段名是 app。否则走 App.ComputerAllow。
// name 在 RPC 非空时作为 computer.allow 的 app。否则传给 App.ComputerAllow。
func (s *Service) ComputerAllow(name string) {
	if s.RPC != nil {
		_, _ = s.call("computer.allow", map[string]any{"app": name})
		return
	}
	s.App.ComputerAllow(name)
}

// ConnectorAuthURL 在 RPC 非空时调用 connectors.auth_url。同进程忽略 ConnectorAuthURL 的错误，只返回 map。
// provider 在 RPC 非空时放进 connectors.auth_url。否则传给 App.ConnectorAuthURL。
// clientID 在 RPC 非空时作为 connectors.auth_url 的 client_id。否则传给 App.ConnectorAuthURL。
// redirect 在 RPC 非空时放进 connectors.auth_url。否则传给 App.ConnectorAuthURL。
func (s *Service) ConnectorAuthURL(provider, clientID, redirect string) map[string]any {
	if s.RPC != nil {
		v, err := s.call("connectors.auth_url", map[string]any{"provider": provider, "client_id": clientID, "redirect": redirect})
		m, _ := decode[map[string]any](v, err)
		return m
	}
	m, _ := s.App.ConnectorAuthURL(provider, clientID, redirect)
	return m
}

// ConnectorStoreToken 在 RPC 非空时调用 connectors.token，否则走 App.ConnectorStoreToken。
// id 在 RPC 非空时放进 connectors.token。否则传给 App.ConnectorStoreToken。
func (s *Service) ConnectorStoreToken(id, token string) error {
	if s.RPC != nil {
		_, err := s.call("connectors.token", map[string]any{"id": id, "token": token})
		return err
	}
	return s.App.ConnectorStoreToken(id, token)
}

// ConnectorComplete 在 RPC 非空时调用 connectors.complete，否则走 App.ConnectorComplete。
// provider 在 RPC 非空时放进 connectors.complete。否则传给 App.ConnectorComplete。
// code 在 RPC 非空时放进 connectors.complete。否则传给 App.ConnectorComplete。
// clientID 在 RPC 非空时作为 connectors.complete 的 client_id。否则传给 App.ConnectorComplete。
// secret 在 RPC 非空时放进 connectors.complete。否则传给 App.ConnectorComplete。
// redirect 在 RPC 非空时放进 connectors.complete。否则传给 App.ConnectorComplete。
func (s *Service) ConnectorComplete(provider, code, clientID, secret, redirect string) (connector.Account, error) {
	if s.RPC != nil {
		v, err := s.call("connectors.complete", map[string]any{
			"provider": provider, "code": code, "client_id": clientID, "secret": secret, "redirect": redirect,
		})
		return decode[connector.Account](v, err)
	}
	return s.App.ConnectorComplete(provider, code, clientID, secret, redirect)
}

// ConnectorDisconnect 在 RPC 非空时调用 connectors.disconnect，并读取 ok。否则走 App.ConnectorDisconnect。
// id 在 RPC 非空时放进 connectors.disconnect。否则传给 App.ConnectorDisconnect。
func (s *Service) ConnectorDisconnect(id string) bool {
	if s.RPC != nil {
		v, _ := s.call("connectors.disconnect", map[string]any{"id": id})
		m, _ := decode[map[string]any](v, nil)
		ok, _ := m["ok"].(bool)
		return ok
	}
	return s.App.ConnectorDisconnect(id)
}

// PersonalSnapshot 在 RPC 非空时调用 personal.snapshot，否则走 App.PersonalSnapshot。
func (s *Service) PersonalSnapshot() map[string]any {
	if s.RPC != nil {
		v, err := s.call("personal.snapshot", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return s.App.PersonalSnapshot()
}

// PersonalAnswer 在 RPC 非空时调用 personal.answer，否则走 App.PersonalAnswer。
// id 在 RPC 非空时放进 personal.answer。否则传给 App.PersonalAnswer。
// text 在 RPC 非空时放进 personal.answer。否则传给 App.PersonalAnswer。
func (s *Service) PersonalAnswer(id, text string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.answer", map[string]any{"id": id, "text": text})
	}
	return s.App.PersonalAnswer(id, text)
}

// PersonalDecide 在 RPC 非空时调用 personal.decide，否则走 App.PersonalDecide。
// hash 在 RPC 非空时放进 personal.decide。否则传给 App.PersonalDecide。
// approve 在 RPC 非空时放进 personal.decide。否则传给 App.PersonalDecide。
func (s *Service) PersonalDecide(id, hash string, approve bool) (any, error) {
	if s.RPC != nil {
		return s.call("personal.decide", map[string]any{"id": id, "hash": hash, "approve": approve})
	}
	return s.App.PersonalDecide(id, hash, approve)
}

// PersonalIdea 在 RPC 非空时调用 personal.idea，否则走 App.PersonalIdea。
// action 在 RPC 非空时放进 personal.idea。否则传给 App.PersonalIdea。
func (s *Service) PersonalIdea(id, action string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.idea", map[string]any{"id": id, "action": action})
	}
	return s.App.PersonalIdea(id, action)
}

// PersonalGoal 在 RPC 非空时调用 personal.goal，否则走 App.PersonalGoal。
// title 在 RPC 非空时放进 personal.goal。否则传给 App.PersonalGoal。
// description 在 RPC 非空时放进 personal.goal。否则传给 App.PersonalGoal。
// milestones 在 RPC 非空时放进 personal.goal。否则传给 App.PersonalGoal。
func (s *Service) PersonalGoal(title, description string, milestones []string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.goal", map[string]any{"title": title, "description": description, "milestones": milestones})
	}
	return s.App.PersonalGoal(title, description, milestones)
}

// PersonalWatch 在 RPC 非空时调用 personal.watch，间隔字段是 interval_minutes。否则走 App.PersonalWatch。
// title 在 RPC 非空时放进 personal.watch。否则传给 App.PersonalWatch。
// url 在 RPC 非空时放进 personal.watch。否则传给 App.PersonalWatch。
// condition 在 RPC 非空时放进 personal.watch。否则传给 App.PersonalWatch。
// value 在 RPC 非空时放进 personal.watch。否则传给 App.PersonalWatch。
func (s *Service) PersonalWatch(title, url, condition, value string, interval int) (any, error) {
	if s.RPC != nil {
		return s.call("personal.watch", map[string]any{"title": title, "url": url, "condition": condition, "value": value, "interval_minutes": interval})
	}
	return s.App.PersonalWatch(title, url, condition, value, interval)
}

// PersonalCancel 在 RPC 非空时调用 personal.cancel，否则走 App.PersonalCancel。
// id 在 RPC 非空时放进 personal.cancel。否则传给 App.PersonalCancel。
func (s *Service) PersonalCancel(id string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.cancel", map[string]any{"id": id})
	}
	return s.App.PersonalCancel(id)
}

// PersonalChoice 在 RPC 非空时调用 personal.choice，否则走 App.PersonalChoice。
// id 在 RPC 非空时放进 personal.choice。否则传给 App.PersonalChoice。
// option 在 RPC 非空时放进 personal.choice。否则传给 App.PersonalChoice。
func (s *Service) PersonalChoice(id, option string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.choice", map[string]any{"id": id, "option": option})
	}
	return s.App.PersonalChoice(id, option)
}

// PersonalPause 在 RPC 非空时调用 personal.pause，否则走 App.PersonalPause。
// id 在 RPC 非空时放进 personal.pause。否则传给 App.PersonalPause。
func (s *Service) PersonalPause(id string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.pause", map[string]any{"id": id})
	}
	return s.App.PersonalPause(id)
}

// PersonalResume 在 RPC 非空时调用 personal.resume，否则走 App.PersonalResume。
// id 在 RPC 非空时放进 personal.resume。否则传给 App.PersonalResume。
func (s *Service) PersonalResume(id string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.resume", map[string]any{"id": id})
	}
	return s.App.PersonalResume(id)
}

// PersonalRetry 在 RPC 非空时调用 personal.retry，否则走 App.PersonalRetry。
// id 在 RPC 非空时放进 personal.retry。否则传给 App.PersonalRetry。
func (s *Service) PersonalRetry(id string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.retry", map[string]any{"id": id})
	}
	return s.App.PersonalRetry(id)
}

// PersonalGoalStatus 在 RPC 非空时调用 personal.goal.status，否则走 App.PersonalGoalStatus。
// id 在 RPC 非空时放进 personal.goal.status。否则传给 App.PersonalGoalStatus。
func (s *Service) PersonalGoalStatus(id, status string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.goal.status", map[string]any{"id": id, "status": status})
	}
	return s.App.PersonalGoalStatus(id, status)
}

// PersonalMilestone 在 RPC 非空时调用 personal.milestone，否则走 App.PersonalMilestone。
// goalID 在 RPC 非空时作为 personal.milestone 的 goal_id。否则传给 App.PersonalMilestone。
// milestoneID 在 RPC 非空时作为 personal.milestone 的 milestone_id。否则传给 App.PersonalMilestone。
// done 在 RPC 非空时放进 personal.milestone。否则传给 App.PersonalMilestone。
func (s *Service) PersonalMilestone(goalID, milestoneID string, done bool) (any, error) {
	if s.RPC != nil {
		return s.call("personal.milestone", map[string]any{"goal_id": goalID, "milestone_id": milestoneID, "done": done})
	}
	return s.App.PersonalMilestone(goalID, milestoneID, done)
}

// PersonalMonitorStatus 在 RPC 非空时调用 personal.monitor.status，否则走 App.PersonalMonitorStatus。
// id 在 RPC 非空时放进 personal.monitor.status。否则传给 App.PersonalMonitorStatus。
func (s *Service) PersonalMonitorStatus(id, status string) (any, error) {
	if s.RPC != nil {
		return s.call("personal.monitor.status", map[string]any{"id": id, "status": status})
	}
	return s.App.PersonalMonitorStatus(id, status)
}

// MemoryWrite 在 RPC 非空时调用 memory.write，否则走 App.MemoryWrite。
// kind 在 RPC 非空时放进 memory.write。否则传给 App.MemoryWrite。
// text 在 RPC 非空时放进 memory.write。否则传给 App.MemoryWrite。
// project 在 RPC 非空时放进 memory.write。否则传给 App.MemoryWrite。
func (s *Service) MemoryWrite(kind, text, project string) memory.Item {
	if s.RPC != nil {
		v, _ := s.call("memory.write", map[string]any{"kind": kind, "text": text, "project": project})
		out, _ := decode[memory.Item](v, nil)
		return out
	}
	return s.App.MemoryWrite(kind, text, project)
}
