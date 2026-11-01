// runtime.go 持有编排引擎、两秒轮次和 SQLite 快照。
// 它不画界面。桌面和 HTTP 通过 View、Decide、SetMode 使用它。

package superagent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"

	_ "modernc.org/sqlite"
)

// Runtime 把引擎、副作用、来源轮询和快照放在一起。两秒轮次调用 Pass。
type Runtime struct {
	mu           sync.Mutex
	eng          *Engine
	effects      *Queue
	exec         *Executor
	ready        func() error
	db           *sql.DB
	cancel       context.CancelFunc
	client       *http.Client
	sources      SourceConfig
	status       map[string]SourceState
	health       string
	at           time.Time
	workers      int
	sourcesSaved bool
	lastGitHub   time.Time
	lastFeishu   time.Time
}

// Open 打开 path 上的快照并还原引擎。path 为空时只放在内存里。policy 在没有快照时作为初始策略。
func Open(path string, policy Policy) (*Runtime, error) {
	eng := NewEngine(policy)
	rt := &Runtime{
		eng:     eng,
		effects: NewQueue(),
		client:  &http.Client{Timeout: 20 * time.Second},
		sources: DefaultSources(),
		status:  map[string]SourceState{},
		health:  "ready",
	}
	if path == "" {
		return rt, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS snapshot (id INTEGER PRIMARY KEY CHECK (id = 1), body TEXT NOT NULL)`); err != nil {
		_ = db.Close()
		return nil, err
	}
	rt.db = db
	if err := rt.load(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return rt, nil
}

// Bind 接上模型和命令。ready 返回错误时派工先推迟。model 是工单代理的补全。shell 执行工单里的命令。
func (rt *Runtime) Bind(ready func() error, model dsh.Completer, shell func(ctx context.Context, command string) (string, error)) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.ready = ready
	if rt.workers < 1 {
		rt.workers = 3
	}
	rt.exec = NewExecutor(rt.eng, NewSlots(rt.workers), model)
	rt.exec.Shell = shell
	rt.effects.Handle("launch", 3, func(payload any) error {
		id, _ := payload.(string)
		if rt.ready != nil {
			if err := rt.ready(); err != nil {
				return fmt.Errorf("%w", ErrLater)
			}
		}
		if rt.exec == nil {
			return fmt.Errorf("%w", ErrLater)
		}
		return rt.exec.Launch(id)
	})
	rt.eng.OnLaunch = func(id string) {
		rt.effects.Enqueue("launch", id, id)
	}
	rt.eng.OnInput = func(id, text string) {
		if rt.exec != nil {
			rt.exec.Deliver(id, text)
		}
	}
}

// Start 每两秒跑一次 Pass。重复调用不会再开一条计时器。
func (rt *Runtime) Start() {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	if rt.cancel != nil {
		rt.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	rt.cancel = cancel
	rt.mu.Unlock()
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				rt.Pass(now)
			}
		}
	}()
}

// Stop 停掉两秒轮次并关掉数据库。没有启动过时只关数据库。
func (rt *Runtime) Stop() {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	cancel := rt.cancel
	rt.cancel = nil
	db := rt.db
	rt.db = nil
	rt.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if db != nil {
		_ = db.Close()
	}
}

// Pass 先执行到期副作用和到期评估，再轮询来源，最后落盘。now 传给副作用和来源的时间比较。
func (rt *Runtime) Pass(now time.Time) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	if rt.effects != nil {
		rt.effects.Pump(now)
	}
	if rt.eng != nil {
		rt.eng.EvaluateDue()
	}
	if rt.effects != nil {
		rt.effects.Pump(time.Now())
	}
	rt.health = "ready"
	rt.at = now
	rt.mu.Unlock()
	rt.poll(now)
	rt.flushOutbox()
	rt.save()
}

// SetEvalTimeout 改评估时限。d 小于等于 0 时改回两分钟。改完立刻落盘。
func (rt *Runtime) SetEvalTimeout(d time.Duration) {
	if rt == nil || rt.eng == nil {
		return
	}
	if d <= 0 {
		d = 120 * time.Second
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.eng.mu.Lock()
	rt.eng.policy.EvalTimeout = d
	rt.eng.mu.Unlock()
	rt.saveLocked()
}

// Decide 评估 triggerID。kind 和 reason 写进决定。成功后立刻泵一次派工并落盘。
func (rt *Runtime) Decide(triggerID, kind, reason string) error {
	if rt == nil || rt.eng == nil {
		return errCode("empty")
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if _, err := rt.eng.Decide(triggerID, kind, reason); err != nil {
		return err
	}
	if rt.effects != nil {
		rt.effects.Pump(time.Now())
	}
	rt.saveLocked()
	return nil
}

// SetMode 换成 mode。mode 只能是 off、observe 或 auto。
func (rt *Runtime) SetMode(mode string) error {
	switch mode {
	case "off", "observe", "auto":
	default:
		return errCode("bad_mode")
	}
	return rt.updatePolicy(func(p Policy) Policy {
		p.Mode = mode
		return p
	})
}

// SetPaused 设置暂停。paused 为真时两秒轮次不接受触发，也不恢复工单。
func (rt *Runtime) SetPaused(paused bool) error {
	return rt.updatePolicy(func(p Policy) Policy {
		p.Paused = paused
		return p
	})
}

// SetAdmission 写下三格准入。listedUnaddressed、unlistedAddressed、unlistedUnaddressed 只能是空、explicit、inferred 或 ignore。
func (rt *Runtime) SetAdmission(listedUnaddressed, unlistedAddressed, unlistedUnaddressed string) error {
	for _, cell := range []string{listedUnaddressed, unlistedAddressed, unlistedUnaddressed} {
		switch cell {
		case "", "explicit", "inferred", "ignore":
		default:
			return errCode("bad_admission")
		}
	}
	return rt.updatePolicy(func(p Policy) Policy {
		p.Admission = Admission{
			ListedUnaddressed:   listedUnaddressed,
			UnlistedAddressed:   unlistedAddressed,
			UnlistedUnaddressed: unlistedUnaddressed,
		}
		return p
	})
}

// SetSources 换成 cfg。测试用它指向假的接口，不读环境变量。
func (rt *Runtime) SetSources(cfg SourceConfig) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	rt.sources = cfg
	rt.sourcesSaved = true
	rt.lastGitHub = time.Time{}
	rt.lastFeishu = time.Time{}
	rt.mu.Unlock()
}

// Deliver 把还没交给编排台的评估标成已交付。有新交付时落盘。返回新交付的轮数。
func (rt *Runtime) Deliver() int {
	if rt == nil || rt.eng == nil {
		return 0
	}
	n := rt.eng.DeliverOpen()
	if n > 0 {
		rt.save()
	}
	return n
}

// Act 执行编排台动作。kind 取 attach、reply、note、handback、reevaluate、dismiss、unsubscribe、stop、retry、requestStatus、markUnrelated、follow 或 timeout。
// ticketID、triggerID、text 和 reason 按动作使用。成功后立刻泵一次派工并落盘。
func (rt *Runtime) Act(kind, ticketID, triggerID, text, reason string) error {
	if rt == nil || rt.eng == nil {
		return errCode("empty")
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	var err error
	switch kind {
	case "attach":
		err = rt.eng.Attach(ticketID, triggerID, text)
	case "reply":
		_, err = rt.eng.Reply(ticketID, reason, triggerID, text)
	case "note":
		err = rt.eng.Note(ticketID, text)
	case "handback":
		_, err = rt.eng.Handback(ticketID, triggerID, reason)
	case "reevaluate":
		_, err = rt.eng.Reevaluate(triggerID)
	case "dismiss":
		err = rt.eng.Dismiss(triggerID)
	case "unsubscribe":
		rt.eng.Unsubscribe(text)
	case "stop":
		err = rt.eng.Stop(ticketID, reason)
	case "retry":
		err = rt.eng.Retry(ticketID)
	case "requestStatus":
		_, err = rt.eng.RequestStatus(ticketID)
	case "markUnrelated":
		err = rt.eng.MarkUnrelated(ticketID, text, reason)
	case "follow":
		_, err = rt.eng.FollowForContext(ticketID, text)
	case "timeout":
		sec, conv := strconv.Atoi(strings.TrimSpace(text))
		d := time.Duration(sec) * time.Second
		if conv != nil || d <= 0 {
			d = 120 * time.Second
		}
		rt.eng.mu.Lock()
		rt.eng.policy.EvalTimeout = d
		rt.eng.mu.Unlock()
	case "allowed":
		err = rt.applyAllowed(text)
	case "workers":
		rt.applyWorkers(text)
	case "sources":
		err = rt.applySources(text)
	default:
		err = errCode("bad_action")
	}
	if err != nil {
		return err
	}
	if rt.effects != nil {
		rt.effects.Pump(time.Now())
	}
	rt.saveLocked()
	return nil
}

// View 返回当前快照。引擎还没有记录时，模式和准入仍来自策略。
func (rt *Runtime) View() View {
	if rt == nil || rt.eng == nil {
		return View{Health: "unavailable"}
	}
	rt.mu.Lock()
	sources := rt.sourceList()
	health := rt.health
	at := ""
	if !rt.at.IsZero() {
		at = rt.at.UTC().Format(time.RFC3339)
	}
	workers := rt.workers
	rt.mu.Unlock()
	snap, err := rt.eng.Export()
	if err != nil {
		return View{Health: err.Error(), Sources: sources, At: at, Tickets: []TicketRow{}, Threads: []ThreadRow{}, Queued: []TriggerRow{}, Evals: []EvalRow{}, Decisions: []DecisionRow{}, Executions: []ExecutionRow{}, Problems: []ProblemRow{}, Allowed: []string{}}
	}
	return buildView(snap, sources, health, at, workers)
}

// updatePolicy 用 edit 改策略并落盘。edit 收到当前策略，返回要写回的策略。
func (rt *Runtime) updatePolicy(edit func(Policy) Policy) error {
	if rt == nil || rt.eng == nil {
		return errCode("empty")
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	snap, err := rt.eng.Export()
	if err != nil {
		return err
	}
	from := snap.Policy.Mode
	next := edit(snap.Policy)
	rt.eng.SetPolicy(next)
	if from != next.Mode {
		rt.eng.PolicyChanged(from, next.Mode)
	}
	rt.saveLocked()
	return nil
}

// sourceList 按 github、feishu 的顺序返回状态。还没轮询过的来源是 unconfigured。
func (rt *Runtime) sourceList() []SourceState {
	out := make([]SourceState, 0, 2)
	for _, name := range []string{"github", "feishu"} {
		st, ok := rt.status[name]
		if !ok {
			st = SourceState{Name: name, State: "unconfigured", Detail: "还没轮询"}
		}
		st.Name = name
		if name == "github" {
			st.Repo = rt.sources.GitHubRepo
			st.API = rt.sources.GitHubAPI
			st.HasSecret = strings.TrimSpace(rt.sources.GitHubToken) != ""
		}
		if name == "feishu" {
			st.Chat = rt.sources.FeishuChat
			st.App = rt.sources.FeishuAppID
			st.API = rt.sources.FeishuAPI
			st.HasSecret = strings.TrimSpace(rt.sources.FeishuSecret) != ""
		}
		out = append(out, st)
	}
	return out
}

// applyAllowed 用 text 里逗号分开的名字重写允许发送者。调用时要已持有锁。
func (rt *Runtime) applyAllowed(text string) error {
	snap, err := rt.eng.Export()
	if err != nil {
		return err
	}
	next := snap.Policy
	allowed := map[string]bool{}
	for _, part := range strings.Split(text, ",") {
		name := strings.TrimSpace(part)
		if name != "" {
			allowed[name] = true
		}
	}
	next.Allowed = allowed
	rt.eng.SetPolicy(next)
	return nil
}

// applyWorkers 把普通工人槽改成 text 里的数字。调用时要已持有锁。小于 1 当作 1，大于 32 当作 32。
func (rt *Runtime) applyWorkers(text string) {
	n, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || n < 1 {
		n = 1
	}
	if n > 32 {
		n = 32
	}
	rt.workers = n
	if rt.exec != nil && rt.exec.Slots != nil {
		rt.exec.Slots.SetMax(n)
	}
}

// sourceForm 是编排台提交的来源表。密钥留空表示保持原值，Clear 为真才清掉。
type sourceForm struct {
	GitHubToken       string `json:"githubToken"`
	GitHubRepo        string `json:"githubRepo"`
	GitHubAPI         string `json:"githubApi"`
	FeishuAppID       string `json:"feishuAppId"`
	FeishuSecret      string `json:"feishuSecret"`
	FeishuChat        string `json:"feishuChat"`
	FeishuAPI         string `json:"feishuApi"`
	ClearGitHubToken  bool   `json:"clearGithubToken"`
	ClearFeishuSecret bool   `json:"clearFeishuSecret"`
}

// applySources 按 form 改来源。仓库、应用号和会话空串会清掉。调用时要已持有锁。
func (rt *Runtime) applySources(text string) error {
	var form sourceForm
	if err := json.Unmarshal([]byte(text), &form); err != nil {
		return err
	}
	next := rt.sources
	next.GitHubRepo = strings.TrimSpace(form.GitHubRepo)
	next.FeishuAppID = strings.TrimSpace(form.FeishuAppID)
	next.FeishuChat = strings.TrimSpace(form.FeishuChat)
	if api := strings.TrimSpace(form.GitHubAPI); api != "" {
		next.GitHubAPI = api
	}
	if api := strings.TrimSpace(form.FeishuAPI); api != "" {
		next.FeishuAPI = api
	}
	if form.ClearGitHubToken {
		next.GitHubToken = ""
	} else if token := strings.TrimSpace(form.GitHubToken); token != "" {
		next.GitHubToken = token
	}
	if form.ClearFeishuSecret {
		next.FeishuSecret = ""
	} else if secret := strings.TrimSpace(form.FeishuSecret); secret != "" {
		next.FeishuSecret = secret
	}
	rt.sources = next
	rt.sourcesSaved = true
	rt.lastGitHub = time.Time{}
	rt.lastFeishu = time.Time{}
	return nil
}

// load 读出快照并还原。没有快照行时保持 Open 时的策略。
func (rt *Runtime) load() error {
	if rt.db == nil {
		return nil
	}
	var body string
	err := rt.db.QueryRow(`SELECT body FROM snapshot WHERE id = 1`).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var stored diskBody
	if err := json.Unmarshal([]byte(body), &stored); err != nil {
		return err
	}
	rt.eng.Import(stored.Snapshot)
	if stored.Sources != nil {
		rt.sources = *stored.Sources
		rt.sourcesSaved = true
	}
	if stored.Workers > 0 {
		rt.workers = stored.Workers
	}
	return nil
}

// diskBody 是落盘的快照。Sources 为空指针表示仍用环境变量。Workers 为 0 表示还没改过槽位数。
type diskBody struct {
	Snapshot
	Sources *SourceConfig `json:"sources,omitempty"`
	Workers int           `json:"workers,omitempty"`
}

// save 在锁内把快照写入数据库。
func (rt *Runtime) save() {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.saveLocked()
}

// saveLocked 把快照写入数据库。调用时要已持有锁。没有数据库时什么都不做。
func (rt *Runtime) saveLocked() {
	if rt.db == nil || rt.eng == nil {
		return
	}
	snap, err := rt.eng.Export()
	if err != nil {
		rt.health = err.Error()
		return
	}
	stored := diskBody{Snapshot: snap, Workers: rt.workers}
	if rt.sourcesSaved {
		cfg := rt.sources
		stored.Sources = &cfg
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		rt.health = err.Error()
		return
	}
	_, err = rt.db.Exec(`INSERT INTO snapshot(id, body) VALUES(1, ?) ON CONFLICT(id) DO UPDATE SET body = excluded.body`, string(raw))
	if err != nil {
		rt.health = err.Error()
	}
}
