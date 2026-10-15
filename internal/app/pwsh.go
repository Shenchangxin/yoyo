package app

import (
	"time"

	"github.com/Shenchangxin/yoyo/internal/pwsh"
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/update"
)

func (a *App) configurePwsh() {
	if a == nil || a.Home == nil {
		return
	}
	pwsh.Configure(a.Home.Runtime(), func(snap map[string]any) {
		if a.Hub == nil {
			return
		}
		a.Hub.Publish(trace.Event{
			TS: time.Now().UTC(), Type: trace.TypeSystem, Source: "runtime", ItemKind: "pwsh", Payload: snap,
		})
	})
	if update.DesktopOS() {
		pwsh.Prefetch()
	}
}

func (a *App) PwshStatus() map[string]any {
	return pwsh.Snapshot()
}

func (a *App) PwshInstall() map[string]any {
	return pwsh.Install()
}
