//go:build !windows

// 本文件是 Windows 以外的工作区和虚拟屏幕。两个函数都返回 0。尺寸公式在 size.go。

package winsize

// PlatformWorkArea 在非 Windows 上返回 0, 0。
func PlatformWorkArea() (int, int) { return 0, 0 }

// PlatformVirtualScreen 在非 Windows 上返回四个 0。
func PlatformVirtualScreen() (x, y, w, h int) { return 0, 0, 0, 0 }
