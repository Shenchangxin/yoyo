// Package connection 保存供应商连接。JSON 文档里的 VaultKey 是钥匙名字，结构体没有密钥原文字段。
// Upsert 把传入的 apiKey 交给 Secrets.Set，落盘前把 HasKey 写成 false。Settings 原样写入，本包不检查其中是否夹带秘密。
// adoptDefaultKey 只复制 GetStored 读到的已保存值，不读进程环境变量。
package connection

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/filestore"
)

// Secrets 是按名字读写钥匙的仓库。Lease 取一次可用值，Get 读取，Set 写入。
type Secrets interface {
	Lease(name string) (string, error)
	Set(name, value string)
	Get(name string) (string, error)
}

// Registry 把连接 JSON 放在 filestore 目录里，钥匙交给 vault。
type Registry struct {
	mu    sync.Mutex
	docs  *filestore.Store
	vault Secrets
}

// Open 用 filestore 打开 dir，并确保有一条本地 CAS 存储连接。
// vault 写入 Registry 的 vault 字段。
func Open(dir string, vault Secrets) (*Registry, error) {
	docs, err := filestore.New(dir)
	if err != nil {
		return nil, err
	}
	r := &Registry{docs: docs, vault: vault}
	_ = r.ensureCAS()
	return r, nil
}

// List 返回具备该能力的连接。cap 为空则返回全部非下划线文档。顺序先按 Priority 从高到低，再按 CreatedAt。
func (r *Registry) List(cap string) ([]Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.listLocked(cap)
}

// listLocked 在已持有锁时列出连接，并给每条 hydrateKey。id 为空或以 _ 开头的文档跳过。
// cap 传给 NormalizeCap。结果非空时，HasCap 不匹配的连接跳过。
func (r *Registry) listLocked(cap string) ([]Connection, error) {
	ids, err := r.docs.IDs("")
	if err != nil {
		return nil, err
	}
	want := NormalizeCap(cap)
	var out []Connection
	for _, id := range ids {
		if id == "" || strings.HasPrefix(id, "_") {
			continue
		}
		var c Connection
		if err := r.docs.Get(id+".json", &c); err != nil {
			continue
		}
		if c.ID == "" {
			c.ID = id
		}
		r.hydrateKey(&c)
		if want != "" && !HasCap(c, want) {
			continue
		}
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		return out[i].CreatedAt < out[j].CreatedAt
	})
	if out == nil {
		out = []Connection{}
	}
	return out, nil
}

// Get 按 id 读取一条连接。找不到返回 connection not found。
func (r *Registry) Get(id string) (Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.getLocked(id)
}

// getLocked 在已持有锁时读取 id.json。id 为空或文件不存在都返回 connection not found。
func (r *Registry) getLocked(id string) (Connection, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Connection{}, fmt.Errorf("connection not found")
	}
	var c Connection
	if err := r.docs.Get(id+".json", &c); err != nil {
		return Connection{}, fmt.Errorf("connection not found")
	}
	if c.ID == "" {
		c.ID = id
	}
	r.hydrateKey(&c)
	return c, nil
}

// hydrateKey 在仓库里该 VaultKey 已有非空保存值，或 Get 成功时，把 HasKey 设为真。不把钥匙原文写进连接。
// c 的 VaultKey 为空或 vault 为 nil 时直接返回。storedVault 非空或 Get 成功时把 HasKey 设为真。
func (r *Registry) hydrateKey(c *Connection) {
	if r.vault == nil || c.VaultKey == "" {
		return
	}
	if storedVault(r.vault, c.VaultKey) != "" {
		c.HasKey = true
		return
	}
	if _, err := r.vault.Get(c.VaultKey); err == nil {
		c.HasKey = true
	}
}

// Defaults 返回各能力当前的默认连接 id。
func (r *Registry) Defaults() Defaults {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.defaultsLocked()
}

// defaultsLocked 读取 _defaults.json。文件缺失或为空时返回空映射。
func (r *Registry) defaultsLocked() Defaults {
	var d Defaults
	_ = r.docs.Get("_defaults.json", &d)
	if d == nil {
		d = Defaults{}
	}
	return d
}

// SetDefault 把该能力的默认连接设为 id。id 为空则删掉这一项。
// cap 传给 NormalizeCap，作为默认表的键。
func (r *Registry) SetDefault(cap, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cap = NormalizeCap(cap)
	d := r.defaultsLocked()
	if id == "" {
		delete(d, cap)
	} else {
		d[cap] = id
	}
	return r.docs.Put("_defaults.json", d)
}

