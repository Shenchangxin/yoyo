//go:build !windows

package desktop

func desktopCursorCSS() (x, y float64, ok bool) {
	return 0, 0, false
}
