// 行为：加载协议插件和用户插件，维护画布通道，并生成模型目录。

package video

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/video/protocol"
)

// loadCanvasPlugins 把内置、插件目录和用户目录里的协议插件注册进 Proto。
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

// installUserPlugin 把上传的插件包写到用户插件目录并重新加载。
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

// pluginRecord 按协议元数据拼一条前端插件记录，供应商和说明会去掉影策品牌。
func (e *Engine) pluginRecord(id, source, fileName, status string) map[string]any {
	name := id
	version := "1"
	author, docs, desc := "", "", ""
	caps := []string{}
	if e.Proto != nil {
		if ad, ok := e.Proto.Resolve(id); ok {
			m := ad.Metadata()
			name = m.Name
			version = m.Version
			id = m.ID
			author = debrandYingce(m.Vendor)
			docs = debrandYingce(m.Documentation)
			desc = debrandYingce(m.Description)
			for _, cap := range m.Categories {
				if cap != "" {
					caps = append(caps, string(cap))
				}
			}
		}
	}
	providers := []any{}
	if len(caps) > 0 {
		providers = append(providers, map[string]any{
			"id":           id,
			"label":        name,
			"capabilities": caps,
			"scopes":       []string{"canvas", "creation", "agent", "user.custom-channel"},
			"create":       map[string]any{"method": "POST", "path": "/"},
			"response":     map[string]any{},
		})
	}
	manifest := map[string]any{
		"apiVersion":  "yingce.plugin/v2",
		"id":          id,
		"name":        name,
		"version":     version,
		"enabled":     true,
		"trusted":     true,
		"permissions": []string{"generation.run", "media.read"},
		"contributes": map[string]any{"providers": providers},
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
		"manifest":    manifest,
		"source":      source,
		"fileName":    fileName,
		"package":     fileName,
		"sha256":      "",
		"installedAt": Now(),
		"updatedAt":   Now(),
		"status":      status,
		"management": map[string]any{
			"origin":             sourceOrigin(source),
			"kind":               "protocol",
			"activationScope":    "system",
			"configurationScope": "user",
		},
	}
}

// sourceOrigin 把 bundled 收成 official，其余来源收成 uploaded。
func sourceOrigin(source string) string {
	if source == "bundled" {
		return "official"
	}
	return "uploaded"
}

// 前端内置的应用插件不进协议注册表。只有用户明确拨过开关时才下发状态，
// 避免用“默认关闭”盖掉这台机器上已经保存的启用记录。
var desktopApplicationPluginIDs = []string{
	"eagle-asset-connector",
	"prompt-optimizer",
	"runninghub-workflow-provider",
	"ai-art-critique",
	"editor-shell",
}

// pluginStateMap 拼一条插件开关状态。
func pluginStateMap(id string, enabled bool) map[string]any {
	return map[string]any{
		"pluginId":          id,
		"platformAvailable": true,
		"userEnabled":       enabled,
		"userConfigured":    true,
		"effectiveEnabled":  enabled,
		"canToggle":         true,
		"canConfigure":      true,
	}
}

// setPluginEnabled 记下插件是否启用，并返回新状态。
func (e *Engine) setPluginEnabled(id string, enabled bool) (map[string]any, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("plugin id required")
	}
	flags := e.loadPluginFlags()
	flags[id] = pluginFlagRec{Enabled: enabled, UpdatedAt: Now()}
	if err := e.savePluginFlags(flags); err != nil {
		return nil, err
	}
	return pluginStateMap(id, enabled), nil
}

