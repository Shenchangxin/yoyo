// 把聊天、搜索、OpenTelemetry 和 MCP 连接补进注册表。抄钥匙时只读已经保存的值。

package connection

import "strings"

// adoptDefaultKey 只通过 storedVault 读取已保存的 default 和本连接钥匙。本连接已有不同钥匙，且不是默认聊天连接时，不覆盖。
func (r *Registry) adoptDefaultKey(c Connection) Connection {
	if r == nil || r.vault == nil {
		return c
	}
	// 只复制已经保存的钥匙。不读进程环境变量，避免把环境里的钥匙写到自定义供应商上。
	key := storedVault(r.vault, "default")
	if key == "" {
		return c
	}
	own := storedVault(r.vault, c.VaultKey)
	if own != "" {
		c.HasKey = true
		if own == key {
			return c
		}
		defID := r.Defaults()[CapChat]
		if c.ID != defID {
			return c
		}
	} else {
		defID := r.Defaults()[CapChat]
		if c.ID != "" && c.ID != defID {
			return c
		}
	}
	out, err := r.Upsert(c, key)
	if err != nil {
		return c
	}
	return out
}

// storedVault 在仓库实现了 GetStored 时返回去掉空白的已保存值。没有该方法或名字为空则返回空串，不调用 Get。
func storedVault(v Secrets, name string) string {
	name = strings.TrimSpace(name)
	if v == nil || name == "" {
		return ""
	}
	if g, ok := v.(interface{ GetStored(string) (string, error) }); ok {
		got, _ := g.GetStored(name)
		return strings.TrimSpace(got)
	}
	return ""
}

// SyncChat 在没有聊天连接时 SeedChat，再按需 Upsert。已有连接时更新供应商、端点和模型，然后 adoptDefaultKey。anthropic 收成 claude。
func (r *Registry) SyncChat(provider, baseURL, model, apiKey string, extraModels ...string) (Connection, error) {
	provider = strings.TrimSpace(provider)
	baseURL = strings.TrimSpace(baseURL)
	model = strings.TrimSpace(model)
	list, err := r.List(CapChat)
	if err != nil {
		return Connection{}, err
	}
	if len(list) == 0 {
		c, err := r.SeedChat(provider, baseURL, model, extraModels...)
		if err != nil || apiKey == "" {
			return c, err
		}
		return r.Upsert(c, apiKey)
	}
	defs := r.Defaults()
	cur := list[0]
	if id := defs[CapChat]; id != "" {
		if c, err := r.Get(id); err == nil {
			cur = c
		}
	}
	if provider != "" {
		if provider == "anthropic" {
			provider = "claude"
		}
		cur.Vendor = provider
		cur.Name = firstNonEmpty(cur.Name, provider)
	}
	if baseURL != "" {
		cur.Endpoint = baseURL
	}
	if model != "" || len(extraModels) > 0 {
		cur.Models = compact(append(append([]string{model}, extraModels...), cur.Models...))
		if model != "" {
			if cur.DefaultModel == nil {
				cur.DefaultModel = map[string]string{}
			}
			cur.DefaultModel[CapChat] = model
		}
	}
	if cur.Protocol == "" {
		cur.Protocol = InferProtocol(cur.Vendor, CapChat)
	}
	out, err := r.Upsert(cur, apiKey)
	if err != nil {
		return out, err
	}
	return r.adoptDefaultKey(out), nil
}

// SeedSearch 调用 SyncSearch。
func (r *Registry) SeedSearch(endpoint, vaultKeyName string) {
	r.SyncSearch(endpoint, vaultKeyName)
}

// SyncSearch 把端点写到搜索连接。vaultKeyName 是钥匙名字，为空时用 search。Upsert 的 apiKey 是空串，不在这里写钥匙原文。端点为空则直接返回。
func (r *Registry) SyncSearch(endpoint, vaultKeyName string) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return
	}
	list, _ := r.List(CapSearch)
	vk := strings.TrimSpace(vaultKeyName)
	if vk == "" {
		vk = "search"
	}
	if len(list) == 0 {
		_, _ = r.Upsert(Connection{
			Name: "Web search", Vendor: "search", Protocol: "search",
			Endpoint: endpoint, VaultKey: vk, Capabilities: []string{CapSearch},
			Active: true, Priority: 10,
		}, "")
		return
	}
	c := list[0]
	c.Endpoint = endpoint
	if c.VaultKey == "" {
		c.VaultKey = vk
	}
	_, _ = r.Upsert(c, "")
}

// SeedOTEL 写入或更新 OpenTelemetry 端点。端点为空则直接返回。Upsert 不带钥匙。
func (r *Registry) SeedOTEL(endpoint string) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return
	}
	list, _ := r.List(CapOTEL)
	if len(list) > 0 {
		c := list[0]
		if c.Endpoint != endpoint {
			c.Endpoint = endpoint
			_, _ = r.Upsert(c, "")
		}
		return
	}
	_, _ = r.Upsert(Connection{
		Name: "OpenTelemetry", Vendor: "otel", Protocol: "otel",
		Endpoint: endpoint, Capabilities: []string{CapOTEL},
		Active: true, Priority: 10,
	}, "")
}

// SeedMCP 按名称或端点更新已有 MCP 连接，否则新建。名称为空时用 mcp。Upsert 不带钥匙。
func (r *Registry) SeedMCP(name, endpoint string) {
	endpoint = strings.TrimSpace(endpoint)
	name = strings.TrimSpace(name)
	if endpoint == "" {
		return
	}
	if name == "" {
		name = "mcp"
	}
	list, _ := r.List(CapMCP)
	for i := range list {
		if list[i].Name == name || list[i].Endpoint == endpoint {
			c := list[i]
			c.Endpoint = endpoint
			c.Name = name
			_, _ = r.Upsert(c, "")
			return
		}
	}
	_, _ = r.Upsert(Connection{
		Name: name, Vendor: "mcp", Protocol: "mcp-http",
		Endpoint: endpoint, Capabilities: []string{CapMCP},
		Active: true,
	}, "")
}

// firstNonEmpty 返回第一段去掉空白后仍非空的字符串。
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
