// registry.go 拥有 ACP 注册表的刷新：每天最多一次，超时 15 秒；校验 version 和 agents 后写进 buddies/registry.json。
// 失败时继续用缓存，没有缓存用内嵌的快照。刷新后重建目录、发出全部视图，再补下载图标。

package buddy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// defaultRegistryURL 是官方注册表地址。
const defaultRegistryURL = "https://cdn.agentclientprotocol.com/registry/v1/latest/registry.json"

// registryTTL 是注册表缓存的有效期。
const registryTTL = 24 * time.Hour

// loadCachedRegistry 读 buddies/registry.json，坏了或读不到时返回假。
func loadCachedRegistry(root string) (Registry, bool) {
	raw, err := os.ReadFile(filepath.Join(root, "registry.json"))
	if err != nil {
		return Registry{}, false
	}
	reg, err := parseRegistry(raw)
	return reg, err == nil
}

// registryURL 选注册表地址：配置里写了 https 地址就用它，其次 Options 里的，最后官方地址。
func (m *Manager) registryURL() string {
	if u := strings.TrimSpace(m.opts.Settings().Registry); strings.HasPrefix(u, "https://") {
		return u
	}
	if m.opts.RegistryURL != "" {
		return m.opts.RegistryURL
	}
	return defaultRegistryURL
}

// refreshRegistry 在缓存过期或 force 时下载注册表。不论下没下载，最后都补一次图标。
func (m *Manager) refreshRegistry(ctx context.Context, force bool) error {
	defer m.fetchIcons(ctx)
	if m.opts.Root == "" {
		return nil
	}
	path := filepath.Join(m.opts.Root, "registry.json")
	if info, err := os.Stat(path); err == nil && !force && m.opts.Now().Sub(info.ModTime()) < registryTTL {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.registryURL(), nil)
	if err != nil {
		return err
	}
	resp, err := m.opts.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry: http %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	reg, err := parseRegistry(raw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.opts.Root, 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	m.mu.Lock()
	m.reg = reg
	m.rebuildLocked()
	m.mu.Unlock()
	m.detectAll(ctx)
	return nil
}
