package app

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/eval"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/journal"
	"github.com/Shenchangxin/yoyo/internal/kernel"
	"github.com/Shenchangxin/yoyo/internal/plugin/mcp"
	wasm "github.com/Shenchangxin/yoyo/internal/plugin/wasm"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/update"
	"github.com/Shenchangxin/yoyo/internal/vault"
	"github.com/Shenchangxin/yoyo/internal/version"
)

type MCPServerConfig struct {
	Name    string   `yaml:"name" json:"name"`
	Command string   `yaml:"command" json:"command"`
	Args    []string `yaml:"args" json:"args"`
}

type Config struct {
	Provider                string            `yaml:"provider" json:"provider"`
	Model                   string            `yaml:"model" json:"model"`
	BaseURL                 string            `yaml:"base_url" json:"base_url"`
	Workspace               string            `yaml:"workspace" json:"workspace"`
	AutoAllow               bool              `yaml:"auto_allow" json:"auto_allow"`
	MaxBudgetUSD            float64           `yaml:"max_budget_usd" json:"max_budget_usd"`
	USDPerMTok              float64           `yaml:"usd_per_mtok" json:"usd_per_mtok"`
	Models                  []string          `yaml:"models" json:"models"`
	MCP                     []MCPServerConfig `yaml:"mcp" json:"mcp"`
	CloseToTray             bool              `yaml:"close_to_tray" json:"close_to_tray"`
	UpdateURL               string            `yaml:"update_url" json:"update_url"`
	Locale                  string            `yaml:"locale" json:"locale"`
	Keymap                  map[string]string `yaml:"keymap" json:"keymap"`
	AlwaysOnTop             bool              `yaml:"always_on_top" json:"always_on_top"`
	StartAtLogin            bool              `yaml:"start_at_login" json:"start_at_login"`
	NotificationsEnabled    bool              `yaml:"notifications_enabled" json:"notifications_enabled"`
	NotifyWhenUnfocusedOnly bool              `yaml:"notify_when_unfocused_only" json:"notify_when_unfocused_only"`
	UIScale                 float64           `yaml:"ui_scale" json:"ui_scale"`
	UpdateChannel           string            `yaml:"update_channel" json:"update_channel"`
	Theme                   string            `yaml:"theme" json:"theme"`
}

type App struct {
	Home         *home.Dir
	Kernel       *kernel.Context
	CAS          *artifact.Store
	Refs         *artifact.Refs
	Traces       *trace.Store
	Caps         *capability.Broker
	Gate         *capability.Gate
	Hub          *Hub
	Vault        *vault.Store
	Journal      *journal.Log
	WASM         *wasm.Host
	MCP          *mcp.Host
	Eval         *eval.Engine
	Evolve       *evolve.Engine
	Archive      *evolve.Archive
	Config       Config
	BundledEvals string

	mu   sync.Mutex
	runs map[string]context.CancelFunc

	shapeMu   sync.Mutex
	lastShape map[string]runtime.ShapeReport
	usageMu   sync.Mutex
	lastUsage map[string]any

	queueMu sync.Mutex
	queue   map[string][]QueuedTurn
	steers  map[string][]string
}

