// Package vault 保存模型供应商的密钥。已保存的值在内存、操作系统钥匙串和权限 0o600 的 vault.json 中。环境变量只在 Get 的名字是 openai 或 default 时回落。
package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// EnvAPIKey 是环境变量 YOYO_API_KEY。Get 在名字为 openai 或 default 时，第一个回落它。
const EnvAPIKey = "YOYO_API_KEY"

// Store 保存供应商密钥。已保存值的查找顺序是内存、操作系统钥匙串、vault.json。
type Store struct {
	mu       sync.Mutex
	mem      map[string]string
	path     string // vault.json 的路径；home 为空时保持空串，不落盘
	keychain bool   // 为真时启用操作系统钥匙串
}

// New 用 home 打开仓库。钥匙串是否启用由本平台的 keychainAvailable 决定。
func New(home string) *Store {
	return open(home, keychainAvailable())
}

// NewFileOnly 只使用 vault.json，不启用操作系统钥匙串，避免测试改写系统钥匙串。
func NewFileOnly(home string) *Store {
	return open(home, false)
}

// open 在 home 非空时把 path 设为该目录下的 vault.json 并载入文件。useKeychain 为真时把内存中已有条目迁入钥匙串。
func open(home string, useKeychain bool) *Store {
	s := &Store{mem: map[string]string{}}
	if home != "" {
		s.path = filepath.Join(home, "vault.json")
		s.loadFile()
	}
	s.keychain = useKeychain
	if s.keychain {
		s.migrateToKeychain()
	}
	return s
}

// Set 在持锁期间保存或删除一条密钥，然后写回文件，落盘错误被忽略。
// value 为空时从内存删除，并调用 keychainDelete，与钥匙串是否启用不相关。
// value 非空时写入内存；只有钥匙串启用才调用 keychainSet。钥匙串错误被忽略。
func (s *Store) Set(name, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mem == nil {
		s.mem = map[string]string{}
	}
	if value == "" {
		delete(s.mem, name)
		_ = keychainDelete(name)
	} else {
		s.mem[name] = value
		if s.keychain {
			_ = keychainSet(name, value)
		}
	}
	_ = s.flushLocked()
}

// Get 先调用 GetStored。结果没有错误，且去掉首尾空白后仍非空时，直接返回 GetStored 的原值，空白只用于判断。
// 只有 name 恰好是 openai 或 default 时，才按顺序回落环境变量 YOYO_API_KEY、OPENAI_API_KEY、ANTHROPIC_API_KEY、DEEPSEEK_API_KEY，取第一个不等于空串的值。
// 都没有则返回 vault: missing key。
func (s *Store) Get(name string) (string, error) {
	if v, err := s.GetStored(name); err == nil && strings.TrimSpace(v) != "" {
		return v, nil
	}
	if name == "openai" || name == "default" {
		if v := os.Getenv(EnvAPIKey); v != "" {
			return v, nil
		}
		if v := os.Getenv("OPENAI_API_KEY"); v != "" {
			return v, nil
		}
		if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
			return v, nil
		}
		if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("vault: missing key %q", name)
}

// GetStored 返回已经保存的密钥。持锁查看内存，非空字符串则返回，空串视为未命中；读钥匙串和文件时已经解锁。
// 钥匙串只在启用时查询，查询失败或值为空则继续读 vault.json，钥匙串错误不返回给调用方。
// 本函数不读进程环境变量，残留的 OPENAI_API_KEY 不会盖住自定义供应商的令牌。三处都没有则返回 vault: missing key。
func (s *Store) GetStored(name string) (string, error) {
	s.mu.Lock()
	if v := s.mem[name]; v != "" {
		s.mu.Unlock()
		return v, nil
	}
	useKeychain := s.keychain
	path := s.path
	s.mu.Unlock()
	if useKeychain {
		if v, err := keychainGet(name); err == nil && v != "" {
			return v, nil
		}
	}
	if v := readFileKey(path, name); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("vault: missing key %q", name)
}

// Lease 调用 Get，把密钥明文交给调用方。
func (s *Store) Lease(name string) (string, error) {
	return s.Get(name)
}

// Status 报告密钥来源，不修改仓库。
// 启用钥匙串时来源是 keychain；内存 map 非空则是 keychain+file。这两种情况都不看环境变量，也不去钥匙串里确认是否真有密钥。
// 未启用钥匙串且内存 map 非空时来源是 file。内存为空，且 YOYO_API_KEY、OPENAI_API_KEY、ANTHROPIC_API_KEY、DEEPSEEK_API_KEY 任一不等于空串时，来源是 env，否则是 empty。
// 返回的 saved 在 source 不是 empty 时为真。path 和 keychain 分别是文件路径和是否启用钥匙串。
func (s *Store) Status() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	saved := len(s.mem) > 0
	source := "empty"
	if s.keychain {
		source = "keychain"
		if saved {
			source = "keychain+file"
		}
	} else if saved {
		source = "file"
	} else if os.Getenv(EnvAPIKey) != "" || os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("DEEPSEEK_API_KEY") != "" {
		source = "env"
	}
	return map[string]any{
		"saved":    saved || source == "env" || source == "keychain" || source == "keychain+file",
		"source":   source,
		"path":     s.path,
		"keychain": s.keychain,
	}
}

// migrateToKeychain 在已持锁时把内存里的非空值写入钥匙串，单条错误被忽略，然后写回文件。内存为空则直接返回。文件里的副本保留。
func (s *Store) migrateToKeychain() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.mem) == 0 {
		return
	}
	for k, v := range s.mem {
		if v == "" {
			continue
		}
		_ = keychainSet(k, v)
	}
	_ = s.flushLocked()
}

// loadFile 把 vault.json 解进 mem。path 为空、读失败、JSON 不能解成对象或结果为 nil 时，不替换现有内存。
func (s *Store) loadFile() {
	if s.path == "" {
		return
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var m map[string]string
	if json.Unmarshal(b, &m) != nil || m == nil {
		return
	}
	s.mem = m
}

// readFileKey 从 path 指向的 JSON 对象取出 name。查找名和取出的值都会去掉首尾空白。path 或 name 为空、读失败或 JSON 非法时返回空串。不读环境变量。
func readFileKey(path, name string) string {
	name = strings.TrimSpace(name)
	if path == "" || name == "" {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var m map[string]string
	if json.Unmarshal(b, &m) != nil || m == nil {
		return ""
	}
	return strings.TrimSpace(m[name])
}

// flushLocked 必须在已持有 s.mu 时调用。path 为空则返回 nil。
// 先以 0o700 创建目录，再以 0o600 把内存写入 path 加 .tmp 的临时文件，最后 Rename 成 path。任一步失败则返回该错误。
func (s *Store) flushLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.mem, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
