package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/browser"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/computeruse"
	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/expert"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/isolation"
	"github.com/Shenchangxin/yoyo/internal/kernel"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/observe"
	"github.com/Shenchangxin/yoyo/internal/project"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

func (a *App) mountFiber(name string, setup func() (func() error, error)) error {
	_, err := a.Kernel.Plugin(name, func(c *kernel.Context) error {
		return c.Effect(setup)
	})
	return err
}

func (a *App) openPersonal() error {
	if err := a.mountFiber("memory", func() (func() error, error) {
		mem, err := memory.Open(a.Home.Memory())
		if err != nil {
			return nil, err
		}
		a.Memory = mem
		return func() error { a.Memory = nil; return nil }, nil
	}); err != nil {
		return err
	}
	if err := a.mountFiber("scheduler", func() (func() error, error) {
		sched, err := schedule.Open(a.Home.Schedule())
		if err != nil {
			return nil, err
		}
		a.Schedule = sched
		a.schedStop = make(chan struct{})
		go a.scheduleLoop()
		return func() error {
			if a.schedStop != nil {
				select {
				case <-a.schedStop:
				default:
					close(a.schedStop)
				}
			}
			a.Schedule = nil
			return nil
		}, nil
	}); err != nil {
		return err
	}
	if err := a.mountFiber("connectors", func() (func() error, error) {
		conn, err := connector.Open(a.Home.Connectors())
		if err != nil {
			return nil, err
		}
		a.Connectors = conn
		a.Connectors.SetTokens(a.Vault)
		return func() error { a.Connectors = nil; return nil }, nil
	}); err != nil {
		return err
	}
	if err := a.mountFiber("browser", func() (func() error, error) {
		br, err := browser.Open(a.Home.Browser())
		if err != nil {
			return nil, err
		}
		a.Browser = br
		return func() error {
			if a.Browser != nil {
				a.Browser.Close()
			}
			a.Browser = nil
			return nil
		}, nil
	}); err != nil {
		return err
	}
	if err := a.mountFiber("office", func() (func() error, error) {
		return func() error { return nil }, nil
	}); err != nil {
		return err
	}
	if err := a.mountFiber("preview", func() (func() error, error) {
		return func() error { return nil }, nil
	}); err != nil {
		return err
	}
	proj, err := project.Open(a.Home.Projects())
	if err != nil {
		return err
	}
	box, err := inbox.Open(a.Home.Inbox())
	if err != nil {
		return err
	}
	cu, err := computeruse.Open(a.Home.Computer())
	if err != nil {
		return err
	}
	a.Projects = proj
	a.Inbox = box
	a.Computer = cu
	a.Observe = observe.Open(a.Home.Observe())
	ep := strings.TrimSpace(os.Getenv("YOYO_OTEL_ENDPOINT"))
	if ep == "" {
		ep = strings.TrimSpace(a.Config.Log.OTELEndpoint)
	}
	if ep != "" {
		a.Observe.SetEndpoint(ep)
	}
	return nil
}

func (a *App) attachPersonal(tools *runtime.WorkspaceTools) {
	if tools == nil {
		return
	}
	tools.Memory = a.Memory
	tools.Schedule = a.Schedule
	tools.Projects = a.Projects
	tools.Connectors = a.Connectors
	tools.Browser = a.Browser
	tools.Computer = a.Computer
	tools.Inbox = a.Inbox
	tools.SearchAPI = a.searchAPI
	tools.ChatOverlay = true
	tools.Advertised = runtime.ApplyChatToolMenu(tools.Advertised)
	if a.Threads != nil && tools.SessionID != "" {
		tools.AllowedConnectors = a.Threads.ConnectorAllow(tools.SessionID)
	}
}

func (a *App) searchAPI(query string) (string, error) {
	endpoint := strings.TrimSpace(a.Config.SearchURL)
	if endpoint == "" {
		return "", fmt.Errorf("no search_url")
	}
	keyName := a.Config.SearchKey
	if keyName == "" {
		keyName = "search"
	}
	key, _ := a.Vault.Lease(keyName)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if strings.Contains(endpoint, "{q}") {
		u := strings.ReplaceAll(endpoint, "{q}", url.QueryEscape(query))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return "", err
		}
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 128<<10))
		return string(b), nil
	}
	payload, _ := json.Marshal(map[string]any{"query": query, "q": query})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 128<<10))
	return string(b), nil
}

