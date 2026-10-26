// provider.go 把连接表投影成图像、视频和语音等供应商行。

package video

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connection"
)

// ListProviders 按服务类型列出连接。连接表为空时返回空切片。空类型时跳过聊天能力，protocol 能力一律跳过。同一连接的同一能力只出现一次。
// serviceType 传给 connection.NormalizeCap；结果为空时跳过聊天能力，非空时只保留与之相等的能力。
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

// GetProvider 按标识读取连接并投影成供应商。连接表为空或读不到时返回 provider not found。
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

// ActiveProvider 先按标识取匹配该能力的连接，没有再取 Active，仍没有再找第一条匹配的连接。都没有时提示到设置里添加。
// serviceType 传给 connection.NormalizeCap 和 Conn.Active，并写进找不到供应商时的错误文本。
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

// firstMatchingConn 在启用且能力匹配的连接里，优先返回有密钥并且声明了该能力的一条。否则返回第一条有密钥的，再否则返回第一条匹配的。都没有时返回 no active connection。
// cap 传给 connection.MatchesCap 和 connection.HasCap，并写进找不到连接时的错误文本。
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

// ProviderModels 解析供应商模型列表文本。
// p 的 Models 传给 connection.ParseModels。
func ProviderModels(p Provider) []string {
	return connection.ParseModels(p.Models)
}

// ResolveProviderModel 优先用 want，其次用供应商当前模型，再次用模型列表的第一项。都空时返回空串。
// p 在 want 去掉首尾空白后为空时，Model 非空就返回 Model，否则用 ProviderModels 的第一项。
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

// UpsertProvider 把供应商写进连接表。服务类型空着时用图像。已有连接会合并能力、模型、默认模型、保险库键、名字、端点、厂商和设置。IsDefault 为真时把该连接设为这种能力的默认。连接表为空时返回 connection registry missing。
// apiKey 传给 Conn.Upsert。
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

// DeleteProvider 删除连接。连接表为空时返回 nil。标识 cas-local 返回 local CAS cannot be removed。
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

// TestProvider 调用连接表的测试。连接表为空时返回 ok 为假、error 为 registry missing。
func (e *Engine) TestProvider(id string) map[string]any {
	if e.Conn == nil {
		return map[string]any{"ok": false, "error": "registry missing"}
	}
	return e.Conn.Test(providerIDFromChannel(id))
}

// uniqueModels 按出现顺序去掉空白和重复模型名。
// in 逐项去掉首尾空白，空串或已经出现过的丢掉。
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

// providerFromConn 把一条连接投影成供应商。能力空着时优先取图像、视频或语音，再取第一项。语音能力的服务类型写成 tts，聊天写成 chat。设置编不成 JSON 时用 {}。
// cap 为空时从 Capabilities 里先找图像、视频或语音，否则取第一项，再传给 connection.NormalizeCap、connection.ModelForCap 和 connection.ModelsJSONFor。
// defs 非 nil 时用 defs[cap] 是否等于连接 ID 决定 IsDefault。
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

// mapProviderPlugin 按厂商和服务类型推断协议名。
// serviceType 作为第二个参数传给 connection.InferProtocol。
// vendor 作为第一个参数传给 connection.InferProtocol。
func mapProviderPlugin(serviceType, vendor string) string {
	return connection.InferProtocol(vendor, serviceType)
}
