package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Shenchangxin/yoyo/internal/api"
	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// Service is the Wails-bound facade. When RPC is set the GUI talks to a
// worker process (YOYO_WORKER=1 / yoyo serve --stdio) so a wedged loop
// cannot stall the window. App is used in-process otherwise.
type Service struct {
	App        *app.App
	RPC        *api.LineClient
	cmd        *exec.Cmd
	gui        *application.App
	win        application.Window
	tray       *application.SystemTray
	menuLocale string
	quitting   atomic.Bool
}

func NewService(a *app.App) *Service { return &Service{App: a} }

func decode[T any](v any, err error) (T, error) {
	var zero T
	if err != nil {
		return zero, err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return zero, err
	}
	var out T
	return out, json.Unmarshal(b, &out)
}

func (s *Service) call(method string, params any) (any, error) {
	if s.RPC == nil {
		return nil, fmt.Errorf("no rpc")
	}
	return s.RPC.Call(method, params)
}

func (s *Service) Health() map[string]any {
	if s.RPC != nil {
		v, err := s.call("health", nil)
		m, _ := decode[map[string]any](v, err)
		if m == nil {
			m = map[string]any{}
		}
		m["isolated"] = true
		return m
	}
	h := s.App.Health()
	h["isolated"] = false
	return h
}

func (s *Service) GetConfig() app.Config {
	if s.RPC != nil {
		v, err := s.call("config.get", nil)
		c, _ := decode[app.Config](v, err)
		return c
	}
	return s.App.Config
}

func (s *Service) SetConfig(cfg app.Config) error {
	if s.RPC != nil {
		_, err := s.call("config.set", cfg)
		if err == nil {
			s.applyDesktop(cfg)
		}
		return err
	}
	s.App.Config = cfg
	if err := s.App.SaveConfig(); err != nil {
		return err
	}
	s.applyDesktop(cfg)
	return nil
}

func (s *Service) SetLocale(locale string) error {
	cfg := s.GetConfig()
	cfg.Locale = locale
	return s.SetConfig(cfg)
}

func (s *Service) BeginQuit() {
	s.quitting.Store(true)
}

func (s *Service) Quitting() bool {
	return s.quitting.Load()
}

func (s *Service) applyDesktop(cfg app.Config) {
	s.menuLocale = cfg.Locale
	if s.win != nil {
		s.win.SetAlwaysOnTop(cfg.AlwaysOnTop)
	}
	_ = SetStartAtLogin(cfg.StartAtLogin)
	s.RebuildMenus()
}

func (s *Service) SetAPIKey(value string) {
	if s.RPC != nil {
		_, _ = s.call("key.set", map[string]any{"value": value})
		return
	}
	s.App.Vault.Set("default", value)
}

