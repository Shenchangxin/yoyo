// registry.go 按标识登记协议适配器，并按能力和使用面列出元数据。

package protocol

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Registry 保存当前进程里的协议适配器。
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

// NewRegistry 依次登记传入的适配器。其中一个失败就返回该错误。
func NewRegistry(adapters ...Adapter) (*Registry, error) {
	r := &Registry{adapters: make(map[string]Adapter, len(adapters))}
	for _, adapter := range adapters {
		if err := r.Register(adapter); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Register 用元数据里的标识登记适配器。适配器为空，或标识、版本为空，或标识已存在时失败。
func (r *Registry) Register(adapter Adapter) error {
	if adapter == nil {
		return fmt.Errorf("protocol adapter is nil")
	}
	metadata := adapter.Metadata()
	metadata.ID = strings.TrimSpace(metadata.ID)
	metadata.Version = strings.TrimSpace(metadata.Version)
	if metadata.ID == "" || metadata.Version == "" {
		return fmt.Errorf("protocol adapter metadata requires id and version")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.adapters[metadata.ID]; exists {
		return fmt.Errorf("protocol adapter %q is already registered", metadata.ID)
	}
	r.adapters[metadata.ID] = adapter
	return nil
}

// Unregister 从当前登记表去掉这个标识。调用方负责持久化，登记表只表示此刻的插件快照。标识不在时返回假。
func (r *Registry) Unregister(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	id = strings.TrimSpace(id)
	if _, ok := r.adapters[id]; !ok {
		return false
	}
	delete(r.adapters, id)
	return true
}

// RegisterManifest 从清单字节装入适配器再登记。
func (r *Registry) RegisterManifest(data []byte) error {
	adapter, err := LoadManifest(data)
	if err != nil {
		return err
	}
	return r.Register(adapter)
}

// Get 按去掉空白后的标识取出适配器。
func (r *Registry) Get(id string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.adapters[strings.TrimSpace(id)]
	return adapter, ok
}

// Resolve 先按标识取，没有再比各适配器的 LegacyAliases。
func (r *Registry) Resolve(id string) (Adapter, bool) {
	if adapter, ok := r.Get(id); ok {
		return adapter, true
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, adapter := range r.adapters {
		for _, alias := range adapter.Metadata().LegacyAliases {
			if strings.TrimSpace(alias) == strings.TrimSpace(id) {
				return adapter, true
			}
		}
	}
	return nil, false
}

// List 按标识排序返回元数据。includeUnavailable 为假时跳过未启用或带不可用原因的适配器。能力和使用面非空时还要匹配。
func (r *Registry) List(surface Surface, capability Capability, includeUnavailable bool) []Metadata {
	r.mu.RLock()
	items := make([]Metadata, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		metadata := adapter.Metadata()
		if !includeUnavailable && (!metadata.Enabled || metadata.UnavailableReason != "") {
			continue
		}
		if capability != "" && !containsCapability(metadata.Categories, capability) {
			continue
		}
		if surface != "" && !containsSurface(metadata.Scopes, surface) {
			continue
		}
		items = append(items, metadata)
	}
	r.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

// IsCapability 在解析得到的适配器已启用、没有不可用原因，且类别里含该能力时为真。
func (r *Registry) IsCapability(id string, capability Capability) bool {
	adapter, ok := r.Resolve(id)
	if !ok {
		return false
	}
	metadata := adapter.Metadata()
	return metadata.Enabled && metadata.UnavailableReason == "" && containsCapability(metadata.Categories, capability)
}

// containsCapability 报告能力切片里有没有这个值。
func containsCapability(values []Capability, value Capability) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

// containsSurface 报告使用面切片里有没有这个值。
func containsSurface(values []Surface, value Surface) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
