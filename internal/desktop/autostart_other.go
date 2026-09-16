//go:build !windows && !darwin

package desktop

import (
	"os"
	"path/filepath"
)

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
