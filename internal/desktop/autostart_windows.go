//go:build windows

package desktop

import (
	"golang.org/x/sys/windows/registry"
)

// SetStartAtLogin 写当前用户的 Run 键，值名是 Yoyo。关闭时删掉该值。打开时写入带引号的可执行文件路径。路径为空则返回 nil。
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