func Open(root, bundledEvals string) (*App, error) {
	h, err := home.Open(root)
	if err != nil {
		return nil, err
	}
	j, err := journal.Open(h.Journal())
	if err != nil {
		return nil, err
	}
	arch, err := evolve.OpenArchive(h.Archive())
	if err != nil {
		return nil, err
	}
	cfg := Config{
		Provider:             "openai",
		Model:                "gpt-4.1-mini",
		BaseURL:              "https://api.openai.com/v1",
		Workspace:            "",
		AutoAllow:            false,
		NotificationsEnabled: true,
		UIScale:              1,
		UpdateChannel:        "nightly",
		Theme:                "system",
	}
	if b, err := os.ReadFile(h.Config()); err == nil {
		raw := map[string]any{}
		_ = yaml.Unmarshal(b, &raw)
		_ = yaml.Unmarshal(b, &cfg)
		if _, ok := raw["notifications_enabled"]; !ok {
			cfg.NotificationsEnabled = true
		}
		if cfg.UIScale <= 0 {
			cfg.UIScale = 1
		}
		if cfg.UpdateChannel == "" {
			cfg.UpdateChannel = "nightly"
		}
		if cfg.Theme == "" {
			cfg.Theme = "system"
		}
	}
	filledDefault := applyDefaultWorkspace(h, &cfg)
	k := kernel.New()
	allow := []capability.Level{capability.ReadWorkspace, capability.WriteWorkspace}
	if cfg.AutoAllow {
		allow = append(allow, capability.Shell)
	}
	gate := capability.NewGate()
	a := &App{
		Home:         h,
		Kernel:       k,
		CAS:          artifact.NewStore(h.CAS()),
		Refs:         artifact.NewRefs(h.Refs()),
		Traces:       trace.NewStore(h.Sessions()),
		Gate:         gate,
		Hub:          NewHub(),
		Vault:        vault.New(h.Root),
		Journal:      j,
		WASM:         wasm.NewHost(),
		MCP:          mcp.NewHost(),
		Eval:         eval.NewEngine(bundledEvals),
		Archive:      arch,
		Config:       cfg,
		BundledEvals: bundledEvals,
		runs:         map[string]context.CancelFunc{},
		lastShape:    map[string]runtime.ShapeReport{},
	}
	a.initQueue()
	a.loadKeymapFile()
	for _, srv := range cfg.MCP {
		if srv.Name == "" || srv.Command == "" {
			continue
		}
		_ = a.MCP.Start(srv.Name, srv.Command, srv.Args)
	}
	a.Caps = capability.NewBroker(capability.AutoPolicy{Allow: allow}, a.approve)
	a.Gate.SetOnOffer(func(o capability.Offer) {
		a.Hub.Publish(trace.Event{
			Type:    trace.TypeApproval,
			Source:  "gate",
			Payload: map[string]any{"id": o.ID, "action": o.Request.Action, "command": o.Request.Command, "path": o.Request.Path},
		})
	})
	a.Evolve = &evolve.Engine{
		CAS:     a.CAS,
		Refs:    a.Refs,
		Eval:    a.Eval,
		Journal: a.Journal,
		Archive: a.Archive,
		Kernel:  a.Kernel,
	}
	if _, err := a.Kernel.Plugin("tcb", func(c *kernel.Context) error {
		if err := c.Provide("cas", a.CAS); err != nil {
			return err
		}
		if err := c.Provide("journal", a.Journal); err != nil {
			return err
		}
		if err := c.Provide("eval", a.Eval); err != nil {
			return err
		}
		if err := c.Provide("vault", a.Vault); err != nil {
			return err
		}
		return c.Provide("capability", a.Caps)
	}); err != nil {
		return nil, err
	}
	if err := a.Seed(); err != nil {
		return nil, err
	}
	if filledDefault {
		_ = a.SaveConfig()
	}
	_, _ = a.Kernel.Plugin("hooks", func(c *kernel.Context) error {
		return c.Effect(func() (func() error, error) {
			off := c.Events().On(runtimeHookPreTool(), func(payload any) (any, error) {
				return nil, nil
			})
			return func() error { off(); return nil }, nil
		})
	})
	return a, nil
}

func runtimeHookPreTool() string { return "agent.pre_tool" }

func (a *App) approve(ctx context.Context, req capability.Request) (capability.Decision, error) {
	if a.Config.AutoAllow && !req.ForceAsk {
		return capability.Always, nil
	}
	return a.Gate.Ask(ctx, req)
}

func (a *App) Interrupt(sessionID string) error {
	a.mu.Lock()
	cancel := a.runs[sessionID]
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (a *App) RememberShape(sessionID string, r runtime.ShapeReport) {
	a.shapeMu.Lock()
	a.lastShape[sessionID] = r
	a.shapeMu.Unlock()
}

func (a *App) ContextUsage(sessionID string) runtime.ShapeReport {
	a.shapeMu.Lock()
	defer a.shapeMu.Unlock()
	return a.lastShape[sessionID]
}

func (a *App) RememberUsage(snap map[string]any) {
	a.usageMu.Lock()
	a.lastUsage = snap
	a.usageMu.Unlock()
}

func (a *App) LastUsage() map[string]any {
	a.usageMu.Lock()
	defer a.usageMu.Unlock()
	if a.lastUsage == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	for k, v := range a.lastUsage {
		out[k] = v
	}
	return out
}

func (a *App) Health() map[string]any {
	return map[string]any{
		"ok":              true,
		"harness":         a.ActiveHash(),
		"model":           a.Config.Model,
		"version":         version.Version,
		"usage":           a.LastUsage(),
		"budget_usd":      a.Config.MaxBudgetUSD,
		"update":          update.Current(""),
		"isolated":        isolated(),
		"models":          a.Config.Models,
		"workspace_ready": WorkspaceReady(a.Config.Workspace),
		"vault":           a.Vault.Status(),
		"update_channel":  a.Config.UpdateChannel,
	}
}

func (a *App) ResolveApproval(id, decision string) error {
	d := capability.Deny
	switch decision {
	case "always":
		d = capability.Always
	case "session":
		d = capability.Session
	case "once", "allow":
		d = capability.Once
	}
	return a.Gate.Resolve(id, d)
}

func (a *App) SaveConfig() error {
	b, err := yaml.Marshal(a.Config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(a.Home.Config(), b, 0o644); err != nil {
		return err
	}
	a.writeKeymapFile()
	return nil
}

func (a *App) Close() error {
	_ = a.MCP.Close()
	_ = a.WASM.Close()
	return a.Kernel.Dispose()
}

func (a *App) Workspace() string {
	if a.Config.Workspace != "" {
		if p, err := filepath.Abs(a.Config.Workspace); err == nil {
			return p
		}
	}
	if a.Home != nil {
		p := a.Home.Workspace()
		_ = os.MkdirAll(p, 0o755)
		return p
	}
	wd, _ := os.Getwd()
	return wd
}

func (a *App) StagingPath() string {
	return filepath.Join(a.Home.Updates(), update.StagingName())
}

func (a *App) ApplyStagedUpdate(exe string) error {
	return update.ApplyStaged(exe, a.StagingPath())
}
