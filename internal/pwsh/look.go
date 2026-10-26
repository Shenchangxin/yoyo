// 本文件按 YOYO_PWSH、再按托管目录查找 pwsh.exe。不搜 PATH。

package pwsh

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Look 返回进程级 Manager 的 Look。先环境变量，再托管安装目录。
func Look() string {
	return current().Look()
}

// Look 先看 YOYO_PWSH。环境变量没有可用文件时再看 installDir。
func (m *Manager) Look() string {
	if p := lookEnv(); p != "" {
		return p
	}
	return m.lookManaged()
}

// lookEnv 读取 YOYO_PWSH。空白返回空。它是文件且 isPwshBin 为真就返回该路径；是目录就 lookDir。Stat 失败返回空。
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

// lookManaged 在 installDir 里查找。
func (m *Manager) lookManaged() string {
	return lookDir(m.installDir())
}

// lookDir 先在 dir 本身找 pwsh 可执行文件，再看它的一级子目录。空白目录或读目录失败返回空。
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

// pwshBin 在 Windows 上只找 pwsh.exe，其他系统先找 pwsh.exe 再找 pwsh。命中 isPwshBin 的第一个路径返回。
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

// isPwshBin 在路径存在且不是目录时返回 true。
func isPwshBin(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// Fallback 在 Windows 上返回 powershell，其他系统返回空字符串。这是智能体 shell 的最后退路。
func Fallback() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	return "powershell"
}

// Fact 在 Look 非空时返回 pwsh、Version 和该路径，中间用空格分开。Look 为空且是 Windows 时返回 powershell 5.1。其他情况返回空字符串。
func Fact() string {
	if p := Look(); p != "" {
		return "pwsh " + Version + " " + p
	}
	if runtime.GOOS == "windows" {
		return "powershell 5.1"
	}
	return ""
}
