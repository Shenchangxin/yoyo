// 技能包的偏好、来源、清单和交给会话的运行时契约。

package skillpack

import "time"

// Pref 是全局配置里 packs.<id> 的一块，只有 Enabled。
type Pref struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// Origin 记录技能包是从 GitHub 还是本地目录装上来的。
type Origin struct {
	Kind      string `json:"kind"`                 // github 或 local
	Repo      string `json:"repo,omitempty"`       // 所有者/仓库名
	Ref       string `json:"ref,omitempty"`        // 分支、标签或提交
	SkillsRel string `json:"skills_rel,omitempty"` // 仓库里的技能目录
	Path      string `json:"path,omitempty"`       // 本地目录
}

// Manifest 是 home/packs/<id>/pack.json 的内容。
type Manifest struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Version        string    `json:"version,omitempty"`
	License        string    `json:"license,omitempty"`
	BootstrapSkill string    `json:"bootstrap_skill,omitempty"`
	Methodology    bool      `json:"methodology,omitempty"`
	Origin         Origin    `json:"origin"`
	Commit         string    `json:"commit,omitempty"`
	SkillCount     int       `json:"skill_count"`
	InstalledAt    time.Time `json:"installed_at"`
}

// Known 是内置目录的一条。它只描述从哪里取、如何引导；安装时仍要复制文件。方法论包设置 BootstrapSkill 和 Methodology。领域包这两项都空着，技能按目录路由、用到时再加载，会话开始时不注入。
type Known struct {
	ID             string
	Name           string
	Description    string
	License        string
	DefaultRepo    string
	DefaultRef     string
	SkillsRel      string
	BootstrapSkill string
	Methodology    bool
}

// WorkspacePref 是 <workspace>/.yoyo/packs.json 里一个 id 的开关。
type WorkspacePref struct {
	Enabled bool `json:"enabled"`
}

// Status 是界面和接口看到的包：内置目录和磁盘上已安装的并集。
type Status struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	License         string    `json:"license,omitempty"`
	Version         string    `json:"version,omitempty"`
	Commit          string    `json:"commit,omitempty"`
	BootstrapSkill  string    `json:"bootstrap_skill,omitempty"`
	Methodology     bool      `json:"methodology"`
	Installed       bool      `json:"installed"`
	Enabled         bool      `json:"enabled"`
	EnableGlobal    bool      `json:"enable_global"`
	EnableWorkspace *bool     `json:"enable_workspace,omitempty"`
	Root            string    `json:"root,omitempty"`
	SkillsDir       string    `json:"skills_dir,omitempty"`
	SkillCount      int       `json:"skill_count"`
	Skills          []string  `json:"skills,omitempty"`
	Origin          Origin    `json:"origin"`
	InstalledAt     time.Time `json:"installed_at,omitempty"`
	Known           bool      `json:"known"`
}

// Session 是交给每一轮的运行时契约：包 id、引导技能、工具映射和方法论标记。
type Session struct {
	ID             string
	BootstrapSkill string
	Mapping        string
	Methodology    bool
}

const (
	// ScopeGlobal 表示全局开关，值为 global。
	ScopeGlobal = "global"
	// ScopeWorkspace 表示工作区开关，值为 workspace。
	ScopeWorkspace = "workspace"
	// ScopeInherit 表示沿用上一级，值为 inherit。
	ScopeInherit = "inherit"
	// KindGitHub 表示来源是 GitHub，值为 github。
	KindGitHub = "github"
	// KindLocal 表示来源是本地目录，值为 local。
	KindLocal = "local"
)
