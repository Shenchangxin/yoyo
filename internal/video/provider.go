package video

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connection"
)

func (e *Engine) ListProviders(serviceType string) ([]Provider, error) {
	if e.Conn == nil {
		return []Provider{}, nil
	}
	want := connection.NormalizeCap(serviceType)
	list, err := e.Conn.List("")
	if err != nil {
		return nil, err
	}
	defs := e.Conn.Defaults()
	out := make([]Provider, 0, len(list))
	seen := map[string]bool{}
	for _, c := range list {
		caps := connection.CapsOf(c)
		for _, cap := range caps {
			if cap == "" || cap == connection.CapProtocol {
				continue
			}
			if want != "" && cap != want {
				continue
			}
			if want == "" && cap == connection.CapChat {
				continue
			}
			key := c.ID + "/" + cap
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, providerFromConn(c, cap, defs))
		}
	}
	if out == nil {
		out = []Provider{}
	}
	return out, nil
}

func (e *Engine) GetProvider(id string) (Provider, error) {
	if e.Conn == nil {
		return Provider{}, fmt.Errorf("provider not found")
	}
	wantCap := catalogSuffixCap(id)
	id = providerIDFromChannel(id)
	c, err := e.Conn.Get(id)
	if err != nil {
		return Provider{}, fmt.Errorf("provider not found")
	}
	return providerFromConn(c, connCapForYingce(wantCap), e.Conn.Defaults()), nil
}

func (e *Engine) ActiveProvider(serviceType, id string) (Provider, error) {
	if e.Conn == nil {
		return Provider{}, fmt.Errorf("no active %s provider — add one in Settings", serviceType)
	}
	cap := connection.NormalizeCap(serviceType)
	id = providerIDFromChannel(id)
	if id != "" {
		if c, err := e.Conn.Get(id); err == nil && connection.MatchesCap(c, cap) {
			return providerFromConn(c, cap, e.Conn.Defaults()), nil
		}
	}
	c, err := e.Conn.Active(serviceType, id)
	if err != nil {
		c, err = e.firstMatchingConn(cap)
		if err != nil {
			return Provider{}, fmt.Errorf("no active %s provider — add one in Settings", serviceType)
		}
	}
	return providerFromConn(c, cap, e.Conn.Defaults()), nil
}

func (e *Engine) firstMatchingConn(cap string) (connection.Connection, error) {
	list, err := e.Conn.List("")
	if err != nil {
		return connection.Connection{}, err
	}
	var keyed, anyMatch connection.Connection
	var hasKeyed, hasAny bool
	for _, c := range list {
		if !c.Active || !connection.MatchesCap(c, cap) {
			continue
		}
		if !hasAny {
			anyMatch = c
			hasAny = true
		}
		if c.HasKey {
			if !hasKeyed {
				keyed = c
				hasKeyed = true
			}
			if connection.HasCap(c, cap) {
				return c, nil
			}
		}
	}
	if hasKeyed {
		return keyed, nil
	}
	if hasAny {
		return anyMatch, nil
	}
	return connection.Connection{}, fmt.Errorf("no active %s connection", cap)
}

func ProviderModels(p Provider) []string {
	return connection.ParseModels(p.Models)
}

func ResolveProviderModel(p Provider, want string) string {
	want = strings.TrimSpace(want)
	if want != "" {
		return want
	}
	if p.Model != "" {
		return p.Model
	}
	if ms := ProviderModels(p); len(ms) > 0 {
		return ms[0]
	}
	return ""
}