func (a *App) IsolationReport() map[string]any {
	iso := isolation.Status()
	vault := a.Vault.Status()
	browser := false
	if a.Browser != nil {
		browser = a.Browser.Isolated()
	}
	return map[string]any{
		"os":               iso,
		"vault":            vault,
		"browser_isolated": browser,
		"computer_display": "virtual",
		"sandbox_badge_ok": iso.Sandbox,
		"gate_mode":        capability.ParseGateMode(a.Config.GateMode),
		"crash_resume":     a.crashResume(),
		"awake_required":   true,
		"lid_close_stops":  true,
	}
}

func (a *App) crashResume() bool {
	if a == nil {
		return true
	}
	return a.Config.CrashResume
}

func (a *App) InboxList() []inbox.Item {
	if a.Inbox == nil {
		return nil
	}
	return a.Inbox.List()
}

func (a *App) ProjectsList() []project.Project {
	if a.Projects == nil {
		return nil
	}
	return a.Projects.List()
}

func (a *App) ProjectCreate(p project.Project) project.Project {
	if a.Projects == nil {
		return p
	}
	return a.Projects.Create(p)
}

func (a *App) MemoryList(q, kind string) []memory.Item {
	if a.Memory == nil {
		return nil
	}
	return a.Memory.Search(q, memory.Kind(kind), 50)
}

func (a *App) MemoryPromote(id string) bool {
	if a.Memory == nil {
		return false
	}
	return a.Memory.Promote(id)
}

func (a *App) ScheduleList() []schedule.Job {
	if a.Schedule == nil {
		return nil
	}
	return a.Schedule.List()
}

func (a *App) ScheduleCreate(j schedule.Job) schedule.Job {
	if a.Schedule == nil {
		return j
	}
	out := a.Schedule.Create(j)
	if a.Inbox != nil {
		a.Inbox.Push(inbox.Item{Kind: inbox.KindSchedule, Title: "Scheduled: " + out.ID, Body: out.Prompt, SessionID: ""})
	}
	return out
}

func (a *App) ScheduleCancel(id string) bool {
	if a.Schedule == nil {
		return false
	}
	return a.Schedule.Cancel(id)
}

func (a *App) ConnectorsList() []connector.Account {
	if a.Connectors == nil {
		return nil
	}
	return a.Connectors.List()
}

func (a *App) ConnectorConnect(acct connector.Account) connector.Account {
	if a.Connectors == nil {
		return acct
	}
	return a.Connectors.Connect(acct)
}

func (a *App) ConnectorDisconnect(id string) bool {
	if a.Connectors == nil {
		return false
	}
	return a.Connectors.Disconnect(id)
}

func (a *App) ComputerAllow(appName string) {
	if a.Computer != nil {
		a.Computer.Allow(appName)
	}
}

func (a *App) ComputerRecording() []computeruse.Event {
	if a.Computer == nil {
		return nil
	}
	return a.Computer.Recording()
}

func (a *App) DistillSkill(name, description, body string) string {
	return expert.Distill(name, description, body)
}

func (a *App) PhoneStatus() map[string]any {
	pending := 0
	if a.Gate != nil {
		pending = len(a.Gate.Pending())
	}
	unread := 0
	if a.Inbox != nil {
		unread = a.Inbox.Unread()
	}
	return map[string]any{
		"ok":        true,
		"pending":   pending,
		"inbox":     unread,
		"running":   a.RunningIDs(),
		"isolation": a.IsolationReport(),
		"jobs":      len(a.ScheduleList()),
	}
}

func (a *App) scheduleLoop() {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-a.schedStop:
			return
		case <-tick.C:
		}
		if a.Schedule == nil {
			continue
		}
		for _, j := range a.Schedule.Due(time.Now().UTC()) {
			j := j
			go a.runIsolatedJob(j)
		}
	}
}

func (a *App) StartMCPHTTP(name, endpoint string) error {
	if err := a.MCP.StartHTTP(name, endpoint); err != nil {
		diaglog.Error("mcp start http failed", "component", "mcp", "name", name, "err", err)
		return err
	}
	diaglog.Info("mcp started", "component", "mcp", "name", name, "transport", "http")
	a.persistMCP()
	return nil
}
