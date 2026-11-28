// lead.go 让同一个 home 上同时开着的桌面和 yoyo daemon 只有一个在跑后台。
// 拿着 lead.lock 的进程跑送达循环（开口短讯、晨晚简报、IM 攒着的消息）、定时例行、个人任务、IM 网关和编排轮次；另一个只服务自己的界面。
// 桌面开着时总是桌面领头：daemon 看到 desktop.lock 有人拿着就让出来，桌面关掉之后几秒内接回去。
// 跟着的进程每两秒读一次主会话文件，把领头进程写进来的短讯、简报和例行结果补发给自己的界面。
// 开口记录、日程、记忆、个人任务、IM 设置和例行结果都在改之前加锁并重读磁盘，两边谁改都不会盖掉对方。
// role 为空是旧的单进程行为（测试和直接 Open 的调用方）：后台在 Open 里就开始，不选主。tool 是一次性的命令行：不选主，也不跑后台。

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/lease"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// leadEvery 是选主、跟随和配置同步的间隔。桌面打开之后，daemon 最多两拍就让出来。
const leadEvery = 2 * time.Second

// leadBox 是这个进程在选主里的样子。
type leadBox struct {
	mu sync.Mutex
	// role 是 desktop、daemon、tool 或空。
	role    string
	held    *lease.Lease
	front   *lease.Lease
	leading bool
	// heldAs 是拿到租约时写下的持有人记录。收尾和取消收尾时只改它的 Draining 再写回去，Since 保持拿到时的样子。
	heldAs lease.Holder
	// drainSince 是这一段收尾从什么时候开始的。零值表示没在收尾。见 drainStep。
	drainSince time.Time
	stop       chan struct{}
	done       chan struct{}
	// tail 是跟着读的每个会话文件读到哪里了。
	tail map[string]int64
	// cfg 是上次读到或写下时 config.yaml 的样子。
	cfg diskStamp
	// personal 是个人任务的生命期，personalStop 停掉这一段的 worker。
	personal     context.Context
	personalStop context.CancelFunc
	// view 是跟随时最近一次读到的编排台，两秒内不重开数据库。
	viewAt time.Time
	view   superagent.View
}

// diskStamp 是一个文件的修改时间和长度。用来看别的进程有没有改过它。
type diskStamp struct {
	mod  time.Time
	size int64
}

func stampOf(path string) diskStamp {
	st, err := os.Stat(path)
	if err != nil {
		return diskStamp{}
	}
	return diskStamp{mod: st.ModTime(), size: st.Size()}
}

// DeskLead 是这台电脑上谁在跑主动链路。Role 是这个进程：lead 或 follow，不选主时为空。Holder 是拿着租约的那个：desktop 或 daemon。
// Draining 为真时拿着租约的那个已经决定让出，正在等手上在跑的回合跑完（见 drainStep），界面显示「收尾中」。
type DeskLead struct {
	Role     string `json:"role,omitempty"`
	Holder   string `json:"holder,omitempty"`
	Since    string `json:"since,omitempty"`
	Draining bool   `json:"draining,omitempty"`
}

// OpenManaged 打开应用并参加选主。role 是 desktop、daemon 或 tool。
// desktop 和 daemon 抢同一份租约，桌面优先；tool 不抢，也不跑后台，给一次性的命令行用。
func OpenManaged(root, bundledEvals, role string) (*App, error) {
	switch role {
	case "desktop", "daemon", "tool":
	default:
		return nil, fmt.Errorf("app: unknown role %q", role)
	}
	a, err := openAt(root, bundledEvals, role)
	if err != nil {
		return nil, err
	}
	a.startLead()
	return a, nil
}

// managed 在这个进程参加选主（或者是 tool）时为真。为假时后台在 Open 里就跑起来。
func (a *App) managed() bool {
	if a == nil {
		return false
	}
	a.lead.mu.Lock()
	defer a.lead.mu.Unlock()
	return a.lead.role != ""
}

// leading 在这个进程该跑后台时为真：不选主，或者拿着租约。
func (a *App) leading() bool {
	if a == nil {
		return false
	}
	a.lead.mu.Lock()
	defer a.lead.mu.Unlock()
	return a.lead.role == "" || a.lead.leading
}

