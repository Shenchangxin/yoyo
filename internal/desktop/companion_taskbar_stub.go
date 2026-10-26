//go:build !windows

package desktop

import "github.com/wailsapp/wails/v3/pkg/application"

// hideCompanionTaskbar 在非 Windows 上不做任务栏处理。
func hideCompanionTaskbar(application.Window) {}