// Upsert 写入连接。VaultKey 为空时用 conn.<id>.key。apiKey 非空则 Set 到该名字；若这是默认聊天槽或还没有默认聊天槽，再 Set 到 default。落盘的副本把 HasKey 写成 false。
func (r *Registry) Upsert(c Connection, apiKey string) (Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := nowISO()
	if c.ID == "" {
		c.ID = newID()
		c.CreatedAt = now
		c.Name = r.uniqueNameLocked(c.Name, c.ID)
	} else if cur, err := r.getLocked(c.ID); err == nil {
		if c.VaultKey == "" {
			c.VaultKey = cur.VaultKey
		}
		if c.CreatedAt == "" {
			c.CreatedAt = cur.CreatedAt
		}
		if c.Settings == nil {
			c.Settings = cur.Settings
		}
	}
	c.UpdatedAt = now
	if c.VaultKey == "" {
		c.VaultKey = "conn." + c.ID + ".key"
	}
	if c.DefaultModel == nil {
		c.DefaultModel = map[string]string{}
	}
	for i, cap := range c.Capabilities {
		c.Capabilities[i] = NormalizeCap(cap)
	}
	if c.Protocol == "" && len(c.Capabilities) > 0 {
		c.Protocol = InferProtocol(c.Vendor, c.Capabilities[0])
	}
	if apiKey != "" && r.vault != nil {
		r.vault.Set(c.VaultKey, apiKey)
		c.HasKey = true
		// 只把默认聊天槽的钥匙再写一份到名字 default，其余供应商保持自己的钥匙。
		if HasCap(c, CapChat) {
			d := r.defaultsLocked()
			if d[CapChat] == c.ID || d[CapChat] == "" {
				r.vault.Set("default", apiKey)
			}
		}
	} else {
		r.hydrateKey(&c)
	}
	pub := c
	pub.HasKey = false
	if err := r.docs.Put(c.ID+".json", stripKey(pub)); err != nil {
		return c, err
	}
	if c.Active {
		for _, cap := range c.Capabilities {
			d := r.defaultsLocked()
			if d[cap] == "" {
				d[cap] = c.ID
				_ = r.docs.Put("_defaults.json", d)
			}
		}
	}
	r.hydrateKey(&c)
	return c, nil
}

// stripKey 返回 HasKey 为 false 的副本，供写入 JSON。它不改 VaultKey。
// c 的 HasKey 写成 false 后返回。
func stripKey(c Connection) Connection {
	c.HasKey = false
	return c
}

// uniqueNameLocked 在名称与其他连接冲突时追加空格和序号，跳过 skipID。
// name 去掉空白后为空则原样返回。与其他连接重名时在后面加空格和从 2 起、小于 100 的序号。到 99 仍冲突则再加空格和 newID 的前 6 个字符。
func (r *Registry) uniqueNameLocked(name, skipID string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		return base
	}
	all, _ := r.listLocked("")
	exists := func(n string) bool {
		for _, c := range all {
			if c.ID != skipID && c.Name == n {
				return true
			}
		}
		return false
	}
	if !exists(base) {
		return base
	}
	for n := 2; n < 100; n++ {
		cand := fmt.Sprintf("%s %d", base, n)
		if !exists(cand) {
			return cand
		}
	}
	return base + " " + newID()[:6]
}

// Delete 把该连接的 VaultKey 在仓库里 Set 成空串，并从默认表里去掉指向它的项，再删除 id.json。
func (r *Registry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, err := r.getLocked(id)
	if err == nil && r.vault != nil && c.VaultKey != "" {
		r.vault.Set(c.VaultKey, "")
	}
	d := r.defaultsLocked()
	changed := false
	for k, v := range d {
		if v == id {
			delete(d, k)
			changed = true
		}
	}
	if changed {
		_ = r.docs.Put("_defaults.json", d)
	}
	return r.docs.Delete(id + ".json")
}

