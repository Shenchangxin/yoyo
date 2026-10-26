// 本文件只回答两件事：手机 WebSocket 上哪些方法直接拒绝，哪些方法必须带 confirm。
// 不执行 RPC。onRPC 验签之后才调用。浏览器那扇明文 HTTP 走 api.Dispatch，不经过这里。

package remote

import "strings"

// PhoneDenied 判断这个方法是不是只能在电脑上做。
// remote.pair.begin、remote.pair.approve、remote.pair.deny、remote.disable、remote.enable 返回真。
// remote.revoke 和 thread.model.set 返回假，已配对的手机可以收回设备，也可以换模型。
// method 去掉空白后等于 remote.pair.begin、remote.pair.approve、remote.pair.deny、remote.disable 或 remote.enable 时返回真。
func PhoneDenied(method string) bool {
	switch strings.TrimSpace(method) {
	case "remote.pair.begin", "remote.pair.approve", "remote.pair.deny", "remote.disable", "remote.enable":
		return true
	default:
		return false
	}
}

// NeedsConfirm 判断手机调用这个方法时必须带上 confirm 真。
// 为真的是 key.set、provider.upsert、provider.delete、workspace.apply_hunks、workspace.reverse_hunks、harness.checkout、harness.rollback、evolve.run、update.apply、mcp.replace、thread.delete、computer.allow、pages.delete、packs.uninstall、skills.uninstall、video.providers.upsert、video.providers.delete、connections.upsert、connections.delete。
// thread.model.set 不在里面。onRPC 先看 PhoneDenied，再看这里。
// method 去掉空白后与 key.set、provider.upsert、provider.delete、workspace.apply_hunks、workspace.reverse_hunks、harness.checkout、harness.rollback、evolve.run、update.apply、mcp.replace、thread.delete、computer.allow、pages.delete、packs.uninstall、skills.uninstall、video.providers.upsert、video.providers.delete、connections.upsert、connections.delete 比较，命中返回真。
func NeedsConfirm(method string) bool {
	switch strings.TrimSpace(method) {
	case "key.set", "provider.upsert", "provider.delete",
		"workspace.apply_hunks", "workspace.reverse_hunks",
		"harness.checkout", "harness.rollback", "evolve.run", "update.apply",
		"mcp.replace", "thread.delete", "computer.allow", "pages.delete",
		"packs.uninstall", "skills.uninstall",
		"video.providers.upsert", "video.providers.delete",
		"connections.upsert", "connections.delete":
		return true
	default:
		return false
	}
}
