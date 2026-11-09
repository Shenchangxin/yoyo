// uv.go 拥有托管 uv 和 uvx 分发的安装：uv 从 GitHub 下载（失败退 ghfast），校验后解到 runtime/uv/<版本>；
// agent 用 uv tool install 装进 buddies/<id>，启动时直接跑装出来的命令。只给 uvx 分发的 agent 用。

package buddy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/fetch"
)

// uvAsset 返回本平台的 uv 归档名。
func uvAsset(p Platform) (string, bool) {
	arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[p.GOARCH]
	if arch == "" {
		return "", false
	}
	var name string
	switch p.GOOS {
	case "windows":
		name = "uv-" + arch + "-pc-windows-msvc.zip"
	case "darwin":
		name = "uv-" + arch + "-apple-darwin.tar.gz"
	case "linux":
		libc := "gnu"
		if p.Musl {
			libc = "musl"
		}
		name = "uv-" + arch + "-unknown-linux-" + libc + ".tar.gz"
	default:
		return "", false
	}
	_, ok := uvSHA256[name]
	return name, ok
}

// exeName 在 Windows 上补 .exe。
func (m *Manager) exeName(name string) string {
	if m.plat.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// ensureUV 准备好托管 uv，返回 uv 可执行文件。
func (m *Manager) ensureUV(ctx context.Context, prog func(Progress), log io.Writer) (string, error) {
	m.uvMu.Lock()
	defer m.uvMu.Unlock()
	if m.opts.RuntimeDir == "" {
		return "", errors.New("no runtime dir")
	}
	dir := filepath.Join(m.opts.RuntimeDir, "uv", uvVersion)
	if exe := findFile(dir, m.exeName("uv")); exe != "" {
		return exe, nil
	}
	asset, ok := uvAsset(m.plat)
	if !ok {
		return "", errors.New(m.say("此系统不支持托管 uv", "managed uv is not available here"))
	}
	prog(Progress{Phase: "node"})
	archive := filepath.Join(m.opts.RuntimeDir, "uv", ".download", asset)
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		return "", err
	}
	defer os.RemoveAll(filepath.Dir(archive))
	raw := "https://github.com/astral-sh/uv/releases/download/" + uvVersion + "/" + asset
	fmt.Fprintf(log, "download %s\n", raw)
	if _, err := fetch.Download(ctx, fetch.Spec{URLs: fetch.MirrorURLs(raw), SHA256: uvSHA256[asset], MaxBytes: 128 << 20, Dest: archive, Client: m.opts.HTTP}); err != nil {
		return "", fmt.Errorf("uv download: %w", err)
	}
	if err := fetch.Extract(archive, dir, fetch.KindOf(asset), 0); err != nil {
		return "", fmt.Errorf("uv extract: %w", err)
	}
	fetch.Unblock(dir)
	exe := findFile(dir, m.exeName("uv"))
	if exe == "" {
		_ = os.RemoveAll(dir)
		return "", errors.New("uv missing after extract")
	}
	if m.plat.GOOS != "windows" {
		_ = os.Chmod(exe, 0o755)
		if uvx := findFile(dir, "uvx"); uvx != "" {
			_ = os.Chmod(uvx, 0o755)
		}
	}
	return exe, nil
}

// installUVX 用 uv tool install 装一个 Python 包，命令放进 buddies/<id>/uv-bin-<版本>。
func (m *Manager) installUVX(ctx context.Context, e Entry, pkg PackageTarget, prog func(Progress), log io.Writer) (*InstallRecord, error) {
	uv, err := m.ensureUV(ctx, prog, log)
	if err != nil {
		return nil, err
	}
	prog(Progress{Phase: "deps", Percent: 40})
	name, version := splitPackage(pkg.Package)
	base := m.installDir(e.ID)
	binDir := filepath.Join(base, "uv-bin-"+safeName(version))
	toolDir := filepath.Join(base, "uv-tools-"+safeName(version))
	_ = os.RemoveAll(binDir)
	_ = os.RemoveAll(toolDir)
	env := m.baseEnv(nil)
	env = setEnv(env, "UV_TOOL_DIR", toolDir)
	env = setEnv(env, "UV_TOOL_BIN_DIR", binDir)
	env = setEnv(env, "UV_CACHE_DIR", filepath.Join(m.opts.Root, ".uv-cache"))
	env = setEnv(env, "UV_PYTHON_INSTALL_DIR", filepath.Join(m.opts.RuntimeDir, "uv", "python"))
	if idx := strings.TrimSpace(m.opts.Settings().PyIndex); idx != "" {
		env = setEnv(env, "UV_DEFAULT_INDEX", idx)
	}
	fmt.Fprintf(log, "uv tool install %s\n", pkg.Package)
	l := acp.Launch{Command: uv, Args: []string{"tool", "install", "--force", pkg.Package}, Env: env, Dir: base}
	if _, err := m.opts.Run(ctx, Cmd{Launch: l, Output: log}); err != nil {
		_ = os.RemoveAll(binDir)
		_ = os.RemoveAll(toolDir)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("uv tool install: %w", err)
	}
	cmd := pickExecutable(binDir, name)
	if cmd == "" {
		return nil, errors.New("uv tool install: no executable")
	}
	return &InstallRecord{
		Kind: "uvx", Version: version, Package: name, Command: cmd, Args: append([]string(nil), pkg.Args...), Env: pkg.Env,
		Runtime: "uv", Dir: binDir,
	}, nil
}

// pickExecutable 在 dir 里选和包同名的命令，没有就按名字排第一个。
func pickExecutable(dir, pkg string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasPrefix(n, ".") {
			continue
		}
		if strings.TrimSuffix(strings.ToLower(n), ".exe") == strings.ToLower(pkg) {
			return filepath.Join(dir, n)
		}
		names = append(names, n)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	return filepath.Join(dir, names[0])
}

// findFile 在 dir 下两层以内找名为 name 的文件。
func findFile(dir, name string) string {
	found := ""
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || found != "" {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() && strings.Count(rel, string(filepath.Separator)) >= 2 {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.EqualFold(d.Name(), name) {
			found = p
		}
		return nil
	})
	return found
}
