//go:build !windows

package desktop

import "github.com/wailsapp/wails/v3/pkg/application"

func hideCompanionTaskbar(application.Window) {}