func (e *Engine) UpsertProvider(p Provider, apiKey string) (Provider, error) {
	if e.Conn == nil {
		return p, fmt.Errorf("connection registry missing")
	}
	cap := connection.NormalizeCap(p.ServiceType)
	if cap == "" {
		cap = connection.CapImage
	}
	models := connection.ParseModels(p.Models)
	if p.Model != "" {
		models = connection.ParseModels(p.Model)
		models = append(models, connection.ParseModels(p.Models)...)
		models = uniqueModels(models)
	}
	c := connection.Connection{
		ID:           providerIDFromChannel(p.ID),
		Name:         p.Name,
		Vendor:       p.Provider,
		Endpoint:     p.BaseURL,
		VaultKey:     p.VaultKey,
		Capabilities: []string{cap},
		Models:       models,
		DefaultModel: map[string]string{cap: p.Model},
		Active:       p.IsActive,
		Priority:     p.Priority,
		Protocol:     mapProviderPlugin(cap, p.Provider),
	}
	if c.ID != "" {
		if cur, err := e.Conn.Get(c.ID); err == nil {
			caps := cur.Capabilities
			if !connection.HasCap(cur, cap) {
				caps = append(caps, cap)
			}
			c.Capabilities = caps
			c.Models = uniqueModels(append(models, connection.ModelsExcept(cur, cap)...))
			if cur.DefaultModel != nil {
				for k, v := range cur.DefaultModel {
					if c.DefaultModel[k] == "" {
						c.DefaultModel[k] = v
					}
				}
			}
			if c.VaultKey == "" {
				c.VaultKey = cur.VaultKey
			}
			c.CreatedAt = cur.CreatedAt
			c.Settings = cur.Settings
			if c.Name == "" {
				c.Name = cur.Name
			}
			if c.Endpoint == "" {
				c.Endpoint = cur.Endpoint
			}
			if c.Vendor == "" {
				c.Vendor = cur.Vendor
			}
		}
	}
	if strings.TrimSpace(p.Settings) != "" && p.Settings != "{}" {
		var st map[string]any
		if json.Unmarshal([]byte(p.Settings), &st) == nil && st != nil {
			if c.Settings == nil {
				c.Settings = map[string]any{}
			}
			for k, v := range st {
				c.Settings[k] = v
			}
		}
	}
	out, err := e.Conn.Upsert(c, apiKey)
	if err != nil {
		return p, err
	}
	if p.IsDefault {
		_ = e.Conn.SetDefault(cap, out.ID)
	}
	return providerFromConn(out, cap, e.Conn.Defaults()), nil
}

func (e *Engine) DeleteProvider(id string) error {
	if e.Conn == nil {
		return nil
	}
	id = providerIDFromChannel(id)
	if id == "cas-local" {
		return fmt.Errorf("local CAS cannot be removed")
	}
	return e.Conn.Delete(id)
}

func (e *Engine) TestProvider(id string) map[string]any {
	if e.Conn == nil {
		return map[string]any{"ok": false, "error": "registry missing"}
	}
	return e.Conn.Test(providerIDFromChannel(id))
}

func uniqueModels(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func providerFromConn(c connection.Connection, cap string, defs connection.Defaults) Provider {
	if cap == "" {
		for _, x := range c.Capabilities {
			if x == connection.CapImage || x == connection.CapVideo || x == connection.CapSpeech {
				cap = x
				break
			}
		}
		if cap == "" && len(c.Capabilities) > 0 {
			cap = c.Capabilities[0]
		}
	}
	cap = connection.NormalizeCap(cap)
	svc := cap
	if svc == connection.CapSpeech {
		svc = "tts"
	}
	if svc == connection.CapChat {
		svc = "chat"
	}
	isDef := false
	if defs != nil {
		isDef = defs[cap] == c.ID
	}
	settings := "{}"
	if len(c.Settings) > 0 {
		if b, err := json.Marshal(c.Settings); err == nil {
			settings = string(b)
		}
	}
	return Provider{
		ID:          c.ID,
		ServiceType: svc,
		Provider:    c.Vendor,
		Name:        c.Name,
		BaseURL:     c.Endpoint,
		VaultKey:    c.VaultKey,
		Model:       connection.ModelForCap(c, cap),
		Models:      connection.ModelsJSONFor(c, cap),
		Priority:    c.Priority,
		IsDefault:   isDef,
		IsActive:    c.Active,
		HasKey:      c.HasKey,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Settings:    settings,
	}
}

func mapProviderPlugin(serviceType, vendor string) string {
	return connection.InferProtocol(vendor, serviceType)
}
