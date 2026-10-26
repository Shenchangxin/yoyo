// Package profile 按 id 保存助手画像，并用拒绝名单过滤工具。
// 种子 researcher 的 DenyTools 是 shell、computer_act、screenshot_region、git_commit、connector_send。
// 种子 secretary 的 DenyTools 是 shell、computer_act、screenshot_region、git_commit。种子 assistant 不写 DenyTools。
// Research 为 false 时再拒绝 web_search、web_fetch、reach_status、browser_open、browser_snapshot、browser_screenshot、cite_sources。
// Memory 为 false 时再拒绝 memory_write、remember_fact。AllowTools 为空不是拒绝。
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultID 是默认画像 id，值是 assistant。Get 在 id 为空时用它。List 把它排在最前。
const DefaultID = "assistant"

// Profile 是一份落在 <id>.json 里的助手画像。
type Profile struct {
	// ID 是文件名和内存 map 的键。Save 经 sanitizeID 后只留小写字母、数字、连字符和下划线。
	ID string `json:"id"`
	// Name 是显示名。Save 见到空字符串就改成 ID。
	Name string `json:"name"`
	// Instructions 是给该画像的说明，原样写入 JSON。
	Instructions string `json:"instructions"`
	// AllowTools 非空时，FilterTools 只留名单里的名字，再加上一组固定保留的工具。空名单不是拒绝。
	AllowTools []string `json:"allow_tools,omitempty"`
	// DenyTools 是显式拒绝的工具名。denySet 会先收进拒绝集合。
	DenyTools []string `json:"deny_tools,omitempty"`
	// SpaceIDs 是画像可用的空间。Save 见到空切片就改成只含 DefaultSpace。
	SpaceIDs []string `json:"space_ids,omitempty"`
	// DefaultSpace 是默认空间。Save 见到空字符串就写成 home。
	DefaultSpace string `json:"default_space,omitempty"`
	// Research 为 false 时，denySet 再拒绝 web_search、web_fetch、reach_status、browser_open、browser_snapshot、browser_screenshot、cite_sources。
	Research bool `json:"research"`
	// Memory 为 false 时，denySet 再拒绝 memory_write、remember_fact。
	Memory bool `json:"memory"`
	// MCPAllow 是 MCP 服务器允许片段。空切片表示 MCPAllowed 全部放行。
	MCPAllow []string `json:"mcp_allow,omitempty"`
	// SpeechConn 是语音连接名，本包不解释它。
	SpeechConn string `json:"speech_connection,omitempty"`
	// CreatedAt 是首次保存的 UTC 时间。已有画像再次 Save 时保留原来的值。
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 是最近一次 Save 的 UTC 时间。
	UpdatedAt time.Time `json:"updated_at"`
}

// Store 是一个目录里的画像集合。每个 id 对应一个 json 文件。
type Store struct {
	// mu 保护 all 和写盘。
	mu sync.Mutex
	// root 是 Open 传入的目录。
	root string
	// all 是已载入的画像，键是 ID。
	all map[string]Profile
}

// Open 以 0o755 创建 root，读入其中的 json，再补上还没有的种子。目录创建失败就返回错误。
func Open(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	s := &Store{root: root, all: map[string]Profile{}}
	s.load()
	s.ensureSeeds()
	return s, nil
}

// List 返回全部画像。ID 等于 DefaultID 的排在最前，其余按 Name 升序。
func (s *Store) List() []Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Profile, 0, len(s.all))
	for _, p := range s.all {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID == DefaultID {
			return true
		}
		if out[j].ID == DefaultID {
			return false
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// Get 按 id 取画像。id 去掉空白后为空就用 DefaultID。找不到时错误文案是 profile not found。
func (s *Store) Get(id string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	if id == "" {
		id = DefaultID
	}
	p, ok := s.all[id]
	if !ok {
		return Profile{}, fmt.Errorf("profile not found")
	}
	return p, nil
}

// Save 把画像写成 <root>/<id>.json，权限 0o644，并放进内存。
// ID 经 sanitizeID 后为空就返回 profile id required。Name 为空则改成 ID。
// 已有同 ID 时保留原来的 CreatedAt，否则 CreatedAt 和 UpdatedAt 都是现在的 UTC。
// DefaultSpace 为空写成 home。SpaceIDs 为空就写成只含 DefaultSpace。
func (s *Store) Save(p Profile) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	p.ID = sanitizeID(p.ID)
	if p.ID == "" {
		return Profile{}, fmt.Errorf("profile id required")
	}
	if p.Name == "" {
		p.Name = p.ID
	}
	if cur, ok := s.all[p.ID]; ok {
		p.CreatedAt = cur.CreatedAt
	} else {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.DefaultSpace == "" {
		p.DefaultSpace = "home"
	}
	if len(p.SpaceIDs) == 0 {
		p.SpaceIDs = []string{p.DefaultSpace}
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return Profile{}, err
	}
	if err := atomicWrite(filepath.Join(s.root, p.ID+".json"), b, 0o644); err != nil {
		return Profile{}, err
	}
	s.all[p.ID] = p
	return p, nil
}

// load 读 root 下每个非目录的 .json。读失败、JSON 无效或 ID 为空的文件跳过。
func (s *Store) load() {
	ents, err := os.ReadDir(s.root)
	if err != nil {
		return
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.root, e.Name()))
		if err != nil {
			continue
		}
		var p Profile
		if json.Unmarshal(b, &p) != nil || p.ID == "" {
			continue
		}
		s.all[p.ID] = p
	}
}

