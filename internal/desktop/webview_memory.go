package desktop

import "github.com/wailsapp/wails/v3/pkg/application"

// setRendererMemory 直接返回 nil。v3.0.0 的 MemoryUsageTargetLevel 在 internal/webview2，窗口接口调不到。
// 页面隐藏时前端仍会传入 low，这里不改 WebView。
func setRendererMemory(_ application.Window, _ string) error {
	return nil
}