func (a *App) leadPath() string  { return filepath.Join(a.Home.Root, "lead.lock") }
func (a *App) frontPath() string { return filepath.Join(a.Home.Root, "desktop.lock") }

// startLead 在 desktop 和 daemon 里开始选主：先同步试一次，桌面在没有 daemon 时打开就领头，然后每两拍看一次。
func (a *App) startLead() {
	if a == nil || a.Home == nil {
		return
	}
	a.lead.mu.Lock()
	role := a.lead.role
	a.lead.cfg = stampOf(a.Home.Config())
	a.lead.mu.Unlock()
	if role != "desktop" && role != "daemon" {
		return
	}
	if role == "desktop" {
		if f, err := lease.Try(a.frontPath(), a.holder()); err == nil {
			a.lead.mu.Lock()
			a.lead.front = f
			a.lead.mu.Unlock()
		}
	}
	a.leadStep()
	stop, done := make(chan struct{}), make(chan struct{})
	a.lead.mu.Lock()
	a.lead.stop, a.lead.done = stop, done
	a.lead.mu.Unlock()
	go a.leadLoop(stop, done)
}

func (a *App) holder() lease.Holder {
	a.lead.mu.Lock()
	role := a.lead.role
	a.lead.mu.Unlock()
	return lease.Holder{Role: role, PID: os.Getpid(), Since: time.Now().UTC().Format(time.RFC3339)}
}

func (a *App) leadLoop(stop <-chan struct{}, done chan struct{}) {
	defer close(done)
	tick := time.NewTicker(leadEvery)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			a.leadStep()
		}
	}
}

// leadStep 是一拍：daemon 看到桌面开着就进入收尾，手上的回合跑完（或等满 leadDrainMax）再让出来；
// 收尾中桌面又关掉了就取消收尾，接着领头；没人领头时去抢；领头时把别的进程改过的 IM 设置接上；两边都同步配置、跟读会话。
func (a *App) leadStep() {
	a.lead.mu.Lock()
	role, leading := a.lead.role, a.lead.leading
	a.lead.mu.Unlock()
	front := role == "daemon" && a.desktopOpen()
	switch {
	case leading && front:
		a.drainStep(time.Now())
	case leading:
		a.cancelDrain()
	case !front:
		a.tryLead()
	}
	if a.leading() && a.IM != nil {
		a.IM.Sync()
	}
	a.syncConfig()
	a.tailForeign()
}

// desktopOpen 在桌面进程开着（拿着 desktop.lock）时为真。
func (a *App) desktopOpen() bool {
	_, ok := lease.ReadHolder(a.frontPath())
	return ok
}

// tryLead 抢租约，抢到就把后台跑起来。拿到时的持有人记录留在 heldAs 里，收尾时只改 Draining，Since 不变。
func (a *App) tryLead() bool {
	h := a.holder()
	l, err := lease.Try(a.leadPath(), h)
	if err != nil {
		return false
	}
	a.lead.mu.Lock()
	a.lead.held = l
	a.lead.heldAs = h
	a.lead.leading = true
	a.lead.drainSince = time.Time{}
	a.lead.mu.Unlock()
	diaglog.Info("lead acquired", "component", "lead", "role", h.Role, "pid", h.PID)
	a.startRoles()
	return true
}

// stepDown 停掉后台再放开租约。先停后放：编排落盘、例行停拍之后，别的进程才拿得到。
func (a *App) stepDown(reason string) {
	a.lead.mu.Lock()
	if !a.lead.leading {
		a.lead.mu.Unlock()
		return
	}
	a.lead.leading = false
	a.lead.drainSince = time.Time{}
	held := a.lead.held
	a.lead.held = nil
	role := a.lead.role
	a.lead.mu.Unlock()
	a.stopRoles()
	held.Release()
	diaglog.Info("lead released", "component", "lead", "role", role, "reason", reason)
}

// leadDrainMax 是收尾最多等多久。手上的回合超过这个时间还没跑完，就照旧停掉让出：
// 会话回合下次续跑，编排工单下次启动时按 RecoverHost 恢复，不会丢，只是这一轮被打断。
const leadDrainMax = 3 * time.Minute

