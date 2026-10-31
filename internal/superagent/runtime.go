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
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"

	_ "modernc.org/sqlite"
)

// Runtime 把引擎、副作用、来源轮询和快照放在一起。两秒轮次调用 Pass。
type Runtime struct {
	mu         sync.Mutex
	eng        *Engine
	effects    *Queue
	exec       *Executor
	ready      func() error
	db         *sql.DB
	cancel     context.CancelFunc
	client     *http.Client
	sources    SourceConfig
	status     map[string]SourceState
	health     string
	at         time.Time
	lastGitHub time.Time
	lastFeishu time.Time
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
	rt.exec = NewExecutor(rt.eng, NewSlots(3), model)
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
	rt.save()
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
	rt.lastGitHub = time.Time{}
	rt.lastFeishu = time.Time{}
	rt.mu.Unlock()
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
	rt.mu.Unlock()
	snap, err := rt.eng.Export()
	if err != nil {
		return View{Health: err.Error(), Sources: sources, At: at}
	}
	return buildView(snap, sources, health, at)
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
	rt.eng.SetPolicy(edit(snap.Policy))
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
		out = append(out, st)
	}
	return out
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
	var snap Snapshot
	if err := json.Unmarshal([]byte(body), &snap); err != nil {
		return err
	}
	rt.eng.Import(snap)
	return nil
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
	raw, err := json.Marshal(snap)
	if err != nil {
		rt.health = err.Error()
		return
	}
	_, err = rt.db.Exec(`INSERT INTO snapshot(id, body) VALUES(1, ?) ON CONFLICT(id) DO UPDATE SET body = excluded.body`, string(raw))
	if err != nil {
		rt.health = err.Error()
	}
}
