// secrets.go 拥有 agent 的密钥：以 buddy.<id>.<NAME> 为名存进 vault，只在启动进程时读出注入。
// 对外只给 hasKey，从不返回值。「从 yoyo 复用」由 Options.YoyoKey 按提供商取 yoyo 自己的钥匙。
// 列表和视图只看「有没有」，结果缓存在 Manager 里：系统钥匙串在 macOS 和 Linux 上每读一次就起一个进程，界面每秒轮询时不能每次都去读。

package buddy

import (
	"slices"
	"sort"
	"strings"
	"time"
)

// yoyoKeyTTL 是「yoyo 有没有这个提供商的钥匙」的缓存时长。人在设置里加了钥匙，最多这么久之后 Buddy 就能看到。
const yoyoKeyTTL = 30 * time.Second

// Vault 是 Manager 用到的那部分密钥仓库。
type Vault interface {
	GetStored(name string) (string, error)
	Set(name, value string)
}

// yoyoKeyHit 是一次「yoyo 有没有钥匙」的查询结果和时间。
type yoyoKeyHit struct {
	ok bool
	at time.Time
}

// secretKey 是 vault 里的名字。
func secretKey(id, name string) string { return "buddy." + id + "." + name }

// storedSecret 读一把已经保存的密钥，没有时返回空串。顺手记下有没有。
func (m *Manager) storedSecret(id, name string) string {
	if m.opts.Vault == nil {
		return ""
	}
	v, err := m.opts.Vault.GetStored(secretKey(id, name))
	if err != nil {
		v = ""
	}
	v = strings.TrimSpace(v)
	m.keyMu.Lock()
	m.keyCache[secretKey(id, name)] = v != ""
	m.keyMu.Unlock()
	return v
}

// hasSecret 判断一把密钥有没有保存。只有第一次会去读 vault，之后读缓存；setSecret 会同步更新缓存。
func (m *Manager) hasSecret(id, name string) bool {
	m.keyMu.Lock()
	has, ok := m.keyCache[secretKey(id, name)]
	m.keyMu.Unlock()
	if ok {
		return has
	}
	return m.storedSecret(id, name) != ""
}

// setSecret 保存或删除（value 为空）一把密钥，并更新缓存。
func (m *Manager) setSecret(id, name, value string) {
	value = strings.TrimSpace(value)
	if m.opts.Vault != nil {
		m.opts.Vault.Set(secretKey(id, name), value)
	}
	m.keyMu.Lock()
	m.keyCache[secretKey(id, name)] = value != "" && m.opts.Vault != nil
	m.keyMu.Unlock()
}

// refreshSecrets 重新从 vault 读一遍这一项的密钥在不在，并让相关的 yoyo 钥匙缓存失效。探测前调用：探测本来就要读真值。
func (m *Manager) refreshSecrets(e Entry, item ItemConfig) {
	for _, name := range secretNames(e, item) {
		m.storedSecret(e.ID, name)
	}
	m.keyMu.Lock()
	for _, s := range e.Secrets {
		delete(m.yoyoKeys, s.Provider)
	}
	m.keyMu.Unlock()
}

// yoyoHas 判断 yoyo 自己有没有这个提供商的钥匙，结果缓存 yoyoKeyTTL。只给视图和必填检查用，注入时另取值。
func (m *Manager) yoyoHas(provider string) bool {
	if provider == "" || m.opts.YoyoKey == nil {
		return false
	}
	now := time.Now()
	m.keyMu.Lock()
	hit, ok := m.yoyoKeys[provider]
	m.keyMu.Unlock()
	if ok && now.Sub(hit.at) < yoyoKeyTTL {
		return hit.ok
	}
	key, found := m.opts.YoyoKey(provider)
	has := found && strings.TrimSpace(key) != ""
	m.keyMu.Lock()
	m.yoyoKeys[provider] = yoyoKeyHit{ok: has, at: now}
	m.keyMu.Unlock()
	return has
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
		v := SecretView{Name: name, Label: spec.Label, Optional: !spec.Required, HasKey: m.hasSecret(e.ID, name)}
		if declared && m.yoyoHas(spec.Provider) {
			v.Reusable = true
			v.Reuse = slices.Contains(item.Reuse, name)
		}
		out = append(out, v)
	}
	return out
}

// missingRequired 返回还没填的必填密钥名。复用打开且 yoyo 有钥匙的算已填。只看有没有，不读值。
func (m *Manager) missingRequired(e Entry, item ItemConfig) []string {
	var out []string
	for _, s := range e.Secrets {
		if !s.Required || item.Env[s.Name] != "" || m.hasSecret(e.ID, s.Name) {
			continue
		}
		if slices.Contains(item.Reuse, s.Name) && m.yoyoHas(s.Provider) {
			continue
		}
		out = append(out, s.Name)
	}
	return out
}
