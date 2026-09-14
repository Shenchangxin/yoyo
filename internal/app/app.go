package app

import (
	"os"
	"path/filepath"

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
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/vault"
)

type Config struct {
	Provider  string `yaml:"provider" json:"provider"`
	Model     string `yaml:"model" json:"model"`
	BaseURL   string `yaml:"base_url" json:"base_url"`
	Workspace string `yaml:"workspace" json:"workspace"`
	AutoAllow bool   `yaml:"auto_allow" json:"auto_allow"`
}

type App struct {
	Home         *home.Dir
	Kernel       *kernel.Context
	CAS          *artifact.Store
	Refs         *artifact.Refs
	Traces       *trace.Store
	Caps         *capability.Broker
	Vault        *vault.Store
	Journal      *journal.Log
	WASM         *wasm.Host
	MCP          *mcp.Host
	Eval         *eval.Engine
	Evolve       *evolve.Engine
	Archive      *evolve.Archive
	Config       Config
	BundledEvals string
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
		Provider:  "openai",
		Model:     "gpt-4.1-mini",
		BaseURL:   "https://api.openai.com/v1",
		Workspace: ".",
		AutoAllow: true,
	}
	if b, err := os.ReadFile(h.Config()); err == nil {
		_ = yaml.Unmarshal(b, &cfg)
	}
	k := kernel.New()
	allow := []capability.Level{capability.ReadWorkspace, capability.WriteWorkspace}
	if cfg.AutoAllow {
		allow = append(allow, capability.Shell)
	}
	a := &App{
		Home:         h,
		Kernel:       k,
		CAS:          artifact.NewStore(h.CAS()),
		Refs:         artifact.NewRefs(h.Refs()),
		Traces:       trace.NewStore(h.Sessions()),
		Caps:         capability.NewBroker(capability.AutoPolicy{Allow: allow}, autoApprover),
		Vault:        vault.New(),
		Journal:      j,
		WASM:         wasm.NewHost(),
		MCP:          mcp.NewHost(),
		Eval:         eval.NewEngine(bundledEvals),
		Archive:      arch,
		Config:       cfg,
		BundledEvals: bundledEvals,
	}
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
	return a, nil
}

func autoApprover(capability.Request) (capability.Decision, error) {
	return capability.Always, nil
}

func (a *App) SaveConfig() error {
	b, err := yaml.Marshal(a.Config)
	if err != nil {
		return err
	}
	return os.WriteFile(a.Home.Config(), b, 0o644)
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
	wd, _ := os.Getwd()
	return wd
}
