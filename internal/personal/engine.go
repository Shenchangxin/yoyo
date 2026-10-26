// Package personal 在 Open 收到的目录里保存任务、目标和 URL 监视，状态文件是 workspace.json。桌面应用把该目录设成主目录的 personal/。
// KindAgent 创建后即为 waiting_input，tick 会跳过它，不另开模型循环。
// 邮件和日历先 Propose，Decide 批准后才经连接器发出。本包不写记忆，也不把 Markdown 写进主目录的 pages/。
package personal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/inbox"
)

// Bind 是挂到引擎上的依赖。Workspace 只用来拼接相对的表单或 CSV 路径，不是页面库目录。
type Bind struct {
	Connectors *connector.Broker
	Inbox      *inbox.Store
	Workspace  string
	Fetch      func(ctx context.Context, url string) (string, error)
	Now        func() time.Time
	Poll       time.Duration
	Lease      time.Duration
	OnChange   func()
	Paused     func() bool
}

// Engine 保存个人目录、workspace.json，以及正在跑的任务租约。
type Engine struct {
	dir        string
	store      *Store
	Connectors *connector.Broker
	Inbox      *inbox.Store
	Workspace  string
	Fetch      func(ctx context.Context, url string) (string, error)
	Now        func() time.Time
	Poll       time.Duration
	Lease      time.Duration
	OnChange   func()
	Paused     func() bool

	mu       sync.Mutex
	running  bool
	inFlight map[string]struct{}
	seq      atomic.Int64
	abort    map[string]context.CancelFunc
	lastTick time.Time
}

// Open 创建 dir 和 dir/files，权限都是 0o755，并打开 workspace.json。Poll 默认 1 秒，Lease 默认 1 分钟，Fetch 默认 SafeGET。
func Open(dir string) (*Engine, error) {
	st, err := openStore(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "files"), 0o755); err != nil {
		return nil, err
	}
	return &Engine{
		dir:      dir,
		store:    st,
		Poll:     time.Second,
		Lease:    time.Minute,
		inFlight: map[string]struct{}{},
		abort:    map[string]context.CancelFunc{},
		Fetch:    SafeGET,
		Now:      func() time.Time { return time.Now().UTC() },
	}, nil
}

// Bind 只覆盖传入的非空字段。Poll 和 Lease 只在大于 0 时覆盖，函数为 nil 时保持原值。
func (e *Engine) Bind(b Bind) {
	if b.Connectors != nil {
		e.Connectors = b.Connectors
	}
	if b.Inbox != nil {
		e.Inbox = b.Inbox
	}
	if b.Workspace != "" {
		e.Workspace = b.Workspace
	}
	if b.Fetch != nil {
		e.Fetch = b.Fetch
	}
	if b.Now != nil {
		e.Now = b.Now
	}
	if b.Poll > 0 {
		e.Poll = b.Poll
	}
	if b.Lease > 0 {
		e.Lease = b.Lease
	}
	if b.OnChange != nil {
		e.OnChange = b.OnChange
	}
	if b.Paused != nil {
		e.Paused = b.Paused
	}
}

// now 调用 Now 并转成 UTC。Now 为 nil 时用当前 UTC。
func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

// newID 用前缀、时分秒毫秒和递增序号拼出 ID。
func (e *Engine) newID(prefix string) string {
	n := e.seq.Add(1)
	return fmt.Sprintf("%s-%s-%d", prefix, e.now().Format("150405.000"), n)
}

// hashID 把各段用空字节隔开后做 SHA-256，返回十六进制的前 16 位。
func hashID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// Snapshot 复制存储里的快照，并填上内存中的 Running 和 LastTickAt。
func (e *Engine) Snapshot() Snapshot {
	s := e.store.snapshot()
	e.mu.Lock()
	s.Worker.Running = e.running
	s.Worker.LastTickAt = e.lastTick
	e.mu.Unlock()
	return s
}

// changed 在 OnChange 非 nil 时调用它。
func (e *Engine) changed() {
	if e.OnChange != nil {
		e.OnChange()
	}
}

// notify 在 Inbox 为 nil 时直接返回，否则推一条收件箱记录并调用 changed。
func (e *Engine) notify(kind inbox.Kind, title, body, taskID, key string) {
	if e.Inbox == nil {
		return
	}
	e.Inbox.Push(inbox.Item{
		Kind:   kind,
		Title:  title,
		Body:   body,
		TaskID: taskID,
		Key:    key,
	})
	e.changed()
}

