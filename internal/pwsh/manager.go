// Package pwsh 把智能体 shell 钉在便携版 PowerShell 7.6.7 上。archiveName 只接受 windows/amd64 和 windows/arm64。
// Look 先看 YOYO_PWSH，再看 runtime/pwsh/7.6.7，不搜 PATH。
// candidateURLs 的顺序是：YOYO_PWSH_URL、ModelScope resolve、ModelScope API、GitHub release，再把同一 GitHub 地址加上 https://ghfast.top/ 前缀。空地址和重复地址丢掉。
// 单个 zip 上限 maxZipBytes 是 200<<20。解压后累计上限 maxUncompressedBytes 是 512<<20。
package pwsh

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// HTTPDoer 是下载用的 HTTP 客户端。测试可以换成别的实现。
type HTTPDoer interface {
	// Do 发出请求并返回响应。
	Do(*http.Request) (*http.Response, error)
}

// HubFunc 把安装快照发给界面。快照里有 phase、percent 和 error。
type HubFunc func(map[string]any)

// Manager 管一棵边车安装树，目录是 runtime/pwsh/<Version>。
type Manager struct {
	// Dir 是 runtime 根目录。空的时候 runtimeDir 改看 YOYO_HOME 或用户主目录下的 .yoyo/runtime。
	Dir string
	// GOOS 是目标系统。空则用 runtime.GOOS。
	GOOS string
	// GOARCH 是目标架构。空则用 runtime.GOARCH。
	GOARCH string
	// HTTP 是下载客户端。空则 newHTTP。
	HTTP HTTPDoer
	// Verify 检查解压出的二进制。空则 verifyBin。
	Verify func(bin string) error
	// Hub 在进度变化时收到 Snapshot。空则 publish 直接返回。
	Hub HubFunc

	// mu 保护下面的进度字段。
	mu sync.Mutex
	// phase 是 resolving、downloading、extracting、verifying、ready、error、idle 之一，或空字符串。
	phase string
	// percent 是 0 到 100 的进度。下载阶段最高写到 90。
	percent int
	// bytes 是已下载字节。
	bytes int64
	// total 是 Content-Length，没有则保持 0。
	total int64
	// err 是最近一次安装错误的文案。
	err string
	// archive 是当前压缩包文件名。
	archive string
	// wait 在安装进行中非空。beginInstall 把同一个通道交给后来的等待者。
	wait chan struct{}
}

var (
	// activeMu 保护进程级 active。
	activeMu sync.Mutex
	// active 是 Configure 写上的进程级 Manager。current 见到 nil 就 New("")。
	active *Manager
)

// defaultOSArch 返回本进程的 GOOS 和 GOARCH。
func defaultOSArch() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}

// New 用 runtimeDir 作为 Dir，填上本机 GOOS、GOARCH、newHTTP 和 verifyBin。
func New(runtimeDir string) *Manager {
	goos, goarch := defaultOSArch()
	return &Manager{
		Dir:    runtimeDir,
		GOOS:   goos,
		GOARCH: goarch,
		HTTP:   newHTTP(),
		Verify: verifyBin,
	}
}

// Configure 把进程级 Manager 设到 runtimeDir，并挂上 hub。Look 和 Ensure 之后都走这一份。
func Configure(runtimeDir string, hub HubFunc) {
	m := New(runtimeDir)
	m.Hub = hub
	activeMu.Lock()
	active = m
	activeMu.Unlock()
}

// current 返回 Configure 设过的 Manager。还没有时返回 New("")。
func current() *Manager {
	activeMu.Lock()
	m := active
	activeMu.Unlock()
	if m != nil {
		return m
	}
	return New("")
}

// platform 返回目标 GOOS 和 GOARCH。字段为空时用 runtime 的值。
func (m *Manager) platform() (string, string) {
	goos, goarch := m.GOOS, m.GOARCH
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	return goos, goarch
}

// runtimeDir 返回安装根。Dir 非空就用 Dir。否则 YOYO_HOME/runtime，再否则用户主目录下的 .yoyo/runtime。主目录也失败时用 ./runtime。
func (m *Manager) runtimeDir() string {
	if m != nil && strings.TrimSpace(m.Dir) != "" {
		return m.Dir
	}
	if v := strings.TrimSpace(os.Getenv("YOYO_HOME")); v != "" {
		return filepath.Join(v, "runtime")
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "runtime")
	}
	return filepath.Join(h, ".yoyo", "runtime")
}

// installDir 是 runtimeDir/pwsh/<Version>。
func (m *Manager) installDir() string {
	return filepath.Join(m.runtimeDir(), "pwsh", Version)
}

// client 返回 HTTP。空则 newHTTP。
func (m *Manager) client() HTTPDoer {
	if m != nil && m.HTTP != nil {
		return m.HTTP
	}
	return newHTTP()
}

// verify 返回校验函数。空则 verifyBin。
func (m *Manager) verify() func(string) error {
	if m != nil && m.Verify != nil {
		return m.Verify
	}
	return verifyBin
}

// errUnsupported 报告 archiveName 不接受的 GOOS/GOARCH，文案以 pwsh: unsupported platform 开头。
func errUnsupported(goos, goarch string) error {
	return fmt.Errorf("pwsh: unsupported platform %s/%s", goos, goarch)
}

// verifyBin 用 90 秒超时执行 bin -NoLogo -v。环境加上 POWERSHELL_TELEMETRY_OPTOUT=1 和 POWERSHELL_UPDATECHECK=Off。
// 命令失败就返回 pwsh: verify。输出小写后既不含 powershell 也不含 Version 时返回 pwsh: unexpected -Version output。
// 既不含 Version 也不含 7.6 时返回 pwsh: version pin。bin 空白时返回 pwsh: extracted binary missing。
func verifyBin(bin string) error {
	if strings.TrimSpace(bin) == "" {
		return fmt.Errorf("pwsh: extracted binary missing")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-NoLogo", "-v")
	cmd.Env = append(os.Environ(), "POWERSHELL_TELEMETRY_OPTOUT=1", "POWERSHELL_UPDATECHECK=Off")
	osutil.HideConsole(cmd)
	out, err := cmd.CombinedOutput()
	s := strings.ToLower(string(out))
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("pwsh: verify: %s", msg)
	}
	if !strings.Contains(s, "powershell") && !strings.Contains(s, strings.ToLower(Version)) {
		return fmt.Errorf("pwsh: unexpected -Version output")
	}
	if !strings.Contains(s, strings.ToLower(Version)) && !strings.Contains(s, "7.6") {
		return fmt.Errorf("pwsh: version pin %s not found in %q", Version, strings.TrimSpace(string(out)))
	}
	return nil
}
