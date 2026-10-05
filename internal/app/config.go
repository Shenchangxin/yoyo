package app

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connection"
)

// ApplyConfigPatch merges JSON into the live Config. Omitted keys keep their
// current values, so a partial Wails/HTTP body cannot wipe MCP, search, or gate.
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
