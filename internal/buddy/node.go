// node.go 拥有托管 Node：按 YOYO_NODE_MIRROR、npmmirror、nodejs.org 的顺序下载钉死的版本，校验 sha256，解到 runtime/node/<版本>，
// 装完用 node --version 确认。musl 等没有官方包的系统退回系统里的 Node。npm 托管安装和启动 node 脚本的 agent 都用它。

package buddy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/fetch"
)

// nodeAsset 返回本平台的归档后缀；musl 和不认识的架构没有。
func nodeAsset(p Platform) (string, bool) {
	if p.Musl {
		return "", false
	}
	arch := map[string]string{"amd64": "x64", "arm64": "arm64"}[p.GOARCH]
	if arch == "" {
		return "", false
	}
	var key string
	switch p.GOOS {
	case "windows":
		key = "win-" + arch + ".zip"
	case "darwin":
		key = "darwin-" + arch + ".tar.gz"
	case "linux":
		key = "linux-" + arch + ".tar.gz"
	default:
		return "", false
	}
	_, ok := nodeSHA256[key]
	return key, ok
}

// managedNodeDir 是托管 Node 的目录。
func (m *Manager) managedNodeDir() string {
	if m.opts.RuntimeDir == "" {
		return ""
	}
	return filepath.Join(m.opts.RuntimeDir, "node", nodeVersion)
}

// managedNodeExe 是托管 node 可执行文件的路径；没配置运行时目录时为空。
func (m *Manager) managedNodeExe() string {
	dir := m.managedNodeDir()
	if dir == "" {
		return ""
	}
	if m.plat.GOOS == "windows" {
		return filepath.Join(dir, "node.exe")
	}
	return filepath.Join(dir, "bin", "node")
}

// npmCLIFor 返回和 node 同一套安装里的 npm-cli.js。Windows 在 node_modules 下，Unix 在 lib/node_modules 下。
func npmCLIFor(nodeExe, goos string) string {
	dir := filepath.Dir(nodeExe)
	if goos == "windows" {
		return filepath.Join(dir, "node_modules", "npm", "bin", "npm-cli.js")
	}
	return filepath.Join(filepath.Dir(dir), "lib", "node_modules", "npm", "bin", "npm-cli.js")
}

// nodeMirrors 返回下载 Node 的地址前缀，按顺序试。
func (m *Manager) nodeMirrors() []string {
	if len(m.opts.NodeMirrors) > 0 {
		return m.opts.NodeMirrors
	}
	var out []string
	if v := strings.TrimSpace(os.Getenv("YOYO_NODE_MIRROR")); strings.HasPrefix(v, "https://") {
		out = append(out, withSlash(v))
	}
	return append(out, "https://npmmirror.com/mirrors/node/", "https://nodejs.org/dist/")
}

// ensureNode 准备好 node 和它的 npm-cli.js。托管版已装好且版本对就直接用；本平台没有官方包时退回系统 Node。
func (m *Manager) ensureNode(ctx context.Context, prog func(Progress), log io.Writer) (string, string, error) {
	m.nodeMu.Lock()
	defer m.nodeMu.Unlock()
	exe := m.managedNodeExe()
	if exe != "" && fileExists(exe) && m.nodeOK(ctx, exe) {
		return exe, npmCLIFor(exe, m.plat.GOOS), nil
	}
	asset, ok := nodeAsset(m.plat)
	if !ok || exe == "" {
		if sys, err := m.opts.LookPath("node"); err == nil {
			if cli := npmCLIFor(sys, m.plat.GOOS); fileExists(cli) {
				fmt.Fprintf(log, "using system node %s\n", sys)
				return sys, cli, nil
			}
		}
		return "", "", errors.New(m.say("此系统不支持托管 Node，先装好系统的 Node.js", "managed Node is not available here; install Node.js first"))
	}
	prog(Progress{Phase: "node"})
	name := "node-" + nodeVersion + "-" + asset
	var urls []string
	for _, mirror := range m.nodeMirrors() {
		urls = append(urls, withSlash(mirror)+nodeVersion+"/"+name)
	}
	dir := m.managedNodeDir()
	archive := filepath.Join(filepath.Dir(dir), ".download", name)
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		return "", "", err
	}
	defer os.RemoveAll(filepath.Dir(archive))
	fmt.Fprintf(log, "download node %s\n", name)
	_, err := fetch.Download(ctx, fetch.Spec{
		URLs: urls, SHA256: nodeSHA256[asset], MaxBytes: 256 << 20, Dest: archive, Client: m.opts.HTTP,
		Progress: func(done, total int64) {
			if total > 0 {
				prog(Progress{Phase: "node", Percent: int(done * 30 / total)})
			}
		},
	})
	if err != nil {
		return "", "", fmt.Errorf("node download: %w", err)
	}
	if err := fetch.Extract(archive, dir, fetch.KindOf(name), 1); err != nil {
		return "", "", fmt.Errorf("node extract: %w", err)
	}
	fetch.Unblock(dir)
	if !m.nodeOK(ctx, exe) {
		_ = os.RemoveAll(dir)
		return "", "", errors.New(m.say("托管 Node 校验失败", "managed Node failed its version check"))
	}
	return exe, npmCLIFor(exe, m.plat.GOOS), nil
}

// nodeOK 跑 node --version，输出是钉死的版本才算好。
func (m *Manager) nodeOK(ctx context.Context, exe string) bool {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := m.opts.Run(ctx, Cmd{Launch: acp.Launch{Command: exe, Args: []string{"--version"}, Env: m.baseEnv(nil)}})
	return err == nil && strings.TrimSpace(string(out)) == nodeVersion
}
