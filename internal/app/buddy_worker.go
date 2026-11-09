// buddy_worker.go 拥有外部 agent 当工人时 app 这一侧的接线：执行器的 Opener（在线检查、启动方式、会话授权、控制面登记、
// 权限请求和 fs 写入的检查）、每个 agent 的并行上限和名字，以及进程里唯一的控制面服务器的建和关。
// 权限请求和 yoyo 自己的工人走同一个 Broker 会话 ticket-<工单号>，命令还要过同一套确定性拒绝名单。
// 不拥有回合、日志和续接（internal/acpworker），不拥有工单状态（internal/superagent），也不拥有安装和检测（internal/buddy）。

package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/acpworker"
	"github.com/Shenchangxin/yoyo/internal/buddy"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/mcpserve"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/superagent"
	"github.com/Shenchangxin/yoyo/internal/version"
)

// errNoBroker 是没有能力代理时的审批结果：一律拒绝。
var errNoBroker = errors.New("no capability broker")

// errNoCommand 是看不到要跑什么命令时的审批结果：拒绝名单查不了，就不批。
var errNoCommand = errors.New("execute request carries no command")

// configureBuddyWorkers 建控制面。它在第一次登记时才开始监听。
func (a *App) configureBuddyWorkers() {
	a.control = mcpserve.New()
}

// dialBuddy 启动并连上 agent。探测、健康检查和工人都走这里，测试换过 buddyDial 时连同进程的假 agent。
func (a *App) dialBuddy(ctx context.Context, opts acp.Options) (*acp.Client, error) {
	if a.buddyDial != nil {
		return a.buddyDial(ctx, opts)
	}
	return acp.Dial(ctx, opts)
}

// closeBuddyWorkers 在应用退出时停掉所有外部 agent 进程（会话留着，下次启动续接），再关掉控制面。
func (a *App) closeBuddyWorkers() {
	acpworker.CloseAll()
	if a.control != nil {
		_ = a.control.Close()
	}
}

