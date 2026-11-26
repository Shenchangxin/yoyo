// 门共用的应用方法。桌面、RPC 和 HTTP 调这些，不再各自摸 vault、内核和归档。

package app

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// SetDefaultKey 把去掉空白后的 value 写进 vault 的 default 槽。
// value 先 TrimSpace，再传给 Vault.Set。
func (a *App) SetDefaultKey(value string) {
	a.Vault.Set("default", strings.TrimSpace(value))
}

// VaultStatus 返回 vault 的状态。不含密钥原文。
func (a *App) VaultStatus() map[string]any {
	return a.Vault.Status()
}

// JournalPath 返回日志链文件路径。还没有链时返回空串。
func (a *App) JournalPath() string {
	if a == nil || a.Journal == nil {
		return ""
	}
	return a.Journal.Path
}

// ArchiveList 返回演进归档里的节点。
func (a *App) ArchiveList() []evolve.Node {
	return a.Archive.List()
}

// PluginList 返回已挂上的 fiber、WASM 模块、MCP 服务和工具。
func (a *App) PluginList() map[string]any {
	return map[string]any{
		"fibers": a.Kernel.Fibers(),
		"wasm":   a.WASM.List(),
		"mcp":    a.MCP.Info(),
		"tools":  a.MCP.Tools(),
	}
}

// UnloadFiber 卸下同名 fiber。没有这条 fiber 时改卸同名 WASM 模块。
// name 先交给 Kernel.Fiber，找到就 Dispose，否则传给 WASM.Unload。
func (a *App) UnloadFiber(name string) error {
	if f := a.Kernel.Fiber(name); f != nil {
		return f.Dispose()
	}
	return a.WASM.Unload(name)
}

// HomeRoot 返回数据根目录。Home 为空时返回空串。
func (a *App) HomeRoot() string {
	if a == nil || a.Home == nil {
		return ""
	}
	return a.Home.Root
}

// SubscribeHub 订阅 Hub 上该会话的轨迹事件。sessionID 为 * 时是全部。
// Hub 为空时返回已关闭的通道。sessionID 传给 Hub.Subscribe。
func (a *App) SubscribeHub(sessionID string) (<-chan trace.Event, func()) {
	if a == nil || a.Hub == nil {
		ch := make(chan trace.Event)
		close(ch)
		return ch, func() {}
	}
	return a.Hub.Subscribe(sessionID)
}
