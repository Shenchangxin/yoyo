package pwsh

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Look returns the managed (or YOYO_PWSH) pwsh.exe. It does not search PATH:
// a random system 7.x would drift the dialect the model is pinned to.
func Look() string {
	return current().Look()
}

func (m *Manager) Look() string {
	if p := lookEnv(); p != "" {
		return p
	}
	return m.lookManaged()
}

func lookEnv() string {
	raw := strings.TrimSpace(os.Getenv(EnvBin))
	if raw == "" {
		return ""
	}
	st, err := os.Stat(raw)
	if err != nil {
		return ""
	}
	if !st.IsDir() {
		if isPwshBin(raw) {
			return raw
		}
		return ""
	}
	return lookDir(raw)
}

func (m *Manager) lookManaged() string {
	return lookDir(m.installDir())
}

func lookDir(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	if p := pwshBin(dir); p != "" {
		return p
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if p := pwshBin(filepath.Join(dir, e.Name())); p != "" {
			return p
		}
	}
	return ""
}

func pwshBin(dir string) string {
	cands := []string{"pwsh.exe", "pwsh"}
	if runtime.GOOS == "windows" {
		cands = []string{"pwsh.exe"}
	}
	for _, name := range cands {
		p := filepath.Join(dir, name)
		if isPwshBin(p) {
			return p
		}
	}
	return ""
}

func isPwshBin(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// Fallback is Windows PowerShell 5.1. Desktop WinRT callers must keep using
// powershell.exe directly; this is only the agent-shell last resort.
func Fallback() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	return "powershell"
}

// Fact is the environment-block token: "pwsh 7.6.7 <abs>" or "powershell 5.1".
func Fact() string {
	if p := Look(); p != "" {
		return "pwsh " + Version + " " + p
	}
	if runtime.GOOS == "windows" {
		return "powershell 5.1"
	}
	return ""
}
