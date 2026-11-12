// launcher.go 拥有一项的启动方式：按 mode 在托管和系统版之间选来源，参数是注册表 args 加用户 args，
// 环境是清洗后的白名单加上注册表 env、用户的非密钥变量和这一项自己的密钥。探测、登录和工人都从这里拿 acp.Launch。

package buddy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// Launch 返回启动 id 的命令和环境。Dir 留空，由调用方设成工单目录。yoyo 自己不经这里。
func (m *Manager) Launch(id string) (acp.Launch, error) {
	l, _, err := m.launch(id)
	return l, err
}

// LaunchRedacted 和 Launch 一样，另外返回这一项的密钥值，调用方打码日志时用。
func (m *Manager) LaunchRedacted(id string) (acp.Launch, []string, error) {
	return m.launch(id)
}

// launch 是 Launch 的实现。
func (m *Manager) launch(id string) (acp.Launch, []string, error) {
	settings := m.opts.Settings()
	m.mu.Lock()
	e, ok := m.entryLocked(id)
	d := m.det[id]
	m.mu.Unlock()
	if !ok || e.Builtin {
		return acp.Launch{}, nil, ErrNotFound
	}
	return m.launchFor(e, settings.Item(id), d)
}

// launchFor 拼出启动方式。来源：mode 为 system 且系统版在时用系统版；否则托管优先，没有托管再用系统版。
func (m *Manager) launchFor(e Entry, item ItemConfig, d detection) (acp.Launch, []string, error) {
	var l acp.Launch
	extraEnv := map[string]string{}
	var pathPrefix []string
	useSystem := d.SystemPath != "" && (item.Mode == "system" || d.Managed == nil)
	switch {
	case e.Custom:
		if d.SystemPath == "" {
			return acp.Launch{}, nil, ErrNotInstalled
		}
		l = commandFor(d.SystemPath, append([]string(nil), e.Args...), m.nodePath)
		for k, v := range e.Env {
			extraEnv[k] = v
		}
	case useSystem:
		args, env := registryArgs(e, m.plat)
		l = commandFor(d.SystemPath, args, m.nodePath)
		for k, v := range env {
			extraEnv[k] = v
		}
	case d.Managed != nil:
		rec := d.Managed
		l = acp.Launch{Command: rec.Command, Args: append([]string(nil), rec.Args...), ViaCmd: rec.ViaCmd}
		if rec.Entry != "" && !fileExists(rec.Command) {
			node, err := m.nodePath()
			if err != nil {
				return acp.Launch{}, nil, fmt.Errorf("%w: node runtime missing", ErrNotInstalled)
			}
			l.Command = node
		}
		if rec.Runtime == "node" {
			node := l.Command
			if rec.Entry == "" {
				node, _ = m.nodePath()
			}
			if node != "" {
				pathPrefix = append(pathPrefix, filepath.Dir(node))
			}
		}
		for k, v := range rec.Env {
			extraEnv[k] = v
		}
	default:
		return acp.Launch{}, nil, ErrNotInstalled
	}
	// 适配器能改用外部 CLI、本机又装着一个时，把它的路径交给适配器：用的是人已经装好的那一个。
	// 文件不在了就不交，适配器退回自带的；人在环境变量里自己写了同名变量时，下面会用人写的盖掉。
	if !e.Custom && e.CLIEnv != "" && d.CLIPath != "" && !item.NoHostCLI && fileExists(d.CLIPath) {
		extraEnv[e.CLIEnv] = d.CLIPath
	}
	l.Args = append(l.Args, item.Args...)
	for k, v := range item.Env {
		if !SecretLike(k) {
			extraEnv[k] = v
		}
	}
	secrets, values := m.secretEnv(e, item)
	for k, v := range secrets {
		extraEnv[k] = v
	}
	l.Env = m.baseEnv(pathPrefix)
	for k, v := range extraEnv {
		l.Env = setEnv(l.Env, k, v)
	}
	if l.Command == "" {
		return acp.Launch{}, nil, errors.New("buddy: empty command")
	}
	return l, values, nil
}

// registryArgs 返回注册表给的启动参数和环境：本平台有二进制就用它的，其次 npx，最后 uvx。
func registryArgs(e Entry, p Platform) ([]string, map[string]string) {
	if t, _, ok := pickTarget(e.Dist.Binary, p); ok {
		return append([]string(nil), t.Args...), t.Env
	}
	if e.Dist.NPX != nil {
		return append([]string(nil), e.Dist.NPX.Args...), e.Dist.NPX.Env
	}
	if e.Dist.UVX != nil {
		return append([]string(nil), e.Dist.UVX.Args...), e.Dist.UVX.Env
	}
	return nil, nil
}

// baseEnv 是清洗后的环境，PATH 换成扩展过的目录表，prefix 里的目录放在最前面。
func (m *Manager) baseEnv(prefix []string) []string {
	env := BuildEnv(m.opts.BaseEnv())
	sep := ":"
	if m.plat.GOOS == "windows" {
		sep = ";"
	}
	parts := append(append([]string(nil), prefix...), m.paths.PathValue())
	value := strings.Join(parts, sep)
	if strings.TrimSpace(value) == "" {
		value = getEnv(env, "PATH")
	}
	return setEnv(env, "PATH", value)
}

// nodePath 返回能用的 node：先托管版，再系统版。
func (m *Manager) nodePath() (string, error) {
	if p := m.managedNodeExe(); p != "" && fileExists(p) {
		return p, nil
	}
	return m.opts.LookPath("node")
}

// probeDir 是探测时的工作目录 buddies/.probe/<id>。
func (m *Manager) probeDir(id string) string {
	dir := filepath.Join(m.opts.Root, ".probe", id)
	if m.opts.Root == "" {
		dir = filepath.Join(os.TempDir(), "yoyo-buddy-probe", id)
	}
	_ = os.MkdirAll(dir, 0o755)
	return dir
}
