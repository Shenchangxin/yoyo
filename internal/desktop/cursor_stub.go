//go:build !windows

package desktop

// desktopCursorCSS 在非 Windows 上返回假。
func desktopCursorCSS() (x, y float64, ok bool) {
	return 0, 0, false
}
