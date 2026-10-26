// 桌面 PowerShell 运行时的配置和状态。安装细节在 internal/pwsh。

package app

import (
	"time"

	"github.com/Shenchangxin/yoyo/internal/pwsh"
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/update"
)

// configurePwsh 在 Home 为空时直接返回。否则按 Home.Runtime 配置 pwsh，并把快照发到 Hub 的 system/pwsh 事件。桌面系统再 Prefetch。
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

// PwshStatus 返回 pwsh.Snapshot。
func (a *App) PwshStatus() map[string]any {
	return pwsh.Snapshot()
}

// PwshInstall 返回 pwsh.Install 的结果。
func (a *App) PwshInstall() map[string]any {
	return pwsh.Install()
}
