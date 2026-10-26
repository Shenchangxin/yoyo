// Package home 解析并创建 Yoyo 的主目录布局。
// 桌面入口和 app.Open 调用 Open。Default 只解析路径，不创建目录。
// Open 按 0o755 建子目录。中途失败时，已经建好的目录留在磁盘上，不删除。
package home

import (
	"os"
	"path/filepath"
)

const (
	// EnvHome 是环境变量 YOYO_HOME。Default 见到它非空就用它当主目录。
	EnvHome = "YOYO_HOME"
	// DirName 是用户主目录下的默认文件夹名 .yoyo。
	DirName = ".yoyo"
)

// Dir 是一块已经确定根路径的主目录。
type Dir struct {
	// 主目录根路径。零值空字符串不能当根用，Open 返回前会写入。
	Root string
}

// Open 确保 root 下的固定子目录存在。root 为空时改用 Default 的路径，调用方传入的 root 原样使用，不再转成绝对路径。
// 目录权限是 0o755。某一级 MkdirAll 失败就返回该错误，已经建好的目录留着，不回滚。
// 还会创建 pages/home，以及 Logs 下面的 crash 和 mcp。
// YOYO_LOG_DIR 非空时，logs、crash、mcp 建在该变量的路径下。pages/home 仍在 Root 里。
// 不创建 config.yaml，不创建 refs/models，也不创建某个会话的 spill。
func Open(root string) (*Dir, error) {
	if root == "" {
		var err error
		root, err = Default()
		if err != nil {
			return nil, err
		}
	}
	d := &Dir{Root: root}
	for _, p := range []string{
		d.CAS(),
		d.Refs(),
		d.Sessions(),
		d.Journal(),
		d.EvalRuns(),
		d.Archive(),
		d.Plugins(),
		d.Tmp(),
		d.Updates(),
		d.Skills(),
		d.Packs(),
		d.Skins(),
		d.Workspace(),
		d.Projects(),
		d.Memory(),
		d.Library(),
		d.Schedule(),
		d.Inbox(),
		d.Personal(),
		d.Pages(),
		filepath.Join(d.Pages(), "home"),
		d.Profiles(),
		d.Connectors(),
		d.Connections(),
		d.Browser(),
		d.Computer(),
		d.Observe(),
		d.Worktrees(),
		d.Video(),
		d.VideoWorkspace(),
		d.Idols(),
		d.Runtime(),
		d.Logs(),
		filepath.Join(d.Logs(), "crash"),
		filepath.Join(d.Logs(), "mcp"),
	} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return nil, err
		}
	}
	return d, nil
}

// Default 解析主目录路径，不创建目录。
// YOYO_HOME 非空时返回它的绝对路径，Abs 失败就把错误返回。
// 否则是用户主目录下的 .yoyo。用户主目录取不到时返回错误，不另猜一条路径。
func Default() (string, error) {
	if v := os.Getenv(EnvHome); v != "" {
		return filepath.Abs(v)
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, DirName), nil
}

// CAS 指向 cas/objects。
func (d *Dir) CAS() string { return filepath.Join(d.Root, "cas", "objects") }

// Refs 指向 refs。
func (d *Dir) Refs() string { return filepath.Join(d.Root, "refs") }

// Sessions 指向 sessions。
func (d *Dir) Sessions() string { return filepath.Join(d.Root, "sessions") }

// Journal 指向 journal 目录。
func (d *Dir) Journal() string { return filepath.Join(d.Root, "journal") }

// EvalRuns 指向 evals/runs。
func (d *Dir) EvalRuns() string { return filepath.Join(d.Root, "evals", "runs") }

// Archive 指向 archive。
func (d *Dir) Archive() string { return filepath.Join(d.Root, "archive") }

// Plugins 指向 plugins。
func (d *Dir) Plugins() string { return filepath.Join(d.Root, "plugins") }

// Tmp 指向 tmp。
func (d *Dir) Tmp() string { return filepath.Join(d.Root, "tmp") }

// Updates 指向 updates。
func (d *Dir) Updates() string { return filepath.Join(d.Root, "updates") }

// Skills 指向 skills。
func (d *Dir) Skills() string { return filepath.Join(d.Root, "skills") }

// Packs 指向 packs。
func (d *Dir) Packs() string { return filepath.Join(d.Root, "packs") }

// Skins 指向 skins。
func (d *Dir) Skins() string { return filepath.Join(d.Root, "skins") }

// Workspace 指向 workspace。
func (d *Dir) Workspace() string {
	return filepath.Join(d.Root, "workspace")
}

// Config 指向 config.yaml。Open 不创建这个文件。
func (d *Dir) Config() string { return filepath.Join(d.Root, "config.yaml") }

// Projects 指向 projects。
func (d *Dir) Projects() string { return filepath.Join(d.Root, "projects") }

// Memory 指向 memory。
func (d *Dir) Memory() string { return filepath.Join(d.Root, "memory") }

// Library 指向 library。
func (d *Dir) Library() string { return filepath.Join(d.Root, "library") }

// Schedule 指向 schedule。
func (d *Dir) Schedule() string { return filepath.Join(d.Root, "schedule") }

// Inbox 指向 inbox。
func (d *Dir) Inbox() string { return filepath.Join(d.Root, "inbox") }

// Personal 指向 personal。
func (d *Dir) Personal() string { return filepath.Join(d.Root, "personal") }

// Pages 指向 pages。
func (d *Dir) Pages() string { return filepath.Join(d.Root, "pages") }

// Profiles 指向 profiles。
func (d *Dir) Profiles() string { return filepath.Join(d.Root, "profiles") }

// Connectors 指向 connectors。
func (d *Dir) Connectors() string { return filepath.Join(d.Root, "connectors") }

// Connections 指向 connections。
func (d *Dir) Connections() string {
	return filepath.Join(d.Root, "connections")
}

// Browser 指向 browser。
func (d *Dir) Browser() string { return filepath.Join(d.Root, "browser") }

// Computer 指向 computer。
func (d *Dir) Computer() string { return filepath.Join(d.Root, "computer") }

// Observe 指向 observe。
func (d *Dir) Observe() string { return filepath.Join(d.Root, "observe") }

// Worktrees 指向 worktrees。
func (d *Dir) Worktrees() string { return filepath.Join(d.Root, "worktrees") }

// Video 指向 video。
func (d *Dir) Video() string { return filepath.Join(d.Root, "video") }

// Idols 指向 idols。
func (d *Dir) Idols() string { return filepath.Join(d.Root, "idols") }

// Runtime 指向 runtime。
func (d *Dir) Runtime() string { return filepath.Join(d.Root, "runtime") }

// VideoWorkspace 指向 video/workspace。
func (d *Dir) VideoWorkspace() string {
	return filepath.Join(d.Video(), "workspace")
}

// Logs 在 YOYO_LOG_DIR 非空时原样返回该值，不转成绝对路径，也不拼到 Root 下。否则指向 logs。
func (d *Dir) Logs() string {
	if v := os.Getenv("YOYO_LOG_DIR"); v != "" {
		return v
	}
	return filepath.Join(d.Root, "logs")
}

// SessionSpill 指向 sessions 下该 id 的 spill。Open 不预先创建它。
func (d *Dir) SessionSpill(id string) string {
	return filepath.Join(d.Sessions(), id, "spill")
}

// ModelsRefs 指向 refs/models。Open 不预先创建它。
func (d *Dir) ModelsRefs() string {
	return filepath.Join(d.Refs(), "models")
}
