package app

import (
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	"github.com/Shenchangxin/yoyo/internal/session"
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
	Threads      *session.Manager
	Vault        *vault.Store
	Journal      *journal.Log
	WASM         *wasm.Host
	MCP          *mcp.Host
	Eval         *eval.Engine
	Evolve       *evolve.Engine
	Archive      *evolve.Archive
	Config       Config
	BundledEvals string
	wasmPub      ed25519.PublicKey

	mu   sync.Mutex
	runs map[string]context.CancelFunc

	shapeMu   sync.Mutex
	lastShape map[string]runtime.ShapeReport
	usageMu   sync.Mutex
	lastUsage map[string]any

	queueMu sync.Mutex
	queue   map[string][]QueuedTurn
	steers  map[string][]string

	askMu  sync.Mutex
	askAns map[string]string
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
		Threads:      session.NewManager(h.Sessions()),
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
		askAns:       map[string]string{},
	}
	a.Hub.Overflow = filepath.Join(h.Sessions(), "_overflow")
	a.MCP.Roots = []string{a.Workspace()}
	a.initQueue()
	a.loadKeymapFile()
	for _, srv := range cfg.MCP {
		if srv.Name == "" || srv.Command == "" {
			continue
		}
		_ = a.MCP.Start(srv.Name, srv.Command, srv.Args)
	}
	a.Caps = capability.NewBroker(capability.AutoPolicy{Allow: allow}, a.approve)
	a.restoreRuns()
	a.Gate.SetOnOffer(func(o capability.Offer) {
		ev := trace.Event{
			TS:        time.Now().UTC(),
			Type:      trace.TypeApproval,
			Source:    "gate",
			SessionID: o.Request.SessionID,
			Payload: map[string]any{
				"id": o.ID, "action": o.Request.Action, "command": o.Request.Command,
				"path": o.Request.Path, "level": string(o.Request.Level),
			},
		}
		if a.Traces != nil && o.Request.SessionID != "" {
			_ = a.Traces.Append(ev)
		}
		a.Hub.Publish(ev)
		if a.Threads != nil && o.Request.SessionID != "" && a.Gate != nil {
			var offers []session.PendingApproval
			for _, p := range a.Gate.Pending() {
				if p.Request.SessionID != o.Request.SessionID {
					continue
				}
				offers = append(offers, session.PendingApproval{
					ID: p.ID, Action: p.Request.Action, Level: string(p.Request.Level),
					Path: p.Request.Path, Command: p.Request.Command,
				})
			}
			a.Threads.SetApprovals(o.Request.SessionID, offers)
		}
	})
	a.Evolve = &evolve.Engine{
		CAS:       a.CAS,
		Refs:      a.Refs,
		Eval:      a.Eval,
		Journal:   a.Journal,
		Archive:   a.Archive,
		Kernel:    a.Kernel,
		TrialRoot: h.EvalRuns(),
	}
	if k := os.Getenv("YOYO_WASM_PUBKEY"); k != "" {
		if pub, err := wasm.ParseKey(k); err == nil {
			a.wasmPub = pub
		}
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
	dec, err := a.Gate.Ask(ctx, req)
	if err != nil {
		return dec, err
	}
	if dec == capability.Session && req.SessionID != "" && a.Threads != nil {
		caps := append(a.Threads.Caps(req.SessionID), string(req.Level))
		a.Threads.SetCaps(req.SessionID, caps)
		a.Threads.SetApprovals(req.SessionID, nil)
	}
	return dec, nil
}

func (a *App) Interrupt(sessionID string) error {
	if a.Threads != nil {
		a.Threads.Interrupt(sessionID)
	}
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
	sessionID = strings.TrimSpace(sessionID)
	if sessionID != "" && a.Running(sessionID) {
		a.shapeMu.Lock()
		live := a.lastShape[sessionID]
		a.shapeMu.Unlock()
		if live.Tokens > 0 || live.PrefixTokens > 0 || live.Window > 0 {
			return live
		}
	}
	return a.measureSessionContext(sessionID)
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
	return a.ResolveApprovalAnswer(id, decision, "")
}

func (a *App) ResolveApprovalAnswer(id, decision, answer string) error {
	if answer != "" {
		a.askMu.Lock()
		if a.askAns == nil {
			a.askAns = map[string]string{}
		}
		a.askAns[id] = answer
		a.askMu.Unlock()
	}
	d := capability.Deny
	switch decision {
	case "always":
		d = capability.Always
	case "session":
		d = capability.Session
	case "once", "allow":
		d = capability.Once
	}
	sessionID := ""
	if a.Gate != nil {
		for _, o := range a.Gate.Pending() {
			if o.ID == id {
				sessionID = o.Request.SessionID
				break
			}
		}
	}
	if err := a.Gate.Resolve(id, d); err != nil {
		return err
	}
	if sessionID != "" {
		ev := trace.Event{
			TS:        time.Now().UTC(),
			Type:      trace.TypeApproval,
			Source:    "user",
			SessionID: sessionID,
			Payload:   map[string]any{"id": id, "decision": decision},
		}
		if a.Traces != nil {
			_ = a.Traces.Append(ev)
		}
		if a.Hub != nil {
			a.Hub.Publish(ev)
		}
	}
	return nil
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

func (a *App) restoreRuns() {
	if a.Threads == nil || a.Home == nil {
		return
	}
	entries, err := os.ReadDir(a.Home.Sessions())
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".run.json") {
			continue
		}
		id := strings.TrimSuffix(name, ".run.json")
		st := a.Threads.Load(id)
		if len(st.Queue) > 0 {
			a.queueMu.Lock()
			for _, t := range st.Queue {
				atts := make([]Attachment, 0, len(t.Attachments))
				for _, x := range t.Attachments {
					atts = append(atts, Attachment{Path: x.Path, Name: x.Name, MIME: x.MIME, DataB64: x.DataB64})
				}
				a.queue[id] = append(a.queue[id], QueuedTurn{Text: t.Text, Plan: t.Plan, Attachments: atts})
			}
			a.queueMu.Unlock()
		}
		if a.Caps != nil && len(st.SessionCaps) > 0 {
			var lv []capability.Level
			for _, c := range st.SessionCaps {
				lv = append(lv, capability.Level(c))
			}
			a.Caps.RestoreSession(id, lv)
		}
		if len(st.Steers) > 0 {
			a.queueMu.Lock()
			a.steers[id] = append(a.steers[id], st.Steers...)
			a.queueMu.Unlock()
		}
	}
}
