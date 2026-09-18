//go:build !windows

package winsize

func PlatformWorkArea() (int, int) { return 0, 0 }

func PlatformVirtualScreen() (x, y, w, h int) { return 0, 0, 0, 0 }
