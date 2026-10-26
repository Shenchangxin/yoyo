//go:build !windows

package desktop

// registerWakeHotkey 在非 Windows 上不注册全局热键。
func registerWakeHotkey(s *Service) {}