// listPluginPayload 列出协议插件。桌面应用插件只在本机已经有开关记录时才下发状态。
func (e *Engine) listPluginPayload() map[string]any {
	plugins := []map[string]any{}
	states := map[string]any{}
	seen := map[string]struct{}{}
	flags := e.loadPluginFlags()
	if e.Proto != nil {
		for _, meta := range e.Proto.List("", "", true) {
			if _, ok := seen[meta.ID]; ok {
				continue
			}
			seen[meta.ID] = struct{}{}
			enabled := true
			if f, ok := flags[meta.ID]; ok {
				enabled = f.Enabled
			}
			status := "enabled"
			if !enabled {
				status = "disabled"
			}
			rec := e.pluginRecord(meta.ID, "bundled", meta.ID+".yingce-plugin", status)
			plugins = append(plugins, rec)
			states[meta.ID] = pluginStateMap(meta.ID, enabled)
		}
	}
	for _, id := range desktopApplicationPluginIDs {
		if _, ok := states[id]; ok {
			continue
		}
		f, ok := flags[id]
		if !ok {
			continue
		}
		states[id] = pluginStateMap(id, f.Enabled)
	}
	statuses := map[string]any{}
	for id, raw := range states {
		if boolAny(asMap(raw)["effectiveEnabled"]) {
			statuses[id] = "enabled"
		} else {
			statuses[id] = "disabled"
		}
	}
	return map[string]any{"plugins": plugins, "states": states, "statuses": statuses}
}

// pluginCatalog 按能力列出画布面上的协议提供者。scope 参数目前未使用。
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

// providerChannelID 给供应商 id 加上 ch- 前缀。已有前缀或空串则原样返回。
func providerChannelID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || strings.HasPrefix(id, "ch-") {
		return id
	}
	return "ch-" + id
}

// providerIDFromChannel 去掉 ch- 前缀和目录能力后缀，得到供应商 id。
func providerIDFromChannel(id string) string {
	return stripCatalogCapSuffix(strings.TrimPrefix(strings.TrimSpace(id), "ch-"))
}

// stripCatalogCapSuffix 去掉 id 末尾的 --text、--image、--video 或 --audio。
func stripCatalogCapSuffix(id string) string {
	for _, suf := range []string{"--text", "--image", "--video", "--audio"} {
		if strings.HasSuffix(id, suf) {
			return strings.TrimSuffix(id, suf)
		}
	}
	return id
}

// catalogSuffixCap 读出 id 末尾的目录能力后缀，没有则返回空串。
func catalogSuffixCap(id string) string {
	for _, suf := range []string{"--text", "--image", "--video", "--audio"} {
		if strings.HasSuffix(id, suf) {
			return strings.TrimPrefix(suf, "--")
		}
	}
	return ""
}

// yingceCapability 把连接能力收成画布用的 text、audio、image 或 video。
func yingceCapability(cap string) string {
	switch connection.NormalizeCap(cap) {
	case connection.CapChat:
		return "text"
	case connection.CapSpeech:
		return "audio"
	case connection.CapImage:
		return "image"
	case connection.CapVideo:
		return "video"
	default:
		return ""
	}
}

// yingceCaps 列出连接具备的画布能力，并去掉重复。
func yingceCaps(c connection.Connection) []string {
	var out []string
	seen := map[string]bool{}
	for _, x := range connection.CapsOf(c) {
		y := yingceCapability(x)
		if y == "" || seen[y] {
			continue
		}
		seen[y] = true
		out = append(out, y)
	}
	return out
}

// connCapForYingce 把画布能力收成连接能力。text 对应 chat，audio 对应 speech。
func connCapForYingce(y string) string {
	switch y {
	case "text":
		return connection.CapChat
	case "audio":
		return connection.CapSpeech
	default:
		return y
	}
}

// isGenerationConnection 判断连接是否具备对话、图像、视频或语音能力。
func isGenerationConnection(c connection.Connection) bool {
	return connection.MatchesCap(c, connection.CapChat) || connection.MatchesCap(c, connection.CapImage) || connection.MatchesCap(c, connection.CapVideo) || connection.MatchesCap(c, connection.CapSpeech)
}

