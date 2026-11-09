// pathenv.go 拥有找命令用的目录表：进程 PATH，macOS/Linux 再加登录 shell 的 PATH，最后补上常见安装目录。
// 图形界面启动的 yoyo 拿不到用户 shell 里的 PATH，这里补齐，结果缓存 30 分钟。检测、启动和装到系统都用它。

package buddy

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// pathCacheTTL 是登录 shell PATH 的缓存时长。
const pathCacheTTL = 30 * time.Minute

// pathMarker 标出登录 shell 输出里 PATH 开始的位置，避开交互 shell 打印的欢迎语。
const pathMarker = "__YOYO_PATH__"

// errNotFound 是目录表里找不到命令。
var errNotFound = errors.New("executable not found")

// PathFinder 在扩展过的目录表里找命令。
type PathFinder struct {
	goos  string
	env   func(string) string
	shell func(ctx context.Context) string
	now   func() time.Time

	mu       sync.Mutex
	dirs     []string
	cachedAt time.Time
}

// newPathFinder 用本进程的环境建一个查找器。
func newPathFinder(goos string, now func() time.Time) *PathFinder {
	return &PathFinder{goos: goos, env: os.Getenv, shell: loginShellPath, now: now}
}

// Dirs 返回去重后的目录表：进程 PATH、登录 shell 的 PATH、常见安装目录。
func (p *PathFinder) Dirs() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.dirs != nil && p.now().Sub(p.cachedAt) < pathCacheTTL {
		return p.dirs
	}
	sep := string(os.PathListSeparator)
	if p.goos == "windows" {
		sep = ";"
	}
	var all []string
	all = append(all, strings.Split(p.env("PATH"), sep)...)
	if p.goos != "windows" && p.shell != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		all = append(all, strings.Split(p.shell(ctx), ":")...)
		cancel()
	}
	all = append(all, p.commonDirs()...)
	seen := map[string]bool{}
	var out []string
	for _, d := range all {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		key := d
		if p.goos == "windows" {
			key = strings.ToLower(d)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, d)
	}
	p.dirs = out
	p.cachedAt = p.now()
	return out
}

// commonDirs 是常见的用户级安装目录。
func (p *PathFinder) commonDirs() []string {
	if p.goos == "windows" {
		if appdata := p.env("APPDATA"); appdata != "" {
			return []string{filepath.Join(appdata, "npm")}
		}
		return nil
	}
	home := p.env("HOME")
	var out []string
	if home != "" {
		out = append(out, filepath.Join(home, ".local", "bin"), filepath.Join(home, ".npm-global", "bin"))
	}
	return append(out, "/opt/homebrew/bin", "/usr/local/bin")
}

// PathValue 返回给子进程用的 PATH 值。
func (p *PathFinder) PathValue() string {
	sep := ":"
	if p.goos == "windows" {
		sep = ";"
	}
	return strings.Join(p.Dirs(), sep)
}

// LookPath 在目录表里找 name。name 带路径时只检查它本身。Windows 上没写扩展名时按 PATHEXT 试。
func (p *PathFinder) LookPath(name string) (string, error) {
	if name == "" {
		return "", errNotFound
	}
	if strings.ContainsAny(name, `/\`) {
		if p.runnable(name) {
			return name, nil
		}
		return "", errNotFound
	}
	for _, dir := range p.Dirs() {
		for _, candidate := range p.candidates(filepath.Join(dir, name)) {
			if p.runnable(candidate) {
				return candidate, nil
			}
		}
	}
	return "", errNotFound
}

// candidates 列出 base 可能的文件名。Windows 上已有可执行扩展名时只试它本身。
func (p *PathFinder) candidates(base string) []string {
	if p.goos != "windows" {
		return []string{base}
	}
	exts := strings.Split(strings.ToLower(firstNonEmpty(p.env("PATHEXT"), ".COM;.EXE;.BAT;.CMD")), ";")
	ext := strings.ToLower(filepath.Ext(base))
	for _, e := range exts {
		if e != "" && e == ext {
			return []string{base}
		}
	}
	var out []string
	for _, e := range exts {
		if e != "" {
			out = append(out, base+e)
		}
	}
	return out
}

// runnable 判断 path 是不是能跑的普通文件。Unix 上要有执行位。
func (p *PathFinder) runnable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if p.goos == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}

// loginShellPath 用登录交互 shell 取 PATH，3 秒内没结果就返回空。
func loginShellPath(ctx context.Context) string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	out, err := exec.CommandContext(ctx, shell, "-ilc", `printf '`+pathMarker+`%s' "$PATH"`).Output()
	if err != nil && len(out) == 0 {
		return ""
	}
	text := string(out)
	i := strings.LastIndex(text, pathMarker)
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(text[i+len(pathMarker):])
}
