package connection

import "strings"

func (r *Registry) adoptDefaultKey(c Connection) Connection {
	if r == nil || r.vault == nil {
		return c
	}
	key, err := r.vault.Get("default")
	if err != nil || strings.TrimSpace(key) == "" {
		return c
	}
	if c.VaultKey != "" {
		if cur, e := r.vault.Get(c.VaultKey); e == nil && cur == key {
			c.HasKey = true
			return c
		}
	}
	out, err := r.Upsert(c, key)
	if err != nil {
		return c
	}
	return out
}

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

func (r *Registry) SeedSearch(endpoint, vaultKeyName string) {
	r.SyncSearch(endpoint, vaultKeyName)
}

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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