// drainStep 是 daemon 看到桌面开着时的一拍。不再当场停掉后台，而是先收尾：
//   - 第一拍：停掉会开新活的送达循环和定时例行，在持有人记录上标 Draining，让两边的界面显示「收尾中」；
//   - 之后每拍：看手上还有没有在跑的回合（busyTurns），没有了、或者等满 leadDrainMax，就 stepDown 让出。
//
// IM 网关和编排在收尾期间照常开着：IM 回合跑完要靠它把回复发出去，编排的工人要靠轮次收报告和落盘。
// 但收尾期间进来的、要开新回合的 IM 消息不再当场开回合，记进交接队列，由接手的进程处理（im_handoff.go），收尾不会被新消息拖长。
// 个人任务停止领新的（personal.Engine.Hold），在跑的那几件算进 busyTurns，跑完再让出，不再被半路取消。
func (a *App) drainStep(now time.Time) {
	a.lead.mu.Lock()
	since := a.lead.drainSince
	first := since.IsZero()
	if first {
		a.lead.drainSince = now
		since = now
	}
	held, h := a.lead.held, a.lead.heldAs
	a.lead.mu.Unlock()
	if first {
		a.stopDeskLoop()
		a.stopSchedule()
		a.Personal.Hold(true)
		h.Draining = true
		_ = held.Update(h)
	}
	busy := a.busyTurns()
	if first {
		diaglog.Info("lead draining", "component", "lead", "role", h.Role, "busy", busy)
	}
	switch {
	case busy == 0:
		a.stepDown("desktop opened")
	case now.Sub(since) >= leadDrainMax:
		a.stepDown(fmt.Sprintf("desktop opened; %d turns still running after %s", busy, leadDrainMax))
	}
}

// cancelDrain 在收尾中桌面又关掉时调用：取消收尾，把停掉的送达循环和定时例行接回来，持有人记录去掉 Draining。
// 没在收尾时什么都不做。
func (a *App) cancelDrain() {
	a.lead.mu.Lock()
	if a.lead.drainSince.IsZero() {
		a.lead.mu.Unlock()
		return
	}
	a.lead.drainSince = time.Time{}
	held, h := a.lead.held, a.lead.heldAs
	a.lead.mu.Unlock()
	h.Draining = false
	_ = held.Update(h)
	a.Personal.Hold(false)
	a.startSchedule()
	a.startDeskRole()
	diaglog.Info("lead drain cancelled", "component", "lead", "role", h.Role)
	// 收尾期间记下的 IM 消息由自己接着处理：没人来接手了。
	go a.imReplay()
}

// busyTurns 是这个进程手上还在跑的回合数：会话回合（主会话、IM、界面里的对话）、编排里启动中或正在跑的工单，
// 加上个人引擎已经领取、还没结束的任务。只看已经开着的编排，不为了数一下去打开它。
func (a *App) busyTurns() int {
	n := len(a.RunningIDs())
	a.deskMu.Lock()
	rt := a.orch
	a.deskMu.Unlock()
	return n + rt.Working() + a.Personal.Busy()
}

// startRoles 把后台跑起来：定时例行、个人任务、IM 网关、送达循环和编排轮次。编排在这里就打开，没有界面也会续上还在做的工单。
// IM 网关起来之后处理上一个领头进程收尾时记下的消息（im_handoff.go）。
func (a *App) startRoles() {
	a.startSchedule()
	a.Personal.Hold(false)
	a.startPersonal()
	if a.IM != nil {
		a.IM.Start()
	}
	a.startDeskRole()
	_ = a.orchestrator()
	go a.imReplay()
}

func (a *App) stopRoles() {
	a.stopDeskLoop()
	a.stopSchedule()
	a.stopPersonal()
	if a.IM != nil {
		a.IM.Close()
	}
	a.dropOrchestrator()
}

// stopLead 停掉选主的那一拍。租约等 Close 把其余的都关好之后再放（releaseLead）。
func (a *App) stopLead() {
	if a == nil {
		return
	}
	a.lead.mu.Lock()
	stop, done := a.lead.stop, a.lead.done
	a.lead.stop = nil
	a.lead.mu.Unlock()
	if stop == nil {
		return
	}
	close(stop)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}
}

