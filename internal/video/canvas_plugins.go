package video

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/protocol"
)

func (e *Engine) loadCanvasPlugins() error {
	reg, err := protocol.NewRegistry()
	if err != nil {
		return err
	}
	if builtins := protocol.Builtins(); builtins != nil {
		for _, meta := range builtins.List("", "", true) {
			if ad, ok := builtins.Get(meta.ID); ok {
				_ = reg.Register(ad)
			}
		}
	}
	resolve := func(id string) (protocol.Adapter, bool) {
		if builtins := protocol.Builtins(); builtins != nil {
			return builtins.Resolve(id)
		}
		return nil, false
	}
	dir := e.PluginDir
	if dir != "" {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.yingce-plugin"))
		for _, file := range matches {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			pkg, err := protocol.ParsePluginPackage(data)
			if err != nil {
				continue
			}
			adapters, err := protocol.LoadInstalledProviders(pkg.ManifestRaw, resolve)
			if err != nil {
				continue
			}
			for _, ad := range adapters {
				_ = reg.Register(ad)
			}
		}
		ents, _ := os.ReadDir(dir)
		for _, ent := range ents {
			if !ent.IsDir() {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, ent.Name(), "manifest.json"))
			if err != nil {
				continue
			}
			adapters, err := protocol.LoadInstalledProviders(raw, resolve)
			if err != nil {
				continue
			}
			for _, ad := range adapters {
				_ = reg.Register(ad)
			}
		}
	}
	userDir := filepath.Join(e.Dir, "plugins", "user")
	if matches, _ := filepath.Glob(filepath.Join(userDir, "*.yingce-plugin")); len(matches) > 0 {
		for _, file := range matches {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			pkg, err := protocol.ParsePluginPackage(data)
			if err != nil {
				continue
			}
			adapters, err := protocol.LoadInstalledProviders(pkg.ManifestRaw, resolve)
			if err != nil {
				continue
			}
			for _, ad := range adapters {
				_ = reg.Register(ad)
			}
		}
	}
	e.Proto = reg
	return nil
}

func (e *Engine) installUserPlugin(raw []byte, fileName string) (map[string]any, error) {
	pkg, err := protocol.ParsePluginPackage(raw)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(e.Dir, "plugins", "user")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if fileName == "" {
		fileName = pkg.Manifest.Metadata.ID + ".yingce-plugin"
	}
	path := filepath.Join(dir, filepath.Base(fileName))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return nil, err
	}
	if err := e.loadCanvasPlugins(); err != nil {
		return nil, err
	}
	return e.pluginRecord(pkg.Manifest.Metadata.ID, "uploaded", fileName, "enabled"), nil
}

func (e *Engine) pluginRecord(id, source, fileName, status string) map[string]any {
	name := id
	version := "1"
	author, docs, desc := "", "", ""
	if e.Proto != nil {
		if ad, ok := e.Proto.Resolve(id); ok {
			m := ad.Metadata()
			name = m.Name
			version = m.Version
			id = m.ID
			author = debrandYingce(m.Vendor)
			docs = debrandYingce(m.Documentation)
			desc = debrandYingce(m.Description)
		}
	}
	manifest := map[string]any{
		"apiVersion": "v1",
		"id":         id,
		"name":       name,
		"version":    version,
		"enabled":    true,
	}
	if author != "" {
		manifest["author"] = author
	}
	if docs != "" {
		manifest["documentation"] = docs
	}
	if desc != "" {
		manifest["description"] = desc
	}
	return map[string]any{
		"manifest": manifest,
		"source":     source,
		"fileName":   fileName,
		"package":    fileName,
		"sha256":     "",
		"installedAt": Now(),
		"updatedAt":  Now(),
		"status":     status,
		"management": map[string]any{
			"origin":             sourceOrigin(source),
			"kind":               "protocol",
			"activationScope":    "system",
			"configurationScope": "user",
		},
	}
}

func sourceOrigin(source string) string {
	if source == "bundled" {
		return "official"
	}
	return "uploaded"
}