// channelFromProvider 把旧供应商记录收成画布通道。
func (e *Engine) channelFromProvider(p Provider) canvasChannel {
	return canvasChannel{
		ID:         providerChannelID(p.ID),
		PluginID:   mapProviderPlugin(p.ServiceType, p.Provider),
		Name:       p.Name,
		Capability: firstNonEmpty(yingceCapability(p.ServiceType), p.ServiceType),
		BaseURL:    p.BaseURL,
		VaultKey:   p.VaultKey,
		Model:      p.Model,
		Models:     p.Models,
		Settings:   p.Settings,
		Enabled:    p.IsActive,
		HasKey:     p.HasKey,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

// channelFromConn 用连接的第一项画布能力收成通道。
func (e *Engine) channelFromConn(c connection.Connection) canvasChannel {
	cap := ""
	if caps := yingceCaps(c); len(caps) > 0 {
		cap = caps[0]
	}
	return e.channelFromConnCap(c, cap)
}

// channelFromConnCap 按指定画布能力从连接取出模型、端点和插件，收成通道。
func (e *Engine) channelFromConnCap(c connection.Connection, yingceCap string) canvasChannel {
	if yingceCap == "" {
		if caps := yingceCaps(c); len(caps) > 0 {
			yingceCap = caps[0]
		}
	}
	ccap := connCapForYingce(yingceCap)
	def := connection.ModelForCap(c, ccap)
	keys := connection.ModelsFor(c, ccap)
	if def != "" {
		found := false
		for _, k := range keys {
			if k == def {
				found = true
				break
			}
		}
		if !found {
			keys = append([]string{def}, keys...)
		}
	}
	models := "[]"
	if b, err := json.Marshal(keys); err == nil {
		models = string(b)
	}
	plugin := connection.InferProtocol(c.Vendor, ccap)
	if plugin == "" {
		plugin = c.Protocol
	}
	return canvasChannel{
		ID: c.ID, PluginID: plugin, Name: c.Name, Capability: yingceCap,
		BaseURL: c.Endpoint, VaultKey: c.VaultKey, Model: def,
		Models: models, Enabled: c.Active || c.HasKey, HasKey: c.HasKey,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

// syncProviderChannel 当前直接返回 nil。
func (e *Engine) syncProviderChannel(p Provider) error { return nil }

// deleteProviderChannel 删除供应商记录和同 id 的画布通道文档。
func (e *Engine) deleteProviderChannel(id string) {
	_ = e.DeleteProvider(id)
	_ = e.delDoc(colChannels, id)
}

// syncAllProviderChannels 当前直接返回 nil。
func (e *Engine) syncAllProviderChannels() error { return nil }

// seedCanvasChannels 调用 syncAllProviderChannels，因此当前返回 nil。
func (e *Engine) seedCanvasChannels() error {
	return e.syncAllProviderChannels()
}

// listCanvasChannels 合并连接表和已保存的画布通道，并按金库标记是否有密钥。
func (e *Engine) listCanvasChannels() []canvasChannel {
	seen := map[string]bool{}
	var out []canvasChannel
	add := func(c canvasChannel) {
		if c.ID == "" || seen[c.ID] {
			return
		}
		seen[c.ID] = true
		if e.Vault != nil && c.VaultKey != "" {
			if _, err := e.Vault.Get(c.VaultKey); err == nil {
				c.HasKey = true
			}
		}
		out = append(out, c)
	}
	if e.Conn != nil {
		list, _ := e.Conn.List("")
		for _, c := range list {
			if !isGenerationConnection(c) {
				continue
			}
			add(e.channelFromConn(c))
		}
	}
	for _, rec := range loadCol[canvasChannel](e, colChannels) {
		add(rec)
	}
	if out == nil {
		out = []canvasChannel{}
	}
	return out
}

// getCanvasChannel 按通道 id 查找。带能力后缀时改写返回的 id 和能力。
func (e *Engine) getCanvasChannel(id string) (canvasChannel, error) {
	id = strings.TrimSpace(id)
	wantCap := catalogSuffixCap(id)
	pid := providerIDFromChannel(id)
	for _, c := range e.listCanvasChannels() {
		if c.ID == id || c.ID == pid {
			if wantCap != "" {
				c.ID = id
				c.Capability = wantCap
			}
			return c, nil
		}
	}
	if e.Conn != nil && pid != "" {
		if c, err := e.Conn.Get(pid); err == nil && isGenerationConnection(c) {
			ch := e.channelFromConnCap(c, wantCap)
			if wantCap != "" {
				ch.ID = id
			}
			return ch, nil
		}
	}
	if p, err := e.GetProvider(pid); err == nil {
		ch := e.channelFromProvider(p)
		if wantCap != "" {
			ch.ID = id
			ch.Capability = wantCap
		}
		return ch, nil
	}
	return canvasChannel{}, fmt.Errorf("channel not found")
}

// upsertCanvasChannel 创建或更新画布通道。有连接表时写进连接，否则写通道文档。
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
	vaultKey := firstNonEmpty(strAnyMap(in, "vaultKey", "vault_key"), "conn."+id+".key")
	enabled := true
	if v, ok := in["enabled"]; ok {
		enabled = boolAny(v)
	}
	if e.Conn != nil {
		c := connection.Connection{
			ID: id, Name: name, Vendor: firstNonEmpty(plugin, "custom"), Protocol: plugin,
			Endpoint: base, VaultKey: vaultKey,
			Capabilities: []string{connection.NormalizeCap(cap), connection.CapProtocol},
			Models:       connection.ParseModels(models), DefaultModel: map[string]string{connection.NormalizeCap(cap): model},
			Active: enabled,
		}
		if model != "" && len(c.Models) == 0 {
			c.Models = []string{model}
		}
		out, err := e.Conn.Upsert(c, apiKey)
		if err != nil {
			return canvasChannel{}, err
		}
		_ = now
		return e.channelFromConn(out), nil
	}
	ch := canvasChannel{ID: id, PluginID: plugin, Name: name, Capability: cap, BaseURL: base, VaultKey: vaultKey, Model: model, Models: models, Enabled: enabled, CreatedAt: now, UpdatedAt: now}
	if apiKey != "" && e.Vault != nil {
		e.Vault.Set(vaultKey, apiKey)
		ch.HasKey = true
	}
	if err := e.putDoc(colChannels, id, ch); err != nil {
		return canvasChannel{}, err
	}
	return e.getCanvasChannel(id)
}

// catalogModels 把通道上的模型键收成目录条目，操作按能力给出默认集。
func catalogModels(c canvasChannel) []map[string]any {
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
	cap := yingceCapability(c.Capability)
	if cap == "" {
		cap = c.Capability
		if cap == "tts" || cap == "speech" {
			cap = "audio"
		}
		if cap == "chat" || cap == "llm" {
			cap = "text"
		}
	}
	models := []map[string]any{}
	for i, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		item := map[string]any{
			"id": c.ID + ":" + key, "modelKey": key, "name": key, "displayName": key,
			"capability": cap, "available": c.Enabled || c.HasKey, "sortOrder": i,
			"pricingMode": "info", "priceLabel": "",
			"priceTiers":     []map[string]any{{"billingMode": "fixed_request", "unitPriceMicrocredits": 0}},
			"capabilitySpec": map[string]any{"version": 1, "capability": cap, "operations": defaultOps(cap)},
		}
		if c.PluginID != "" {
			item["protocol"] = c.PluginID
		}
		models = append(models, item)
	}
	return models
}

// modelCatalog 按连接的每项能力展开通道，并带上已保存的画布通道。
func (e *Engine) modelCatalog() map[string]any {
	channels := []map[string]any{}
	seen := map[string]bool{}
	add := func(c canvasChannel) {
		if !c.Enabled || seen[c.ID] {
			return
		}
		seen[c.ID] = true
		channels = append(channels, map[string]any{
			"id": c.ID, "name": c.Name, "displayName": c.Name, "sortOrder": c.SortOrder, "models": catalogModels(c),
		})
	}
	if e.Conn != nil {
		list, _ := e.Conn.List("")
		for _, c := range list {
			if !isGenerationConnection(c) {
				continue
			}
			caps := yingceCaps(c)
			if len(caps) == 0 {
				continue
			}
			for _, cap := range caps {
				ch := e.channelFromConnCap(c, cap)
				if len(caps) > 1 {
					ch.ID = c.ID + "--" + cap
					ch.Name = c.Name + " · " + cap
				}
				add(ch)
			}
		}
	}
	for _, rec := range loadCol[canvasChannel](e, colChannels) {
		if rec.Capability != "" && yingceCapability(rec.Capability) != "" {
			rec.Capability = yingceCapability(rec.Capability)
		}
		add(rec)
	}
	return map[string]any{"source": "system", "channels": channels, "models": []any{}}
}

// defaultOps 按能力返回默认可执行的操作名。
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
