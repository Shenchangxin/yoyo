// secrets.go 拥有 agent 的密钥：以 buddy.<id>.<NAME> 为名存进 vault，只在启动进程时读出注入。
// 对外只给 hasKey，从不返回值。「从 yoyo 复用」由 Options.YoyoKey 按提供商取 yoyo 自己的钥匙。

package buddy

import (
	"slices"
	"sort"
	"strings"
)

// Vault 是 Manager 用到的那部分密钥仓库。
type Vault interface {
	GetStored(name string) (string, error)
	Set(name, value string)
}

// secretKey 是 vault 里的名字。
func secretKey(id, name string) string { return "buddy." + id + "." + name }

// storedSecret 读一把已经保存的密钥，没有时返回空串。
func (m *Manager) storedSecret(id, name string) string {
	if m.opts.Vault == nil {
		return ""
	}
	v, err := m.opts.Vault.GetStored(secretKey(id, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

// secretNames 是这一项要关心的密钥名：精选表声明的，加上人在配置里存过的，排好序。
func secretNames(e Entry, item ItemConfig) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range e.Secrets {
		if !seen[s.Name] {
			seen[s.Name] = true
			out = append(out, s.Name)
		}
	}
	extra := append([]string(nil), item.Secrets...)
	sort.Strings(extra)
	for _, name := range extra {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

// secretEnv 取出这一项的全部密钥，返回要注入的变量和用于打码的值。复用开着时，自己没存的用 yoyo 同类提供商的钥匙补上。
func (m *Manager) secretEnv(e Entry, item ItemConfig) (map[string]string, []string) {
	env := map[string]string{}
	var values []string
	for _, name := range secretNames(e, item) {
		if v := m.storedSecret(e.ID, name); v != "" {
			env[name] = v
			values = append(values, v)
		}
	}
	for _, spec := range e.Secrets {
		if env[spec.Name] != "" || spec.Provider == "" || !slices.Contains(item.Reuse, spec.Name) || m.opts.YoyoKey == nil {
			continue
		}
		if v, ok := m.opts.YoyoKey(spec.Provider); ok && strings.TrimSpace(v) != "" {
			env[spec.Name] = strings.TrimSpace(v)
			values = append(values, env[spec.Name])
		}
	}
	return env, values
}

// secretViews 是界面上的密钥行：名字、标签、是否已保存、能否复用。
func (m *Manager) secretViews(e Entry, item ItemConfig) []SecretView {
	specs := map[string]SecretSpec{}
	for _, s := range e.Secrets {
		specs[s.Name] = s
	}
	out := []SecretView{}
	for _, name := range secretNames(e, item) {
		spec, declared := specs[name]
		v := SecretView{Name: name, Label: spec.Label, Optional: !spec.Required, HasKey: m.storedSecret(e.ID, name) != ""}
		if declared && spec.Provider != "" && m.opts.YoyoKey != nil {
			if key, ok := m.opts.YoyoKey(spec.Provider); ok && key != "" {
				v.Reusable = true
				v.Reuse = slices.Contains(item.Reuse, name)
			}
		}
		out = append(out, v)
	}
	return out
}

// missingRequired 返回还没填的必填密钥名。复用打开且 yoyo 有钥匙的算已填。
func (m *Manager) missingRequired(e Entry, item ItemConfig) []string {
	env, _ := m.secretEnv(e, item)
	var out []string
	for _, s := range e.Secrets {
		if s.Required && env[s.Name] == "" && item.Env[s.Name] == "" {
			out = append(out, s.Name)
		}
	}
	return out
}
