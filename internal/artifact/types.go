// Package artifact 是内容寻址对象和 refs 指针。对象按哈希落盘；裸字节已在库里就不再重写，Put 的信封含写入时间，同一载荷会变成新对象。
// 能改指向的名字有 active、canary、HEAD、staging，以及 archive/<哈希前 12 位>、models/<指纹>/active 和 canary。Set 也接受其他相对路径。
package artifact

import "time"

// Kind 是信封里的对象种类。
type Kind string

const (
	// KindPromptFragment 是提示词片段，kind 为 prompt_fragment。
	KindPromptFragment Kind = "prompt_fragment"
	// KindSkill 是技能，kind 为 skill。
	KindSkill Kind = "skill"
	// KindPlaybook 是剧本，kind 为 playbook。
	KindPlaybook Kind = "playbook"
	// KindToolSpec 是工具声明，kind 为 tool_spec。
	KindToolSpec Kind = "tool_spec"
	// KindLoopPreset 是循环预设，kind 为 loop_preset。
	KindLoopPreset Kind = "loop_preset"
	// KindPolicyPack 是策略包，kind 为 policy_pack。
	KindPolicyPack Kind = "policy_pack"
	// KindEvalSuite 是评测套件，kind 为 eval_suite。
	KindEvalSuite Kind = "eval_suite"
	// KindModelBinding 是模型绑定，kind 为 model_binding。
	KindModelBinding Kind = "model_binding"
	// KindHarnessSnapshot 是 harness 快照，kind 为 harness_snapshot。
	KindHarnessSnapshot Kind = "harness_snapshot"
	// KindWASMPlugin 是 WASM 插件，kind 为 wasm_plugin。
	KindWASMPlugin Kind = "wasm_plugin"
)

// Envelope 是落盘的信封。Put 对整份信封做哈希，不只对载荷。
type Envelope struct {
	Kind Kind   `json:"kind"`
	ID   string `json:"id"`
	// CreatedAt 由 Put 写成当前 UTC，并参与哈希。
	CreatedAt time.Time `json:"created_at"`
	Payload   any       `json:"payload"`
}

// PromptFragment 是按槽位存放的一段提示词。
type PromptFragment struct {
	ID      string `json:"id"`
	Slot    string `json:"slot"`
	Text    string `json:"text"`
	Surface string `json:"surface"`
}

// Skill 是解析后的技能。Dir 只在来源路径以 skill.md 结尾时有目录，空来源和 builtin 留空。
type Skill struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	DisplayName   string `json:"display_name,omitempty"`
	Icon          string `json:"icon,omitempty"`
	License       string `json:"license,omitempty"`
	Compatibility string `json:"compatibility,omitempty"`
	// Metadata 是压平后的点分键。解析不到叶子时为 nil，不是空映射。
	Metadata map[string]string `json:"metadata,omitempty"`
	// AllowedTools 是收成一段的工具名。前置信息里的列表会用空格拼起来。
	AllowedTools string `json:"allowed_tools,omitempty"`
	// DisableModelInvocation 为真时不允许隐式调用。零值还要再看 Metadata 里的同类开关。
	DisableModelInvocation bool   `json:"disable_model_invocation,omitempty"`
	Body                   string `json:"body"`
	Source                 string `json:"source,omitempty"`
	// Dir 不是工作目录。来源不以 skill.md 结尾时为零值。
	Dir string `json:"dir,omitempty"`
}

// PlaybookBullet 是剧本中的一条。Helpful 和 Harmful 是计数，同文案合并时相加。
type PlaybookBullet struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Helpful int    `json:"helpful"`
	Harmful int    `json:"harmful"`
	Source  string `json:"source,omitempty"`
}

// Playbook 是可增量合并的条目集。
type Playbook struct {
	ID      string           `json:"id"`
	Bullets []PlaybookBullet `json:"bullets"`
}

// ToolImplKind 标明工具实现落在宿主、WASM 还是 MCP。
type ToolImplKind string

const (
	// ToolImplHost 表示由宿主实现，值为 host。
	ToolImplHost ToolImplKind = "host"
	// ToolImplWASM 表示在 WASM 模块里，值为 wasm。
	ToolImplWASM ToolImplKind = "wasm"
	// ToolImplMCP 表示走 MCP，值为 mcp。
	ToolImplMCP ToolImplKind = "mcp"
)

// ToolSpec 是工具声明。Impl 决定实现落在哪里。
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

