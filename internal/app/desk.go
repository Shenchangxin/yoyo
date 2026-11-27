// desk.go 持有桌面会话的 dsh 宿主，以及编排台的运行时。
// 发送回合调用 agentDesk。编排台的快照、决定和来源轮询调用 orchestrator。

package app

import (
	"context"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/buddy"
	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/dshdesk"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// agentDesk 返回进程内的桌面代理宿主。第一次调用时创建。
func (appHost *deskHost) agentDesk() *dshdesk.Host {
	if appHost.app == nil {
		return dshdesk.New()
	}
	appHost.app.deskMu.Lock()
	defer appHost.app.deskMu.Unlock()
	if appHost.app.desk == nil {
		appHost.app.desk = dshdesk.New()
	}
	return appHost.app.desk
}

// orchestrator 返回编排运行时。第一次调用时打开 SQLite、接上模型和工人，并开始两秒轮次。应用关掉之后返回 nil。
func (appHost *deskHost) orchestrator() *superagent.Runtime {
	if appHost.app == nil {
		return nil
	}
	appHost.app.deskMu.Lock()
	if appHost.app.orch != nil || appHost.app.orchShut {
		rt := appHost.app.orch
		appHost.app.deskMu.Unlock()
		return rt
	}
	path := ""
	dir := ""
	if appHost.app.Home != nil {
		dir = appHost.app.Home.Root
		path = filepath.Join(dir, "superagent.sqlite")
	}
	rt, err := superagent.Open(path, superagent.Policy{Mode: "observe", Version: 1})
	if err != nil {
		rt, _ = superagent.Open("", superagent.Policy{Mode: "observe", Version: 1})
	}
	if rt == nil {
		appHost.app.deskMu.Unlock()
		return nil
	}
	work := dir
	rt.SetWorkRoot(work)
	rt.Bind(func() error {
		_, err := appHost.app.ClientFor(SessionMeta{})
		return err
	}, liveModel{app: appHost.app}, orchestratorShell(work))
	rt.SetMount(appHost.app.mountWorker)
	rt.SetOpener(appHost.app.openBuddyWorker)
	rt.SetAgentLimit(appHost.app.buddyLimit)
	rt.SetAgentName(appHost.app.buddyName)
	rt.SetRoots(func(ticketID string) []string { return appHost.app.namedWorkRoots(ticketID, dir) })
	rt.SetProfile(appHost.app.workerProfile)
	rt.SetSkills(appHost.workerSkills)
	rt.SetOnReport(appHost.app.relayReport)
	rt.SetOnNotice(appHost.app.relayNotice)
	rt.SetOnWake(appHost.app.wakeRouter)
	rt.SetPrepareRunner(orchestratorPrepare(work))
	rt.SetIntake(appHost.app.connectorTriggers)
	appHost.app.orch = rt
	appHost.app.deskMu.Unlock()
	// 应用本身没暂停时，清掉快照里留下的编排暂停。
	// 那一档没有单独开关，留着真值时人点的任务不会开工，页面却像正在执行。
	if !appHost.app.Config.Paused {
		_ = rt.SetPaused(false)
	}
	rt.Start()
	go func() {
		time.Sleep(time.Second)
		appHost.app.relayWaiting()
	}()
	return rt
}

// workerProfileBytes 是工人系统提示里操作者画像的上限。
const workerProfileBytes = 1500

// workerProfile 是交给工人的操作者画像：偏好和本人说过的话。没有记忆库时为空。
func (appHost *deskHost) workerProfile() string {
	if appHost.app == nil || appHost.app.Memory == nil {
		return ""
	}
	return appHost.app.Memory.OperatorPin(workerProfileBytes)
}

// workerSkills 是交给外部工人的技能清单。只收有目录的技能：外部 agent 要能自己读到 SKILL.md。
// 每次派工时现取，装上或卸掉的技能下一张工单就能看到。
func (appHost *deskHost) workerSkills() []superagent.SkillRef {
	if appHost.app == nil {
		return nil
	}
	var out []superagent.SkillRef
	for _, s := range appHost.app.collectSkills("") {
		if s.Name == "" || s.Dir == "" {
			continue
		}
		out = append(out, superagent.SkillRef{Name: s.Name, Description: s.Description, Dir: s.Dir})
	}
	return out
}

// relayWaiting 把还等人的工单补进当前主会话。已经写过的同一段正文不会再写。
func (appHost *deskHost) relayWaiting() {
	if appHost.app == nil {
		return
	}
	appHost.app.deskMu.Lock()
	rt := appHost.app.orch
	appHost.app.deskMu.Unlock()
	if rt == nil {
		return
	}
	view := rt.View()
	for _, row := range view.Tickets {
		state, summary, question, ok := handoffAsk(row)
		if !ok {
			continue
		}
		appHost.app.relayReport(row.ID, state, summary, question)
	}
}

// stopOrchestrator 停掉两秒轮次，之后不再打开。还没打开过时只记下已关。
func (appHost *deskHost) stopOrchestrator() {
	if appHost.app == nil {
		return
	}
	appHost.app.deskMu.Lock()
	rt := appHost.app.orch
	appHost.app.orch = nil
	appHost.app.orchShut = true
	appHost.app.deskMu.Unlock()
	if rt != nil {
		rt.Stop()
	}
}

// OrchestratorView 返回编排台快照。还没有运行时时会先打开。
func (appHost *deskHost) OrchestratorView() superagent.View {
	rt := appHost.app.orchestrator()
	if rt == nil {
		return superagent.View{Health: "unavailable"}
	}
	return rt.View()
}

// OrchestratorDecide 评估 triggerID。kind 与 reason 写进决定。
func (appHost *deskHost) OrchestratorDecide(triggerID, kind, reason string) error {
	rt := appHost.app.orchestrator()
	if rt == nil {
		return errString("orchestrator unavailable")
	}
	if kind == "accept" {
		return rt.Confirm(triggerID, kind, reason)
	}
	return rt.Decide(triggerID, kind, reason)
}

// OrchestratorMode 换成 mode。mode 只能是 off、observe 或 auto。
func (appHost *deskHost) OrchestratorMode(mode string) error {
	rt := appHost.app.orchestrator()
	if rt == nil {
		return errString("orchestrator unavailable")
	}
	return rt.SetMode(mode)
}

// OrchestratorPaused 设置暂停。paused 为真时两秒轮次不接受新触发。
func (appHost *deskHost) OrchestratorPaused(paused bool) error {
	rt := appHost.app.orchestrator()
	if rt == nil {
		return errString("orchestrator unavailable")
	}
	return rt.SetPaused(paused)
}

// OrchestratorDeliver 把还没交给编排台的评估标成已交付。返回新交付的轮数。
func (appHost *deskHost) OrchestratorDeliver() int {
	rt := appHost.app.orchestrator()
	if rt == nil {
		return 0
	}
	return rt.Deliver()
}

// WorkerTranscript 返回工单已经落下的会话。ticketID 为空时为空。
func (appHost *deskHost) WorkerTranscript(ticketID string) []superagent.TranscriptLine {
	rt := appHost.app.orchestrator()
	if rt == nil {
		return []superagent.TranscriptLine{}
	}
	lines := rt.WorkerTranscript(ticketID)
	if lines == nil {
		return []superagent.TranscriptLine{}
	}
	return lines
}

// OrchestratorAct 执行编排台动作。kind、ticketID、triggerID、text 和 reason 交给运行时。
// 换人时 text 是 agent 编号，空着是 yoyo；现在派不了的先拒掉，不让工单落到一个开不起来的工人上。
func (appHost *deskHost) OrchestratorAct(kind, ticketID, triggerID, text, reason string) error {
	rt := appHost.app.orchestrator()
	if rt == nil {
		return errString("orchestrator unavailable")
	}
	if kind == "reassign" {
		agent := strings.ToLower(strings.TrimSpace(text))
		if agent == "" {
			agent = buddy.YoyoID
		}
		if why := appHost.app.agentRoutable(agent); why != "" {
			return errString(agent + " " + why)
		}
		text = agent
	}
	return rt.Act(kind, ticketID, triggerID, text, reason)
}

// OrchestratorAdmission 写下三格准入。listedUnaddressed、unlistedAddressed、unlistedUnaddressed 只能是空、explicit、inferred 或 ignore。
func (appHost *deskHost) OrchestratorAdmission(listedUnaddressed, unlistedAddressed, unlistedUnaddressed string) error {
	rt := appHost.app.orchestrator()
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

// orchestratorShell 返回执行命令的函数。dir 非空时用它，否则用 fallback。command 交给系统 shell，最多跑 10 分钟。
func orchestratorShell(fallback string) func(ctx context.Context, dir, command string) (string, error) {
	return func(ctx context.Context, dir, command string) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		var cmd *exec.Cmd
		if goruntime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", command)
		}
		if dir == "" {
			dir = fallback
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