// CreateTask 把空种类收成 agent。prompt 和 title 都空则返回错误。agent 的状态是 waiting_input，题面写明 worker 不另开模型循环。
func (e *Engine) CreateTask(title, prompt string, kind Kind, input map[string]any, goalID, sessionID string) (Task, error) {
	kind = Kind(strings.ToLower(strings.TrimSpace(string(kind))))
	switch kind {
	case KindAgent, KindDocument, KindMonitor, KindFinance, KindPlan:
	default:
		kind = KindAgent
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" && title == "" {
		return Task{}, errf("prompt required")
	}
	if title == "" {
		title = clip(prompt, 80)
	}
	if input == nil {
		input = map[string]any{}
	}
	now := e.now()
	t := Task{
		ID:        e.newID("task"),
		Title:     clip(title, 160),
		Prompt:    prompt,
		Kind:      kind,
		Status:    StatusQueued,
		GoalID:    goalID,
		Input:     input,
		State:     map[string]any{},
		CreatedAt: now,
		UpdatedAt: now,
		SessionID: sessionID,
		Plan:      defaultPlan(kind),
	}
	if kind == KindAgent {
		t.Status = StatusWaitingInput
		t.Question = "Continue this work in chat, then call complete_work with the result. The worker does not start a second model loop."
	}
	e.store.putTask(t)
	e.changed()
	return t, nil
}

// defaultPlan 按种类给出步骤标题。未列出的种类只有一步，写明在对话里完成。
func defaultPlan(kind Kind) []Step {
	switch kind {
	case KindDocument:
		return []Step{
			{ID: "1", Title: "Inspect the document", Status: "pending"},
			{ID: "2", Title: "Write a filled copy", Status: "pending"},
			{ID: "3", Title: "Prepare a reply", Status: "pending"},
			{ID: "4", Title: "Wait for send approval", Status: "pending"},
		}
	case KindMonitor:
		return []Step{{ID: "1", Title: "Check the page", Status: "pending"}}
	case KindFinance:
		return []Step{{ID: "1", Title: "Analyze the CSV", Status: "pending"}}
	case KindPlan:
		return []Step{{ID: "1", Title: "Write milestones", Status: "pending"}}
	default:
		return []Step{{ID: "1", Title: "Do the work in chat", Status: "pending"}}
	}
}

// Answer 只接受 waiting_input。document 要解析出字段，plan 要按行给出里程碑，然后把状态改回 queued。
func (e *Engine) Answer(id, text string) (Task, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Task{}, errf("empty answer")
	}
	t, ok := e.store.task(id)
	if !ok {
		return Task{}, errf("unknown task")
	}
	if t.Status != StatusWaitingInput {
		return Task{}, errf("task is not waiting for input")
	}
	input := cloneMap(t.Input)
	state := cloneMap(t.State)
	state["answer"] = text
	switch t.Kind {
	case KindDocument:
		fields := parseFields(text)
		if len(fields) == 0 {
			return Task{}, errf("reply with JSON object of field values")
		}
		input["fields"] = fields
	case KindPlan:
		var ms []string
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(strings.TrimLeft(line, "-* "))
			if line != "" {
				ms = append(ms, line)
			}
		}
		if len(ms) == 0 {
			return Task{}, errf("reply with milestone titles, one per line")
		}
		input["milestones"] = ms
	}
	out, ok := e.store.casTask(id, string(StatusWaitingInput), "", func(cur *Task) bool {
		cur.Input = input
		cur.State = state
		cur.Status = StatusQueued
		cur.Question = ""
		cur.Error = ""
		cur.UpdatedAt = e.now()
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		return true
	})
	if !ok {
		return t, errf("task changed")
	}
	e.changed()
	return out, nil
}

// parseFields 在正文以 { 开头时按 JSON 对象解析，否则按行切分冒号或等号。
func parseFields(text string) map[string]string {
	text = strings.TrimSpace(text)
	out := map[string]string{}
	if strings.HasPrefix(text, "{") {
		var raw map[string]any
		if json.Unmarshal([]byte(text), &raw) == nil {
			for k, v := range raw {
				out[k] = strings.TrimSpace(fmt.Sprint(v))
			}
			return out
		}
	}
	for _, line := range strings.Split(text, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			k, v, ok = strings.Cut(line, "=")
		}
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if k != "" {
			out[k] = v
		}
	}
	return out
}

