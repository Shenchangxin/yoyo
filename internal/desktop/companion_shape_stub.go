//go:build !windows

package desktop

import "github.com/wailsapp/wails/v3/pkg/application"

func applyCompanionShape(application.Window, bool) {}

func setCompanionPassthrough(win application.Window, ignore bool) {
	if win == nil {
		return
	}
	win.SetIgnoreMouseEvents(ignore)
}

func companionCursorLocalNative(application.Window) (lx, ly, ww, wh float64, ok bool) {
	return 0, 0, 0, 0, false
}

func leftMouseDown() bool { return false }