// openBuddyWorker 是执行器的 Opener：spec.Agent 在线时为这张工单打开一个外部工人，不在线时返回包着 ErrAgentOffline 的错误。
// 要先登录时错误里带上 ReasonAuthRequired，执行器就等人登录，不换人。
// 工人说明、续接编号和回调已经由执行器填好；这里补上启动方式、控制面、审批和写入检查。
func (a *App) openBuddyWorker(ctx context.Context, spec superagent.WorkerSpec) (superagent.Worker, error) {
	m := a.Buddies()
	if m == nil || a.control == nil {
		return nil, fmt.Errorf("%s: %w", spec.Agent, superagent.ErrAgentOffline)
	}
	id := spec.Agent
	m.Prepare(id)
	name := firstNonEmpty(a.buddyName(id), id)
	if !m.Online(id) {
		// 刚启动还没人探测过，或上次健康检查失败时，先探一次再说离线。
		_, _ = m.Probe(ctx, id)
	}
	if !m.Online(id) {
		if v, err := m.Get(id); err == nil && v.State == buddy.StateNeedsAuth {
			return nil, fmt.Errorf("%s: %s: %w", name, superagent.ReasonAuthRequired, superagent.ErrAgentOffline)
		}
		return nil, fmt.Errorf("%s: %w", name, superagent.ErrAgentOffline)
	}
	launch, secrets, err := m.LaunchRedacted(id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	a.applySessionAuth("ticket-"+spec.TicketID, capability.AuthFull)
	token := mcpserve.NewToken()
	if err := a.control.Register(token, spec.TicketID, controlTools(spec.Tools), superagent.WorkerToolSchemas()); err != nil {
		return nil, err
	}
	exe, _ := os.Executable()
	ctl := a.control
	return acpworker.Open(acpworker.Config{
		Spec:        spec,
		Launch:      launch,
		Dial:        a.dialBuddy,
		Version:     version.Version,
		MCPURL:      ctl.URL(),
		MCPToken:    token,
		Exe:         exe,
		Model:       m.Model(id),
		Approve:     a.buddyApprover(spec, m.Approval(id)),
		CheckWrite:  a.buddyWriteCheck(spec),
		TurnTimeout: time.Duration(a.buddySettings().TurnMinutesOf()) * time.Minute,
		OnAuth:      func(detail string) { m.NoteAuthFailure(id, detail) },
		OnRun:       func(err error) { m.NoteRun(id, err) },
		Redact:      func(s string) string { return buddy.Redact(s, secrets...) },
		OnClose:     func() { ctl.Unregister(token) },
	}), nil
}

// controlTools 只留 orchestrator_* 工具。控制面不给外部 agent 别的东西。
func controlTools(tools []dsh.Tool) []dsh.Tool {
	var out []dsh.Tool
	for _, tool := range tools {
		if strings.HasPrefix(tool.Name, "orchestrator_") {
			out = append(out, tool)
		}
	}
	return out
}

// buddyLimit 是 agent 同时最多几张工单。yoyo 只受全局槽限制，返回 0。
func (a *App) buddyLimit(agent string) int {
	if agent == "" || agent == buddy.YoyoID {
		return 0
	}
	if m := a.Buddies(); m != nil {
		return m.MaxParallel(agent)
	}
	return 0
}

// buddyName 是 agent 给人看的名字，找不到时为空。
func (a *App) buddyName(agent string) string {
	m := a.Buddies()
	if m == nil || agent == "" {
		return ""
	}
	v, err := m.Get(agent)
	if err != nil {
		return ""
	}
	return v.Name
}

// buddyApprover 把外部 agent 的权限请求换成 Broker 的能力检查。mode 是这一项的审批方式：yoyo 交给会话授权和门禁，
// ask（每次问我）时除了 orchestrator_* 和 think 都强制问人。
func (a *App) buddyApprover(spec superagent.WorkerSpec, mode string) func(context.Context, acpworker.Approval) (bool, error) {
	session := "ticket-" + spec.TicketID
	ask := mode == "ask"
	home := ""
	if a.Home != nil {
		home = a.Home.Root
	}
	readable := append(slices.Clone(spec.Roots), spec.ReadRoots...)
	check := func(ctx context.Context, kind string, level capability.Level, path, command string, force bool) error {
		if a.Caps == nil {
			return errNoBroker
		}
		roots := spec.Roots
		if level == capability.ReadWorkspace {
			roots = readable
		}
		return a.Caps.CheckCtx(ctx, capability.Request{
			Level: level, Action: "buddy." + kind, Path: path, Command: command,
			SessionID: session, Workspace: spec.Dir, Roots: slices.Clone(roots), ForceAsk: force || ask,
		})
	}
	return func(ctx context.Context, req acpworker.Approval) (bool, error) {
		if strings.HasPrefix(req.Name, "orchestrator_") || req.Kind == "think" {
			return true, nil
		}
		var err error
		switch req.Kind {
		case "edit", "delete", "move":
			if len(req.Paths) == 0 {
				err = check(ctx, req.Kind, capability.WriteWorkspace, "", "", false)
			}
			for _, p := range req.Paths {
				if capability.WithinAny(spec.ReadRoots, p) {
					err = fmt.Errorf("%s is read-only for this ticket", p)
				} else {
					err = check(ctx, req.Kind, capability.WriteWorkspace, p, "", false)
				}
				if err != nil {
					break
				}
			}
		case "read", "search":
			if len(req.Paths) == 0 {
				err = check(ctx, req.Kind, capability.ReadWorkspace, spec.Dir, "", false)
			}
			for _, p := range req.Paths {
				if err = check(ctx, req.Kind, capability.ReadWorkspace, p, "", false); err != nil {
					break
				}
			}
		case "execute":
			command := firstNonEmpty(strings.TrimSpace(req.Command), strings.TrimSpace(req.Title))
			if command == "" {
				err = errNoCommand
			} else if err = runtime.ShellDenied(command, spec.Dir, nil, readable...); err == nil {
				if err = runtime.DenyHomePlanes(runtime.SplitShellArgv(command), home); err == nil {
					err = check(ctx, req.Kind, capability.Shell, spec.Dir, command, false)
				}
			}
		case "fetch":
			err = check(ctx, req.Kind, capability.Network, "", fetchTarget(req), false)
		case "switch_mode":
			err = check(ctx, req.Kind, capability.HighRisk, "", req.Label(), true)
		default:
			if ask {
				err = check(ctx, req.Kind, capability.HighRisk, "", req.Label(), true)
			}
		}
		if err != nil {
			return false, err
		}
		return true, nil
	}
}

// fetchTarget 是 fetch 请求的地址：rawInput 里的 url，没有就用标题。
func fetchTarget(req acpworker.Approval) string {
	var in struct {
		URL string `json:"url"`
	}
	if len(req.Input) > 0 && json.Unmarshal(req.Input, &in) == nil && strings.TrimSpace(in.URL) != "" {
		return strings.TrimSpace(in.URL)
	}
	return strings.TrimSpace(req.Title)
}

// buddyWriteCheck 是 fs 写入前的检查：和 yoyo 自己的工人写文件一样，走 Broker 的 WriteWorkspace。
func (a *App) buddyWriteCheck(spec superagent.WorkerSpec) func(context.Context, string) error {
	session := "ticket-" + spec.TicketID
	return func(ctx context.Context, path string) error {
		if a.Caps == nil {
			return errNoBroker
		}
		return a.Caps.CheckCtx(ctx, capability.Request{
			Level: capability.WriteWorkspace, Action: "buddy.write", Path: path,
			SessionID: session, Workspace: spec.Dir, Roots: slices.Clone(spec.Roots),
		})
	}
}
