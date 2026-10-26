//go:build !windows

package desktop

import "github.com/wailsapp/wails/v3/pkg/application"

// applyCompanionShape 在非 Windows 上不裁窗口。
func applyCompanionShape(application.Window, bool) {}

// setCompanionPassthrough 在非 Windows 上调用 SetIgnoreMouseEvents。窗口为空时返回。
func setCompanionPassthrough(win application.Window, ignore bool) {
	if win == nil {
		return
	}
	win.SetIgnoreMouseEvents(ignore)
}

// companionCursorLocalNative 在非 Windows 上返回假，改由 CSS 光标换算。
func companionCursorLocalNative(application.Window) (lx, ly, ww, wh float64, ok bool) {
	return 0, 0, 0, 0, false
}

// leftMouseDown 在非 Windows 上返回假，拖动不靠这条路径。
func leftMouseDown() bool { return false }
