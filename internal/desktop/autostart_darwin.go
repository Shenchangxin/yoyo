//go:build darwin

package desktop

import (
	"os"
	"path/filepath"
)

// SetStartAtLogin 在打开时把 com.yoyo.workstation.plist 写到 ~/Library/LaunchAgents，RunAtLoad 为真，程序是当前可执行文件。关闭时删除该 plist。可执行文件路径为空则返回 nil。
// enabled 为假时删除该 plist。为真时在可执行文件路径非空的情况下写入它。
func SetStartAtLogin(enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, "Library", "LaunchAgents")
	path := filepath.Join(dir, "com.yoyo.workstation.plist")
	if !enabled {
		return os.Remove(path)
	}
	exe := executablePath()
	if exe == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	body := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.yoyo.workstation</string>
  <key>ProgramArguments</key>
  <array><string>` + exe + `</string></array>
  <key>RunAtLoad</key><true/>
</dict>
</plist>
`
	_ = os.WriteFile(path, []byte(body), 0o644)
	return nil
}
