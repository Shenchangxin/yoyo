// Package filestore 是进程内的 JSON 文档树。写入先写临时文件再改名。
// 互斥锁串行化同一进程里的任务和 RPC。Yoyo 桌面按单进程使用这份存储。
package filestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrNotFound 表示要读的文件不存在。Get 在文件缺失时返回它。Delete 对不存在的路径返回 nil。
var ErrNotFound = errors.New("filestore: not found")

// Store 是一份以 Root 为根的 JSON 文档树。写入用临时文件加改名。
type Store struct {
	Root string
	mu   sync.Mutex
}

// New 创建 root，权限 0o755，并返回指向它的 Store。
func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Store{Root: root}, nil
}

// Abs 把相对路径清洗后接到 Root 上。反斜杠先换成斜杠。
// rel 里的反斜杠换成斜杠，前面加斜杠后 Clean，再去掉开头的分隔符，接到 Root 上。
func (s *Store) Abs(rel string) string {
	rel = filepath.Clean("/" + strings.ReplaceAll(rel, "\\", "/"))
	rel = strings.TrimPrefix(rel, string(filepath.Separator))
	return filepath.Join(s.Root, rel)
}

// Put 在锁内把 v 写成 rel 对应的 JSON 文件。
func (s *Store) Put(rel string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.putLocked(rel, v)
}

// putLocked 创建父目录，权限 0o755。JSON 缩进两个空格并在末尾加换行。
// 先写到路径加 .tmp，权限 0o644，再改成目标名。改名失败时删掉临时文件。
// rel 交给 Abs，得到要写的路径。
// v 交给 json.MarshalIndent，前缀是空串，缩进是两个空格，末尾再加换行。
func (s *Store) putLocked(rel string, v any) error {
	path := s.Abs(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Get 在锁内把 rel 的 JSON 解到 v。
func (s *Store) Get(rel string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(rel, v)
}

// getLocked 读取文件并 json 解码。文件不存在时返回 ErrNotFound。
// rel 交给 Abs 后读取该文件。文件不存在时返回 ErrNotFound。
// v 交给 json.Unmarshal。
func (s *Store) getLocked(rel string, v any) error {
	b, err := os.ReadFile(s.Abs(rel))
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	return json.Unmarshal(b, v)
}

// Delete 删掉 rel。路径不存在时返回 nil。
func (s *Store) Delete(rel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.Abs(rel))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Exists 在 Stat 没有错误时返回真。
// rel 交给 Abs 后 Stat。错误为 nil 时返回 true。
func (s *Store) Exists(rel string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := os.Stat(s.Abs(rel))
	return err == nil
}

// IDs 列出 dir 下的文档 id。目录不存在时返回 nil 和 nil。
// 跳过子目录、不以 .json 结尾的名字，以及以 .tmp 结尾的名字。id 是去掉 .json 后的文件名。
func (s *Store) IDs(dir string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ents, err := os.ReadDir(s.Abs(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, ent := range ents {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(name, ".json"))
	}
	return ids, nil
}

// LoadAll 按 IDs 的顺序加载 dir 下每个 id.json。Get 返回 ErrNotFound 时跳过该条。其他错误带上 dir 和 id。
func LoadAll[T any](s *Store, dir string) ([]T, error) {
	ids, err := s.IDs(dir)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(ids))
	for _, id := range ids {
		var v T
		if err := s.Get(filepath.Join(dir, id+".json"), &v); err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("%s/%s: %w", dir, id, err)
		}
		out = append(out, v)
	}
	return out, nil
}

// Rel 拼出 col/id.json。id 先去掉首尾空白，再把 /、\ 和 .. 换成下划线。
func Rel(col, id string) string {
	id = strings.TrimSpace(id)
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, "\\", "_")
	id = strings.ReplaceAll(id, "..", "_")
	return filepath.Join(col, id+".json")
}
