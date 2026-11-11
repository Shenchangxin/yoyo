// adopt.go 在 agent 刚装好、还没自己配过时，把 yoyo 当前的模型和钥匙抄过来。
// 已经写过的模型和钥匙不动。复用开关仍是人另开的，这里不碰。

package buddy

import (
	"slices"
	"strings"
)

// adoptYoyoChat 给 id 填上 yoyo 的模型和对应钥匙。模型空着才填；一把钥匙都没有才写入。
// 端点不是这家的官方地址时，把地址记进环境变量，钥匙才能打到同一处。
func (m *Manager) adoptYoyoChat(id string) {
	if m == nil || m.opts.YoyoChat == nil || m.opts.Settings == nil {
		return
	}
	model, provider, key, endpoint := m.opts.YoyoChat()
	model = strings.TrimSpace(model)
	provider = strings.TrimSpace(provider)
	key = strings.TrimSpace(key)
	endpoint = strings.TrimSpace(endpoint)
	if model == "" && key == "" {
		return
	}
	m.mu.Lock()
	e, ok := m.entryLocked(id)
	m.mu.Unlock()
	if !ok || e.Builtin {
		return
	}
	item := m.opts.Settings().Item(id)
	name := ""
	if key != "" && !m.hasOwnKey(e, item) {
		name = chatSecretName(e, provider)
	}
	envName, envVal := "", ""
	if name != "" {
		envName, envVal = baseURLEnv(provider, endpoint)
		if envName == "" || item.Env[envName] != "" {
			envName, envVal = "", ""
		}
	}
	if item.Model != "" && name == "" && envName == "" {
		return
	}
	if model == "" && name == "" && envName == "" {
		return
	}
	if name != "" {
		m.setSecret(id, name, key)
	}
	if m.opts.SaveSettings == nil {
		return
	}
	_ = m.opts.SaveSettings(func(s *Settings) {
		if s.Items == nil {
			s.Items = map[string]ItemConfig{}
		}
		cur := s.Items[id]
		if cur.Model == "" && model != "" {
			cur.Model = model
		}
		if name != "" && !slices.Contains(cur.Secrets, name) {
			cur.Secrets = append(cur.Secrets, name)
			slices.Sort(cur.Secrets)
		}
		if envName != "" && cur.Env[envName] == "" {
			if cur.Env == nil {
				cur.Env = map[string]string{}
			}
			cur.Env[envName] = envVal
		}
		s.Items[id] = cur
	})
}

// hasOwnKey 在这项已经存过任何一把钥匙时为真。
func (m *Manager) hasOwnKey(e Entry, item ItemConfig) bool {
	for _, name := range secretNames(e, item) {
		if m.hasSecret(e.ID, name) {
			return true
		}
	}
	return false
}

// chatSecretName 决定钥匙写进哪个变量。对得上提供商的用它声明的名字；
// 只声明了一把、而且没写提供商或提供商相同的，用那一把；其余用 yoyo 这边的常规名字。
func chatSecretName(e Entry, provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	for _, s := range e.Secrets {
		if s.Provider != "" && strings.EqualFold(s.Provider, provider) {
			return s.Name
		}
	}
	if len(e.Secrets) == 1 {
		spec := e.Secrets[0]
		if spec.Provider == "" || strings.EqualFold(spec.Provider, provider) {
			return spec.Name
		}
	}
	switch provider {
	case "anthropic", "claude":
		return "ANTHROPIC_API_KEY"
	case "gemini", "google":
		return "GEMINI_API_KEY"
	default:
		return "OPENAI_API_KEY"
	}
}

// baseURLEnv 在端点不是官方地址时，返回要写入的变量名和值。官方地址或空端点不写。
func baseURLEnv(provider, endpoint string) (string, string) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" || officialChatEndpoint(provider, endpoint) {
		return "", ""
	}
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic", "claude":
		return "ANTHROPIC_BASE_URL", endpoint
	case "gemini", "google":
		return "", ""
	default:
		return "OPENAI_BASE_URL", endpoint
	}
}

// officialChatEndpoint 判断端点是不是这家的官方地址。空端点算官方。
func officialChatEndpoint(provider, endpoint string) bool {
	ep := strings.ToLower(strings.TrimSpace(endpoint))
	if ep == "" {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic", "claude":
		return strings.Contains(ep, "api.anthropic.com")
	case "openai", "azure":
		return strings.Contains(ep, "api.openai.com") || strings.Contains(ep, "openai.azure.com")
	case "gemini", "google":
		return strings.Contains(ep, "generativelanguage.googleapis.com")
	default:
		return false
	}
}
