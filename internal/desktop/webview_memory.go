package desktop

import "github.com/wailsapp/wails/v3/pkg/application"

// setRendererMemory 目前不改 WebView 内存，直接返回 nil。
func setRendererMemory(_ application.Window, _ string) error {
	return nil
}