func (a *App) releaseLead() {
	if a == nil {
		return
	}
	a.lead.mu.Lock()
	held, front := a.lead.held, a.lead.front
	a.lead.held, a.lead.front = nil, nil
	a.lead.leading = false
	a.lead.mu.Unlock()
	held.Release()
	front.Release()
}

func (a *App) startDeskRole() {
	stop := make(chan struct{})
	a.bookMu.Lock()
	a.deskStop = stop
	a.bookMu.Unlock()
	go a.deskLoop(stop)
}

func (a *App) startSchedule() {
	if a.Schedule == nil {
		return
	}
	stop := make(chan struct{})
	a.lead.mu.Lock()
	a.schedStop = stop
	a.lead.mu.Unlock()
	go a.scheduleLoopUntil(stop)
}

func (a *App) stopSchedule() {
	a.lead.mu.Lock()
	stop := a.schedStop
	a.schedStop = nil
	a.lead.mu.Unlock()
	if stop != nil {
		select {
		case <-stop:
		default:
			close(stop)
		}
	}
}

func (a *App) startPersonal() {
	eng := a.Personal
	a.lead.mu.Lock()
	base := a.lead.personal
	if base == nil || eng == nil {
		a.lead.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(base)
	a.lead.personalStop = cancel
	a.lead.mu.Unlock()
	go eng.Run(ctx)
}

func (a *App) stopPersonal() {
	a.lead.mu.Lock()
	cancel := a.lead.personalStop
	a.lead.personalStop = nil
	a.lead.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// dropOrchestrator 让出编排：先落盘停轮次，再关掉工人。和 stopOrchestrator 不同，之后再领头还能重开。
func (a *App) dropOrchestrator() {
	a.deskMu.Lock()
	rt := a.orch
	a.orch = nil
	a.deskMu.Unlock()
	if rt != nil {
		rt.Stop()
		rt.CloseWorkers()
	}
}

// followView 是跟随时的编排台：从数据库读领头进程写下的那一份，不跑轮次，也不落盘。两秒内复用上一次。
func (a *App) followView() (view superagent.View) {
	a.lead.mu.Lock()
	if !a.lead.viewAt.IsZero() && time.Since(a.lead.viewAt) < 2*time.Second {
		v := a.lead.view
		a.lead.mu.Unlock()
		return v
	}
	a.lead.mu.Unlock()
	if a.Home == nil {
		return superagent.View{Health: "unavailable"}
	}
	defer func() {
		if r := recover(); r != nil {
			view = superagent.View{Health: "unavailable"}
		}
	}()
	rt, err := superagent.Open(filepath.Join(a.Home.Root, "superagent.sqlite"), superagent.Policy{Mode: "observe", Version: 1})
	if err != nil || rt == nil {
		return superagent.View{Health: "unavailable"}
	}
	view = rt.View()
	rt.Discard()
	a.lead.mu.Lock()
	a.lead.view, a.lead.viewAt = view, time.Now()
	a.lead.mu.Unlock()
	return view
}

// LeadState 是这个进程在选主里的位置，以及谁拿着租约。不选主时为空。
func (a *App) LeadState() DeskLead {
	if a == nil || a.Home == nil {
		return DeskLead{}
	}
	a.lead.mu.Lock()
	role, leading := a.lead.role, a.lead.leading
	draining := leading && !a.lead.drainSince.IsZero()
	a.lead.mu.Unlock()
	if role == "" {
		return DeskLead{}
	}
	out := DeskLead{Role: "follow", Draining: draining}
	if leading {
		out.Role = "lead"
	}
	// 跟随方从持有人记录里看对方是不是在收尾；领头方以自己内存里的为准（记录写失败时也不会显示错）。
	if h, ok := lease.ReadHolder(a.leadPath()); ok {
		out.Holder, out.Since = h.Role, h.Since
		if !leading {
			out.Draining = h.Draining
		}
	}
	return out
}

// LeadSummary 是 yoyo serve / daemon 启动时打印的一行。
func (a *App) LeadSummary() string {
	s := a.LeadState()
	switch {
	case s.Role == "lead":
		return "proactive: this process sends briefs, runs routines and serves IM"
	case s.Holder == "desktop":
		return "proactive: the desktop app leads; this process takes over when it closes"
	case s.Holder != "":
		return "proactive: another yoyo process (" + s.Holder + ") leads"
	default:
		return "proactive: waiting for the lead"
	}
}

// syncConfig 在 config.yaml 被别的进程改过时接上主动相关的几项：简报时刻和主动程度、暂停、界面语言。
func (a *App) syncConfig() {
	if a.Home == nil {
		return
	}
	path := a.Home.Config()
	st := stampOf(path)
	a.lead.mu.Lock()
	same := st == a.lead.cfg
	a.lead.mu.Unlock()
	if same || st.mod.IsZero() {
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var c Config
	if yaml.Unmarshal(b, &c) != nil {
		return
	}
	a.lead.mu.Lock()
	a.lead.cfg = st
	a.lead.mu.Unlock()
	a.Config.Desk = c.Desk
	a.Config.Paused = c.Paused
	a.Config.Locale = c.Locale
	a.Touch("config", "")
}

// markConfig 记下自己刚写下的 config.yaml，免得下一拍当成别人改的。
func (a *App) markConfig() {
	if a == nil || a.Home == nil {
		return
	}
	st := stampOf(a.Home.Config())
	a.lead.mu.Lock()
	a.lead.cfg = st
	a.lead.mu.Unlock()
}

// tailForeign 把别的进程写进主会话（和界面正在看的会话）的事件补发给自己的界面。第一次看到一个会话时从文件尾开始，不重放历史。
func (a *App) tailForeign() {
	if a.Traces == nil || a.Hub == nil || a.Home == nil {
		return
	}
	ids := map[string]bool{}
	if id := a.readMainID(); id != "" {
		ids[id] = true
	}
	for _, id := range a.Hub.watched() {
		ids[id] = true
	}
	a.lead.mu.Lock()
	if a.lead.tail == nil {
		a.lead.tail = map[string]int64{}
	}
	cursors := make(map[string]int64, len(a.lead.tail))
	for id, at := range a.lead.tail {
		cursors[id] = at
	}
	a.lead.mu.Unlock()
	next := map[string]int64{}
	for id := range ids {
		at, seen := cursors[id]
		if !seen {
			next[id] = a.Traces.Size(id)
			continue
		}
		evs, pos, err := a.Traces.ForeignSince(id, at)
		if err != nil {
			next[id] = at
			continue
		}
		next[id] = pos
		for _, ev := range evs {
			a.Hub.Publish(ev)
		}
	}
	a.lead.mu.Lock()
	a.lead.tail = next
	a.lead.mu.Unlock()
}

// watched 是有人单独订阅的会话。通配订阅不算。
func (h *Hub) watched() []string {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []string
	for id, subs := range h.subs {
		if id == "" || id == "*" || len(subs) == 0 {
			continue
		}
		out = append(out, id)
	}
	return out
}

// seenFile 是领头进程最近一轮判断的落盘，给跟随的进程的界面读。
type seenFile struct {
	Briefing initiative.Briefing `json:"briefing"`
	At       time.Time           `json:"at"`
}

func (a *App) seenPath() string {
	if a == nil || a.Home == nil {
		return ""
	}
	return filepath.Join(a.Home.Memory(), "desk-seen.json")
}

// saveSeen 在选主的进程里把最近一轮判断写下来。不选主时不写。
func (a *App) saveSeen(b initiative.Briefing, at time.Time) {
	if !a.managed() {
		return
	}
	path := a.seenPath()
	if path == "" {
		return
	}
	raw, err := json.Marshal(seenFile{Briefing: b, At: at})
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	tmp := path + ".tmp"
	if os.WriteFile(tmp, raw, 0o600) == nil {
		_ = os.Rename(tmp, path)
	}
}

// seenBriefing 是界面读到的最近一轮判断。跟随时读领头进程写下的那一份。
func (a *App) seenBriefing() (initiative.Briefing, time.Time) {
	if a.managed() && !a.leading() {
		raw, err := os.ReadFile(a.seenPath())
		if err != nil {
			return initiative.Briefing{}, time.Time{}
		}
		var f seenFile
		if json.Unmarshal(raw, &f) != nil {
			return initiative.Briefing{}, time.Time{}
		}
		return f.Briefing, f.At
	}
	a.bookMu.Lock()
	defer a.bookMu.Unlock()
	return a.deskSeen, a.deskSeenAt
}
