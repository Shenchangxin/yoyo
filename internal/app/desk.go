// desk.go 持有桌面会话的 dsh 宿主，以及编排台的运行时。
// 发送回合调用 agentDesk。编排台的快照、决定和来源轮询调用 orchestrator。

package app

import (
	"context"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/dshdesk"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// agentDesk 返回进程内的桌面代理宿主。第一次调用时创建。
func (a *App) agentDesk() *dshdesk.Host {
	if a == nil {
		return dshdesk.New()
	}
	a.deskMu.Lock()
	defer a.deskMu.Unlock()
	if a.desk == nil {
		a.desk = dshdesk.New()
	}
	return a.desk
}

// orchestrator 返回编排运行时。第一次调用时打开 SQLite、接上模型，并开始两秒轮次。
func (a *App) orchestrator() *superagent.Runtime {
	if a == nil {
		return nil
	}
	a.deskMu.Lock()
	defer a.deskMu.Unlock()
	if a.orch != nil {
		return a.orch
	}
	path := ""
	dir := ""
	if a.Home != nil {
		dir = a.Home.Root
		path = filepath.Join(dir, "superagent.sqlite")
	}
	rt, err := superagent.Open(path, superagent.Policy{Mode: "observe", Version: 1})
	if err != nil {
		rt, _ = superagent.Open("", superagent.Policy{Mode: "observe", Version: 1})
	}
	if rt == nil {
		return nil
	}
	work := dir
	rt.Bind(func() error {
		_, err := a.ClientFor(SessionMeta{})
		return err
	}, liveModel{app: a}, orchestratorShell(work))
	rt.Start()
	a.orch = rt
	return rt
}

// stopOrchestrator 停掉两秒轮次。还没打开过时什么都不做。
func (a *App) stopOrchestrator() {
	if a == nil {
		return
	}
	a.deskMu.Lock()
	rt := a.orch
	a.orch = nil
	a.deskMu.Unlock()
	if rt != nil {
		rt.Stop()
	}
}

// OrchestratorView 返回编排台快照。还没有运行时时会先打开。
func (a *App) OrchestratorView() superagent.View {
	rt := a.orchestrator()
	if rt == nil {
		return superagent.View{Health: "unavailable"}
	}
	return rt.View()
}

// OrchestratorDecide 评估 triggerID。kind 与 reason 写进决定。
func (a *App) OrchestratorDecide(triggerID, kind, reason string) error {
	rt := a.orchestrator()
	if rt == nil {
		return errString("orchestrator unavailable")
	}
	return rt.Decide(triggerID, kind, reason)
}

// OrchestratorMode 换成 mode。mode 只能是 off、observe 或 auto。
func (a *App) OrchestratorMode(mode string) error {
	rt := a.orchestrator()
	if rt == nil {
		return errString("orchestrator unavailable")
	}
	return rt.SetMode(mode)
}

// OrchestratorPaused 设置暂停。paused 为真时两秒轮次不接受新触发。
func (a *App) OrchestratorPaused(paused bool) error {
	rt := a.orchestrator()
	if rt == nil {
		return errString("orchestrator unavailable")
	}
	return rt.SetPaused(paused)
}

// OrchestratorDeliver 把还没交给编排台的评估标成已交付。返回新交付的轮数。
func (a *App) OrchestratorDeliver() int {
	rt := a.orchestrator()
	if rt == nil {
		return 0
	}
	return rt.Deliver()
}

// OrchestratorAct 执行编排台动作。kind、ticketID、triggerID、text 和 reason 交给运行时。
func (a *App) OrchestratorAct(kind, ticketID, triggerID, text, reason string) error {
	rt := a.orchestrator()
	if rt == nil {
		return errString("orchestrator unavailable")
	}
	return rt.Act(kind, ticketID, triggerID, text, reason)
}

// OrchestratorAdmission 写下三格准入。listedUnaddressed、unlistedAddressed、unlistedUnaddressed 只能是空、explicit、inferred 或 ignore。
func (a *App) OrchestratorAdmission(listedUnaddressed, unlistedAddressed, unlistedUnaddressed string) error {
	rt := a.orchestrator()
	if rt == nil {
		return errString("orchestrator unavailable")
	}
	return rt.SetAdmission(listedUnaddressed, unlistedAddressed, unlistedUnaddressed)
}

// liveModel 每次补全都取当前配置的客户端。占位模型名 yoyo-dsh 换成配置里的模型。
type liveModel struct {
	app *App
}

// Complete 调用当前客户端。req 的消息原样交给桥。ctx 取消时停。
func (m liveModel) Complete(ctx context.Context, req dsh.Request) (dsh.Assistant, error) {
	if m.app == nil {
		return dsh.Assistant{}, errString("orchestrator unavailable")
	}
	client, err := m.app.ClientFor(SessionMeta{})
	if err != nil {
		return dsh.Assistant{}, err
	}
	if req.Model == "" || req.Model == "yoyo-dsh" {
		req.Model = m.app.Config.Model
	}
	bridge := &dshdesk.Bridge{Client: client, Model: m.app.Config.Model, ShowThinking: m.app.Config.ShowThinking}
	return bridge.Complete(ctx, req)
}

// orchestratorShell 返回在 dir 里执行命令的函数。command 交给系统 shell，最多跑 10 分钟。
func orchestratorShell(dir string) func(ctx context.Context, command string) (string, error) {
	return func(ctx context.Context, command string) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		var cmd *exec.Cmd
		if goruntime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", command)
		}
		if dir != "" {
			cmd.Dir = dir
		}
		out, err := cmd.CombinedOutput()
		text := string(out)
		if len(text) > 8000 {
			text = text[:8000]
		}
		return text, err
	}
}
