package artifact

import "time"

// Kind identifies a versioned harness object stored in CAS.
type Kind string

const (
	KindPromptFragment  Kind = "prompt_fragment"
	KindSkill           Kind = "skill"
	KindPlaybook        Kind = "playbook"
	KindToolSpec        Kind = "tool_spec"
	KindLoopPreset      Kind = "loop_preset"
	KindPolicyPack      Kind = "policy_pack"
	KindEvalSuite       Kind = "eval_suite"
	KindModelBinding    Kind = "model_binding"
	KindHarnessSnapshot Kind = "harness_snapshot"
	KindWASMPlugin      Kind = "wasm_plugin"
)

// Envelope is the CAS-stored wrapper around a typed payload.
type Envelope struct {
	Kind      Kind      `json:"kind"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Payload   any       `json:"payload"`
}

type PromptFragment struct {
	ID      string `json:"id"`
	Slot    string `json:"slot"`
	Text    string `json:"text"`
	Surface string `json:"surface"`
}

type Skill struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  string            `json:"allowed_tools,omitempty"`
	Body          string            `json:"body"`
	Source        string            `json:"source,omitempty"`
	Dir           string            `json:"dir,omitempty"`
}

type PlaybookBullet struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Helpful int    `json:"helpful"`
	Harmful int    `json:"harmful"`
	Source  string `json:"source,omitempty"`
}

type Playbook struct {
	ID      string           `json:"id"`
	Bullets []PlaybookBullet `json:"bullets"`
}

type ToolImplKind string

const (
	ToolImplHost ToolImplKind = "host"
	ToolImplWASM ToolImplKind = "wasm"
	ToolImplMCP  ToolImplKind = "mcp"
)

type ToolSpec struct {
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Parameters      map[string]any `json:"parameters"`
	Capability      string         `json:"capability"`
	Impl            ToolImplKind   `json:"impl"`
	Pointer         string         `json:"pointer,omitempty"`
	ReadOnly        bool           `json:"read_only,omitempty"`
	ConcurrencySafe bool           `json:"concurrency_safe,omitempty"`
	OpenWorld       bool           `json:"open_world,omitempty"`
	Destructive     bool           `json:"destructive,omitempty"`
	Exclusive       bool           `json:"exclusive,omitempty"`
}

type LoopPreset struct {
	ID              string `json:"id"`
	MaxTurns        int    `json:"max_turns"`
	MaxToolMessages int    `json:"max_tool_messages"`
	// CompactionKeep is messages after system, expanded to a pairing boundary.
	CompactionKeep int `json:"compaction_keep"`
	// CompactionTokens is the Harbor/eval Shape budget (estimateTokens). Chat
	// uses ModelContextWindow when set; this field stays the eval-stable ceiling.
	CompactionTokens int `json:"compaction_tokens,omitempty"`
	// ToolResultBudget is runes per tool body inside Shape.
	ToolResultBudget int `json:"tool_result_budget,omitempty"`
	MicroKeep        int `json:"micro_keep,omitempty"`
	// PlaybookTokens is bytes kept by RenderBudget (trusted pin).
	PlaybookTokens int `json:"playbook_tokens,omitempty"`
	// RulesTokens is runes kept from the first-hit YOYO.md/AGENTS.md/CLAUDE.md.
	RulesTokens int `json:"rules_tokens,omitempty"`
	// AllowLLMCompact is chat-only. Harbor and subagents force this false.
	AllowLLMCompact bool    `json:"allow_llm_compact,omitempty"`
	MaxBudgetUSD    float64 `json:"max_budget_usd,omitempty"`
	PlanMode        bool    `json:"plan_mode,omitempty"`
	MaxParallel     int     `json:"max_parallel,omitempty"`
	Bootstrap       string  `json:"bootstrap"`
	Execution       string  `json:"execution"`
	Verification    string  `json:"verification"`
	FailureRecovery string  `json:"failure_recovery"`
	// MaxRecentToolErrors is L1 middleware: after this many consecutive
	// tool errors the runtime injects ToolErrorInstruction. Topology (L3)
	// is unaffected.
	MaxRecentToolErrors  int    `json:"max_recent_tool_errors,omitempty"`
	ToolErrorInstruction string `json:"tool_error_instruction,omitempty"`
	// TaskInstruction is prepended to depth-1 subagent prompts (L1).
	TaskInstruction string `json:"task_instruction,omitempty"`
}

type PolicyPack struct {
	ID              string            `json:"id"`
	Mode            string            `json:"mode,omitempty"` // auto | ask | plan | bypass
	DefaultAllow    []string          `json:"default_allow"`
	RequireApproval []string          `json:"require_approval"`
	NetworkAllow    []string          `json:"network_allow"`
	Notes           map[string]string `json:"notes,omitempty"`
}

type EvalSuite struct {
	ID         string   `json:"id"`
	TaskDir    string   `json:"task_dir"`
	HeldIn     []string `json:"held_in"`
	HeldOut    []string `json:"held_out"`
	Safety     []string `json:"safety,omitempty"`
	Repeats    int      `json:"repeats"`
	TimeoutSec int      `json:"timeout_sec"`
	Sealed     bool     `json:"sealed"`
	// Transfer task ids are never shown to the proposer and are not part of
	// ShouldPromote. They are a post-canary migration check only.
	Transfer []string `json:"transfer,omitempty"`
}

type ModelBinding struct {
	ID        string  `json:"id"`
	Provider  string  `json:"provider"`
	Model     string  `json:"model"`
	BaseURL   string  `json:"base_url,omitempty"`
	MaxTokens int     `json:"max_tokens,omitempty"`
	Temp      float64 `json:"temperature,omitempty"`
}

type HarnessSnapshot struct {
	ID               string   `json:"id"`
	ModelFingerprint string   `json:"model_fingerprint"`
	PromptFragments  []string `json:"prompt_fragments"`
	Skills           []string `json:"skills"`
	Playbook         string   `json:"playbook,omitempty"`
	Tools            []string `json:"tools"`
	LoopPreset       string   `json:"loop_preset"`
	PolicyPack       string   `json:"policy_pack"`
	EvalSuite        string   `json:"eval_suite,omitempty"`
	EvalTools        []string `json:"eval_tools,omitempty"`
	WASMPlugins      []string `json:"wasm_plugins,omitempty"`
	Parent           string   `json:"parent,omitempty"`
	Note             string   `json:"note,omitempty"`
}

type WASMPlugin struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Capability  string `json:"capability"`
	ModuleHash  string `json:"module_hash"`
	Export      string `json:"export"`
	MemoryPages int    `json:"memory_pages"`
	TimeoutMS   int    `json:"timeout_ms"`
	// SigHex is an ed25519 signature over the raw module bytes. Empty means
	// unsigned — Checkout must reject the snapshot.
	SigHex string `json:"sig_hex,omitempty"`
}
