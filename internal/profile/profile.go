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

const DefaultID = "assistant"

type Profile struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Instructions string    `json:"instructions"`
	AllowTools   []string  `json:"allow_tools,omitempty"`
	DenyTools    []string  `json:"deny_tools,omitempty"`
	SpaceIDs     []string  `json:"space_ids,omitempty"`
	DefaultSpace string    `json:"default_space,omitempty"`
	Research     bool      `json:"research"`
	Memory       bool      `json:"memory"`
	MCPAllow     []string  `json:"mcp_allow,omitempty"`
	SpeechConn   string    `json:"speech_connection,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Store struct {
	mu   sync.Mutex
	root string
	all  map[string]Profile
}

func Open(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	s := &Store{root: root, all: map[string]Profile{}}
	s.load()
	s.ensureSeeds()
	return s, nil
}

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

func FilterTools(have []string, p Profile) []string {
	deny := map[string]bool{}
	for _, n := range p.DenyTools {
		deny[strings.TrimSpace(n)] = true
	}
	if !p.Research {
		for _, n := range []string{"web_search", "web_fetch", "browser_open", "browser_snapshot", "browser_screenshot", "cite_sources"} {
			deny[n] = true
		}
	}
	if !p.Memory {
		for _, n := range []string{"memory_write", "remember_fact"} {
			deny[n] = true
		}
	}
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

func atomicWrite(path string, b []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
