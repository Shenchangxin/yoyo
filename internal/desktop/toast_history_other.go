//go:build !windows

package desktop

// clearToastHistory 在非 Windows 上返回 nil。其他平台没有操作中心历史可清。
func clearToastHistory(string) error { return nil }
