// 聊天供应商探测的薄入口。TestProvider 不带连接 id，密钥和请求在 chat_providers.go。

package app

import "github.com/Shenchangxin/yoyo/internal/connection"

// TestProvider 用空 id 调用 TestChatProvider，探测当前默认聊天连接。
func (a *App) TestProvider() map[string]any {
	return a.TestChatProvider("")
}

// providerProbeError 把 HTTP 状态和网关原文交给 connection.ProbeError。
// status 传给 connection.ProbeError。
// raw 传给 connection.ProbeError。
func providerProbeError(status int, raw string) string {
	return connection.ProbeError(status, raw)
}
