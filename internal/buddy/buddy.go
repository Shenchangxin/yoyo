// Package buddy 管理本机的外部 agent 应用：目录（精选表叠在 ACP 注册表上）、检测、托管或系统安装、配置与密钥、登录、探测和健康检查。
// app 用 Manager 提供 buddy.* RPC；编排台的工人用 Launch、Online 和 Roster 找人干活。
// 不负责跑工单（那在 internal/acpworker），也不碰 yoyo 自己的会话。
package buddy

import (
	"maps"
	"slices"
)

// YoyoID 是内置 agent 的编号，永远排第一、永远在线。
const YoyoID = "yoyo"

// State 是一个 agent 在 Buddy 里的状态。
type State string

// 状态取值，和前端 BuddyState 一一对应。
const (
	StateNotInstalled State = "not_installed"
	StateInstalling   State = "installing"
	StateProbing      State = "probing"
	StateNeedsAuth    State = "needs_auth"
	StateNeedsConfig  State = "needs_config"
	StateOnline       State = "online"
	StateRecentlyLost State = "recently_lost"
	StateOffline      State = "offline"
	StateError        State = "error"
	StateUnsupported  State = "unsupported"
)

// Settings 是配置文件里的 buddies 一节。app 用类型别名把它挂进 Config。
type Settings struct {
	NPMRegistry  string                `yaml:"npm_registry,omitempty" json:"npm_registry,omitempty"`
	PyIndex      string                `yaml:"py_index,omitempty" json:"py_index,omitempty"`
	Registry     string                `yaml:"registry,omitempty" json:"registry,omitempty"`
	DefaultAgent string                `yaml:"default_agent,omitempty" json:"default_agent,omitempty"`
	AutoPick     bool                  `yaml:"auto_pick,omitempty" json:"auto_pick,omitempty"`
	TurnMinutes  int                   `yaml:"turn_minutes,omitempty" json:"turn_minutes,omitempty"`
	Items        map[string]ItemConfig `yaml:"items,omitempty" json:"items,omitempty"`
	Custom       []CustomAgent         `yaml:"custom,omitempty" json:"custom,omitempty"`
}

