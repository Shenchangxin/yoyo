// 设置里的聊天供应商。记录存在视频模块的连接库，capability 为 chat。默认连接会写回 Config 的厂商、BaseURL 和模型。自定义供应商不借用 OpenAI 环境变量密钥。

package app

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// ChatProvider 是给设置页的一条聊天连接。HasKey 表示库里有密钥，IsDefault 表示它是 chat 的默认连接。
type ChatProvider struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Vendor    string   `json:"vendor"`
	Endpoint  string   `json:"endpoint"`
	Model     string   `json:"model"`
	Models    []string `json:"models"`
	HasKey    bool     `json:"has_key"`
	IsDefault bool     `json:"is_default"`
	Active    bool     `json:"active"`
}

// ChatProviderIn 是新建或更新聊天连接的入参。Active 为 nil 时保留原有启用状态。MakeDefault 为真时设为 chat 默认连接。
type ChatProviderIn struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Vendor      string   `json:"vendor"`
	Endpoint    string   `json:"endpoint"`
	Model       string   `json:"model"`
	Models      []string `json:"models"`
	Active      *bool    `json:"active"`
	MakeDefault bool     `json:"make_default"`
}

// connReg 返回视频模块上的连接库。视频模块不存在时返回 nil。
func (a *App) connReg() *connection.Registry {
	if a != nil && a.Video != nil {
		return a.Video.Conn
	}
	return nil
}

// ListChatProviders 列出 capability 为 chat 的连接。库不存在或列出失败时返回空切片。
func (a *App) ListChatProviders() []ChatProvider {
	r := a.connReg()
	if r == nil {
		return []ChatProvider{}
	}
	list, err := r.List(connection.CapChat)
	if err != nil {
		return []ChatProvider{}
	}
	defs := r.Defaults()
	defID := defs[connection.CapChat]
	out := make([]ChatProvider, 0, len(list))
	for _, c := range list {
		out = append(out, chatProviderOf(c, defID))
	}
	return out
}

// UpsertChatProvider 规范化端点，把 anthropic 写成 claude，空厂商写成 custom。没有默认连接或 MakeDefault 为真时设为默认，并 promoteChatDefault。
// in 用来读并改写 Endpoint、Vendor、Name、Model、Models 和 ID；Vendor 为空则写成 custom，Name 为空则用 Vendor，Active 为 nil 时沿用已有连接的启用状态，并读 MakeDefault。
// apiKey 去掉首尾空白后传给 Registry.Upsert。
func (a *App) UpsertChatProvider(in ChatProviderIn, apiKey string) (ChatProvider, error) {
	r := a.connReg()
	if r == nil {
		return ChatProvider{}, fmt.Errorf("connection registry missing")
	}
	apiKey = strings.TrimSpace(apiKey)
	in.Endpoint = runtime.NormalizeBaseURL(in.Endpoint)
	in.Vendor = strings.TrimSpace(in.Vendor)
	if in.Vendor == "anthropic" {
		in.Vendor = "claude"
	}
	if in.Vendor == "" {
		in.Vendor = "custom"
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		in.Name = in.Vendor
	}
	in.Model = strings.TrimSpace(in.Model)
	models := connection.ParseModels(strings.Join(in.Models, ","))
	if in.Model != "" {
		models = uniqueStrings(append([]string{in.Model}, models...))
	}
	cur := connection.Connection{
		ID:           strings.TrimSpace(in.ID),
		Name:         in.Name,
		Vendor:       in.Vendor,
		Endpoint:     in.Endpoint,
		Protocol:     connection.InferProtocol(in.Vendor, connection.CapChat),
		Capabilities: []string{connection.CapChat},
		Models:       models,
		DefaultModel: map[string]string{connection.CapChat: in.Model},
		Active:       true,
		Priority:     50,
	}
	if in.Active != nil {
		cur.Active = *in.Active
	} else if cur.ID != "" {
		if existing, err := r.Get(cur.ID); err == nil {
			cur.Active = existing.Active
			cur.Priority = existing.Priority
		}
	}
	out, err := r.Upsert(cur, apiKey)
	if err != nil {
		return ChatProvider{}, err
	}
	defs := r.Defaults()
	isDef := in.MakeDefault || defs[connection.CapChat] == out.ID || defs[connection.CapChat] == ""
	if in.MakeDefault || defs[connection.CapChat] == "" {
		if err := r.SetDefault(connection.CapChat, out.ID); err != nil {
			return ChatProvider{}, err
		}
		isDef = true
	}
	if isDef {
		a.promoteChatDefault(out)
	}
	return chatProviderOf(out, r.Defaults()[connection.CapChat]), nil
}

