// 把部分 JSON 合并进当前配置，并在变化时同步连接登记。

package app

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connection"
)

// ApplyConfigPatch 把 JSON 合并进当前 Config。没出现的键保持原值，半份请求不会清掉 MCP、搜索或 gate。空正文或 null 只把当前配置再存一次。
// raw 去掉首尾空白后，长度为 0 或正文是 null 时只调用 SaveConfig，否则 Unmarshal 进 Config。
func (a *App) ApplyConfigPatch(raw []byte) error {
	if a == nil {
		return nil
	}
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return a.SaveConfig()
	}
	prev := a.Config
	if err := json.Unmarshal(raw, &a.Config); err != nil {
		return err
	}
	if err := a.SaveConfig(); err != nil {
		return err
	}
	a.syncConnectionsFromConfig(prev)
	return nil
}

// otelEndpoint 先看环境变量 YOYO_OTEL_ENDPOINT，再看已选的 OTEL 连接，最后用配置里的端点。
func (a *App) otelEndpoint() string {
	if ep := strings.TrimSpace(os.Getenv("YOYO_OTEL_ENDPOINT")); ep != "" {
		return ep
	}
	if a.Video != nil && a.Video.Conn != nil {
		if c, err := a.Video.Conn.Active(connection.CapOTEL, ""); err == nil {
			if ep := strings.TrimSpace(c.Endpoint); ep != "" {
				return ep
			}
		}
	}
	return strings.TrimSpace(a.Config.Log.OTELEndpoint)
}

// seedConnections 把聊天、搜索、OTEL 和带端点的 MCP 种进连接登记。
func (a *App) seedConnections() {
	if a == nil || a.Video == nil || a.Video.Conn == nil {
		return
	}
	r := a.Video.Conn
	_, _ = r.SeedChat(a.Config.Provider, a.Config.BaseURL, a.Config.Model, a.Config.Models...)
	_, _ = r.SyncChat(a.Config.Provider, a.Config.BaseURL, a.Config.Model, "", a.Config.Models...)
	r.SeedSearch(a.Config.SearchURL, a.Config.SearchKey)
	r.SeedOTEL(a.Config.Log.OTELEndpoint)
	for _, m := range a.Config.MCP {
		if strings.TrimSpace(m.Endpoint) != "" {
			r.SeedMCP(m.Name, m.Endpoint)
		}
	}
}

// syncConnectionsFromConfig 在聊天、搜索、OTEL 或 MCP 端点相对旧配置有变化时写回连接登记。
// prev 用来和当前 Config 比较 Provider、BaseURL、Model、Models、SearchURL、SearchKey 和 Log.OTELEndpoint。
func (a *App) syncConnectionsFromConfig(prev Config) {
	if a.Video == nil || a.Video.Conn == nil {
		return
	}
	r := a.Video.Conn
	if prev.Provider != a.Config.Provider || prev.BaseURL != a.Config.BaseURL || prev.Model != a.Config.Model || !sameStrings(prev.Models, a.Config.Models) {
		_, _ = r.SyncChat(a.Config.Provider, a.Config.BaseURL, a.Config.Model, "", a.Config.Models...)
	}
	if prev.SearchURL != a.Config.SearchURL || prev.SearchKey != a.Config.SearchKey {
		r.SyncSearch(a.Config.SearchURL, a.Config.SearchKey)
	}
	if prev.Log.OTELEndpoint != a.Config.Log.OTELEndpoint {
		r.SeedOTEL(a.Config.Log.OTELEndpoint)
	}
	for _, m := range a.Config.MCP {
		if strings.TrimSpace(m.Endpoint) != "" {
			r.SeedMCP(m.Name, m.Endpoint)
		}
	}
}

// sameStrings 判断两条字符串切片是否等长且逐项相同。
// b 与 a 比长度，并逐项比较。
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