// Complete 把未成功且未取消的任务标成 succeeded，并按 task-done 通知收件箱。
func (e *Engine) Complete(id, result string) (Task, error) {
	out, ok := e.store.casTask(id, "", "", func(cur *Task) bool {
		if cur.Status == StatusSucceeded || cur.Status == StatusCancelled {
			return false
		}
		cur.Status = StatusSucceeded
		cur.Result = result
		cur.Error = ""
		cur.Question = ""
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = e.now()
		return true
	})
	if !ok {
		return Task{}, errf("cannot complete task")
	}
	e.notify(inbox.KindWork, out.Title, result, out.ID, "task-done:"+out.ID)
	return out, nil
}

// Fail 把未成功且未取消的任务标成 failed，并写下错误文本。
func (e *Engine) Fail(id, errMsg string) (Task, error) {
	out, ok := e.store.casTask(id, "", "", func(cur *Task) bool {
		if cur.Status == StatusSucceeded || cur.Status == StatusCancelled {
			return false
		}
		cur.Status = StatusFailed
		cur.Error = errMsg
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = e.now()
		return true
	})
	if !ok {
		return Task{}, errf("cannot fail task")
	}
	e.changed()
	return out, nil
}

// Cancel 把未成功且未取消的任务标成 cancelled。仍在 awaiting 的关联提案改为 cancelled，监视改为 stopped，并中止该任务的租约。
func (e *Engine) Cancel(id string) (Task, error) {
	var actionID string
	out, ok := e.store.casTask(id, "", "", func(cur *Task) bool {
		if cur.Status == StatusSucceeded || cur.Status == StatusCancelled {
			return false
		}
		actionID = cur.ActionID
		cur.Status = StatusCancelled
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = e.now()
		return true
	})
	if !ok {
		return Task{}, errf("cannot cancel task")
	}
	if actionID != "" {
		if p, ok := e.store.proposal(actionID); ok && p.Status == ProposalAwaiting {
			p.Status = ProposalCancelled
			p.Activity = append(p.Activity, Activity{At: e.now(), Status: string(ProposalCancelled), Detail: "task cancelled"})
			e.store.putProposal(p)
		}
	}
	e.abortTask(id)
	for _, m := range e.store.snapshot().Monitors {
		if m.TaskID == id {
			m.Status = "stopped"
			e.store.putMonitor(m)
		}
	}
	e.changed()
	return out, nil
}

// CreateGoal 要求标题非空。类别为空时写成 Personal。空的里程碑标题会被跳过。
func (e *Engine) CreateGoal(title, description, category string, milestones []string) (Goal, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Goal{}, errf("goal title required")
	}
	if category == "" {
		category = "Personal"
	}
	now := e.now()
	g := Goal{
		ID:          e.newID("goal"),
		Title:       clip(title, 160),
		Description: clip(description, 4000),
		Category:    clip(category, 80),
		Status:      "active",
		CreatedAt:   now,
	}
	for i, m := range milestones {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		g.Milestones = append(g.Milestones, Milestone{ID: fmt.Sprintf("m%d", i+1), Title: clip(m, 200)})
	}
	e.store.putGoal(g)
	e.changed()
	return g, nil
}

