// 打开记忆、页面、档案、日程、连接器、浏览器和个人引擎，并把它们挂到回合工具上。搜索有 20 秒超时。日程循环每 30 秒看一次到期任务。

package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/browser"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/computeruse"
	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/expert"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/isolation"
	"github.com/Shenchangxin/yoyo/internal/kernel"
	"github.com/Shenchangxin/yoyo/internal/learn"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/observe"
	"github.com/Shenchangxin/yoyo/internal/personal"
	"github.com/Shenchangxin/yoyo/internal/profile"
	"github.com/Shenchangxin/yoyo/internal/project"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/schedule"
	"github.com/Shenchangxin/yoyo/internal/tool"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// mountFiber 用内核插件挂上一个可撤销的效果。
// name 传给 Kernel.Plugin。
// setup 传给 Effect。
func (a *App) mountFiber(name string, setup func() (func() error, error)) error {
	_, err := a.Kernel.Plugin(name, func(c *kernel.Context) error {
		return c.Effect(setup)
	})
	return err
}

// openPersonal 依次打开记忆、页面、档案、日程、连接器、浏览器，以及空的 office 和 preview 纤维，然后打开项目、收件箱、电脑使用、观测和个人引擎。日程循环在 scheduler 纤维里启动。个人变更发到 Hub 的 personal 事件。
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
	if err := a.mountFiber("pages", func() (func() error, error) {
		store, err := learn.Open(a.Home.Learn(), a.Home.Pages())
		if err != nil {
			return nil, err
		}
		store.Embed = a.embedTexts
		a.Learn = store
		a.Pages = store
		return func() error {
			a.Pages = nil
			a.Learn = nil
			return store.Close()
		}, nil
	}); err != nil {
		return err
	}
	if err := a.mountFiber("profiles", func() (func() error, error) {
		store, err := profile.Open(a.Home.Profiles())
		if err != nil {
			return nil, err
		}
		a.Profiles = store
		return func() error { a.Profiles = nil; return nil }, nil
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
	if ep := a.otelEndpoint(); ep != "" {
		a.Observe.SetEndpoint(ep)
	}
	return a.mountFiber("personal-os", func() (func() error, error) {
		eng, err := personal.Open(a.Home.Personal())
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithCancel(context.Background())
		eng.Bind(personal.Bind{
			Connectors: a.Connectors,
			Inbox:      a.Inbox,
			Workspace:  a.Workspace(),
			OnChange: func() {
				a.Hub.Publish(trace.Event{
					TS:       time.Now().UTC(),
					Type:     trace.TypePersonal,
					Source:   "personal",
					ItemKind: "personal",
				})
			},
			Paused: a.IsPaused,
		})
		a.Personal = eng
		go eng.Run(ctx)
		return func() error {
			cancel()
			a.Personal = nil
			return nil
		}, nil
	})
}

// attachPersonal 把记忆、日程、项目、连接器、个人引擎、浏览器、电脑使用、收件箱和页面挂到工具上，并打开 ChatOverlay。档案的 AllowTools 非空时，用宿主工具名加已广告名滤出 AllowedTools 并置上 ToolJail。
// tools 为 nil 则返回，否则挂上记忆、日程、项目、连接器、个人引擎、浏览器、电脑使用、收件箱、页面和 SearchAPI，并打开 ChatOverlay。
func (a *App) attachPersonal(tools *runtime.WorkspaceTools) {
	if tools == nil {
		return
	}
	tools.Memory = a.Memory
	tools.Schedule = a.Schedule
	tools.Projects = a.Projects
	tools.Connectors = a.Connectors
	tools.Personal = a.Personal
	tools.Browser = a.Browser
	tools.Computer = a.Computer
	tools.Inbox = a.Inbox
	tools.Pages = a.Pages
	tools.SearchAPI = a.searchAPI
	tools.ChatOverlay = true
	tools.Advertised = runtime.ApplyChatToolMenu(tools.Advertised)
	if p := a.sessionProfile(tools.SessionID); p != nil {
		tools.Advertised = profile.FilterTools(tools.Advertised, *p)
		tools.DeniedTools = profile.DeniedNames(*p)
		// allow_tools 才是调用监禁。把开场 schema 抄进 AllowedTools 不算：browser_verify 这类稍后才挂上的宿主工具，在 tool_search 之后仍必须能调用。
		if len(p.AllowTools) > 0 {
			names := make([]string, 0, len(tool.HostSpecs())+len(tools.Advertised))
			for _, s := range tool.HostSpecs() {
				names = append(names, s.Name)
			}
			names = append(names, tools.Advertised...)
			tools.AllowedTools = profile.FilterTools(names, *p)
			tools.ToolJail = true
		}
	}
	if a.Threads != nil && tools.SessionID != "" {
		tools.AllowedConnectors = a.Threads.ConnectorAllow(tools.SessionID)
	}
	a.attachLearn(tools)
}

// searchAPI 优先用视频连接里 capability 为 search 的活动连接，否则用 Config.SearchURL。密钥名默认 search。超时 20 秒。两端都空则返回 no search_url。
// query 传给 connection.QuerySearch。
func (a *App) searchAPI(query string) (string, error) {
	endpoint := strings.TrimSpace(a.Config.SearchURL)
	keyName := a.Config.SearchKey
	if keyName == "" {
		keyName = "search"
	}
	key, _ := a.Vault.Lease(keyName)
	var searchConn connection.Connection
	if a.Video != nil && a.Video.Conn != nil {
		if c, err := a.Video.Conn.Active("search", ""); err == nil {
			searchConn = c
			if ep := strings.TrimSpace(c.Endpoint); ep != "" {
				endpoint = ep
			}
			if k, e := a.Video.Conn.Lease(c); e == nil && strings.TrimSpace(k) != "" {
				key = k
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if searchConn.ID != "" && strings.TrimSpace(searchConn.Endpoint) != "" {
		return connection.QuerySearch(ctx, searchConn, key, query)
	}
	if endpoint == "" {
		return "", fmt.Errorf("no search_url")
	}
	searchConn.Endpoint = endpoint
	searchConn.Protocol = "search"
	return connection.QuerySearch(ctx, searchConn, key, query)
}

// IsolationReport 返回 isolation.Status、保险库状态和浏览器是否隔离。computer_display 写成 virtual，awake_required 和 lid_close_stops 写成 true，sandbox_badge_ok 取 Status 的 Sandbox 字段。
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

// crashResume 在 App 为空时返回 true，否则返回 Config.CrashResume。
func (a *App) crashResume() bool {
	if a == nil {
		return true
	}
	return a.Config.CrashResume
}

// InboxList 返回收件箱列表。收件箱为空时返回 nil。
func (a *App) InboxList() []inbox.Item {
	if a.Inbox == nil {
		return nil
	}
	return a.Inbox.List()
}

// ProjectsList 返回项目列表。项目库为空时返回 nil。
func (a *App) ProjectsList() []project.Project {
	if a.Projects == nil {
		return nil
	}
	return a.Projects.List()
}

// ProjectCreate 创建项目。项目库为空时返回入参本身。
// p 在项目库为空时原样返回，否则传给 Projects.Create。
func (a *App) ProjectCreate(p project.Project) project.Project {
	if a.Projects == nil {
		return p
	}
	return a.Projects.Create(p)
}

// MemoryList 按查询和种类搜索记忆，最多 50 条。记忆库为空时返回 nil。
// q 传给 Memory.Search。
// kind 转成 memory.Kind 后传给 Memory.Search。
func (a *App) MemoryList(q, kind string) []memory.Item {
	if a.Memory == nil {
		return nil
	}
	return a.Memory.Search(q, memory.Kind(kind), 50)
}

// MemoryPromote 把 id 交给 Memory.Promote。记忆库为空返回 false。晋升条件由记忆库决定。
func (a *App) MemoryPromote(id string) bool {
	if a.Memory == nil {
		return false
	}
	return a.Memory.Promote(id)
}

// ScheduleList 返回日程列表。日程为空时返回 nil。
func (a *App) ScheduleList() []schedule.Job {
	if a.Schedule == nil {
		return nil
	}
	return a.Schedule.List()
}

// ScheduleCreate 创建日程，并往收件箱推一条 KindSchedule。日程为空时返回入参。
// j 在日程为空时原样返回，否则传给 Schedule.Create。
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

// ScheduleCancel 取消一条日程。日程为空返回 false。
// id 传给 Schedule.Cancel。
func (a *App) ScheduleCancel(id string) bool {
	if a.Schedule == nil {
		return false
	}
	return a.Schedule.Cancel(id)
}

// ConnectorsList 返回连接器账号。连接器为空时返回 nil。
func (a *App) ConnectorsList() []connector.Account {
	if a.Connectors == nil {
		return nil
	}
	return a.Connectors.List()
}

// ConnectorConnect 连接一个账号。连接器为空时返回入参。
// acct 在连接器为空时原样返回，否则传给 Connectors.Connect。
func (a *App) ConnectorConnect(acct connector.Account) connector.Account {
	if a.Connectors == nil {
		return acct
	}
	return a.Connectors.Connect(acct)
}

// ConnectorDisconnect 断开账号。成功且个人引擎存在时 InvalidateAccount。
func (a *App) ConnectorDisconnect(id string) bool {
	if a.Connectors == nil {
		return false
	}
	ok := a.Connectors.Disconnect(id)
	if ok && a.Personal != nil {
		a.Personal.InvalidateAccount(id)
	}
	return ok
}

// ComputerAllow 把应用名交给电脑使用的 Allow。电脑使用为空则返回。
// appName 在电脑使用非空时传给 Computer.Allow。
func (a *App) ComputerAllow(appName string) {
	if a.Computer != nil {
		a.Computer.Allow(appName)
	}
}

// ComputerRecording 返回电脑使用录制。电脑使用为空时返回 nil。
func (a *App) ComputerRecording() []computeruse.Event {
	if a.Computer == nil {
		return nil
	}
	return a.Computer.Recording()
}

// DistillSkill 把名称、描述和正文交给 expert.Distill。
// name 传给 expert.Distill。
// description 传给 expert.Distill。
// body 传给 expert.Distill。
func (a *App) DistillSkill(name, description, body string) string {
	return expert.Distill(name, description, body)
}

// PhoneStatus 汇总待审批数、未读收件箱、正在跑的会话、隔离报告、日程条数和个人事务计数。
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
		"personal":  a.personalCounts(),
	}
}

// personalCounts 统计 waiting_input、waiting_approval、状态为 new 的想法，以及 awaiting_review 的提案。引擎为空返回空 map。
func (a *App) personalCounts() map[string]any {
	if a.Personal == nil {
		return map[string]any{}
	}
	s := a.Personal.Snapshot()
	waiting, ideas, proposals := 0, 0, 0
	for _, t := range s.Tasks {
		if t.Status == "waiting_input" || t.Status == "waiting_approval" {
			waiting++
		}
	}
	for _, it := range s.Ideas {
		if it.Status == "new" {
			ideas++
		}
	}
	for _, p := range s.Proposals {
		if p.Status == "awaiting_review" {
			proposals++
		}
	}
	return map[string]any{"waiting": waiting, "ideas": ideas, "proposals": proposals, "running": s.Worker.Running}
}

// scheduleLoop 每 30 秒取出到期日程并调用 runScheduledJob。暂停或日程为空时跳过这一拍。schedStop 关闭后退出。
func (a *App) scheduleLoop() {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-a.schedStop:
			return
		case <-tick.C:
		}
		if a.IsPaused() || a.Schedule == nil {
			continue
		}
		for _, j := range a.Schedule.Due(time.Now().UTC()) {
			j := j
			go a.runScheduledJob(j)
		}
	}
}

// StartMCPHTTP 按名字和端点启动 HTTP MCP，成功后 persistMCP。失败时记下诊断并返回错误。
// name 传给 MCP.StartHTTP。
// endpoint 传给 MCP.StartHTTP。
func (a *App) StartMCPHTTP(name, endpoint string) error {
	if err := a.MCP.StartHTTP(name, endpoint); err != nil {
		diaglog.Error("mcp start http failed", "component", "mcp", "name", name, "err", err)
		return err
	}
	diaglog.Info("mcp started", "component", "mcp", "name", name, "transport", "http")
	a.persistMCP()
	return nil
}
