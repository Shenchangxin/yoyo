//go:build windows

package desktop

import (
	"golang.org/x/sys/windows/registry"
)

func SetStartAtLogin(enabled bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !enabled {
		_ = k.DeleteValue("Yoyo")
		return nil
	}
	exe := executablePath()
	if exe == "" {
		return nil
	}
	return k.SetStringValue("Yoyo", `"`+exe+`"`)
}