// DeleteChatProvider 拒绝删到一条都不剩。删掉的是默认连接时，把剩下的第一条设为默认并写回配置。
func (a *App) DeleteChatProvider(id string) error {
	r := a.connReg()
	if r == nil {
		return fmt.Errorf("connection registry missing")
	}
	id = strings.TrimSpace(id)
	list, err := r.List(connection.CapChat)
	if err != nil {
		return err
	}
	if len(list) <= 1 {
		return fmt.Errorf("keep at least one chat provider")
	}
	defs := r.Defaults()
	wasDefault := defs[connection.CapChat] == id
	if err := r.Delete(id); err != nil {
		return err
	}
	if !wasDefault {
		return nil
	}
	rest, _ := r.List(connection.CapChat)
	if len(rest) == 0 {
		return nil
	}
	next := rest[0]
	_ = r.SetDefault(connection.CapChat, next.ID)
	a.promoteChatDefault(next)
	return nil
}

// SetDefaultChatProvider 把指定 chat 连接设为默认，并写回配置。不是 chat 连接时返回 not a chat provider。
func (a *App) SetDefaultChatProvider(id string) error {
	r := a.connReg()
	if r == nil {
		return fmt.Errorf("connection registry missing")
	}
	c, err := r.Get(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if !connection.HasCap(c, connection.CapChat) {
		return fmt.Errorf("not a chat provider")
	}
	if err := r.SetDefault(connection.CapChat, c.ID); err != nil {
		return err
	}
	a.promoteChatDefault(c)
	return nil
}

// TestChatProvider 在有 id 时测试该连接。id 为空时测试当前活动的 chat 连接。库不可用时改走 probeConfigChat。
func (a *App) TestChatProvider(id string) map[string]any {
	r := a.connReg()
	id = strings.TrimSpace(id)
	if r != nil {
		if id != "" {
			return r.Test(id)
		}
		if c, err := r.Active(connection.CapChat, ""); err == nil {
			return connection.TestConnection(c, r)
		}
	}
	return a.probeConfigChat()
}

// probeConfigChat 用 Config.BaseURL 和 vault 的 default 密钥探测 chat/completions。缺地址或密钥时 ok 为 false。
func (a *App) probeConfigChat() map[string]any {
	base := ""
	if a != nil {
		base = a.Config.BaseURL
	}
	key := ""
	if a != nil && a.Vault != nil {
		key, _ = a.Vault.Get("default")
	}
	base = runtime.NormalizeBaseURL(base)
	key = strings.TrimSpace(key)
	if base == "" {
		return map[string]any{"ok": false, "error": "base_url empty"}
	}
	if key == "" {
		return map[string]any{"ok": false, "error": "missing API key — paste the token and Save"}
	}
	model := ""
	vendor := ""
	if a != nil {
		model = a.Config.Model
		vendor = a.Config.Provider
	}
	out := connection.ProbeChatCompletions(base, key, model, vendor)
	if err, _ := out["error"].(string); err == "missing vault key" {
		out["error"] = "missing API key — paste the token and Save"
	}
	return out
}

// promoteChatDefault 把连接的厂商、端点和模型写入 Config，并把租到的密钥放进 vault 的 default，然后 SaveConfig。
// c 传给 Lease；Vendor 去掉首尾空白后为空则写成 custom，Endpoint 经 NormalizeBaseURL 写入 BaseURL，模型经 ModelFor 写入 Model，Models 复制进 Config。
func (a *App) promoteChatDefault(c connection.Connection) {
	if a == nil {
		return
	}
	r := a.connReg()
	if r != nil {
		if key, err := r.Lease(c); err == nil && strings.TrimSpace(key) != "" && a.Vault != nil {
			a.Vault.Set("default", strings.TrimSpace(key))
		}
	}
	a.Config.Provider = strings.TrimSpace(c.Vendor)
	if a.Config.Provider == "" {
		a.Config.Provider = "custom"
	}
	a.Config.BaseURL = runtime.NormalizeBaseURL(c.Endpoint)
	a.Config.Model = connection.ModelFor(c, connection.CapChat)
	a.Config.Models = append([]string{}, c.Models...)
	_ = a.SaveConfig()
}

// chatProviderOf 把连接库记录收成 ChatProvider。默认模型放进 Models 的前面。
// defID 与连接 ID 相等且该 ID 非空时把 IsDefault 设为真。
func chatProviderOf(c connection.Connection, defID string) ChatProvider {
	models := append([]string{}, c.Models...)
	model := connection.ModelFor(c, connection.CapChat)
	if model != "" {
		models = uniqueStrings(append([]string{model}, models...))
	}
	return ChatProvider{
		ID:        c.ID,
		Name:      c.Name,
		Vendor:    c.Vendor,
		Endpoint:  c.Endpoint,
		Model:     model,
		Models:    models,
		HasKey:    c.HasKey,
		IsDefault: c.ID != "" && c.ID == defID,
		Active:    c.Active,
	}
}

// splitSessionModel 解析 connectionId::model。左段必须是已有的 chat 连接，否则整段当作模型名、连接 id 为空。
// raw 去掉首尾空白后按 :: 切开；切不开、左段为空或右段去掉空白后为空，或左段不是已有 chat 连接时，返回空连接 id 和 raw。
func (a *App) splitSessionModel(raw string) (connID, model string) {
	raw = strings.TrimSpace(raw)
	left, right, ok := strings.Cut(raw, "::")
	if !ok || left == "" || strings.TrimSpace(right) == "" {
		return "", raw
	}
	r := a.connReg()
	if r == nil {
		return "", raw
	}
	c, err := r.Get(left)
	if err != nil || !connection.HasCap(c, connection.CapChat) {
		return "", raw
	}
	return c.ID, strings.TrimSpace(right)
}

// resolveChatConnection 先用会话的 ConnectionID。否则按模型在活动 chat 连接里匹配，多个命中时优先默认连接。仍没有则用活动 chat 连接。
// meta 的 ConnectionID 去掉首尾空白后非空时用来 Get；Model 去掉首尾空白后非空时用来在活动 chat 连接里匹配。
func (a *App) resolveChatConnection(meta SessionMeta) (connection.Connection, error) {
	r := a.connReg()
	if r == nil {
		return connection.Connection{}, fmt.Errorf("connection registry missing")
	}
	if id := strings.TrimSpace(meta.ConnectionID); id != "" {
		if c, err := r.Get(id); err == nil && connection.HasCap(c, connection.CapChat) {
			return c, nil
		}
	}
	model := strings.TrimSpace(meta.Model)
	if model != "" {
		list, _ := r.List(connection.CapChat)
		var matches []connection.Connection
		for _, c := range list {
			if !c.Active {
				continue
			}
			if connection.ModelFor(c, connection.CapChat) == model {
				matches = append(matches, c)
				continue
			}
			for _, m := range c.Models {
				if m == model {
					matches = append(matches, c)
					break
				}
			}
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		defID := r.Defaults()[connection.CapChat]
		for _, c := range matches {
			if c.ID == defID {
				return c, nil
			}
		}
		if len(matches) > 0 {
			return matches[0], nil
		}
	}
	return r.Active(connection.CapChat, "")
}

// defaultChatSlot 返回活动 chat 连接的模型和 id。库不可用时只返回 Config.Model。
func (a *App) defaultChatSlot() (model, connID string) {
	model = strings.TrimSpace(a.Config.Model)
	r := a.connReg()
	if r == nil {
		return model, ""
	}
	c, err := r.Active(connection.CapChat, "")
	if err != nil {
		return model, ""
	}
	if m := connection.ModelFor(c, connection.CapChat); m != "" {
		model = m
	}
	return model, c.ID
}

// chatRoutes 先放会话选中的连接和模型，再放该连接的其他模型，然后是默认连接和其他活动连接。没有密钥的连接不进入路线。
// meta 传给 resolveChatConnection；Model 去掉首尾空白后为空时改用该连接的 ModelFor，仍为空再用 Config.Model。
func (a *App) chatRoutes(meta SessionMeta) []runtime.ChatRoute {
	r := a.connReg()
	if r == nil {
		return nil
	}
	list, err := r.List(connection.CapChat)
	if err != nil || len(list) == 0 {
		return nil
	}
	defID := r.Defaults()[connection.CapChat]
	primary, err := a.resolveChatConnection(meta)
	if err != nil || strings.TrimSpace(primary.ID) == "" {
		if c, e := r.Active(connection.CapChat, ""); e == nil {
			primary = c
		}
	}
	primaryModel := strings.TrimSpace(meta.Model)
	if primaryModel == "" {
		primaryModel = connection.ModelFor(primary, connection.CapChat)
	}
	if primaryModel == "" {
		primaryModel = strings.TrimSpace(a.Config.Model)
	}

	var routes []runtime.ChatRoute
	seen := map[string]bool{}
	add := func(c connection.Connection, model string) {
		model = strings.TrimSpace(model)
		if !c.Active || strings.TrimSpace(c.ID) == "" || model == "" {
			return
		}
		key := c.ID + "\x00" + model
		if seen[key] {
			return
		}
		cl := a.openAIForConn(c)
		if cl == nil || strings.TrimSpace(cl.APIKey) == "" {
			return
		}
		seen[key] = true
		name := strings.TrimSpace(c.Name)
		if name == "" {
			name = c.Vendor
		}
		routes = append(routes, runtime.ChatRoute{
			ID:     c.ID,
			Name:   name,
			Model:  model,
			Client: cl,
		})
	}

	if strings.TrimSpace(primary.ID) != "" {
		add(primary, primaryModel)
		for _, m := range uniqueStrings([]string{connection.ModelFor(primary, connection.CapChat)}, primary.Models) {
			add(primary, m)
		}
	}
	var rest []connection.Connection
	var def *connection.Connection
	for i := range list {
		c := list[i]
		if !c.Active || c.ID == primary.ID {
			continue
		}
		if c.ID == defID {
			cc := c
			def = &cc
			continue
		}
		rest = append(rest, c)
	}
	if def != nil {
		rest = append([]connection.Connection{*def}, rest...)
	}
	for _, c := range rest {
		for _, m := range uniqueStrings([]string{connection.ModelFor(c, connection.CapChat)}, c.Models) {
			add(c, m)
		}
	}
	return routes
}

// openAIForConn 用连接端点；端点空则用 Config.BaseURL。密钥先向库租用。默认连接租用失败时读 Vault.GetStored("default")，该调用不读环境变量。只有 chatAllowsEnvKey 为真才再 Vault.Get。密钥仍空则返回 nil。
func (a *App) openAIForConn(c connection.Connection) *runtime.OpenAIClient {
	base := strings.TrimSpace(c.Endpoint)
	if base == "" && a != nil {
		base = a.Config.BaseURL
	}
	key := ""
	r := a.connReg()
	if r != nil {
		if k, e := r.Lease(c); e == nil {
			key = k
		}
	}
	key = strings.TrimSpace(key)
	if key == "" && r != nil && a != nil && a.Vault != nil {
		defID := r.Defaults()[connection.CapChat]
		if c.ID != "" && c.ID == defID {
			if stored, err := a.Vault.GetStored("default"); err == nil {
				key = strings.TrimSpace(stored)
			}
		}
	}
	if key == "" && chatAllowsEnvKey(c) && a != nil && a.Vault != nil {
		if k, err := a.Vault.Get("default"); err == nil {
			key = strings.TrimSpace(k)
		}
	}
	if strings.TrimSpace(key) == "" {
		return nil
	}
	return runtime.NewOpenAIClient(base, key)
}

// chatAllowsEnvKey 在厂商为 openai 或 azure，或端点包含 api.openai.com、openai.azure.com 时返回真。
func chatAllowsEnvKey(c connection.Connection) bool {
	v := strings.ToLower(strings.TrimSpace(c.Vendor))
	if v == "openai" || v == "azure" {
		return true
	}
	ep := strings.ToLower(strings.TrimSpace(c.Endpoint))
	return strings.Contains(ep, "api.openai.com") || strings.Contains(ep, "openai.azure.com")
}

// envChatFallback 仅当 Config 的厂商和端点通过 chatAllowsEnvKey 时，用 Vault.Get("default") 做客户端。否则返回 nil。
func (a *App) envChatFallback() runtime.Client {
	if a == nil || a.Vault == nil {
		return nil
	}
	c := connection.Connection{Vendor: a.Config.Provider, Endpoint: a.Config.BaseURL}
	if !chatAllowsEnvKey(c) {
		return nil
	}
	key, err := a.Vault.Get("default")
	if err != nil || strings.TrimSpace(key) == "" {
		return nil
	}
	return runtime.NewOpenAIClient(a.Config.BaseURL, key)
}
