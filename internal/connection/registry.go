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

type Secrets interface {
	Lease(name string) (string, error)
	Set(name, value string)
	Get(name string) (string, error)
}

type Registry struct {
	mu    sync.Mutex
	docs  *filestore.Store
	vault Secrets
}

func Open(dir string, vault Secrets) (*Registry, error) {
	docs, err := filestore.New(dir)
	if err != nil {
		return nil, err
	}
	r := &Registry{docs: docs, vault: vault}
	_ = r.ensureCAS()
	return r, nil
}

func (r *Registry) List(cap string) ([]Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.listLocked(cap)
}

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

func (r *Registry) Get(id string) (Connection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.getLocked(id)
}

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

func (r *Registry) hydrateKey(c *Connection) {
	if r.vault == nil || c.VaultKey == "" {
		return
	}
	if _, err := r.vault.Get(c.VaultKey); err == nil {
		c.HasKey = true
	}
}

func (r *Registry) Defaults() Defaults {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.defaultsLocked()
}

func (r *Registry) defaultsLocked() Defaults {
	var d Defaults
	_ = r.docs.Get("_defaults.json", &d)
	if d == nil {
		d = Defaults{}
	}
	return d
}

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
		// keep chat slot alias so existing Client() and env fallback still work
		if HasCap(c, CapChat) {
			r.vault.Set("default", apiKey)
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

func stripKey(c Connection) Connection {
	c.HasKey = false
	return c
}

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

func (r *Registry) Lease(c Connection) (string, error) {
	if r.vault == nil {
		return "", fmt.Errorf("vault missing")
	}
	if c.VaultKey == "" {
		return "", fmt.Errorf("missing vault key")
	}
	return r.vault.Lease(c.VaultKey)
}

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

func (r *Registry) SeedChat(provider, baseURL, model string, extraModels ...string) (Connection, error) {
	list, err := r.List(CapChat)
	if err != nil {
		return Connection{}, err
	}
	if len(list) > 0 {
		return r.adoptDefaultKey(list[0]), nil
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

func hostOf(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	if i := strings.IndexByte(u, '/'); i >= 0 {
		u = u[:i]
	}
	return u
}

func nowISO() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func DirOfHome(root string) string {
	return filepath.Join(root, "connections")
}

func MustExist(dir string) error {
	return os.MkdirAll(dir, 0o755)
}
