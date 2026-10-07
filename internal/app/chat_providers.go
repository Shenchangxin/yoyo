package app

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

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

func (a *App) connReg() *connection.Registry {
	if a != nil && a.Video != nil {
		return a.Video.Conn
	}
	return nil
}

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