// ItemConfig 是一个 agent 的设置。密钥只记名字，值在 vault 的 buddy.<id>.<NAME>。
type ItemConfig struct {
	Disabled    bool              `yaml:"disabled,omitempty" json:"disabled,omitempty"`
	Mode        string            `yaml:"mode,omitempty" json:"mode,omitempty"`
	Args        []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Secrets     []string          `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Reuse       []string          `yaml:"reuse,omitempty" json:"reuse,omitempty"`
	Model       string            `yaml:"model,omitempty" json:"model,omitempty"`
	Approval    string            `yaml:"approval,omitempty" json:"approval,omitempty"`
	MaxParallel int               `yaml:"max_parallel,omitempty" json:"max_parallel,omitempty"`
	Strengths   string            `yaml:"strengths,omitempty" json:"strengths,omitempty"`
}

// CustomAgent 是人手动加的 ACP agent：一条命令加参数。
type CustomAgent struct {
	ID      string            `yaml:"id" json:"id"`
	Name    string            `yaml:"name" json:"name"`
	Command string            `yaml:"command" json:"command"`
	Args    []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

// Clone 深拷贝设置，改副本不会影响正在被读的原值。
func (s Settings) Clone() Settings {
	out := s
	if s.Items != nil {
		out.Items = make(map[string]ItemConfig, len(s.Items))
		for id, item := range s.Items {
			out.Items[id] = item.Clone()
		}
	}
	out.Custom = make([]CustomAgent, 0, len(s.Custom))
	for _, c := range s.Custom {
		c.Args = slices.Clone(c.Args)
		c.Env = maps.Clone(c.Env)
		out.Custom = append(out.Custom, c)
	}
	if len(out.Custom) == 0 {
		out.Custom = nil
	}
	return out
}

// Clone 深拷贝一个 agent 的设置。
func (c ItemConfig) Clone() ItemConfig {
	c.Args = slices.Clone(c.Args)
	c.Env = maps.Clone(c.Env)
	c.Secrets = slices.Clone(c.Secrets)
	c.Reuse = slices.Clone(c.Reuse)
	return c
}

// Item 返回 id 的设置，没有时返回零值。
func (s Settings) Item(id string) ItemConfig {
	return s.Items[id]
}

// MaxParallelOf 返回同时最多几个任务：没设时 2，夹在 1 到 4 之间。
func (c ItemConfig) MaxParallelOf() int {
	switch {
	case c.MaxParallel <= 0:
		return 2
	case c.MaxParallel > 4:
		return 4
	default:
		return c.MaxParallel
	}
}

// TurnMinutesOf 返回单轮时长上限（分钟），没设时 60。
func (s Settings) TurnMinutesOf() int {
	if s.TurnMinutes <= 0 {
		return 60
	}
	return s.TurnMinutes
}

// View 是给界面的一项，字段和前端 BuddyView 对齐。不含任何密钥值。CanManage 为真表示本平台能一键托管安装。
type View struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Vendor        string             `json:"vendor,omitempty"`
	Builtin       bool               `json:"builtin,omitempty"`
	Featured      bool               `json:"featured,omitempty"`
	Custom        bool               `json:"custom,omitempty"`
	Order         int                `json:"order,omitempty"`
	RegistryID    string             `json:"registryId,omitempty"`
	Description   string             `json:"description,omitempty"`
	Icon          string             `json:"icon,omitempty"`
	Docs          string             `json:"docs,omitempty"`
	Version       string             `json:"version,omitempty"`
	Latest        string             `json:"latest,omitempty"`
	Aliases       []string           `json:"aliases,omitempty"`
	Install       InstallView        `json:"install"`
	CanManage     bool               `json:"canManage,omitempty"`
	System        *SystemView        `json:"system,omitempty"`
	CLIOnly       bool               `json:"cliOnly,omitempty"`
	State         State              `json:"state"`
	Detail        string             `json:"detail,omitempty"`
	CheckedAt     string             `json:"checkedAt,omitempty"`
	Working       int                `json:"working,omitempty"`
	Progress      *Progress          `json:"progress,omitempty"`
	Auth          *AuthView          `json:"auth,omitempty"`
	Caps          *CapsView          `json:"caps,omitempty"`
	Config        ConfigView         `json:"config"`
	SystemInstall *SystemInstallView `json:"systemInstall,omitempty"`
}

// InstallView 说明这项怎么装的。Unverified 为真表示下载时对方没给校验和。
type InstallView struct {
	Kind       string `json:"kind"`
	Source     string `json:"source,omitempty"`
	Path       string `json:"path,omitempty"`
	SizeBytes  int64  `json:"sizeBytes,omitempty"`
	Unverified bool   `json:"unverified,omitempty"`
}

// SystemView 是 PATH 上找到的系统版本。
type SystemView struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
}

// Progress 是安装进度。Phase 取 queued、node、download、extract、deps、verify、done。
type Progress struct {
	Phase   string `json:"phase"`
	Percent int    `json:"percent,omitempty"`
	Line    string `json:"line,omitempty"`
}

// AuthView 列出对方声明的登录方式。
type AuthView struct {
	Methods []AuthMethodView `json:"methods"`
}

// AuthMethodView 是一种登录方式。Type 取 agent 或 terminal；env_var 型的在配置里填密钥，不在这里出现。
type AuthMethodView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// CapsView 是对方声明的能力里界面关心的几项。
type CapsView struct {
	Resume       bool `json:"resume,omitempty"`
	Load         bool `json:"load,omitempty"`
	Close        bool `json:"close,omitempty"`
	SystemPrompt bool `json:"systemPrompt,omitempty"`
	MCPHTTP      bool `json:"mcpHttp,omitempty"`
	Image        bool `json:"image,omitempty"`
	Dirs         bool `json:"dirs,omitempty"`
	Logout       bool `json:"logout,omitempty"`
}

// ConfigView 是一项的设置。密钥只给 hasKey。
type ConfigView struct {
	Enabled     bool              `json:"enabled"`
	Mode        string            `json:"mode"`
	Model       string            `json:"model,omitempty"`
	Models      []ModelView       `json:"models,omitempty"`
	Approval    string            `json:"approval"`
	MaxParallel int               `json:"maxParallel"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Strengths   string            `json:"strengths,omitempty"`
	Secrets     []SecretView      `json:"secrets"`
}

// ModelView 是一个可选模型。
type ModelView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SecretView 是一把密钥的状态。Reusable 为真表示 yoyo 有同类提供商的钥匙，Reuse 表示已经打开复用。
type SecretView struct {
	Name     string `json:"name"`
	Label    string `json:"label,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	HasKey   bool   `json:"hasKey"`
	Reusable bool   `json:"reusable,omitempty"`
	Reuse    bool   `json:"reuse,omitempty"`
}

// SystemInstallView 说明能不能装到系统、用什么命令；不能时 Reason 给原因。
type SystemInstallView struct {
	Available bool   `json:"available"`
	Command   string `json:"command,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// Prefs 是 Buddy 的偏好，字段和前端 BuddyPrefs 对齐。
type Prefs struct {
	NPMRegistry  string `json:"npmRegistry"`
	PyIndex      string `json:"pyIndex"`
	DefaultAgent string `json:"defaultAgent"`
	AutoPick     bool   `json:"autoPick"`
	TurnMinutes  int    `json:"turnMinutes"`
}

// RosterItem 是给路由看的一项：谁能派、在做几个、擅长什么。
type RosterItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Vendor      string   `json:"vendor,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	State       State    `json:"state"`
	Ready       bool     `json:"ready"`
	Working     int      `json:"working"`
	MaxParallel int      `json:"maxParallel"`
	Strengths   string   `json:"strengths,omitempty"`
}