// ensureSeeds 把 Seeds 里还没有的 id 写到磁盘。编码或写盘的错误被丢掉。
func (s *Store) ensureSeeds() {
	for _, p := range Seeds() {
		if _, ok := s.all[p.ID]; ok {
			continue
		}
		now := time.Now().UTC()
		p.CreatedAt = now
		p.UpdatedAt = now
		b, _ := json.MarshalIndent(p, "", "  ")
		_ = atomicWrite(filepath.Join(s.root, p.ID+".json"), b, 0o644)
		s.all[p.ID] = p
	}
}

// Seeds 返回三份内置画像：assistant、researcher、secretary。
// researcher 拒绝 shell、computer_act、screenshot_region、git_commit、connector_send。
// secretary 拒绝 shell、computer_act、screenshot_region、git_commit。assistant 不写 DenyTools，Research 和 Memory 都为 true。
func Seeds() []Profile {
	return []Profile{
		{
			ID: DefaultID, Name: "Assistant", DefaultSpace: "home", SpaceIDs: []string{"home"},
			Research: true, Memory: true,
			Instructions: "Be a practical local assistant. Prefer workspace artifacts, cited research, and approved Pages over advice.",
		},
		{
			ID: "researcher", Name: "Researcher", DefaultSpace: "home", SpaceIDs: []string{"home"},
			Research: true, Memory: true,
			DenyTools:    []string{"shell", "computer_act", "screenshot_region", "git_commit", "connector_send"},
			Instructions: "Investigate public sources and authorized Pages. Draft findings for review_page. Do not run shell, computer_act, or send as the operator.",
		},
		{
			ID: "secretary", Name: "Secretary", DefaultSpace: "home", SpaceIDs: []string{"home"},
			Research: false, Memory: true,
			DenyTools:    []string{"shell", "computer_act", "screenshot_region", "git_commit"},
			Instructions: "Handle mail, calendar, and Pages as a secretary. Mail and calendar writes stay hashed send_as_you proposals. No shell or computer_act.",
		},
	}
}

// DeniedNames 返回这份画像的拒绝名单：DenyTools，再加上 Research、Memory 开关带上的名字。AllowTools 为空不是拒绝。结果按名字排序。
func DeniedNames(p Profile) []string {
	deny := denySet(p)
	out := make([]string, 0, len(deny))
	for n := range deny {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// denySet 收集拒绝集合。先放入去掉空白后的 DenyTools。
// Research 为 false 时加入 web_search、web_fetch、reach_status、browser_open、browser_snapshot、browser_screenshot、cite_sources。
// Memory 为 false 时加入 memory_write、remember_fact。空字符串不加入。
func denySet(p Profile) map[string]bool {
	deny := map[string]bool{}
	add := func(names ...string) {
		for _, n := range names {
			n = strings.TrimSpace(n)
			if n != "" {
				deny[n] = true
			}
		}
	}
	add(p.DenyTools...)
	if !p.Research {
		add("web_search", "web_fetch", "reach_status", "browser_open", "browser_snapshot", "browser_screenshot", "cite_sources")
	}
	if !p.Memory {
		add("memory_write", "remember_fact")
	}
	return deny
}

// FilterTools 按画像从 have 里留下可用工具。拒绝集合里的名字一律丢掉。
// AllowTools 为空时，其余名字都留。AllowTools 非空时，只留名单里的名字，或 tool_search、ask_user、load_skill、list_skills、recall_context、update_plan、read_skill_file。
// 空名字和重复名字丢掉。顺序跟 have 里第一次出现的顺序。
func FilterTools(have []string, p Profile) []string {
	deny := denySet(p)
	allow := map[string]bool{}
	for _, n := range p.AllowTools {
		allow[strings.TrimSpace(n)] = true
	}
	keep := func(n string) bool {
		if deny[n] {
			return false
		}
		if len(p.AllowTools) == 0 {
			return true
		}
		return allow[n] || n == "tool_search" || n == "ask_user" || n == "load_skill" || n == "list_skills" || n == "recall_context" || n == "update_plan" || n == "read_skill_file"
	}
	var out []string
	seen := map[string]bool{}
	for _, n := range have {
		if n == "" || seen[n] || !keep(n) {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// MCPAllowed 判断 MCP 工具名是否被 allow 放行。allow 为空返回 true。
// 比较不区分大小写。命中条件是全名相等、名字以该片段加 __ 开头，或名字包含该片段。空白片段跳过。
func MCPAllowed(allow []string, name string) bool {
	if len(allow) == 0 {
		return true
	}
	name = strings.ToLower(strings.TrimSpace(name))
	for _, a := range allow {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" {
			continue
		}
		if name == a || strings.HasPrefix(name, a+"__") || strings.Contains(name, a) {
			return true
		}
	}
	return false
}

// sanitizeID 去掉两端空白并转成小写，只保留 a-z、0-9、连字符和下划线。
func sanitizeID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// atomicWrite 先把字节写到 path.tmp，再改名为 path。写临时文件失败时不改名。
func atomicWrite(path string, b []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
