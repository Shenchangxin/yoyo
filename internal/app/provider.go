package app

import "github.com/Shenchangxin/yoyo/internal/connection"

func (a *App) TestProvider() map[string]any {
	return a.TestChatProvider("")
}

func providerProbeError(status int, raw string) string {
	return connection.ProbeError(status, raw)
}