// LoopPreset 是一轮运行的预算和文案。哪些字段算拓扑由 LoopTopologyEqual 决定。
type LoopPreset struct {
	ID              string `json:"id"`
	MaxTurns        int    `json:"max_turns"`
	MaxToolMessages int    `json:"max_tool_messages"`
	// CompactionKeep 是系统消息之后保留的条数，截断时会扩到成对边界。
	CompactionKeep int `json:"compaction_keep"`
	// CompactionTokens 是 Harbor 和评测的 Shape 预算，按 estimateTokens 计。聊天若设置了模型窗口就用窗口；这个字段仍是评测用的稳定上限。
	CompactionTokens int `json:"compaction_tokens,omitempty"`
	// ToolResultBudget 是 Shape 里每段工具正文保留的 rune 数。
	ToolResultBudget int `json:"tool_result_budget,omitempty"`
	MicroKeep        int `json:"micro_keep,omitempty"`
	// PlaybookTokens 是 RenderBudget 保留的字节数，当作受信任的钉选。
	PlaybookTokens int `json:"playbook_tokens,omitempty"`
	// RulesTokens 是从先命中的 YOYO.md、AGENTS.md、CLAUDE.md 里保留的 rune 数。
	RulesTokens int `json:"rules_tokens,omitempty"`
	// AllowLLMCompact 只给聊天使用。Harbor 和子代理会把它强制写成 false。
	AllowLLMCompact bool    `json:"allow_llm_compact,omitempty"`
	MaxBudgetUSD    float64 `json:"max_budget_usd,omitempty"`
	PlanMode        bool    `json:"plan_mode,omitempty"`
	MaxParallel     int     `json:"max_parallel,omitempty"`
	Bootstrap       string  `json:"bootstrap"`
	// Execution 同时承接槽位 execution 和 runtime。
	Execution       string `json:"execution"`
	Verification    string `json:"verification"`
	FailureRecovery string `json:"failure_recovery"`
	// MaxRecentToolErrors 是 L1 中间件：连续工具错误达到这个次数后，运行时注入 ToolErrorInstruction。它不计入 L3 拓扑。
	MaxRecentToolErrors int `json:"max_recent_tool_errors,omitempty"`
	// ToolErrorInstruction 由槽位 middleware 和 tool_error 写入。
	ToolErrorInstruction string `json:"tool_error_instruction,omitempty"`
	// TaskInstruction 会加在一层子代理的提示词前面，属于 L1。
	TaskInstruction string `json:"task_instruction,omitempty"`
	// KeepTokens 是聊天检查点保留的尾部预算，属于 L3。Harbor 仍用 CompactionKeep。零值表示用运行时的聊天保底。
	KeepTokens int `json:"keep_tokens,omitempty"`
	// ContextBuffer 在检查点之前从模型窗口里减去，做法同 OpenCode。Harbor 忽略它，模型窗口保持 0。
	ContextBuffer int `json:"context_buffer,omitempty"`
	// OutputReserve 用来覆盖按比例留出的输出预留。零值表示按窗口的 10% 留，不低于 8000，不超过 32000。
	OutputReserve int `json:"output_reserve,omitempty"`
	// TaskMaxTurns 与 TaskMaxToolMessages 决定一层子代理的规模，属于 L3。零值表示运行时默认 24 轮、60 条工具消息。
	TaskMaxTurns        int `json:"task_max_turns,omitempty"`
	TaskMaxToolMessages int `json:"task_max_tool_messages,omitempty"`
	// StackAgentsMD 以前会把从 git 根到当前目录的 AGENTS.md 接在一起。聊天关掉它：根文件留在钉选里，嵌套文件要等读到对应路径才加载。
	StackAgentsMD bool `json:"stack_agents_md,omitempty"`
}

// PolicyPack 是能力放行策略。
type PolicyPack struct {
	ID string `json:"id"`
	// Mode 的取值是 auto、ask、plan、bypass。
	Mode            string            `json:"mode,omitempty"`
	DefaultAllow    []string          `json:"default_allow"`
	RequireApproval []string          `json:"require_approval"`
	NetworkAllow    []string          `json:"network_allow"`
	Notes           map[string]string `json:"notes,omitempty"`
}

// EvalSuite 是评测任务的划分、重复次数和时限。
type EvalSuite struct {
	ID         string   `json:"id"`
	TaskDir    string   `json:"task_dir"`
	HeldIn     []string `json:"held_in"`
	HeldOut    []string `json:"held_out"`
	Safety     []string `json:"safety,omitempty"`
	Repeats    int      `json:"repeats"`
	TimeoutSec int      `json:"timeout_sec"`
	Sealed     bool     `json:"sealed"`
	// EvolveIn 是提议者能看见的划分。挖掘、提议和内层试跑用这份列表。空表示 HeldIn 同时充当演化集和晋升集。
	EvolveIn []string `json:"evolve_in,omitempty"`
	// Transfer 里的任务编号不会给提议者看，也不参与 ShouldPromote。只在金丝雀之后做迁移检查。
	Transfer []string `json:"transfer,omitempty"`
}

// ModelBinding 记录供应商、模型，以及可选的基址、最大输出和温度。本包不发起请求。
type ModelBinding struct {
	ID        string  `json:"id"`
	Provider  string  `json:"provider"`
	Model     string  `json:"model"`
	BaseURL   string  `json:"base_url,omitempty"`
	MaxTokens int     `json:"max_tokens,omitempty"`
	Temp      float64 `json:"temperature,omitempty"`
}

// HarnessSnapshot 是一份 harness 的指针表，正文不嵌在里面。
// PromptFragments、Skills、Playbook、LoopPreset、PolicyPack、EvalSuite、WASMPlugins 是对象哈希。
// Tools 和 EvalTools 是工具名。Parent 是父快照的哈希。ID、ModelFingerprint、Note 是普通字符串。
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

// WASMPlugin 是一个 WASM 工具模块的描述。ModuleHash 指向裸模块字节，不是这份描述自己的哈希。
type WASMPlugin struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Capability  string `json:"capability"`
	// ModuleHash 是裸模块字节的内容哈希。签名盖的是这串字节。
	ModuleHash  string `json:"module_hash"`
	Export      string `json:"export"`
	MemoryPages int    `json:"memory_pages"`
	TimeoutMS   int    `json:"timeout_ms"`
	// SigHex 是对裸模块字节的 ed25519 签名。空表示未签名，Checkout 必须拒绝这份快照。
	SigHex string `json:"sig_hex,omitempty"`
}