// Active 先按 id 找，再按该能力的默认项，然后在已启用且有钥匙的连接里选。存储能力的 cas 协议可以没有钥匙。都没有则返回错误。
// cap 传给 NormalizeCap。非空时要求 HasCap 匹配。也作为默认表的键传给 listLocked，并写进找不到连接时的错误。
func (r *Registry) Active(cap, id string) (Connection, error) {
	cap = NormalizeCap(cap)
	if id != "" {
		if c, err := r.Get(id); err == nil && (cap == "" || HasCap(c, cap)) {
			return c, nil
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	d := r.defaultsLocked()
	if def := d[cap]; def != "" {
		if c, err := r.getLocked(def); err == nil && c.Active && (cap == "" || HasCap(c, cap)) {
			r.hydrateKey(&c)
			if c.HasKey || cap == CapStorage {
				return c, nil
			}
		}
	}
	list, err := r.listLocked(cap)
	if err != nil {
		return Connection{}, err
	}
	var keyed, anyActive Connection
	var hasKeyed, hasAny bool
	for _, c := range list {
		if !c.Active {
			continue
		}
		if !hasAny {
			anyActive = c
			hasAny = true
		}
		if c.HasKey || cap == CapStorage && c.Protocol == "cas" {
			if d[cap] == c.ID {
				return c, nil
			}
			if !hasKeyed {
				keyed = c
				hasKeyed = true
			}
		}
	}
	if hasKeyed {
		return keyed, nil
	}
	if hasAny {
		return anyActive, nil
	}
	return Connection{}, fmt.Errorf("no active %s connection — add one in Settings", cap)
}

// Lease 按连接的 VaultKey 向仓库租用钥匙原文。仓库缺失或 VaultKey 为空时返回错误。
// c 的 VaultKey 为空时返回 missing vault key。否则传给 vault.Lease。
func (r *Registry) Lease(c Connection) (string, error) {
	if r.vault == nil {
		return "", fmt.Errorf("vault missing")
	}
	if c.VaultKey == "" {
		return "", fmt.Errorf("missing vault key")
	}
	return r.vault.Lease(c.VaultKey)
}

// ensureCAS 在还没有 protocol 为 cas 的存储连接时写入 cas-local，并在存储默认项为空时把它设为默认。
func (r *Registry) ensureCAS() error {
	all, _ := r.listLocked(CapStorage)
	for _, c := range all {
		if c.Protocol == "cas" {
			return nil
		}
	}
	now := nowISO()
	c := Connection{
		ID: "cas-local", Name: "Local CAS", Vendor: "cas", Protocol: "cas",
		Endpoint: "local", VaultKey: "", Capabilities: []string{CapStorage},
		Active: true, Priority: 100, CreatedAt: now, UpdatedAt: now,
		Settings: map[string]any{"provider": "cas"},
	}
	if err := r.docs.Put(c.ID+".json", stripKey(c)); err != nil {
		return err
	}
	d := r.defaultsLocked()
	if d[CapStorage] == "" {
		d[CapStorage] = c.ID
		_ = r.docs.Put("_defaults.json", d)
	}
	return nil
}

// SeedChat 在已有聊天连接时不新建，只对当前默认或列表第一条调用 adoptDefaultKey。没有时按供应商创建一条并设为默认。
// provider 只在还没有聊天连接时使用。去掉空白后为空则写成 openai，等于 anthropic 时写成 claude，并作为 Name 和 Vendor。
// baseURL 只在还没有聊天连接时使用。为空则写成 https://api.openai.com/v1，作为 Endpoint。
// model 只在还没有聊天连接时使用。放进 DefaultModel 的聊天项，并和 extraModels 一起经 compact 放进 Models。
// extraModels 只在还没有聊天连接时使用。接在 model 后面放进 Models。
func (r *Registry) SeedChat(provider, baseURL, model string, extraModels ...string) (Connection, error) {
	list, err := r.List(CapChat)
	if err != nil {
		return Connection{}, err
	}
	if len(list) > 0 {
		cur := list[0]
		if id := r.Defaults()[CapChat]; id != "" {
			if c, err := r.Get(id); err == nil {
				cur = c
			}
		}
		return r.adoptDefaultKey(cur), nil
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "openai"
	}
	if provider == "anthropic" {
		provider = "claude"
	}
	name := provider
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	c, err := r.Upsert(Connection{
		Name: name, Vendor: provider, Endpoint: baseURL,
		Protocol:     InferProtocol(provider, CapChat),
		Capabilities: []string{CapChat}, Models: compact(append([]string{model}, extraModels...)),
		DefaultModel: map[string]string{CapChat: model}, Active: true, Priority: 50,
	}, "")
	if err != nil {
		return c, err
	}
	_ = r.SetDefault(CapChat, c.ID)
	return r.adoptDefaultKey(c), nil
}

// ImportProvider 用给出的 vaultKey 名字导入一条连接，apiKey 传空串，因此这里不写入钥匙原文。isDefault 为真时设成该能力的默认项。
// serviceType 传给 NormalizeCap。结果为空时改用 CapImage，作为能力、InferProtocol 的能力和默认模型的键。
// vendor 写入 Vendor，并传给 InferProtocol。
// name 写入 Name。
// baseURL 写入 Endpoint。
// model 非空时放进模型列表开头。DefaultModel 里该能力的值就是 model。
// models 传给 ParseModels。
// priority 写入 Priority。
// active 写入 Active。
// settings 去掉空白后非空且不是 {} 时按 JSON 对象解进 Settings。
// created 写入 CreatedAt。
// updated 写入 UpdatedAt。
func (r *Registry) ImportProvider(serviceType, vendor, name, baseURL, vaultKey, model, models string, priority int, active, isDefault bool, settings string, created, updated, id string) (Connection, error) {
	cap := NormalizeCap(serviceType)
	if cap == "" {
		cap = CapImage
	}
	ms := ParseModels(models)
	if model != "" {
		ms = compact(append([]string{model}, ms...))
	}
	st := map[string]any{}
	if strings.TrimSpace(settings) != "" && settings != "{}" {
		_ = json.Unmarshal([]byte(settings), &st)
	}
	if id == "" {
		id = newID()
	}
	c := Connection{
		ID: id, Name: name, Vendor: vendor, Endpoint: baseURL, VaultKey: vaultKey,
		Protocol: InferProtocol(vendor, cap), Capabilities: []string{cap},
		Models: ms, DefaultModel: map[string]string{cap: model},
		Active: active, Priority: priority, Settings: st,
		CreatedAt: created, UpdatedAt: updated,
	}
	if c.VaultKey == "" {
		c.VaultKey = "conn." + c.ID + ".key"
	}
	out, err := r.Upsert(c, "")
	if err != nil {
		return out, err
	}
	if isDefault {
		_ = r.SetDefault(cap, out.ID)
	}
	return out, nil
}

// MergeSameAccount 把同一供应商、同一主机的连接并到创建最早的那条。两边都有钥匙且 VaultKey 不同则不并。本地 cas 存储不参与。
func (r *Registry) MergeSameAccount() {
	list, err := r.List("")
	if err != nil {
		return
	}
	type key struct{ vendor, host string }
	by := map[key][]Connection{}
	for _, c := range list {
		if HasCap(c, CapStorage) && c.Protocol == "cas" {
			continue
		}
		k := key{strings.ToLower(c.Vendor), strings.ToLower(hostOf(c.Endpoint))}
		if k.vendor == "" || k.host == "" {
			continue
		}
		by[k] = append(by[k], c)
	}
	for _, group := range by {
		if len(group) < 2 {
			continue
		}
		sort.Slice(group, func(i, j int) bool { return group[i].CreatedAt < group[j].CreatedAt })
		keep := group[0]
		caps := append([]string{}, keep.Capabilities...)
		models := append([]string{}, keep.Models...)
		defaults := map[string]string{}
		for k, v := range keep.DefaultModel {
			defaults[k] = v
		}
		for _, extra := range group[1:] {
			if extra.VaultKey != keep.VaultKey && extra.HasKey && keep.HasKey {
				continue
			}
			for _, cap := range extra.Capabilities {
				if !HasCap(keep, cap) {
					caps = append(caps, cap)
				}
				if ModelFor(extra, cap) != "" {
					defaults[cap] = ModelFor(extra, cap)
				}
			}
			models = compact(append(models, extra.Models...))
			defs := r.Defaults()
			for cap, id := range defs {
				if id == extra.ID {
					_ = r.SetDefault(cap, keep.ID)
				}
			}
			_ = r.Delete(extra.ID)
		}
		keep.Capabilities = caps
		keep.Models = models
		keep.DefaultModel = defaults
		_, _ = r.Upsert(keep, "")
	}
}

// hostOf 去掉协议和路径，返回端点里的主机部分。
// u 去掉空白，再去掉 https:// 或 http:// 前缀，然后丢掉第一个 / 及其后面的部分。
func hostOf(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	if i := strings.IndexByte(u, '/'); i >= 0 {
		u = u[:i]
	}
	return u
}

// nowISO 返回当前 UTC 的 RFC3339Nano 文本。
func nowISO() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// newID 返回 16 字节随机数的十六进制。
func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// DirOfHome 返回 root 下的 connections 目录。调用方决定 root 是不是主目录。
func DirOfHome(root string) string {
	return filepath.Join(root, "connections")
}

// MustExist 按 0o755 创建目录。
// dir 传给 os.MkdirAll，权限 0o755。
func MustExist(dir string) error {
	return os.MkdirAll(dir, 0o755)
}
