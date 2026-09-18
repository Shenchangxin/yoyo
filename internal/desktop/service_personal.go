package desktop

import (
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
	if path == "" {
		ws := s.GetConfig().Workspace
		if ws == "" && s.App != nil {
			ws = s.App.Workspace()
		}
		path = ws + "/.yoyo/captures/shot.png"
	}
	return osutil.Screenshot(path)
}

func (s *Service) RaiseWindow() {
	Raise(s.win)
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

func (s *Service) MemoryWrite(kind, text, project string) memory.Item {
	if s.RPC != nil {
		v, _ := s.call("memory.write", map[string]any{"kind": kind, "text": text, "project": project})
		out, _ := decode[memory.Item](v, nil)
		return out
	}
	return s.App.MemoryWrite(kind, text, project)
}