func (e *Engine) listPluginPayload() map[string]any {
	plugins := []map[string]any{}
	states := map[string]any{}
	seen := map[string]struct{}{}
	if e.Proto != nil {
		for _, meta := range e.Proto.List("", "", true) {
			if _, ok := seen[meta.ID]; ok {
				continue
			}
			seen[meta.ID] = struct{}{}
			rec := e.pluginRecord(meta.ID, "bundled", meta.ID+".yingce-plugin", "enabled")
			plugins = append(plugins, rec)
			states[meta.ID] = map[string]any{
				"pluginId":           meta.ID,
				"platformAvailable":  true,
				"userEnabled":        true,
				"userConfigured":     true,
				"effectiveEnabled":   true,
				"canToggle":          true,
				"canConfigure":       true,
			}
		}
	}
	statuses := map[string]any{}
	for id := range states {
		statuses[id] = "enabled"
	}
	return map[string]any{"plugins": plugins, "states": states, "statuses": statuses}
}

func (e *Engine) pluginCatalog(scope, capability string) map[string]any {
	_ = scope
	items := []map[string]any{}
	if e.Proto != nil {
		var cap protocol.Capability
		switch capability {
		case "image":
			cap = protocol.CapabilityImage
		case "video":
			cap = protocol.CapabilityVideo
		case "audio":
			cap = protocol.CapabilityAudio
		case "text":
			cap = protocol.CapabilityText
		}
		for _, meta := range e.Proto.List(protocol.SurfaceCanvas, cap, true) {
			items = append(items, map[string]any{
				"id": meta.ID, "name": meta.Name, "vendor": debrandYingce(meta.Vendor), "version": meta.Version,
				"categories": meta.Categories, "enabled": meta.Enabled,
			})
		}
	}
	return map[string]any{"providers": items}
}

type canvasChannel struct {
	ID         string `json:"id"`
	PluginID   string `json:"pluginId"`
	Name       string `json:"name"`
	Capability string `json:"capability"`
	BaseURL    string `json:"baseUrl"`
	VaultKey   string `json:"vaultKey"`
	Model      string `json:"model"`
	Models     string `json:"models"`
	Settings   string `json:"settings"`
	SortOrder  int    `json:"sortOrder"`
	Enabled    bool   `json:"enabled"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
	HasKey     bool   `json:"hasKey"`
}

func (e *Engine) seedCanvasChannels() error {
	var n int
	_ = e.DB.QueryRow(`SELECT COUNT(*) FROM canvas_channels`).Scan(&n)
	if n > 0 {
		return nil
	}
	providers, err := e.ListProviders("")
	if err != nil {
		return err
	}
	for i, p := range providers {
		plugin := mapProviderPlugin(p.ServiceType, p.Provider)
		now := Now()
		id := "ch-" + p.ID
		_, _ = e.DB.Exec(`INSERT OR IGNORE INTO canvas_channels(id, plugin_id, name, capability, base_url, vault_key, model, models, settings, sort_order, enabled, created_at, updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, plugin, p.Name, p.ServiceType, p.BaseURL, p.VaultKey, p.Model, p.Models, p.Settings, i, 1, now, now)
	}
	return nil
}

func mapProviderPlugin(serviceType, vendor string) string {
	v := strings.ToLower(vendor)
	switch strings.ToLower(serviceType) {
	case "image":
		switch v {
		case "openai":
			return "openai-images"
		case "gemini":
			return "google-gemini-image"
		case "volcengine":
			return "volcengine-ark-seedream"
		default:
			return "openai-images"
		}
	case "video":
		switch v {
		case "openai":
			return "openai-videos"
		case "gemini":
			return "google-gemini-veo"
		case "minimax", "hailuo":
			return "minimax-hailuo-video-v2"
		case "wan", "dashscope":
			return "dashscope-wan-video"
		case "volcengine", "seedance":
			return "volcengine-ark-seedance"
		default:
			return "openai-videos"
		}
	case "tts", "audio":
		return "openai-audio"
	default:
		return "openai-chat-completions"
	}
}

func (e *Engine) listCanvasChannels() []canvasChannel {
	rows, err := e.DB.Query(`SELECT id, plugin_id, name, capability, base_url, vault_key, model, models, settings, sort_order, enabled, created_at, updated_at FROM canvas_channels ORDER BY sort_order ASC, created_at ASC`)
	if err != nil {
		return []canvasChannel{}
	}
	defer rows.Close()
	var out []canvasChannel
	for rows.Next() {
		var c canvasChannel
		var enabled int
		if err := rows.Scan(&c.ID, &c.PluginID, &c.Name, &c.Capability, &c.BaseURL, &c.VaultKey, &c.Model, &c.Models, &c.Settings, &c.SortOrder, &enabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			continue
		}
		c.Enabled = enabled != 0
		if e.Vault != nil && c.VaultKey != "" {
			if _, err := e.Vault.Get(c.VaultKey); err == nil {
				c.HasKey = true
			}
		}
		out = append(out, c)
	}
	if out == nil {
		out = []canvasChannel{}
	}
	return out
}

