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

// HTTPDoer is the download/list client. Tests swap it.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// HubFunc publishes install snapshots (phase/percent/error) to the UI.
type HubFunc func(map[string]any)

// Manager is one sidecar install tree: $YOYO_HOME/runtime/pwsh/<Version>.
type Manager struct {
	Dir    string
	GOOS   string
	GOARCH string
	HTTP   HTTPDoer
	Verify func(bin string) error
	Hub    HubFunc

	mu      sync.Mutex
	phase   string
	percent int
	bytes   int64
	total   int64
	err     string
	archive string
	wait    chan struct{}
}

var (
	activeMu sync.Mutex
	active   *Manager
)

func defaultOSArch() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}

// New builds a manager rooted at runtimeDir ($YOYO_HOME/runtime).
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

// Configure sets the process-wide manager. App.Open calls this with the
// home runtime directory so Look/Ensure never wander into another profile.
func Configure(runtimeDir string, hub HubFunc) {
	m := New(runtimeDir)
	m.Hub = hub
	activeMu.Lock()
	active = m
	activeMu.Unlock()
}

func current() *Manager {
	activeMu.Lock()
	m := active
	activeMu.Unlock()
	if m != nil {
		return m
	}
	return New("")
}

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

func (m *Manager) installDir() string {
	return filepath.Join(m.runtimeDir(), "pwsh", Version)
}

func (m *Manager) client() HTTPDoer {
	if m != nil && m.HTTP != nil {
		return m.HTTP
	}
	return newHTTP()
}

func (m *Manager) verify() func(string) error {
	if m != nil && m.Verify != nil {
		return m.Verify
	}
	return verifyBin
}

func errUnsupported(goos, goarch string) error {
	return fmt.Errorf("pwsh: unsupported platform %s/%s", goos, goarch)
}

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
