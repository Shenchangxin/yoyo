// 会话元数据、发送回合，以及评测和演进入口。发送进入 runtime.Run。

package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/eval"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/observe"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// profilePinBytes 是写进提示的画像摘录字节上限，大约对应两百个已确认事实的 token。
const profilePinBytes = 800

// SessionMeta 是一份会话的持久元数据。Isolate 为真时表示使用工作区副本。
type SessionMeta struct {
	// ID 在创建时定下，之后不变。轨迹在 sessions/{id}.jsonl，spill 在 sessions/{id}/spill。DumpSession 按这个 id 返回两者全文。
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	// Channel 是会话车道，默认 agent，也可以是 video。视频会话和代理会话不共用标识。
	Channel         string `json:"channel,omitempty"`
	Workspace       string `json:"workspace"`
	OriginWorkspace string `json:"origin_workspace,omitempty"`
	Worktree        string `json:"worktree,omitempty"`
	// Isolate 为真时工具落在 Worktree 这份工作区副本上。这不是进程隔离。
	Isolate          bool      `json:"isolate,omitempty"`
	Harness          string    `json:"harness"`
	HarnessPolicy    string    `json:"harness_policy,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
	Preview          string    `json:"preview,omitempty"`
	ModelFingerprint string    `json:"model_fingerprint"`
	Title            string    `json:"title,omitempty"`
	Archived         bool      `json:"archived"`
	Pinned           bool      `json:"pinned"`
	Model            string    `json:"model,omitempty"`
	ConnectionID     string    `json:"connection_id,omitempty"`
	LoadedSkills     []string  `json:"loaded_skills,omitempty"`
	PinnedSkills     []string  `json:"pinned_skills,omitempty"`
	PlanText         string    `json:"plan_text,omitempty"`
	ParentID         string    `json:"parent_id,omitempty"`
	PinnedPaths      []string  `json:"pinned_paths,omitempty"`
	AuthMode         string    `json:"auth_mode,omitempty"`
	ProfileID        string    `json:"profile_id,omitempty"`
	PageID           string    `json:"page_id,omitempty"`
	// LibraryID 是这段对话绑定的文库。模型检索只能用这一份。
	LibraryID string `json:"library_id,omitempty"`
	// PathID 是这段对话绑定的掌握路径。
	PathID string `json:"path_id,omitempty"`
	// MaterialID 是这段对话绑定的阅读材料。
	MaterialID    string `json:"material_id,omitempty"`
	ResearchBrief string `json:"research_brief,omitempty"`
	ResearchShot  string `json:"research_shot,omitempty"`
	// Interrupted 和 Queued 是列表时填上的现场标注，不是持久 meta。writeSession 写出前会把它们清掉。
	Interrupted bool `json:"interrupted,omitempty"`
	Queued      int  `json:"queued,omitempty"`
}

// ProjectRoot 有 OriginWorkspace 时返回它，否则返回 Workspace。
func (m SessionMeta) ProjectRoot() string {
	if strings.TrimSpace(m.OriginWorkspace) != "" {
		return m.OriginWorkspace
	}
	return m.Workspace
}

// ToolRoot 在 Isolate 为真且 Worktree 非空时返回那份工作区副本，否则返回 Workspace。
func (m SessionMeta) ToolRoot() string {
	if m.Isolate && strings.TrimSpace(m.Worktree) != "" {
		return m.Worktree
	}
	return m.Workspace
}

// NewSession 在给定工作区建立一条代理车道的会话。
// workspace 传给 NewSessionOn。
func (a *App) NewSession(workspace string) (SessionMeta, error) {
	return a.NewSessionOn(workspace, session.ChannelAgent)
}

// NewSessionOn 建立会话并写入 meta。工作区为空时，视频车道用视频工作区，其余用 Workspace。新会话的授权是完整权限。
// workspace 为空时，视频车道用视频工作区，其余用 Workspace，然后写入会话的 Workspace。
// channel 经 NormalizeChannel 后写入 Channel，并用来决定空工作区的回落和标题。
func (a *App) NewSessionOn(workspace, channel string) (SessionMeta, error) {
	ch := session.NormalizeChannel(channel)
	if workspace == "" {
		if ch == session.ChannelVideo {
			workspace = a.VideoWorkspace()
		} else {
			workspace = a.Workspace()
		}
	}
	id := newID()
	title := "session"
	if ch == session.ChannelVideo {
		title = "video"
	}
	if ch == session.ChannelLearn {
		title = "learn"
	}
	meta := SessionMeta{
		ID:               id,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		Channel:          ch,
		Workspace:        workspace,
		Harness:          a.ActiveHash(),
		HarnessPolicy:    session.FollowActive,
		ModelFingerprint: Fingerprint(a.Config),
		Title:            title,
		// 新会话从完整权限开始。已经保存过的会话保持当时的模式。
		AuthMode:  capability.AuthFull,
		ProfileID: "assistant",
	}
	meta.Model, meta.ConnectionID = a.defaultChatSlot()
	b, _ := json.MarshalIndent(meta, "", "  ")
	path := filepath.Join(a.Home.Sessions(), id+".meta.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return meta, err
	}
	a.applySessionAuth(id, capability.AuthFull)
	return a.attachAuthMode(meta), nil
}

// SetSessionChannel 改写会话车道并保存。
// id 传给 GetSession。
// channel 经 NormalizeChannel 后写入 Channel。
func (a *App) SetSessionChannel(id, channel string) (SessionMeta, error) {
	m, err := a.GetSession(id)
	if err != nil {
		return m, err
	}
	m.Channel = session.NormalizeChannel(channel)
	if err := a.writeSession(m); err != nil {
		return m, err
	}
	return a.attachAuthMode(m), nil
}

// GetSession 读取 meta.json。文件缺失时返回只带 id 的元数据和错误。
func (a *App) GetSession(id string) (SessionMeta, error) {
	b, err := os.ReadFile(filepath.Join(a.Home.Sessions(), id+".meta.json"))
	if err != nil {
		return SessionMeta{ID: id}, err
	}
	var m SessionMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return m, err
	}
	return a.attachAuthMode(m), nil
}

// attachAuthMode 用运行状态里的授权模式覆盖返回值；没有则沿用 meta 里的模式。
func (a *App) attachAuthMode(m SessionMeta) SessionMeta {
	mode := capability.AuthDefault
	if a != nil && a.Threads != nil && m.ID != "" {
		if st := a.Threads.Load(m.ID); strings.TrimSpace(st.AuthMode) != "" {
			mode = capability.ParseAuthMode(st.AuthMode)
		} else if strings.TrimSpace(m.AuthMode) != "" {
			mode = capability.ParseAuthMode(m.AuthMode)
		}
	} else if strings.TrimSpace(m.AuthMode) != "" {
		mode = capability.ParseAuthMode(m.AuthMode)
	}
	m.AuthMode = mode
	return m
}

// applySessionAuth 把解析后的授权模式写进线程状态和能力代理。
// id 传给 SetAuthMode 和 ApplyAuthMode。
// mode 经 ParseAuthMode 后传给 SetAuthMode 和 ApplyAuthMode。
func (a *App) applySessionAuth(id, mode string) {
	mode = capability.ParseAuthMode(mode)
	caps := capability.AuthModeCapStrings(mode)
	if a.Threads != nil {
		a.Threads.SetAuthMode(id, mode, caps)
	}
	if a.Caps != nil {
		a.Caps.ApplyAuthMode(id, mode)
	}
}

// writeSession 把 meta 写成 json。写出前清掉 Interrupted 和 Queued，因为它们不是持久字段。
func (a *App) writeSession(m SessionMeta) error {
	m.Interrupted = false
	m.Queued = 0
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(a.Home.Sessions(), m.ID+".meta.json"), b, 0o644); err != nil {
		return err
	}
	a.Touch("session", m.ID)
	return nil
}

// titleFrom 用消息的前一段做成标题。空消息得到 session。
// message 先把换行换成空格并 TrimSpace，空串返回 session，长于 42 个 rune 时截断。
func titleFrom(message string) string {
	s := strings.TrimSpace(strings.ReplaceAll(message, "\n", " "))
	if s == "" {
		return "session"
	}
	r := []rune(s)
	if len(r) > 42 {
		return string(r[:42]) + "…"
	}
	return s
}

// touchSessionMeta 更新时间，有预览时写入，然后保存。
// m 更新 UpdatedAt，预览非空时写入 Preview，然后保存。
// preview 去掉首尾空白后非空时经 titleFrom 写入 Preview。
func (a *App) touchSessionMeta(m SessionMeta, preview string) SessionMeta {
	m.UpdatedAt = time.Now().UTC()
	if s := strings.TrimSpace(preview); s != "" {
		m.Preview = titleFrom(s)
	}
	_ = a.writeSession(m)
	return m
}

// Trajectory 返回给界面回放的轨迹，不附带每条写入的全文。
// id 传给 ReadTrace。
func (a *App) Trajectory(id string) ([]trace.Event, error) {
	evs, err := a.ReadTrace(id)
	if err != nil {
		return nil, err
	}
	// 给界面回放时不把每条 write_file 正文整包送出。DumpSession、压缩和上下文测量仍读存储里的原文。
	return runtime.UITrajectory(evs), nil
}

// TrajectoryPage 先取 Hub 头再按回合分页，避免事件落在两次读取的缝里。
// id 传给 PageTurns。
// beforeSeq 传给 PageTurns。
// turns 传给 PageTurns。
func (a *App) TrajectoryPage(id string, beforeSeq int64, turns int) (runtime.TrajectoryPage, error) {
	if a.Traces == nil {
		return runtime.TrajectoryPage{}, fmt.Errorf("no trace store")
	}
	// 先取 Hub 的头再读文件：事件先落盘再发布，所以序号不大于 hubSeq 的事件在读文件时已经在盘上。
	// 反过来会让事件落在两次读取之间，既不在这一页，也不在从 hubSeq 开始的补拉里。
	hubSeq := int64(0)
	if a.Hub != nil {
		hubSeq = a.Hub.Head()
	}
	raw, err := a.Traces.PageTurns(id, beforeSeq, turns)
	if err != nil {
		return runtime.TrajectoryPage{}, err
	}
	return runtime.ProjectPage(raw, hubSeq, trace.DefaultPageBytes), nil
}

// TrajectoryOutline 返回该会话的回合大纲。没有轨迹存储则返回错误。
// id 传给 OutlineTurns。
func (a *App) TrajectoryOutline(id string) ([]trace.OutlineTurn, error) {
	if a.Traces == nil {
		return []trace.OutlineTurn{}, fmt.Errorf("no trace store")
	}
	return a.Traces.OutlineTurns(id)
}

// TrajectoryAround 先取 Hub 头，再取某个用户序号附近的回合页。
// id 传给 PageAroundUser。
// userSeq 传给 PageAroundUser 和 ProjectPageKeep。
// turns 传给 PageAroundUser。
func (a *App) TrajectoryAround(id string, userSeq int64, turns int) (runtime.TrajectoryPage, error) {
	if a.Traces == nil {
		return runtime.TrajectoryPage{}, fmt.Errorf("no trace store")
	}
	// 先取 Hub 的头再读文件，原因和 TrajectoryPage 相同。
	hubSeq := int64(0)
	if a.Hub != nil {
		hubSeq = a.Hub.Head()
	}
	raw, err := a.Traces.PageAroundUser(id, userSeq, turns)
	if err != nil {
		return runtime.TrajectoryPage{}, err
	}
	return runtime.ProjectPageKeep(raw, hubSeq, trace.DefaultPageBytes, userSeq), nil
}

// LiveSince 从 Hub 取出序号大于 after 的实时事件，并收成界面用的瘦轨迹。
// sessionID 传给 Hub.Since。
func (a *App) LiveSince(sessionID string, after int64) []trace.Event {
	if a.Hub == nil {
		return nil
	}
	return runtime.SlimTrajectory(a.Hub.Since(after, sessionID))
}

// LiveSync 和 LiveSince 一样取实时事件，但保留 Gap 和 More，让调用方区分空洞和还有下一页。
// sessionID 传给 Hub.Batch。
// after 传给 Hub.Batch。
func (a *App) LiveSync(sessionID string, after int64) LiveBatch {
	if a.Hub == nil {
		return LiveBatch{Events: []trace.Event{}}
	}
	b := a.Hub.Batch(after, sessionID)
	b.Events = runtime.SlimTrajectory(b.Events)
	return b
}

// ReadTrace 读取该会话的全部轨迹事件。没有存储则返回错误。
// id 传给 Traces.Read。
func (a *App) ReadTrace(id string) ([]trace.Event, error) {
	if a.Traces == nil {
		return nil, fmt.Errorf("no trace store")
	}
	return a.Traces.Read(id)
}

// Client 按空的会话元数据选择聊天客户端。
func (a *App) Client() (runtime.Client, error) {
	return a.ClientFor(SessionMeta{})
}

// ClientFor 按会话的路由做故障转移客户端。没有可用密钥则返回错误。
// meta 传给 chatRoutes。
func (a *App) ClientFor(meta SessionMeta) (runtime.Client, error) {
	if routes := a.chatRoutes(meta); len(routes) > 0 {
		if cl := runtime.NewFailoverClient(routes); cl != nil {
			return cl, nil
		}
	}
	if cl := a.envChatFallback(); cl != nil {
		return cl, nil
	}
	return nil, fmt.Errorf("missing API key — paste the token in Settings")
}

// Send 发送一轮，不进入计划模式，并进入 runtime.Run。
func (a *App) Send(ctx context.Context, sessionID, message string, client runtime.Client, onEvent func(trace.Event)) (string, error) {
	return a.SendOpts(ctx, sessionID, message, client, onEvent, false)
}

// Running 在槽存在且 done 仍为假时返回真。
// sessionID 用作 runs 的键。
func (a *App) Running(sessionID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	slot := a.runs[sessionID]
	return slot != nil && !slot.done
}

// RunningIDs 返回尚未 done 的会话 id，按字典序排列。
func (a *App) RunningIDs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := make([]string, 0, len(a.runs))
	for id, slot := range a.runs {
		if slot == nil || slot.done {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// markRunDone 把槽的 done 设为真。槽还留着，所以下一轮发送仍会入队。
// sessionID 用作 runs 的键。
func (a *App) markRunDone(sessionID string) {
	a.mu.Lock()
	if slot := a.runs[sessionID]; slot != nil {
		slot.done = true
	}
	a.mu.Unlock()
}

// RunStatus 是一条仍在跑的会话：id、开始时间和最后一次工具名。
type RunStatus struct {
	ID        string    `json:"id"`
	StartedAt time.Time `json:"started_at"`
	LastTool  string    `json:"last_tool,omitempty"`
}

// RunningStatus 列出尚未 done 的槽。
func (a *App) RunningStatus() []RunStatus {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]RunStatus, 0, len(a.runs))
	for id, slot := range a.runs {
		if slot == nil || slot.done {
			continue
		}
		out = append(out, RunStatus{ID: id, StartedAt: slot.started, LastTool: slot.lastTool})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// sessionPlanText 优先用 meta 里的计划文本，否则从工具根读取计划文件。
// toolRoot 传给 LoadPlanFile。
func sessionPlanText(meta SessionMeta, toolRoot string) string {
	if s := strings.TrimSpace(meta.PlanText); s != "" {
		return s
	}
	return runtime.LoadPlanFile(toolRoot, meta.ID)
}

// noteChildRun 在子运行开始时记下槽并写子会话 meta，结束时删掉槽。子会话继承父会话的 Isolate 副本标记，harness 策略钉住。
// parentID 写入子会话的 ParentID。
// childID 先 TrimSpace，空串则返回；done 为真时从 runs 删掉，否则用作 runs 的键并写入子会话 ID。
// prompt 经 titleFrom 写入标题。
// parent 用来抄 Channel、Workspace、OriginWorkspace、Worktree、Isolate、Harness、ModelFingerprint、Model、ConnectionID 和 AuthMode。
// done 为真时只删掉该子会话的运行槽。
func (a *App) noteChildRun(parentID, childID, prompt string, parent SessionMeta, done bool) {
	childID = strings.TrimSpace(childID)
	if childID == "" {
		return
	}
	if done {
		a.mu.Lock()
		delete(a.runs, childID)
		a.mu.Unlock()
		return
	}
	a.mu.Lock()
	if a.runs[childID] == nil {
		a.runs[childID] = &runSlot{started: time.Now().UTC(), lastTool: "task"}
	}
	a.mu.Unlock()
	title := "task · " + titleFrom(prompt)
	child := SessionMeta{
		ID:               childID,
		ParentID:         parentID,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		Channel:          parent.Channel,
		Workspace:        parent.Workspace,
		OriginWorkspace:  parent.OriginWorkspace,
		Worktree:         parent.Worktree,
		Isolate:          parent.Isolate,
		Harness:          parent.Harness,
		HarnessPolicy:    session.Pin,
		ModelFingerprint: parent.ModelFingerprint,
		Title:            title,
		Model:            parent.Model,
		ConnectionID:     parent.ConnectionID,
		AuthMode:         parent.AuthMode,
	}
	_ = a.writeSession(child)
}

// noteRunTool 把运行槽的最后工具名设成这一次调用。
// sessionID 用作 runs 的键。
// name 先 TrimSpace，空串则返回，否则写入槽的 lastTool。
func (a *App) noteRunTool(sessionID, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	a.mu.Lock()
	if slot := a.runs[sessionID]; slot != nil {
		slot.lastTool = name
	}
	a.mu.Unlock()
}

// acquireRun 为会话占一个带取消函数的槽。已经有槽则返回正忙。释放时取消、删槽并清掉待审批。
// sessionID 用作 runs 的键，释放时删掉并传给 SetApprovals。
// ctx 传给 context.WithCancel。
func (a *App) acquireRun(sessionID string, ctx context.Context) (context.Context, func(), error) {
	ctx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	if a.runs[sessionID] != nil {
		a.mu.Unlock()
		cancel()
		return nil, nil, errBusy
	}
	a.runs[sessionID] = &runSlot{cancel: cancel, started: time.Now().UTC()}
	a.mu.Unlock()
	return ctx, func() {
		cancel()
		a.mu.Lock()
		delete(a.runs, sessionID)
		a.mu.Unlock()
		if a.Threads != nil {
			a.Threads.SetApprovals(sessionID, nil)
		}
	}, nil
}

// StartSend 不带附件地开始一轮发送。
func (a *App) StartSend(sessionID, message string, plan bool) error {
	return a.StartSendOpts(sessionID, message, plan, nil)
}

// StartSendOpts 在暂停时直接失败。正在跑且是续跑则返回正忙；正在跑的新消息入队并返回 ErrQueued；否则启动发送。
func (a *App) StartSendOpts(sessionID, message string, plan bool, atts []Attachment) error {
	if a.IsPaused() {
		return fmt.Errorf("paused")
	}
	atts, resume := takeResume(message, atts)
	atts, requestID := takeRequestID(atts)
	if resume {
		message = ""
	}
	if !resume {
		var err error
		message, atts, err = a.prepareAttachments(sessionID, message, atts)
		if err != nil {
			return err
		}
	}
	atts = WithRequestID(atts, requestID)
	if resume && a.Running(sessionID) {
		return errBusy
	}
	if a.Running(sessionID) {
		a.Enqueue(sessionID, QueuedTurn{Text: message, Plan: plan, Attachments: atts})
		return ErrQueued
	}
	return a.launchSend(sessionID, message, plan, atts, resume)
}

// RetrySession 按已有历史继续当前回合，不追加新的用户消息。会话正在跑时返回正忙，不入队。
// sessionID 传给 Running 和 launchSend。
func (a *App) RetrySession(sessionID string) error {
	if a.Running(sessionID) {
		return errBusy
	}
	return a.launchSend(sessionID, "", false, nil, true)
}

// requestIDName 是附件里携带乐观发送 id 的名字，不会当成真实附件。
const requestIDName = "__request_id__"

// WithRequestID 把合成器的乐观发送 id 记在附件上，这样开始发送不用另改签名。空 id 则原样返回。
// atts 先经 takeRequestID 去掉已有的乐观发送 id，id 非空时再附上一条。
func WithRequestID(atts []Attachment, id string) []Attachment {
	id = strings.TrimSpace(id)
	if id == "" {
		return atts
	}
	kept, _ := takeRequestID(atts)
	return append(kept, Attachment{Name: requestIDName, Path: id})
}

// requestIDOf 返回附件里带着的乐观发送 id。
// atts 传给 takeRequestID。
func requestIDOf(atts []Attachment) string {
	_, id := takeRequestID(atts)
	return id
}

// takeRequestID 从附件里拆出乐观发送 id，其余附件照旧返回。
// atts 长度为 0 时原样返回，否则拆出名字是 requestIDName 的项。
func takeRequestID(atts []Attachment) ([]Attachment, string) {
	if len(atts) == 0 {
		return atts, ""
	}
	kept := make([]Attachment, 0, len(atts))
	id := ""
	for _, a := range atts {
		if a.Name == requestIDName {
			id = strings.TrimSpace(a.Path)
			continue
		}
		kept = append(kept, a)
	}
	return kept, id
}

// takeResume 在消息为空且附件里有续跑标记时视为续跑，并拿掉该标记。消息非空则不是续跑。
// message 去掉首尾空白后非空则不是续跑，并原样返回 atts。
// atts 在消息去掉空白后为空时，拿掉名字或路径为 __resume__ 的项。
func takeResume(message string, atts []Attachment) ([]Attachment, bool) {
	if strings.TrimSpace(message) != "" {
		return atts, false
	}
	kept := make([]Attachment, 0, len(atts))
	resume := false
	for _, a := range atts {
		if a.Name == "__resume__" || a.Path == "__resume__" {
			resume = true
			continue
		}
		kept = append(kept, a)
	}
	return kept, resume
}

// chatTurnContext 给异步的桌面或命令行回合一个可取消的上下文，不设墙钟截止。
func chatTurnContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// launchSend 占住运行槽后异步调用 sendLocked。不是续跑时先 commitUser，把用户行写入 JSONL。回合结束后 kickQueue 取下一条。
func (a *App) launchSend(sessionID, message string, plan bool, atts []Attachment, resume bool) error {
	ctx, cancel := chatTurnContext()
	ctx, release, err := a.acquireRun(sessionID, ctx)
	if err != nil {
		cancel()
		if err == errBusy {
			if resume {
				return errBusy
			}
			a.Enqueue(sessionID, QueuedTurn{Text: message, Plan: plan, Attachments: atts})
			return ErrQueued
		}
		return err
	}
	turnID := diaglog.NewTurnID()
	committed := false
	if !resume {
		if err := a.commitUser(sessionID, message, atts, turnID); err != nil {
			cancel()
			release()
			return err
		}
		committed = userCommitNeeded(message, atts)
	}
	if a.Threads != nil {
		a.Threads.MarkTurnInFlight(sessionID, true, plan)
	}
	go func() {
		defer cancel()
		defer release()
		defer func() {
			if rec := recover(); rec != nil {
				path := diaglog.WriteCrash(fmt.Sprintf("loop panic: %v", rec), rec)
				payload := map[string]any{"error": fmt.Sprintf("loop panic: %v", rec)}
				if path != "" {
					payload["crash_path"] = path
				}
				a.Hub.Publish(trace.Event{
					Type: trace.TypeError, Source: "runtime", SessionID: sessionID,
					Payload: payload,
				})
			}
		}()
		defer a.kickQueue(sessionID)
		_, _ = a.sendLocked(ctx, sessionID, message, nil, nil, plan, atts, resume, committed, turnID)
	}()
	return nil
}

// SendOpts 占住运行槽后同步调用 sendLocked。它不预先写用户行，也不是续跑。
func (a *App) SendOpts(ctx context.Context, sessionID, message string, client runtime.Client, onEvent func(trace.Event), plan bool) (string, error) {
	ctx, release, err := a.acquireRun(sessionID, ctx)
	if err != nil {
		return "", err
	}
	defer release()
	return a.sendLocked(ctx, sessionID, message, client, onEvent, plan, nil, false, false, "")
}

// sendLocked 在运行槽已经占用后执行这一轮，并调用 runtime.Run。模型调用发生在 Run 里。
// 进入 Run 之前会解析 harness：按会话的 HarnessPolicy、钉住的哈希和当前 ActiveHash 调用 session.ResolveHarness，再装入快照和材料。
// 然后对循环预设调用 runtime.ApplyChatHorizon。历史用 ReadFromCheckpoint 取出；超预算时 runtime.MaybeCheckpoint 做检查点。
// Hub 非空时包装 onEvent，事件先 Publish 进 Hub，再交给原来的回调。
// 非 resume 且用户行尚未落盘时，commitUser 先把用户事件写入 JSONL。交给 Run 的 UserCommitted 仅在该用户行已经写入且不是 resume 时为真，避免再发第二条用户事件。resume 不写新用户行，UserCommitted 为假。
// Isolate 为真时先确保 Worktree 这份工作区副本，工具用 ToolRoot。PullSteer 指向 pullSteer：Steer 只在会话仍在跑时追加文本，Run 在下一次模型调用前把它一次收走，不取消回合。StopTurn 才取消运行槽的 context。
func (a *App) sendLocked(ctx context.Context, sessionID, message string, client runtime.Client, onEvent func(trace.Event), plan bool, atts []Attachment, resume, alreadyCommitted bool, turnID string) (out string, runErr error) {
	if turnID == "" {
		turnID = diaglog.NewTurnID()
	}
	ctx = diaglog.With(ctx, diaglog.Fields{SessionID: sessionID, TurnID: turnID, Component: "runtime"})
	var turnSpan observe.Span
	if a.Observe != nil {
		turnSpan = a.Observe.Start("turn", sessionID, map[string]any{"plan": plan, "resume": resume, "turn_id": turnID})
		turnSpan.TurnID = turnID
		ctx = observe.ContextWithSpan(ctx, turnSpan)
		ctx = diaglog.With(ctx, diaglog.Fields{TraceID: turnSpan.TraceID, SpanID: turnSpan.SpanID})
		defer func() { a.Observe.End(turnSpan, runErr) }()
	}
	diaglog.InfoContext(ctx, "turn start", "component", "runtime", "plan", plan, "resume", resume)
	defer func() {
		if runErr != nil {
			diaglog.ErrorContext(ctx, "turn end", "component", "runtime", "err", runErr)
		} else {
			diaglog.InfoContext(ctx, "turn end", "component", "runtime")
		}
	}()
	if strings.TrimSpace(sessionID) == "" {
		return "", fmt.Errorf("empty session id")
	}
	meta, err := a.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("unknown session %s: %w", sessionID, err)
	}
	if meta.HarnessPolicy == "" {
		meta.HarnessPolicy = session.FollowActive
	}
	hash := session.ResolveHarness(meta.HarnessPolicy, meta.Harness, a.ActiveHash())
	if hash == "" {
		return "", fmt.Errorf("session %s has no harness", sessionID)
	}
	snap, err := a.LoadSnapshot(hash)
	if err != nil {
		return "", err
	}
	loop, frags, pb, skills, _, pol, err := a.Materials(hash)
	if err != nil {
		return "", err
	}
	if plan {
		loop.PlanMode = true
		pol.Mode = "plan"
	}
	// 桌面回合在进 Run 前套聊天视界。RulesTokens 会被赋成 1600，不论快照里原来是更大还是更小。TaskMaxTurns 低于 40 会抬到 40。
	loop = runtime.ApplyChatHorizon(loop)
	if a.Config.MaxBudgetUSD > 0 {
		// 配置里的美元上限只有在更小，或循环自己还没设上限时才写进去。配置是 0 表示不在这里收紧。
		if loop.MaxBudgetUSD <= 0 || a.Config.MaxBudgetUSD < loop.MaxBudgetUSD {
			loop.MaxBudgetUSD = a.Config.MaxBudgetUSD
		}
	}
	meter := &runtime.Meter{USDPerMTok: a.Config.USDPerMTok}
	if client == nil {
		client, err = a.ClientFor(meta)
		if err != nil {
			return "", err
		}
	}
	skillRoot := meta.ProjectRoot()
	toolRoot := meta.ToolRoot()
	// 会话隔离是工作区副本。这和 YOYO_ISOLATE 的进程拆分不是一回事。副本没建成就仍用原来的工具根。
	if meta.Isolate {
		ensured, e2 := a.EnsureSessionWorktree(sessionID)
		if e2 == nil {
			meta = ensured
			skillRoot = meta.ProjectRoot()
			toolRoot = meta.ToolRoot()
		}
	}
	disk := runtime.LoadSkillDirs(a.skillRoots(skillRoot)...)
	skills = runtime.MergeSkills(skills, disk)
	skills = runtime.FilterSkills(skills, loop.PlanMode || plan)
	skillBodies, skillDirs, skillMeta := runtime.MapSkills(skills)
	model := a.Config.Model
	if meta.Model != "" {
		model = meta.Model
	}
	preHooks, stopHooks, preCompact := runtime.LoadHookFile(toolRoot)
	if a.MCP != nil && strings.TrimSpace(toolRoot) != "" {
		a.MCP.SetRoots([]string{toolRoot})
	}
	tools := &runtime.WorkspaceTools{
		Workspace:          toolRoot,
		Home:               a.Home.Root,
		SessionID:          sessionID,
		Caps:               a.Caps,
		Skills:             skillBodies,
		SkillDirs:          skillDirs,
		SkillMeta:          skillMeta,
		Loaded:             uniqueStrings(meta.PinnedSkills, meta.LoadedSkills),
		Policy:             pol,
		PlanMode:           loop.PlanMode,
		Ctx:                ctx,
		Extra:              a.extraTools(sessionID),
		Spill:              runtime.BindSpill(a.Home.Root, toolRoot, sessionID),
		PlanText:           sessionPlanText(meta, toolRoot),
		Advertised:         append([]string(nil), snap.Tools...),
		AskUser:            a.askUserFn(ctx, sessionID),
		Packs:              a.packSessions(skillRoot),
		SkillAutoInvoke:    runtime.NormalizeAutoInvoke(a.Config.Skills.AutoInvoke),
		SkillCatalogBudget: a.Config.Skills.CatalogBudgetChars,
		CatalogLane:        session.CatalogLane(meta.Channel),
		Pages:              a.Pages,
		OnCapture: func(kind, path, summary string) {
			a.rememberCapture(sessionID, kind, path, summary)
		},
	}
	a.attachPersonal(tools)
	a.attachDramaTools(tools, sessionID)
	a.attachCanvasTools(tools, sessionID)
	a.attachCutTools(tools, sessionID)
	a.attachStudioTools(tools, sessionID)
	a.attachConductorTools(tools, sessionID)
	tools.Sessions = func(id string) []runtime.Message {
		evs, err := a.Traces.ReadFromCheckpoint(id)
		if err != nil {
			return nil
		}
		return runtime.MessagesFromEventsOpts(evs, runtime.BindSpill(a.Home.Root, toolRoot, id))
	}
	tools.ListThreads = func() []runtime.ThreadRef {
		list, err := a.ListSessions()
		if err != nil {
			return nil
		}
		out := make([]runtime.ThreadRef, 0, len(list))
		for _, m := range list {
			if m.Archived {
				continue
			}
			out = append(out, runtime.ThreadRef{ID: m.ID, Title: m.Title})
		}
		return out
	}
	tools.SendThread = func(id, text string) error {
		text = strings.TrimSpace(text)
		if text == "" {
			return fmt.Errorf("empty message")
		}
		if a.Running(id) {
			return a.Steer(id, text)
		}
		a.Enqueue(id, QueuedTurn{Text: text})
		return nil
	}
	userCommitted := alreadyCommitted
	// 先把用户行写入轨迹，再告诉 Run 不要重写。resume 和已经提交过的回合跳过。
	if !resume && !alreadyCommitted {
		if err := a.commitUser(sessionID, message, atts, turnID); err != nil {
			return "", err
		}
		userCommitted = userCommitNeeded(message, atts)
	}
	hist := []runtime.Message{}
	roundSeq := 0
	// 历史只取最后一次检查点之后的事件。更早的原文留在 JSONL，不进这一轮的模型消息。
	if evs, err := a.Traces.ReadFromCheckpoint(sessionID); err == nil {
		hist = runtime.MessagesFromEventsOpts(evs, tools.Spill)
		roundSeq = runtime.MaxAssistantRound(evs)
	}
	if userCommitted {
		// 刚写入的用户行已经在轨迹里。从历史末尾拿掉同一句，避免模型看见两条。
		drop := runtime.NormalizeUserText(message)
		if drop == "" && len(atts) > 0 {
			drop = "(image attached)"
		}
		hist = runtime.DropTrailingUser(hist, drop)
	}
	window := runtime.EffectiveModelWindow(model, a.Config.ContextWindow)
	hist = runtime.MaybeCheckpoint(a.Traces, sessionID, hist, loop, tools.Spill, client, model, window, frags)
	wrapped := onEvent
	if a.Hub != nil {
		prev := wrapped
		wrapped = func(ev trace.Event) {
			sid := ev.SessionID
			if sid == "" {
				sid = sessionID
			}
			if ev.Type == trace.TypeToolCall {
				name, _ := ev.Payload["name"].(string)
				a.noteRunTool(sid, name)
			}
			// 界面订阅的是 Hub。轨迹另由 runtime 写入 JSONL。这里先发布，再交给原来的 onEvent。
			a.Hub.Publish(ev)
			if prev != nil {
				prev(ev)
			}
		}
	}

	inject := ""
	voice := runtime.OperatorVoice(message, hist)
	if resume {
		message = ""
		atts = nil
		voice = runtime.OperatorVoice(message, hist)
	} else {
		if meta.Title == "" || meta.Title == "session" {
			meta.Title = titleFrom(message)
		}
		meta = a.touchSessionMeta(meta, message)
		note := hash
		if snap.Note != "" {
			note = hash + " " + snap.Note
		}
		var refs []runtime.Mention
		// 这份注入按字节累计，上限 3200。普通文件正文另按 800 个 rune 截断。主文件不占这 3200，只注入提纲。
		inject, refs = runtime.ExpandMentionsPacks(toolRoot, message, note, skillBodies, skillDirs, 3200)
		// 会话提及再占最多 1600 字节，接在前面的注入后面。
		if extra := runtime.FormatSessionMentions(refs, sessionID, tools, 1600); extra != "" {
			if inject != "" {
				inject += "\n"
			}
			inject += extra
		}
		mentioned := runtime.MentionSkillNames(refs)
		tools.MarkSkillsLoaded(mentioned...)
		// 技能自动注入用剩下的字节。已经用掉超过 2800 时，仍给技能留 400，避免这一段被提及挤没。
		remain := 3200 - len(inject)
		if remain < 400 {
			remain = 400
		}
		lane := ""
		autoInject, hints, auto := runtime.ApplySkillMatch(runtime.SkillMatchReq{
			Voice:     voice,
			Skills:    skills,
			Packs:     tools.Packs,
			Loaded:    tools.Loaded,
			Mentioned: mentioned,
			Depth:     tools.Depth,
			Mode:      tools.SkillAutoInvoke,
			Lane:      lane,
		}, skillBodies, skillDirs, remain)
		tools.SkillHints = hints
		tools.Autoloaded = auto
		if autoInject != "" {
			if inject != "" {
				inject += "\n"
			}
			inject += autoInject
			tools.MarkSkillsLoaded(auto...)
		}
		// 附件说明最多 2400 字节，接在提及和技能注入后面。
		if extra := runtime.ExpandAttachments(toolRoot, toRuntimeAtts(atts), 2400); extra != "" {
			if inject != "" {
				inject += "\n"
			}
			inject += extra
		}
	}
	userParts := runtime.ImageParts(toolRoot, toRuntimeAtts(atts))
	profile := ""
	if a.Memory != nil {
		if p := a.sessionProfile(sessionID); p == nil || p.Memory {
			profile = a.Memory.ProfilePin(profilePinBytes)
		}
	}
	tools.OperatorVoice = voice
	frags = append(append([]artifact.PromptFragment(nil), frags...), runtime.ChatConduct(runtime.ConductOpts{Plan: loop.PlanMode, Methodology: runtime.HasMethodologyPack(tools.Packs)})...)
	if pin := runtime.LanguagePin(voice); pin.Text != "" {
		frags = append(frags, pin)
	}
	if a.studioLane(sessionID) {
		loop, frags = applyStudioLanePrompt(loop, frags)
	} else if a.conductorLane(sessionID) {
		loop, frags = applyConductorLanePrompt(loop, frags)
	}
	if p := a.sessionProfile(sessionID); p != nil && strings.TrimSpace(p.Instructions) != "" {
		frags = append(frags, artifact.PromptFragment{Slot: "role", Text: p.Instructions})
	}
	pageCtx := ""
	if meta.PageID != "" && a.Pages != nil {
		if pg, err := a.Pages.Get("", meta.PageID); err == nil {
			pageCtx = runtime.PageInject(pg.Title, pg.Content, 3200)
		}
	}
	if tools.ChatOverlay && tools.Depth == 0 && a.Learn != nil && session.NormalizeChannel(meta.Channel) == session.ChannelLearn {
		if journal, err := a.Learn.JournalInject(); err == nil && journal != "" {
			if pageCtx != "" {
				pageCtx += "\n\n"
			}
			pageCtx += journal
		}
	}
	out, runErr = runtime.Run(ctx, runtime.RunRequest{
		SessionID:        sessionID,
		TurnID:           turnID,
		TraceID:          turnSpan.TraceID,
		Observe:          a.Observe,
		User:             message,
		UserCommitted:    userCommitted && !resume,
		RequestID:        requestIDOf(atts),
		UserParts:        userParts,
		Inject:           inject,
		Workspace:        toolRoot,
		Home:             a.Home.Root,
		Harness:          snap,
		HarnessHash:      hash,
		ModelFingerprint: Fingerprint(a.Config),
		Model:            model,
		ModelWindow:      window,
		MaxOutputTokens:  maxOutputTokensFor(model),
		Loop:             loop,
		Policy:           pol,
		Fragments:        frags,
		Playbook:         pb,
		Skills:           skills,
		History:          hist,
		RoundSeq:         roundSeq,
		Tools:            tools,
		Client:           client,
		Trace:            a.Traces,
		Events:           a.Kernel.Events(),
		FileHooks:        preHooks,
		StopHooks:        stopHooks,
		PreCompactHooks:  preCompact,
		ProfileMemory:    profile,
		PageContext:      pageCtx,
		OnEvent:          wrapped,
		OnChild: func(id, prompt string, done bool) {
			a.noteChildRun(sessionID, id, prompt, meta, done)
		},
		OnShape: func(r runtime.ShapeReport) { a.RememberShape(sessionID, r) },
		Meter:   meter,
		// 转向文本在下一次模型调用之前被拉走。流式生成中途不插入。
		PullSteer: func() string { return a.pullSteer(sessionID) },
		// 桌面这条路径打开软地平线。停轮看 192 和停滞窗口，for 不会因为 MaxTurns 用尽而退出。
		SoftHorizon:  true,
		ShowThinking: a.Config.ShowThinking,
	})
	a.markRunDone(sessionID)
	if a.Threads != nil {
		a.Threads.SetApprovals(sessionID, nil)
	}
	loaded, planText := tools.Pins()
	meta.LoadedSkills = loaded
	meta.PlanText = planText
	if strings.TrimSpace(planText) != "" {
		runtime.WritePlanFile(toolRoot, sessionID, planText)
	}
	_ = a.writeSession(meta)
	a.RememberUsage(meter.Snapshot())
	_, _ = a.StageACE(sessionID)
	if a.Threads != nil {
		var blob strings.Builder
		blob.WriteString(meta.Title)
		blob.WriteByte(' ')
		blob.WriteString(message)
		a.Threads.IndexBlob(sessionID, blob.String())
		a.Threads.MarkTurnInFlight(sessionID, false, false)
	}
	return out, runErr
}

// userCommitNeeded 在用户原文或附件非空时为真，这时才需要先落一条用户事件。
// message 经 NormalizeUserText 后非空则为真。
// atts 长度大于 0 则为真。
func userCommitNeeded(message string, atts []Attachment) bool {
	return runtime.NormalizeUserText(message) != "" || len(atts) > 0
}

// commitUser 把本轮用户原文或附件写成一条持久用户事件，追加进 JSONL，并 Publish 到 Hub。原文和附件都空则什么都不写。
// sessionID 传给 GetSession，并写入事件的 SessionID 和载荷 id。
// message 经 NormalizeUserText 得到正文。
// atts 传给 ImageParts 和 attachmentMeta；规范化正文和附件都空则返回。
// turnID 非空时写入 payload 的 turn_id，并写入事件的 TurnID 和载荷 id。
func (a *App) commitUser(sessionID, message string, atts []Attachment, turnID string) error {
	if a.Traces == nil {
		return fmt.Errorf("no trace store")
	}
	text := runtime.NormalizeUserText(message)
	meta, err := a.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("unknown session %s: %w", sessionID, err)
	}
	parts := runtime.ImageParts(meta.ToolRoot(), toRuntimeAtts(atts))
	if text == "" && len(parts) == 0 && len(atts) == 0 {
		return nil
	}
	if text == "" {
		if len(parts) > 0 {
			text = "(image attached)"
		} else {
			text = runtime.StampFileMentions("", toRuntimeAtts(atts))
			if text == "" {
				text = "(file attached)"
			}
		}
	}
	payload := map[string]any{"text": text, "id": fmt.Sprintf("%s:user:%s", sessionID, turnID)}
	if len(parts) > 0 {
		payload["parts"] = parts
	}
	atts, requestID := takeRequestID(atts)
	if requestID != "" {
		payload["request_id"] = requestID
	}
	if metas := attachmentMeta(atts); len(metas) > 0 {
		payload["attachments"] = metas
	}
	if turnID != "" {
		payload["turn_id"] = turnID
	}
	hash := session.ResolveHarness(meta.HarnessPolicy, meta.Harness, a.ActiveHash())
	ev := trace.Event{
		Type:             trace.TypeUser,
		Source:           "user",
		SessionID:        sessionID,
		TurnID:           turnID,
		HarnessSnapshot:  hash,
		ModelFingerprint: Fingerprint(a.Config),
		Payload:          payload,
	}
	if err := a.Traces.AppendDurable(ev); err != nil {
		return err
	}
	if a.Hub != nil {
		a.Hub.Publish(ev)
	}
	a.touchSessionMeta(meta, text)
	return nil
}

// InterruptResult 是 StopTurn 的结果：会话 id 和 idle、cancelled 或 already_done。
type InterruptResult struct {
	SessionID string `json:"session_id"`
	State     string `json:"state"`
}

// StopTurn 取消该会话运行槽的 context，并让线程管理器打断 actor。它不追加 steer。sessionID 为空时对每个正在跑的会话各做一次，返回最后一次的结果。没在跑则是 idle；取消后仍显示在跑则是 cancelled，否则是 already_done。
func (a *App) StopTurn(sessionID string) InterruptResult {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		ids := a.RunningIDs()
		if len(ids) == 0 {
			return InterruptResult{State: "idle"}
		}
		var last InterruptResult
		for _, id := range ids {
			last = a.StopTurn(id)
		}
		return last
	}
	was := a.Running(sessionID)
	if a.Threads != nil {
		a.Threads.Interrupt(sessionID)
	}
	a.mu.Lock()
	slot := a.runs[sessionID]
	a.mu.Unlock()
	if slot != nil && slot.cancel != nil {
		slot.cancel()
	}
	if !was {
		return InterruptResult{SessionID: sessionID, State: "idle"}
	}
	if a.Running(sessionID) {
		return InterruptResult{SessionID: sessionID, State: "cancelled"}
	}
	return InterruptResult{SessionID: sessionID, State: "already_done"}
}

// errBusy 表示该会话已经有运行槽，不能再占一个。
const errBusy errString = "session already running"

// extraTools 把当前 MCP 工具和已加载的 WASM 插件收成额外工具。调用前走能力检查，MCP 还按会话画像的允许名单过滤。
// sessionID 传给 sessionProfile，并写入能力请求的 SessionID。
func (a *App) extraTools(sessionID string) map[string]runtime.ExtraTool {
	extra := map[string]runtime.ExtraTool{}
	var mcpAllow []string
	if p := a.sessionProfile(sessionID); p != nil {
		mcpAllow = p.MCPAllow
	}
	if a.MCP != nil {
		for _, t := range a.MCP.Tools() {
			t := t
			name := "mcp__" + t.Server + "__" + t.Name
			if !profileMCPAllowed(mcpAllow, name, t.Server) {
				continue
			}
			params := t.InputSchema
			if params == nil {
				params = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			extra[name] = runtime.ExtraTool{
				JSON: runtime.ToolJSON{Type: "function", Function: map[string]any{
					"name": name, "description": t.Description, "parameters": params,
				}},
				ReadOnly:  t.ReadOnlyHint,
				OpenWorld: t.OpenWorldHint,
				Exclusive: t.OpenWorldHint || t.DestructiveHint,
				Call: func(argsJSON string) runtime.ToolResult {
					if err := a.Caps.Check(capability.Request{
						Level: capability.Network, Action: name, SessionID: sessionID, ForceAsk: !t.ReadOnlyHint,
					}); err != nil {
						return runtime.ToolResult{Err: err}
					}
					out, err := a.MCP.Call(t.Server, t.Name, argsJSON)
					return runtime.ToolResult{Content: out, Err: err}
				},
			}
		}
	}
	if a.WASM != nil {
		for _, id := range a.WASM.List() {
			id := id
			name := "wasm__" + id
			extra[name] = runtime.ExtraTool{
				JSON: runtime.ToolJSON{Type: "function", Function: map[string]any{
					"name":        name,
					"description": "Call admitted WASM plugin with JSON in/out (no WASI FS).",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"input": map[string]any{"type": "string"},
							"a":     map[string]any{"type": "integer"},
							"b":     map[string]any{"type": "integer"},
						},
					},
				}},
				Call: func(argsJSON string) runtime.ToolResult {
					if err := a.Caps.Check(capability.Request{
						Level: capability.HighRisk, Action: name, SessionID: sessionID, ForceAsk: true,
					}); err != nil {
						return runtime.ToolResult{Err: err}
					}
					mod := a.WASM.Get(id)
					if mod == nil {
						return runtime.ToolResult{Err: fmt.Errorf("wasm %s not loaded", id)}
					}
					if out, err := mod.CallJSON([]byte(argsJSON)); err == nil {
						text := string(out)
						if logs := mod.Logs(); logs != "" {
							text += "\nlog:" + logs
						}
						return runtime.ToolResult{Content: text}
					}
					var args struct {
						A uint64 `json:"a"`
						B uint64 `json:"b"`
					}
					_ = json.Unmarshal([]byte(argsJSON), &args)
					n, err := mod.CallI32(args.A, args.B)
					if err != nil {
						return runtime.ToolResult{Err: err}
					}
					out := fmt.Sprintf("%d", n)
					if logs := mod.Logs(); logs != "" {
						out += "\nlog:" + logs
					}
					return runtime.ToolResult{Content: out}
				},
			}
		}
	}
	return extra
}

// askUserFn 返回向用户提问的函数：先把问题写入轨迹和 Hub，再经 Gate 询问。有事先存下的回答就用它。
// ctx 传给 Gate.AskOffer。
// sessionID 写入提问事件和能力请求的 SessionID。
func (a *App) askUserFn(ctx context.Context, sessionID string) func(string) (string, error) {
	return func(question string) (string, error) {
		shown := runtime.AskPrompt(question)
		ev := trace.Event{
			TS:        time.Now().UTC(),
			Type:      trace.TypeAsk,
			Source:    "agent",
			SessionID: sessionID,
			Payload:   map[string]any{"question": shown, "text": shown, "form": question},
		}
		if a.Traces != nil {
			_ = a.Traces.Append(ev)
		}
		if a.Hub != nil {
			a.Hub.Publish(ev)
		}
		if a.Gate == nil {
			return runtime.AskIdleAnswer(question), nil
		}
		dec, offer, err := a.Gate.AskOffer(ctx, capability.Request{
			Level: capability.ReadWorkspace, Action: "ask_user", Command: question,
			SessionID: sessionID, ForceAsk: true,
		})
		if err != nil {
			return "", err
		}
		a.askMu.Lock()
		ans := a.askAns[offer.ID]
		a.askMu.Unlock()
		if strings.TrimSpace(ans) == "" {
			return "operator decision: " + string(dec), nil
		}
		return ans, nil
	}
}

// RunEval 用当前 harness 跑评测，不改套件。
// ctx 传给 RunEvalMut。
// client 传给 RunEvalMut。
func (a *App) RunEval(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, nil)
}

// RunEvalAt 对指定快照跑评测，不移动 refs/active。
// ctx 传给 runEvalAt。
// client 传给 runEvalAt。
// hash 传给 runEvalAt。
func (a *App) RunEvalAt(ctx context.Context, client runtime.Client, hash string) (eval.RunReport, error) {
	return a.runEvalAt(ctx, client, "", hash, nil)
}

// RunEvalOpts 在 safety 为真时把 no-escape 放进套件的 Safety 再跑。
// ctx 传给 RunEvalMut。
// client 传给 RunEvalMut。
func (a *App) RunEvalOpts(ctx context.Context, client runtime.Client, safety bool) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, func(suite *artifact.EvalSuite) {
		if safety {
			already := false
			for _, id := range suite.Safety {
				if id == "no-escape" {
					already = true
					break
				}
			}
			if !already {
				suite.Safety = append(append([]string{}, suite.Safety...), "no-escape")
			}
		}
	})
}

// RunEvalTB 给套件加上 held-in 的 mkdir-note 和 held-out 的 copy-seed，并把重复次数至少设为 2。
// ctx 传给 RunEvalMut。
// client 传给 RunEvalMut。
func (a *App) RunEvalTB(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, func(suite *artifact.EvalSuite) {
		suite.HeldIn = append(append([]string{}, suite.HeldIn...), "mkdir-note")
		suite.HeldOut = append(append([]string{}, suite.HeldOut...), "copy-seed")
		if suite.Repeats < 2 {
			suite.Repeats = 2
		}
	})
}

// RunEvalSealed 先把套件密封再跑。
// ctx 传给 RunEvalMut。
// client 传给 RunEvalMut。
func (a *App) RunEvalSealed(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, eval.ApplySealed)
}

// RunEvalTransfer 密封后只把 Transfer 当作 held-out，并清掉 held-in、transfer 和 safety 再跑。
// ctx 传给 RunEvalMut。
// client 传给 RunEvalMut。
func (a *App) RunEvalTransfer(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, func(suite *artifact.EvalSuite) {
		eval.ApplySealed(suite)
		suite.HeldIn = nil
		suite.HeldOut = append([]string{}, suite.Transfer...)
		suite.Transfer = nil
		suite.Safety = nil
	})
}

// RunEvalIndex 只按索引迁移调整套件再跑。
// ctx 传给 RunEvalMut。
// client 传给 RunEvalMut。
func (a *App) RunEvalIndex(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, eval.ApplyIndexTransferOnly)
}

// RunEvalBehavior 把套件改成行为探针，held-in 用行为题，其余集合清空，重复次数为 1。
// ctx 传给 RunEvalMut。
// client 传给 RunEvalMut。
func (a *App) RunEvalBehavior(ctx context.Context, client runtime.Client) (eval.RunReport, error) {
	return a.RunEvalMut(ctx, client, func(suite *artifact.EvalSuite) {
		suite.ID = "behavior-probes"
		suite.HeldIn = append([]string{}, eval.BehaviorIDs...)
		suite.HeldOut = nil
		suite.Transfer = nil
		suite.Safety = nil
		suite.EvolveIn = nil
		suite.Repeats = 1
	})
}

// RunEvalMut 对当前 harness 跑评测，mut 可以改套件。
// ctx 传给 runEvalAt。
// client 传给 runEvalAt。
func (a *App) RunEvalMut(ctx context.Context, client runtime.Client, mut func(*artifact.EvalSuite)) (eval.RunReport, error) {
	return a.runEvalAt(ctx, client, "", "", mut)
}

// RunEvalHash 对指定哈希跑评测，mut 可以改套件。
// ctx 传给 runEvalAt。
// client 传给 runEvalAt。
// hash 传给 runEvalAt。
func (a *App) RunEvalHash(ctx context.Context, client runtime.Client, hash string, mut func(*artifact.EvalSuite)) (eval.RunReport, error) {
	return a.runEvalAt(ctx, client, "", hash, mut)
}

// runEval 用指定模型跑评测，快照用当前 active。
// ctx 传给 runEvalAt。
// client 传给 runEvalAt。
// model 传给 runEvalAt。
// mut 传给 runEvalAt。
func (a *App) runEval(ctx context.Context, client runtime.Client, model string, mut func(*artifact.EvalSuite)) (eval.RunReport, error) {
	return a.runEvalAt(ctx, client, model, "", mut)
}

// runEvalAt 装入材料，可选改套件，调用评测引擎，并把报告记下来。
// ctx 传给 Eval.Run。
// client 为 nil 时改用 Client，否则传给 Eval.Run。
// model 为空时改用 Config.Model，再传给 Eval.Run。
// hash 去掉首尾空白后为空时改用 ActiveHash，再传给 LoadSnapshot 和 Materials。
// mut 非 nil 时用来改套件。
func (a *App) runEvalAt(ctx context.Context, client runtime.Client, model, hash string, mut func(*artifact.EvalSuite)) (eval.RunReport, error) {
	if strings.TrimSpace(hash) == "" {
		hash = a.ActiveHash()
	}
	snap, err := a.LoadSnapshot(hash)
	if err != nil {
		return eval.RunReport{}, err
	}
	loop, frags, pb, skills, suite, _, err := a.Materials(hash)
	if err != nil {
		return eval.RunReport{}, err
	}
	if mut != nil {
		mut(&suite)
	}
	if client == nil {
		client, err = a.Client()
		if err != nil {
			return eval.RunReport{}, err
		}
	}
	if model == "" {
		model = a.Config.Model
	}
	meter := &runtime.Meter{USDPerMTok: a.Config.USDPerMTok}
	rep, err := a.Eval.Run(ctx, eval.RunOpts{
		Suite:     suite,
		Snapshot:  snap,
		Hash:      hash,
		Client:    client,
		Loop:      loop,
		Fragments: frags,
		Playbook:  pb,
		Skills:    skills,
		Trace:     a.Traces,
		SessionID: "eval-" + newID()[:8],
		Model:     model,
		Meter:     meter,
	})
	if err != nil {
		return rep, err
	}
	a.rememberEval(rep)
	return rep, nil
}

// EvolveOnce 用候选数 3 跑一轮演进。
// ctx 传给 EvolveWith。
// client 传给 EvolveWith。
// failed 传给 EvolveWith。
func (a *App) EvolveOnce(ctx context.Context, client runtime.Client, failed map[string]string) (evolve.CycleResult, error) {
	return a.EvolveWith(ctx, client, failed, EvolveRun{K: 3})
}

// EvolveK 用给定的候选数跑一轮演进。
// ctx 传给 EvolveWith。
// client 传给 EvolveWith。
// failed 传给 EvolveWith。
// k 写入 EvolveRun 的 K。
func (a *App) EvolveK(ctx context.Context, client runtime.Client, failed map[string]string, k int) (evolve.CycleResult, error) {
	return a.EvolveWith(ctx, client, failed, EvolveRun{K: k})
}

// EvolveRun 是操作者侧的演进参数。PromoteActive 默认关闭，接受的改动只落到 refs/canary。
type EvolveRun struct {
	K              int
	Rounds         int
	PromoteActive  bool
	Sealed         bool
	MaxWall        time.Duration
	MaxUSD         float64
	PromoteRepeats int
	Behavior       bool
	IndexTransfer  bool
	Baselines      int
}

// EvolveWith 在轮数大于 1 时跑完循环并返回最后一轮，否则只跑一轮。
// ctx 在 Rounds 大于 1 时传给 EvolveLoop，否则传给 evolveOnce。
// client 在 Rounds 大于 1 时传给 EvolveLoop，否则传给 evolveOnce。
// failed 在 Rounds 大于 1 时传给 EvolveLoop，否则传给 evolveOnce。
// run 的 Rounds 大于 1 时传给 EvolveLoop，否则传给 evolveOnce。
func (a *App) EvolveWith(ctx context.Context, client runtime.Client, failed map[string]string, run EvolveRun) (evolve.CycleResult, error) {
	if run.Rounds > 1 {
		all, err := a.EvolveLoop(ctx, client, failed, run)
		if err != nil {
			return evolve.CycleResult{}, err
		}
		if len(all) == 0 {
			return evolve.CycleResult{}, fmt.Errorf("no evolve rounds")
		}
		return all[len(all)-1], nil
	}
	return a.evolveOnce(ctx, client, failed, run)
}

// EvolveLoop 按轮数重复演进，上下文取消、墙钟或金额上限到达时停下。
// ctx 每轮先看 Err，再传给 evolveOnce。
// client 传给 evolveOnce。
// failed 只在第一轮传给 evolveOnce，之后改成 nil。
// run 用来读 Rounds、MaxWall 和 MaxUSD，并传给 evolveOnce；Rounds 小于等于 0 时按 1 轮。
func (a *App) EvolveLoop(ctx context.Context, client runtime.Client, failed map[string]string, run EvolveRun) ([]evolve.CycleResult, error) {
	n := run.Rounds
	if n <= 0 {
		n = 1
	}
	start := time.Now()
	var out []evolve.CycleResult
	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if run.MaxWall > 0 && time.Since(start) > run.MaxWall {
			break
		}
		if run.MaxUSD > 0 && a.spentUSD() >= run.MaxUSD {
			break
		}
		res, err := a.evolveOnce(ctx, client, failed, run)
		if err != nil {
			return out, err
		}
		out = append(out, res)
		failed = nil
	}
	return out, nil
}

// spentUSD 返回最近用量快照里的美元金额，没有则返回 0。
func (a *App) spentUSD() float64 {
	a.usageMu.Lock()
	defer a.usageMu.Unlock()
	if a.lastUsage == nil {
		return 0
	}
	v, _ := a.lastUsage["usd"].(float64)
	return v
}

// evolveOnce 选定父快照后跑一轮演进。有迁移题时给候选做标记；要求基线搜索时把比较结果附上。
// ctx 传给 Evolve.Cycle，有迁移题时再传给 markTransfer，基线搜索时再传给 SearchBaselines。
// client 为 nil 时改用 Client，写入 CycleOpts 后传给 Cycle，同一份选项在基线搜索时再传给 SearchBaselines；有迁移题时再传给 markTransfer。
// failed 为 nil 时改成空 map，再写入 FailedTasks。
// run 用来读 Sealed、IndexTransfer、K、PromoteActive、PromoteRepeats、MaxUSD、MaxWall、Behavior 和 Baselines。
func (a *App) evolveOnce(ctx context.Context, client runtime.Client, failed map[string]string, run EvolveRun) (evolve.CycleResult, error) {
	activeHash := a.ActiveHash()
	parentHash := evolve.SelectParent(a.Archive.List(), activeHash)
	if parentHash == "" {
		parentHash = activeHash
	}
	if _, err := a.LoadSnapshot(parentHash); err != nil {
		parentHash = activeHash
	}
	parentSet, err := a.materialSet(parentHash)
	if err != nil {
		return evolve.CycleResult{}, err
	}
	baseSet, err := a.materialSet(activeHash)
	if err != nil {
		return evolve.CycleResult{}, err
	}
	if client == nil {
		client, err = a.Client()
		if err != nil {
			return evolve.CycleResult{}, err
		}
	}
	if failed == nil {
		failed = map[string]string{}
	}
	_, _, _, _, suite, _, err := a.Materials(activeHash)
	if err != nil {
		return evolve.CycleResult{}, err
	}
	if run.Sealed {
		eval.ApplySealed(&suite)
	}
	if run.IndexTransfer {
		eval.ApplyIndexTransfer(&suite)
	}
	k := run.K
	if k <= 0 {
		k = 3
	}
	sid := "evolve-" + newID()[:8]
	opts := evolve.CycleOpts{
		Parent:         parentSet,
		Baseline:       baseSet,
		Suite:          suite,
		Client:         client,
		Trace:          a.Traces,
		SessionID:      sid,
		Model:          a.Config.Model,
		FailedTasks:    failed,
		K:              k,
		PromoteActive:  run.PromoteActive,
		HeldOut:        suite.HeldOut,
		StagingHash:    a.Refs.GetOrEmpty(artifact.RefStaging),
		PromoteRepeats: run.PromoteRepeats,
		MaxUSD:         run.MaxUSD,
		MaxWall:        run.MaxWall,
		Behavior:       run.Behavior,
		IndexTransfer:  run.IndexTransfer,
		USDPerMTok:     a.Config.USDPerMTok,
	}
	res, err := a.Evolve.Cycle(ctx, opts)
	if err != nil {
		a.rememberEvolve(res)
		return res, err
	}
	if res.Promoted != "" && len(suite.Transfer) > 0 {
		a.markTransfer(ctx, client, activeHash, res.Promoted, suite)
	}
	if run.Baselines > 0 {
		bopts := opts
		bopts.SessionID = sid + "-lab"
		if run.MaxUSD > 0 {
			left := run.MaxUSD - res.Spend.USD
			if left < 0 {
				left = 0
			}
			bopts.MaxUSD = left
		}
		cmp, berr := a.Evolve.SearchBaselines(ctx, bopts, run.Baselines)
		if berr != nil {
			a.rememberEvolve(res)
			return res, berr
		}
		res.Compare = &cmp
	}
	a.addSpendUsage(res.Spend)
	if res.Compare != nil {
		a.addSpendUsage(res.Compare.Spend)
	}
	a.rememberEvolve(res)
	return res, nil
}

// SearchBaselines 用当前 active 的材料搜索基线，并把花费记进用量。
// ctx 传给 Evolve.SearchBaselines。
// client 为 nil 时改用 Client，否则写入 CycleOpts。
// run 用来读 Sealed、IndexTransfer、PromoteRepeats、MaxUSD、MaxWall 和 Behavior。
func (a *App) SearchBaselines(ctx context.Context, client runtime.Client, n int, run EvolveRun) (evolve.LabCompare, error) {
	activeHash := a.ActiveHash()
	baseSet, err := a.materialSet(activeHash)
	if err != nil {
		return evolve.LabCompare{}, err
	}
	if client == nil {
		client, err = a.Client()
		if err != nil {
			return evolve.LabCompare{}, err
		}
	}
	_, _, _, _, suite, _, err := a.Materials(activeHash)
	if err != nil {
		return evolve.LabCompare{}, err
	}
	if run.Sealed {
		eval.ApplySealed(&suite)
	}
	if run.IndexTransfer {
		eval.ApplyIndexTransfer(&suite)
	}
	opts := evolve.CycleOpts{
		Parent:         baseSet,
		Baseline:       baseSet,
		Suite:          suite,
		Client:         client,
		Trace:          a.Traces,
		SessionID:      "lab-" + newID()[:8],
		Model:          a.Config.Model,
		HeldOut:        suite.HeldOut,
		PromoteRepeats: run.PromoteRepeats,
		MaxUSD:         run.MaxUSD,
		MaxWall:        run.MaxWall,
		Behavior:       run.Behavior,
		USDPerMTok:     a.Config.USDPerMTok,
	}
	cmp, err := a.Evolve.SearchBaselines(ctx, opts, n)
	if err != nil {
		return cmp, err
	}
	a.addSpendUsage(cmp.Spend)
	a.rememberEvolve(evolve.CycleResult{Compare: &cmp, Spend: cmp.Spend})
	return cmp, nil
}

// markTransfer 用迁移题分别跑基线和候选。候选的 held-out 通过数更少时，在归档里标上。
// ctx 传给两次 Eval.Run。
// client 传给两次 Eval.Run。
// baseHash 传给 materialSet。
// canary 传给 materialSet，并传给 FlagTransfer。
// suite 的 Transfer 当作 held-out，HeldIn、Transfer 和 Safety 清空后再跑。
func (a *App) markTransfer(ctx context.Context, client runtime.Client, baseHash, canary string, suite artifact.EvalSuite) {
	baseSet, err := a.materialSet(baseHash)
	if err != nil {
		return
	}
	candSet, err := a.materialSet(canary)
	if err != nil {
		return
	}
	xfer := suite
	xfer.HeldIn = nil
	xfer.HeldOut = append([]string{}, suite.Transfer...)
	xfer.Transfer = nil
	xfer.Safety = nil
	baseRep, err := a.Eval.Run(ctx, eval.RunOpts{
		Suite: xfer, Snapshot: baseSet.Snap, Hash: baseSet.Hash, Client: client,
		Loop: baseSet.Loop, Fragments: baseSet.Fragments, Playbook: baseSet.Playbook, Skills: baseSet.Skills,
		SessionID: "xfer-base", Model: a.Config.Model, Meter: &runtime.Meter{USDPerMTok: a.Config.USDPerMTok},
	})
	if err != nil {
		return
	}
	candRep, err := a.Eval.Run(ctx, eval.RunOpts{
		Suite: xfer, Snapshot: candSet.Snap, Hash: candSet.Hash, Client: client,
		Loop: candSet.Loop, Fragments: candSet.Fragments, Playbook: candSet.Playbook, Skills: candSet.Skills,
		SessionID: "xfer-cand", Model: a.Config.Model, Meter: &runtime.Meter{USDPerMTok: a.Config.USDPerMTok},
	})
	if err != nil {
		return
	}
	fail := candRep.Metrics.HeldOutPass < baseRep.Metrics.HeldOutPass
	_ = a.Archive.FlagTransfer(canary, fail)
}

// materialSet 把快照和它解出的循环、提示、playbook、技能收成一套材料。
// hash 传给 LoadSnapshot 和 Materials，并写入材料的 Hash。
func (a *App) materialSet(hash string) (evolve.MaterialSet, error) {
	snap, err := a.LoadSnapshot(hash)
	if err != nil {
		return evolve.MaterialSet{}, err
	}
	loop, frags, pb, skills, _, _, err := a.Materials(hash)
	if err != nil {
		return evolve.MaterialSet{}, err
	}
	return evolve.MaterialSet{
		Snap: snap, Hash: hash, Loop: loop, Fragments: frags, Playbook: pb, Skills: skills,
	}, nil
}

// StageACE 在上一轮成功结束时，把整理后的 playbook 写成新快照并设到 refs/staging。它不移动 refs/active。
// sessionID 传给 GetSession 和 Traces.Read，并写入笔记事件和 journal。
func (a *App) StageACE(sessionID string) (string, error) {
	meta, err := a.GetSession(sessionID)
	if err != nil {
		return "", nil
	}
	hash := session.ResolveHarness(session.NormalizePolicy(meta.HarnessPolicy), meta.Harness, a.ActiveHash())
	if hash == "" {
		return "", nil
	}
	events, err := a.Traces.Read(sessionID)
	if err != nil || len(events) == 0 {
		return "", err
	}
	var lastEnd *trace.Event
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == trace.TypeTurnEnd {
			lastEnd = &events[i]
			break
		}
	}
	if lastEnd == nil {
		return "", nil
	}
	ok, _ := lastEnd.Payload["ok"].(bool)
	if !ok {
		return "", nil
	}
	base, err := a.LoadSnapshot(hash)
	if err != nil {
		return "", err
	}
	_, _, pb, _, _, _, err := a.Materials(hash)
	if err != nil {
		return "", err
	}
	if st := a.Refs.GetOrEmpty(artifact.RefStaging); st != "" {
		if ss, err := a.LoadSnapshot(st); err == nil && ss.Playbook != "" {
			if v, _, err := artifact.Decode[artifact.Playbook](a.CAS, ss.Playbook); err == nil {
				pb = v
			}
		}
	}
	stripped := stripACEBodies(events)
	ws := ""
	if m, err := a.GetSession(sessionID); err == nil {
		ws = m.ToolRoot()
	}
	if notes := runtime.ReadNotes(runtime.BindSpill(a.Home.Root, ws, sessionID)); notes != "" {
		stripped = append([]trace.Event{{
			Type:      trace.TypeUser,
			SessionID: sessionID,
			Payload:   map[string]any{"text": notes},
		}}, stripped...)
	}
	ref := evolve.Reflect(stripped, nil, pb)
	curated := evolve.Curate(pb, ref)
	if evolve.PlaybooksEqual(pb, curated) {
		return "", nil
	}
	pbHash, err := a.CAS.Put(artifact.KindPlaybook, curated.ID, curated)
	if err != nil {
		return "", err
	}
	next := artifact.CloneSnapshot(base)
	next.Parent = hash
	next.Playbook = pbHash
	next.Note = "online-ace-stage"
	staged, err := a.CAS.PutSnapshot(next)
	if err != nil {
		return "", err
	}
	if err := a.Refs.Set(artifact.RefStaging, staged); err != nil {
		return "", err
	}
	_ = a.Archive.Add(evolve.Node{
		ID: staged, Parent: hash, Snapshot: staged, ProposalID: "online-ace-stage",
		Note: "online-ace-stage",
	})
	_, _ = a.Journal.Append("ace.stage", map[string]string{"hash": staged, "session": sessionID})
	return staged, nil
}

// stripACEBodies 把工具结果正文收短：错误截断，其余只留工具名和调用 id。
// evs 先整份拷贝；工具结果正文以 ERROR: 开头且长于 240 时截断，否则改成工具名加调用 id。
func stripACEBodies(evs []trace.Event) []trace.Event {
	out := make([]trace.Event, len(evs))
	copy(out, evs)
	for i := range out {
		if out[i].Type != trace.TypeToolResult {
			continue
		}
		p := map[string]any{}
		for k, v := range out[i].Payload {
			p[k] = v
		}
		name, _ := p["name"].(string)
		id, _ := p["id"].(string)
		content, _ := p["content"].(string)
		if strings.HasPrefix(content, "ERROR:") {
			if len(content) > 240 {
				content = content[:240]
			}
			p["content"] = content
		} else {
			p["content"] = name + " " + id
		}
		out[i].Payload = p
	}
	return out
}

// ListHarnesses 返回引用名到快照哈希的映射。
func (a *App) ListHarnesses() (map[string]string, error) {
	return a.Refs.List()
}

// BestOfN 把同一套评测重复跑 n 次，并留下得分最高的一次。
// ctx 传给 BestOfNMut。
// client 传给 BestOfNMut。
func (a *App) BestOfN(ctx context.Context, n int, client runtime.Client) (BestOfNReport, error) {
	return a.BestOfNMut(ctx, n, client, nil)
}

// BestOfNMut 重复跑评测，mut 可以改套件，并按 held-in 与 held-out 的通过数选最好的一次。
// ctx 传给 runEval。
// client 传给 runEval。
func (a *App) BestOfNMut(ctx context.Context, n int, client runtime.Client, mut func(*artifact.EvalSuite)) (BestOfNReport, error) {
	if n < 1 {
		n = 3
	}
	rep := BestOfNReport{N: n, Kind: "repeat"}
	for i := 0; i < n; i++ {
		r, err := a.runEval(ctx, client, "", mut)
		if err != nil {
			return rep, err
		}
		rep.Reports = append(rep.Reports, r)
		if i == 0 || evalScore(r) > evalScore(rep.Best) {
			rep.Best = r
		}
	}
	return rep, nil
}

// BestOfModels 给每个模型各跑一次评测，并选通过数最高的模型。
// ctx 传给 runEval。
// models 长度为 0 时改用 Config.Model 和 Config.Models，逐项去掉空白，空串和重复丢掉。
// clients 按模型名取客户端；该名为空且 clients 非 nil 时再取键为 * 的客户端。
func (a *App) BestOfModels(ctx context.Context, models []string, clients map[string]runtime.Client) (BestOfNReport, error) {
	if len(models) == 0 {
		models = append([]string{a.Config.Model}, a.Config.Models...)
	}
	seen := map[string]bool{}
	var clean []string
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		clean = append(clean, m)
	}
	if len(clean) == 0 {
		clean = []string{a.Config.Model}
	}
	type row struct {
		model string
		r     eval.RunReport
		err   error
	}
	ch := make(chan row, len(clean))
	for _, m := range clean {
		m := m
		go func() {
			cl := clients[m]
			if cl == nil && clients != nil {
				cl = clients["*"]
			}
			r, err := a.runEval(ctx, cl, m, nil)
			ch <- row{model: m, r: r, err: err}
		}()
	}
	rep := BestOfNReport{N: len(clean), Kind: "models"}
	for i := 0; i < len(clean); i++ {
		got := <-ch
		if got.err != nil {
			return rep, got.err
		}
		rep.Models = append(rep.Models, ModelRun{Model: got.model, Report: got.r})
		rep.Reports = append(rep.Reports, got.r)
		if i == 0 || evalScore(got.r) > evalScore(rep.Best) {
			rep.Best = got.r
			rep.BestModel = got.model
		}
	}
	return rep, nil
}

// BestOfNReport 是重复或跨模型评测的各次报告，以及最好的一次。
type BestOfNReport struct {
	N         int              `json:"n"`
	Kind      string           `json:"kind,omitempty"`
	Reports   []eval.RunReport `json:"reports"`
	Best      eval.RunReport   `json:"best"`
	BestModel string           `json:"best_model,omitempty"`
	Models    []ModelRun       `json:"models,omitempty"`
}

// ModelRun 是一个模型名和它的一次评测报告。
type ModelRun struct {
	Model  string         `json:"model"`
	Report eval.RunReport `json:"report"`
}

// evalScore 把 held-in 与 held-out 的通过数相加。
func evalScore(r eval.RunReport) int {
	return r.Metrics.HeldInPass + r.Metrics.HeldOutPass
}

// DiffDetail 比较两份快照的材料差异，并带上文本、字段和两边快照。
// aHash 传给 LoadSnapshot，作为左边。
// bHash 传给 LoadSnapshot，作为右边。
func (a *App) DiffDetail(aHash, bHash string) (map[string]any, error) {
	left, err := a.LoadSnapshot(aHash)
	if err != nil {
		return nil, err
	}
	right, err := a.LoadSnapshot(bHash)
	if err != nil {
		return nil, err
	}
	changes := artifact.MaterialDiff(a.CAS, left, right)
	text := artifact.MaterialDiffText(changes)
	if text == "" {
		text = artifact.SnapshotDiff(left, right)
	}
	return map[string]any{
		"text":    text,
		"fields":  artifact.SnapshotDiffFields(left, right),
		"changes": changes,
		"a":       left,
		"b":       right,
	}, nil
}

// Diff 只返回两份快照的差异文本。
// aHash 传给 DiffDetail。
// bHash 传给 DiffDetail。
func (a *App) Diff(aHash, bHash string) (string, error) {
	d, err := a.DiffDetail(aHash, bHash)
	if err != nil {
		return "", err
	}
	text, _ := d["text"].(string)
	return text, nil
}

// rememberEval 记下评测报告，写入 last.json，并更新用量。
// rep 赋给 lastEval，并写入 last.json 和用量。
func (a *App) rememberEval(rep eval.RunReport) {
	a.evalMu.Lock()
	a.lastEval = rep
	a.evalMu.Unlock()
	b, err := json.MarshalIndent(rep, "", "  ")
	if err == nil && a.Home != nil {
		_ = os.WriteFile(filepath.Join(a.Home.EvalRuns(), "last.json"), b, 0o644)
	}
	a.RememberUsage(map[string]any{
		"usd":           rep.USD,
		"input_tokens":  rep.TokensIn,
		"output_tokens": rep.TokensOut,
		"wall_ms":       rep.WallMs,
	})
}

// LastEval 返回内存里的评测报告；还没有时从 last.json 读。
func (a *App) LastEval() eval.RunReport {
	a.evalMu.Lock()
	rep := a.lastEval
	a.evalMu.Unlock()
	if rep.Suite != "" || len(rep.Results) > 0 {
		return rep
	}
	if a.Home == nil {
		return eval.RunReport{}
	}
	b, err := os.ReadFile(filepath.Join(a.Home.EvalRuns(), "last.json"))
	if err != nil {
		return eval.RunReport{}
	}
	_ = json.Unmarshal(b, &rep)
	a.evalMu.Lock()
	a.lastEval = rep
	a.evalMu.Unlock()
	return rep
}

// rememberEvolve 记下演进结果，并写入 last-evolve.json。
// res 赋给 lastEvolve，并写入 last-evolve.json。
func (a *App) rememberEvolve(res evolve.CycleResult) {
	a.evolveMu.Lock()
	a.lastEvolve = res
	a.evolveMu.Unlock()
	b, err := json.MarshalIndent(res, "", "  ")
	if err == nil && a.Home != nil {
		_ = os.WriteFile(filepath.Join(a.Home.EvalRuns(), "last-evolve.json"), b, 0o644)
	}
}

// LastEvolve 返回内存里的演进结果；还没有时从 last-evolve.json 读。
func (a *App) LastEvolve() evolve.CycleResult {
	a.evolveMu.Lock()
	res := a.lastEvolve
	a.evolveMu.Unlock()
	if len(res.Tried) > 0 || res.Promoted != "" || res.Compare != nil {
		return res
	}
	if a.Home == nil {
		return evolve.CycleResult{}
	}
	b, err := os.ReadFile(filepath.Join(a.Home.EvalRuns(), "last-evolve.json"))
	if err != nil {
		return evolve.CycleResult{}
	}
	_ = json.Unmarshal(b, &res)
	a.evolveMu.Lock()
	a.lastEvolve = res
	a.evolveMu.Unlock()
	return res
}

// addSpendUsage 把这一次花费加进最近的用量快照。
func (a *App) addSpendUsage(s evolve.Spend) {
	prev := a.LastUsage()
	usd, _ := prev["usd"].(float64)
	in := intFromAny(prev["input_tokens"])
	out := intFromAny(prev["output_tokens"])
	wall := int64(intFromAny(prev["wall_ms"]))
	a.RememberUsage(map[string]any{
		"usd":           usd + s.USD,
		"input_tokens":  in + s.TokensIn,
		"output_tokens": out + s.TokensOut,
		"wall_ms":       wall + s.WallMs,
	})
}

// intFromAny 把 int、int64 或 float64 收成 int，其他类型返回 0。
// v 是 int、int64 或 float64 时收成 int，其他类型返回 0。
func intFromAny(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		return 0
	}
}

// applyStudioLanePrompt 换上工作室引导，去掉要求先巡视工作区的片段，并附上工作室运行说明。
// loop 的 Bootstrap 换成 studioBootstrap。
// frags 丢掉正文里含 inspect the workspace 的片段，并附上 studioRuntimePin。
func applyStudioLanePrompt(loop artifact.LoopPreset, frags []artifact.PromptFragment) (artifact.LoopPreset, []artifact.PromptFragment) {
	loop.Bootstrap = studioBootstrap
	kept := make([]artifact.PromptFragment, 0, len(frags)+1)
	for _, f := range frags {
		if strings.Contains(strings.ToLower(f.Text), "inspect the workspace") {
			continue
		}
		kept = append(kept, f)
	}
	kept = append(kept, artifact.PromptFragment{Slot: "runtime", Text: studioRuntimePin})
	return loop, kept
}

// applyConductorLanePrompt 换上指挥引导，去掉要求先巡视工作区的片段，并附上指挥运行说明。
// loop 的 Bootstrap 换成 conductorBootstrap。
// frags 丢掉正文里含 inspect the workspace 的片段，并附上 conductorRuntimePin。
func applyConductorLanePrompt(loop artifact.LoopPreset, frags []artifact.PromptFragment) (artifact.LoopPreset, []artifact.PromptFragment) {
	loop.Bootstrap = conductorBootstrap
	kept := make([]artifact.PromptFragment, 0, len(frags)+1)
	for _, f := range frags {
		if strings.Contains(strings.ToLower(f.Text), "inspect the workspace") {
			continue
		}
		kept = append(kept, f)
	}
	kept = append(kept, artifact.PromptFragment{Slot: "runtime", Text: conductorRuntimePin})
	return loop, kept
}

// newID 返回 8 字节随机数的十六进制，用作新的会话 id。
func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
