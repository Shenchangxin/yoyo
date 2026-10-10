package skillpack

import "time"

// Pref is the global config.yaml packs.<id> block.
type Pref struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// Origin records how a pack landed on disk.
type Origin struct {
	Kind      string `json:"kind"`                 // github | local
	Repo      string `json:"repo,omitempty"`       // owner/name
	Ref       string `json:"ref,omitempty"`        // branch, tag, or commit
	SkillsRel string `json:"skills_rel,omitempty"` // directory inside the repo
	Path      string `json:"path,omitempty"`       // local checkout
}

// Manifest is ~/.yoyo/packs/<id>/pack.json.
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

// Known is a built-in catalog entry. Installation still copies files; this
// only describes what Yoyo can fetch and how to bootstrap it.
//
// Methodology packs set BootstrapSkill and Methodology. Domain packs leave
// both empty: skills are catalog-routed and loaded on demand, with no
// session-start inject.
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

// WorkspacePref is one key in .yoyo/packs.json.
type WorkspacePref struct {
	Enabled bool `json:"enabled"`
}

// Status is the UI/API view of a pack: known catalog ∪ installed disk.
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

// Session is the per-turn runtime contract handed to the loop.
type Session struct {
	ID             string
	BootstrapSkill string
	Mapping        string
	Methodology    bool
}

const (
	ScopeGlobal    = "global"
	ScopeWorkspace = "workspace"
	ScopeInherit   = "inherit"
	KindGitHub     = "github"
	KindLocal      = "local"
)