func (s *Service) CreateSession(workspace string) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("thread.start", map[string]any{"workspace": workspace}))
	} else {
		m, err = s.App.NewSession(workspace)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

func (s *Service) ListSessions() ([]app.SessionMeta, error) {
	if s.RPC != nil {
		return decode[[]app.SessionMeta](s.call("thread.list", nil))
	}
	return s.App.ListSessions()
}

func (s *Service) ForkSession(id string) (app.SessionMeta, error) {
	var m app.SessionMeta
	var err error
	if s.RPC != nil {
		m, err = decode[app.SessionMeta](s.call("thread.fork", map[string]any{"session": id}))
	} else {
		m, err = s.App.ForkSession(id)
	}
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

func (s *Service) RenameSession(id, title string) (app.SessionMeta, error) {
	var err error
	if s.RPC != nil {
		_, err = s.call("thread.rename", map[string]any{"session": id, "title": title})
		if err != nil {
			return app.SessionMeta{}, err
		}
		list, e2 := decode[[]app.SessionMeta](s.call("thread.list", nil))
		if e2 != nil {
			return app.SessionMeta{ID: id, Title: title}, nil
		}
		for _, m := range list {
			if m.ID == id {
				s.emitSessions(m)
				return m, nil
			}
		}
		s.emitSessions(id)
		return app.SessionMeta{ID: id, Title: title}, nil
	}
	err = s.App.RenameSession(id, title)
	if err != nil {
		return app.SessionMeta{}, err
	}
	m, err := s.App.GetSession(id)
	if err == nil {
		s.emitSessions(m)
	}
	return m, err
}

func (s *Service) Playbook() (any, error) {
	if s.RPC != nil {
		return s.call("playbook.get", nil)
	}
	return s.App.Playbook()
}

func (s *Service) RatePlaybook(id string, helpful bool) (any, error) {
	if s.RPC != nil {
		return s.call("playbook.rate", map[string]any{"id": id, "helpful": helpful})
	}
	return s.App.RatePlaybook(id, helpful)
}

func (s *Service) WorkspaceDiff(workspace string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("workspace.diff", map[string]any{"workspace": workspace})
		if err != nil {
			return "", err
		}
		m, _ := v.(map[string]any)
		if m != nil {
			if d, ok := m["diff"].(string); ok {
				return d, nil
			}
		}
		return fmt.Sprint(v), nil
	}
	return s.App.WorkspaceDiff(workspace)
}

func (s *Service) WorkspaceHunks(workspace string) (any, error) {
	if s.RPC != nil {
		return s.call("workspace.hunks", map[string]any{"workspace": workspace})
	}
	return s.App.WorkspaceHunks(workspace)
}

func (s *Service) ApplyHunks(workspace string, ids []string) error {
	if s.RPC != nil {
		_, err := s.call("workspace.apply_hunks", map[string]any{"workspace": workspace, "ids": ids})
		return err
	}
	return s.App.ApplyWorkspaceHunks(workspace, ids)
}

func (s *Service) ReverseHunks(workspace string, ids []string, snapshot string) error {
	if s.RPC != nil {
		_, err := s.call("workspace.reverse_hunks", map[string]any{"workspace": workspace, "ids": ids, "snapshot": snapshot})
		return err
	}
	return s.App.ReverseWorkspaceHunks(workspace, ids, snapshot)
}

func (s *Service) RunEvalSafety() (any, error) {
	if s.RPC != nil {
		return s.call("eval.run_safety", nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return s.App.RunEvalOpts(ctx, nil, true)
}

func (s *Service) RunEvalTB() (any, error) {
	if s.RPC != nil {
		return s.call("eval.run_tb", nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return s.App.RunEvalTB(ctx, nil)
}

func (s *Service) Trajectory(id string) ([]trace.Event, error) {
	if s.RPC != nil {
		return decode[[]trace.Event](s.call("trajectory.get", map[string]any{"session": id}))
	}
	return s.App.Trajectory(id)
}

func (s *Service) SessionTrace(id string) (any, error) {
	if s.RPC != nil {
		return s.call("trace.get", map[string]any{"session": id})
	}
	return s.App.SessionTrace(id)
}

func (s *Service) SpillBlob(sessionID, blobID string) (any, error) {
	if s.RPC != nil {
		return s.call("spill.get", map[string]any{"session": sessionID, "id": blobID})
	}
	return s.App.SpillBlob(sessionID, blobID)
}

func (s *Service) Send(sessionID, text string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("turn.start", map[string]any{"session": sessionID, "text": text, "wait": true})
		if err != nil {
			return "", err
		}
		m, _ := v.(map[string]any)
		if m != nil {
			if t, ok := m["text"].(string); ok {
				return t, nil
			}
		}
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return s.App.Send(ctx, sessionID, text, nil, s.App.Hub.Publish)
}

func (s *Service) StartSend(sessionID, text string, plan bool) error {
	return s.StartSendOpts(sessionID, text, plan, nil)
}

func (s *Service) StartSendOpts(sessionID, text string, plan bool, atts []app.Attachment) error {
	if s.RPC != nil {
		_, err := s.call("turn.start", map[string]any{"session": sessionID, "text": text, "plan": plan, "wait": false, "attachments": atts})
		return err
	}
	err := s.App.StartSendOpts(sessionID, text, plan, atts)
	if errors.Is(err, app.ErrQueued) {
		return nil
	}
	return err
}

func (s *Service) RetrySession(sessionID string) error {
	if s.RPC != nil {
		_, err := s.call("turn.retry", map[string]any{"session": sessionID})
		return err
	}
	return s.App.RetrySession(sessionID)
}

func (s *Service) Interrupt(sessionID string) error {
	if s.RPC != nil {
		_, err := s.call("turn.interrupt", map[string]any{"session": sessionID})
		return err
	}
	return s.App.Interrupt(sessionID)
}

func (s *Service) Running(sessionID string) bool {
	if s.RPC != nil {
		v, err := s.call("turn.running", map[string]any{"session": sessionID})
		m, _ := decode[map[string]any](v, err)
		b, _ := m["running"].(bool)
		return b
	}
	return s.App.Running(sessionID)
}

func (s *Service) RunningIDs() []string {
	if s.RPC != nil {
		v, err := s.call("turn.running_ids", nil)
		ids, _ := decode[[]string](v, err)
		if ids == nil {
			return []string{}
		}
		return ids
	}
	return s.App.RunningIDs()
}

func (s *Service) ContextUsage(sessionID string) any {
	if s.RPC != nil {
		v, _ := s.call("context.get", map[string]any{"session": sessionID})
		return v
	}
	return s.App.ContextUsage(sessionID)
}

func (s *Service) PendingApprovals() any {
	if s.RPC != nil {
		v, _ := s.call("approvals.list", nil)
		return v
	}
	return s.App.Gate.Pending()
}

func (s *Service) ResolveApproval(id, decision string) error {
	if s.RPC != nil {
		_, err := s.call("approvals.resolve", map[string]any{"id": id, "decision": decision})
		return err
	}
	return s.App.ResolveApproval(id, decision)
}

func (s *Service) StartMCP(name, command string, args []string) error {
	if s.RPC != nil {
		_, err := s.call("mcp.start", map[string]any{"name": name, "command": command, "args": args})
		return err
	}
	return s.App.StartMCP(name, command, args)
}

func (s *Service) Harness() (map[string]any, error) {
	if s.RPC != nil {
		return decode[map[string]any](s.call("harness.get", nil))
	}
	refs, err := s.App.ListHarnesses()
	if err != nil {
		return nil, err
	}
	snap, _ := s.App.LoadSnapshot(s.App.ActiveHash())
	return map[string]any{"active": s.App.ActiveHash(), "refs": refs, "snapshot": snap}, nil
}

func (s *Service) Checkout(hash string) error {
	if s.RPC != nil {
		_, err := s.call("harness.checkout", map[string]any{"hash": hash})
		return err
	}
	return s.App.Checkout(hash)
}

func (s *Service) CheckoutL3(hash string) error {
	if s.RPC != nil {
		_, err := s.call("harness.checkout", map[string]any{"hash": hash, "l3": true})
		return err
	}
	return s.App.CheckoutOpts(hash, app.CheckoutOpts{ConfirmL3: true})
}

func (s *Service) Rollback() error {
	if s.RPC != nil {
		_, err := s.call("harness.rollback", nil)
		return err
	}
	return s.App.Rollback()
}

func (s *Service) Diff(a, b string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("harness.diff", map[string]any{"a": a, "b": b})
		m, err2 := decode[map[string]any](v, err)
		if err2 != nil {
			return "", err2
		}
		t, _ := m["text"].(string)
		return t, nil
	}
	return s.App.Diff(a, b)
}

func (s *Service) DiffDetail(a, b string) (map[string]any, error) {
	if s.RPC != nil {
		return decode[map[string]any](s.call("harness.diff", map[string]any{"a": a, "b": b}))
	}
	return s.App.DiffDetail(a, b)
}

func (s *Service) RunEval() (any, error) {
	if s.RPC != nil {
		return s.call("eval.run", nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return s.App.RunEval(ctx, nil)
}

func (s *Service) BestOfN(n int) (app.BestOfNReport, error) {
	if s.RPC != nil {
		return decode[app.BestOfNReport](s.call("eval.best_of_n", map[string]any{"n": n}))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	return s.App.BestOfN(ctx, n, nil)
}

func (s *Service) BestOfModels(models []string) (app.BestOfNReport, error) {
	if s.RPC != nil {
		return decode[app.BestOfNReport](s.call("eval.best_of_models", map[string]any{"models": models}))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	return s.App.BestOfModels(ctx, models, nil)
}

func (s *Service) Evolve() (evolve.CycleResult, error) {
	return s.EvolveK(3)
}

func (s *Service) EvolveK(k int) (evolve.CycleResult, error) {
	if s.RPC != nil {
		return decode[evolve.CycleResult](s.call("evolve.run", map[string]any{"k": k}))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	return s.App.EvolveK(ctx, nil, nil, k)
}

func (s *Service) Archive() []evolve.Node {
	if s.RPC != nil {
		v, err := s.call("archive.list", nil)
		n, _ := decode[[]evolve.Node](v, err)
		return n
	}
	return s.App.Archive.List()
}

func (s *Service) Plugins() map[string]any {
	if s.RPC != nil {
		v, err := s.call("plugins.list", nil)
		m, _ := decode[map[string]any](v, err)
		return m
	}
	return map[string]any{"fibers": s.App.Kernel.Fibers(), "wasm": s.App.WASM.List(), "mcp": s.App.MCP.Info(), "tools": s.App.MCP.Tools()}
}

func (s *Service) UnloadFiber(name string) error {
	if s.RPC != nil {
		_, err := s.call("fiber.unload", map[string]any{"name": name})
		return err
	}
	if f := s.App.Kernel.Fiber(name); f != nil {
		return f.Dispose()
	}
	return s.App.WASM.Unload(name)
}

func (s *Service) ApplyUpdate() error {
	if s.RPC != nil {
		_, err := s.call("update.apply", map[string]any{"exe": ""})
		return err
	}
	return s.App.ApplyStagedUpdate("")
}

func (s *Service) Close() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}
