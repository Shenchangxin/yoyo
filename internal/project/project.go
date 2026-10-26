// Package project 把项目列表写在目录下的 projects.json。
// Open 按 0o755 建目录。Create 在 ID 为空时用名称的小写并把空格换成连字符；名称也空就用 UTC 时间格式 p-20060102T150405。
// List 按 CreatedAt 从新到旧排。flush 先写 .tmp 再改名，权限 0o644。load 的错误被 Open 忽略。
package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Project 是列表里的一个项目。
type Project struct {
	// ID 是项目标识。Create 只在它为空时生成。
	ID string `json:"id"`
	// Name 是显示名，也是空 ID 时生成标识的原料。
	Name string `json:"name"`
	// Root 是项目根路径，本包不创建这个目录。
	Root string `json:"root"`
	// Instructions 是项目说明，可空。
	Instructions string `json:"instructions,omitempty"`
	// Links 是附带链接，可空。
	Links []string `json:"links,omitempty"`
	// CreatedAt 是创建时间。Create 见到零值就写成现在的 UTC。
	CreatedAt time.Time `json:"created_at"`
}

// Store 读写一份 projects.json。
type Store struct {
	// mu 保护 all 和写盘。
	mu sync.Mutex
	// path 是 projects.json 的完整路径。
	path string
	// all 是当前内存里的项目，顺序与文件一致，List 会另做排序。
	all []Project
}

// Open 以 0o755 创建 dir，并把路径设为 dir/projects.json。load 失败时仍返回空列表的 Store。
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "projects.json")}
	_ = s.load()
	return s, nil
}

// Create 把项目追加到列表并 flush。ID 为空时用名称小写、空格换连字符；那样还是空就用 UTC 的 p-20060102T150405。CreatedAt 为零就写成现在的 UTC。flush 的错误被丢掉，仍然返回这份项目。
func (s *Store) Create(p Project) Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = strings.ToLower(strings.ReplaceAll(p.Name, " ", "-"))
		if p.ID == "" {
			p.ID = time.Now().UTC().Format("p-20060102T150405")
		}
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	s.all = append(s.all, p)
	_ = s.flush()
	return p
}

// List 返回副本，按 CreatedAt 从新到旧排。
func (s *Store) List() []Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Project(nil), s.all...)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// Get 按 ID 精确查找。找不到时返回零值和 false。
func (s *Store) Get(id string) (Project, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.all {
		if p.ID == id {
			return p, true
		}
	}
	return Project{}, false
}

// load 把 projects.json 解进 all。文件不存在或 JSON 无效时返回该错误，all 保持原样。
func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.all)
}

// flush 把 all 缩进编码后写到 path.tmp，再改名为 path，权限 0o644。
func (s *Store) flush() error {
	b, err := json.MarshalIndent(s.all, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
