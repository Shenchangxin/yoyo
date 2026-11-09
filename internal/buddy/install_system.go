// install_system.go 拥有装到系统：npm 项用系统里的 npm 跑 npm install -g（输出进安装日志）；二进制项把精选表里的官方安装命令
// 放进可见终端执行（界面先把完整命令给人确认）。条件不满足时 systemInstall.reason 给出原因。

package buddy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// systemPackage 返回装到系统要用的 npm 包和版本：精选表写了包名就用它；注册表只有 npx 分发的也用 npx 的包。
func systemPackage(e Entry) (string, string) {
	if e.System.NPM != "" {
		if e.Dist.NPX != nil {
			name, version := splitPackage(e.Dist.NPX.Package)
			if name == e.System.NPM {
				return name, version
			}
		}
		return e.System.NPM, "latest"
	}
	if e.Dist.NPX != nil && len(e.Dist.Binary) == 0 {
		return splitPackage(e.Dist.NPX.Package)
	}
	return "", ""
}

// systemInstallView 说明能不能装到系统。调用方持有 m.mu（读 sysNPM）。
func (m *Manager) systemInstallView(e Entry) *SystemInstallView {
	if e.Builtin || e.Custom {
		return &SystemInstallView{}
	}
	if declaredUnsupported(e, m.plat) {
		return &SystemInstallView{Reason: m.say("此系统不可用", "Not available on this system")}
	}
	if pkg, version := systemPackage(e); pkg != "" && m.sysNPM != "" {
		return &SystemInstallView{Available: true, Command: "npm install -g " + pkg + "@" + firstNonEmpty(version, "latest")}
	}
	if cmd := e.System.Command(m.plat.GOOS); cmd != "" {
		return &SystemInstallView{Available: true, Command: cmd}
	}
	if pkg, _ := systemPackage(e); pkg != "" {
		return &SystemInstallView{Reason: m.say("需要系统 Node.js", "Needs Node.js on the system")}
	}
	return &SystemInstallView{Reason: m.say("没有官方的系统安装方式", "No official system install")}
}

// installSystem 装到系统。npm 项直接跑；其余在可见终端里跑官方命令，然后等它出现在 PATH 上。
func (m *Manager) installSystem(ctx context.Context, e Entry, prog func(Progress), log io.Writer) error {
	m.mu.Lock()
	npm := m.sysNPM
	m.mu.Unlock()
	if pkg, version := systemPackage(e); pkg != "" && npm != "" {
		prog(Progress{Phase: "deps", Percent: 30})
		registry := m.npmRegistryFor(ctx, pkg, version)
		spec := pkg + "@" + firstNonEmpty(version, "latest")
		fmt.Fprintf(log, "npm install -g %s --registry %s\n", spec, registry)
		l := m.systemNPM(npm, "install", "-g", spec, "--registry", registry, "--no-audit", "--no-fund", "--loglevel=error")
		if _, err := m.opts.Run(ctx, Cmd{Launch: l, Output: log}); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("npm install -g: %w", err)
		}
		m.paths.reset()
		return nil
	}
	cmd := e.System.Command(m.plat.GOOS)
	if cmd == "" {
		return errors.New(m.say("没有官方的系统安装方式", "No official system install"))
	}
	fmt.Fprintf(log, "terminal: %s\n", cmd)
	prog(Progress{Phase: "deps", Percent: 30, Line: cmd})
	res, err := m.opts.Terminal(ctx, acp.Launch{Command: shellFor(m.plat.GOOS), Args: shellArgs(m.plat.GOOS, cmd), Env: withoutEnv(m.baseEnv(nil), "TERM", "NO_COLOR")})
	if err != nil {
		return err
	}
	if res.Copy != "" {
		return fmt.Errorf("%s %s", m.say("没有可用的终端，手动运行：", "No terminal available; run it yourself:"), res.Copy)
	}
	return m.waitDetected(ctx, e, res.Waited)
}

// waitDetected 等系统版出现在 PATH 上。终端已经结束时只看一次；否则每 5 秒看一次，最多 10 分钟。
func (m *Manager) waitDetected(ctx context.Context, e Entry, waited bool) error {
	deadline := time.Now().Add(10 * time.Minute)
	for {
		m.paths.reset()
		if d := m.detectFast(e); d.SystemPath != "" {
			return nil
		}
		if waited || time.Now().After(deadline) {
			return errors.New(m.say("装完后没在 PATH 上找到它，重开 yoyo 或手动检查", "not found on PATH after installing; restart yoyo or check manually"))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// systemNPM 返回跑系统 npm 的启动方式。Windows 上 npm 旁边有 node.exe 和 npm-cli.js 时直接用 node 跑，避开 npm.cmd。
func (m *Manager) systemNPM(npm string, args ...string) acp.Launch {
	env := m.baseEnv([]string{filepath.Dir(npm)})
	if m.plat.GOOS == "windows" {
		dir := filepath.Dir(npm)
		node := filepath.Join(dir, "node.exe")
		cli := filepath.Join(dir, "node_modules", "npm", "bin", "npm-cli.js")
		if fileExists(node) && fileExists(cli) {
			return acp.Launch{Command: node, Args: append([]string{cli}, args...), Env: env}
		}
	}
	l := commandFor(npm, args, m.nodePath)
	l.Env = env
	return l
}

// shellFor 返回跑一整行命令的 shell。
func shellFor(goos string) string {
	if goos == "windows" {
		if c := os.Getenv("ComSpec"); c != "" {
			return c
		}
		return "cmd.exe"
	}
	return "/bin/sh"
}

// shellArgs 返回让 shell 跑 line 的参数。
func shellArgs(goos, line string) []string {
	if goos == "windows" {
		return []string{"/d", "/s", "/c", line}
	}
	return []string{"-c", line}
}

// reset 清掉目录表缓存，刚装完东西时重新找。
func (p *PathFinder) reset() {
	p.mu.Lock()
	p.dirs = nil
	p.mu.Unlock()
}