// Watch 要求标题和非空 URL。除 sample:// 外必须是 http 或 https，并拒绝 blockedHost。条件默认 change，间隔默认 15 分钟，最大 10080。任务随后改成 scheduled。
func (e *Engine) Watch(title, rawURL, condition, value string, intervalMinutes int) (Monitor, Task, error) {
	title = strings.TrimSpace(title)
	rawURL = strings.TrimSpace(rawURL)
	if title == "" || rawURL == "" {
		return Monitor{}, Task{}, errf("title and url required")
	}
	if !strings.HasPrefix(rawURL, "sample://") {
		uok := strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")
		if !uok {
			return Monitor{}, Task{}, errf("url must be http(s) or sample://")
		}
		if u, err := parsePublicURL(rawURL); err != nil {
			return Monitor{}, Task{}, err
		} else if blockedHost(u) {
			return Monitor{}, Task{}, errf("host is not allowed")
		}
	}
	condition = strings.ToLower(strings.TrimSpace(condition))
	if condition == "" {
		condition = "change"
	}
	switch condition {
	case "change", "contains", "price_below":
	default:
		return Monitor{}, Task{}, errf("condition must be change, contains, or price_below")
	}
	if condition != "change" && strings.TrimSpace(value) == "" {
		return Monitor{}, Task{}, errf("enter a condition value")
	}
	if condition == "price_below" {
		n := 0.0
		fmt.Sscanf(value, "%f", &n)
		if n <= 0 {
			return Monitor{}, Task{}, errf("enter a positive price")
		}
	}
	if intervalMinutes < 1 {
		intervalMinutes = 15
	}
	if intervalMinutes > 10080 {
		intervalMinutes = 10080
	}
	now := e.now()
	t, err := e.CreateTask(title, "Watch "+rawURL, KindMonitor, map[string]any{
		"title": title, "url": rawURL, "condition": condition, "value": value, "interval_minutes": intervalMinutes,
	}, "", "")
	if err != nil {
		return Monitor{}, Task{}, err
	}
	m := Monitor{
		ID:              e.newID("watch"),
		TaskID:          t.ID,
		Title:           clip(title, 160),
		URL:             rawURL,
		Condition:       condition,
		Value:           value,
		IntervalMinutes: intervalMinutes,
		Status:          "active",
		NextCheckAt:     now,
	}
	e.store.putMonitor(m)
	e.store.casTask(t.ID, "", "", func(cur *Task) bool {
		cur.Input["monitorId"] = m.ID
		cur.Status = StatusScheduled
		cur.NextRunAt = &now
		cur.UpdatedAt = now
		return true
	})
	t, _ = e.store.task(t.ID)
	e.changed()
	return m, t, nil
}

// parsePublicURL 去掉协议、路径和端口，只返回主机名。空主机名返回错误。
func parsePublicURL(raw string) (string, error) {
	// 只取出主机名，交给 blockedHost。
	s := raw
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	host, _, _ := strings.Cut(s, "/")
	host, _, _ = strings.Cut(host, ":")
	if host == "" {
		return "", errf("need public http(s) URL")
	}
	return host, nil
}

// PresentChoices 在选项为空时放入一个 OK。任一选项的 Source 非空就当成比较，最多保留 3 项，并向收件箱推一条询问。
func (e *Engine) PresentChoices(sessionID, title string, options []ChoiceOption) Choice {
	if title == "" {
		title = "Choose"
	}
	if len(options) == 0 {
		options = []ChoiceOption{{ID: "ok", Label: "OK"}}
	}
	comparison := false
	for _, o := range options {
		if strings.TrimSpace(o.Source) != "" {
			comparison = true
			break
		}
	}
	if comparison && len(options) > 3 {
		options = options[:3]
	}
	c := Choice{ID: e.newID("choice"), SessionID: sessionID, Title: title, Options: options, CreatedAt: e.now()}
	e.store.putChoice(c)
	var body strings.Builder
	for _, o := range options {
		fmt.Fprintf(&body, "- %s: %s\n", o.ID, o.Label)
	}
	e.notify(inbox.KindAsk, title, body.String(), "", "choice:"+c.ID)
	return c
}

// SelectChoice 按选项 ID 或忽略大小写的标签选中一项。未知选项返回错误。
func (e *Engine) SelectChoice(id, option string) (Choice, error) {
	c, ok := e.store.choice(id)
	if !ok {
		return Choice{}, errf("unknown choice")
	}
	option = strings.TrimSpace(option)
	okOpt := false
	for _, o := range c.Options {
		if o.ID == option || strings.EqualFold(o.Label, option) {
			option = o.ID
			okOpt = true
			break
		}
	}
	if !okOpt {
		return Choice{}, errf("unknown option")
	}
	c.Selected = option
	e.store.putChoice(c)
	e.changed()
	return c, nil
}

// filesDir 返回 dir/files 下该任务的目录，并按 0o755 创建。
func (e *Engine) filesDir(taskID string) string {
	p := filepath.Join(e.dir, "files", taskID)
	_ = os.MkdirAll(p, 0o755)
	return p
}

// clip 去掉首尾空白，并按字节数截断。n 不大于 0 时不截。
func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if n > 0 && len(s) > n {
		return s[:n]
	}
	return s
}

// strMap 在值为 nil 时返回空串，否则 Sprint 后再去掉首尾空白。
func strMap(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

// SetSample 把 sample:// 使用的键和正文写入存储。
func (e *Engine) SetSample(key, text string) {
	e.store.sample(key, text)
}
