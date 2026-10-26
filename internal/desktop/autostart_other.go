//go:build !windows && !darwin

package desktop

import (
	"os"
	"path/filepath"
)

// SetStartAtLogin 在打开时把 yoyo.desktop 写到 ~/.config/autostart，Exec 是当前可执行文件。关闭时删掉该文件并忽略删除错误。路径为空则返回 nil。
// enabled 为假时删除 yoyo.desktop 并忽略删除错误。为真时在可执行文件路径非空的情况下写入该文件。
func SetStartAtLogin(enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", "autostart")
	path := filepath.Join(dir, "yoyo.desktop")
	if !enabled {
		_ = os.Remove(path)
		return nil
	}
	exe := executablePath()
	if exe == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	body := `[Desktop Entry]
Type=Application
Name=Yoyo
Exec=` + exe + `
X-GNOME-Autostart-enabled=true
`
	return os.WriteFile(path, []byte(body), 0o644)
}
