package desktop

import (
	"context"
	"time"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// Service is the Wails-bound facade.
type Service struct {
	App *app.App
}

func NewService(a *app.App) *Service { return &Service{App: a} }

func (s *Service) Health() map[string]any {
	return map[string]any{"ok": true, "harness": s.App.ActiveHash(), "model": s.App.Config.Model}
}

func (s *Service) GetConfig() app.Config { return s.App.Config }

func (s *Service) SetConfig(cfg app.Config) error {
	s.App.Config = cfg
	return s.App.SaveConfig()
}

func (s *Service) SetAPIKey(value string) {
	s.App.Vault.Set("default", value)
}

func (s *Service) CreateSession(workspace string) (app.SessionMeta, error) {
	return s.App.NewSession(workspace)
}

func (s *Service) ListSessions() ([]app.SessionMeta, error) {
	return s.App.ListSessions()
}

func (s *Service) Trajectory(id string) ([]trace.Event, error) {
	return s.App.Trajectory(id)
}

func (s *Service) Send(sessionID, text string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return s.App.Send(ctx, sessionID, text, nil, nil)
}

func (s *Service) Harness() (map[string]any, error) {
	refs, err := s.App.ListHarnesses()
	if err != nil {
		return nil, err
	}
	snap, _ := s.App.LoadSnapshot(s.App.ActiveHash())
	return map[string]any{"active": s.App.ActiveHash(), "refs": refs, "snapshot": snap}, nil
}

func (s *Service) Checkout(hash string) error { return s.App.Checkout(hash) }

func (s *Service) Rollback() error { return s.App.Rollback() }

func (s *Service) Diff(a, b string) (string, error) { return s.App.Diff(a, b) }

func (s *Service) RunEval() (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return s.App.RunEval(ctx, nil)
}

func (s *Service) Evolve() (evolve.CycleResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	return s.App.EvolveOnce(ctx, nil, nil)
}

func (s *Service) Archive() []evolve.Node { return s.App.Archive.List() }

func (s *Service) Plugins() map[string]any {
	return map[string]any{"fibers": s.App.Kernel.Fibers(), "wasm": s.App.WASM.List(), "mcp": s.App.MCP.List()}
}

func (s *Service) UnloadFiber(name string) error {
	if f := s.App.Kernel.Fiber(name); f != nil {
		return f.Dispose()
	}
	return s.App.WASM.Unload(name)
}