func (e *Engine) getCanvasChannel(id string) (canvasChannel, error) {
	for _, c := range e.listCanvasChannels() {
		if c.ID == id {
			return c, nil
		}
	}
	// fall back to drama providers as channels
	if p, err := e.GetProvider(id); err == nil {
		return canvasChannel{
			ID: p.ID, PluginID: mapProviderPlugin(p.ServiceType, p.Provider), Name: p.Name,
			Capability: p.ServiceType, BaseURL: p.BaseURL, VaultKey: p.VaultKey, Model: p.Model,
			Models: p.Models, Settings: p.Settings, Enabled: p.IsActive, HasKey: p.HasKey,
			CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		}, nil
	}
	return canvasChannel{}, fmt.Errorf("channel not found")
}

func (e *Engine) upsertCanvasChannel(in map[string]any, apiKey string) (canvasChannel, error) {
	id := strAnyMap(in, "id")
	if id == "" {
		id = NewID()
	}
	now := Now()
	name := firstNonEmpty(strAnyMap(in, "name", "displayName"), "Channel")
	cap := firstNonEmpty(strAnyMap(in, "capability"), "image")
	plugin := strAnyMap(in, "pluginId", "plugin_id")
	base := strAnyMap(in, "baseUrl", "base_url")
	model := strAnyMap(in, "model")
	models := strAnyMap(in, "models")
	if models == "" {
		models = "[]"
	}
	vaultKey := firstNonEmpty(strAnyMap(in, "vaultKey", "vault_key"), "video.canvas."+id)
	enabled := 1
	if v, ok := in["enabled"]; ok && !boolAny(v) {
		enabled = 0
	}
	_, err := e.DB.Exec(`INSERT INTO canvas_channels(id, plugin_id, name, capability, base_url, vault_key, model, models, settings, sort_order, enabled, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET plugin_id=excluded.plugin_id, name=excluded.name, capability=excluded.capability, base_url=excluded.base_url, model=excluded.model, models=excluded.models, settings=excluded.settings, enabled=excluded.enabled, updated_at=excluded.updated_at`,
		id, plugin, name, cap, base, vaultKey, model, models, "{}", 0, enabled, now, now)
	if err != nil {
		return canvasChannel{}, err
	}
	if apiKey != "" && e.Vault != nil {
		e.Vault.Set(vaultKey, apiKey)
	}
	return e.getCanvasChannel(id)
}

func (e *Engine) modelCatalog() map[string]any {
	channels := []map[string]any{}
	for _, c := range e.listCanvasChannels() {
		if !c.Enabled {
			continue
		}
		keys := []string{}
		if c.Models != "" && c.Models != "[]" {
			_ = json.Unmarshal([]byte(c.Models), &keys)
		}
		if c.Model != "" {
			found := false
			for _, k := range keys {
				if k == c.Model {
					found = true
					break
				}
			}
			if !found {
				keys = append([]string{c.Model}, keys...)
			}
		}
		models := []map[string]any{}
		for i, key := range keys {
			if strings.TrimSpace(key) == "" {
				continue
			}
			cap := c.Capability
			if cap == "tts" {
				cap = "audio"
			}
			models = append(models, map[string]any{
				"id": c.ID + ":" + key, "modelKey": key, "name": key, "displayName": key,
				"capability": cap, "available": c.HasKey, "sortOrder": i,
				"pricingMode": "info", "priceLabel": "",
				"capabilitySpec": map[string]any{"version": 1, "capability": cap, "operations": defaultOps(cap)},
			})
		}
		channels = append(channels, map[string]any{
			"id": c.ID, "name": c.Name, "displayName": c.Name, "sortOrder": c.SortOrder, "models": models,
		})
	}
	return map[string]any{"source": "system", "channels": channels, "models": []any{}}
}

func defaultOps(cap string) []string {
	switch cap {
	case "image":
		return []string{"text_to_image", "image_to_image", "inpaint"}
	case "video":
		return []string{"text_to_video", "image_to_video", "reference_to_video"}
	case "audio":
		return []string{"text_to_speech"}
	default:
		return []string{"chat"}
	}
}
