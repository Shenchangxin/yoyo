// Package memory 把记忆写在目录里的 memory.json。
// 拥有：能核对的事实直接留下，重复的合并，像密钥一样的文本不收。
// 不拥有：让人在「留下」和「忘掉」之间选。模型写进来的也直接留下，路由仍然只读。
package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Kind 是一条记忆的种类。
type Kind string

const (
	// KindProfile 是档案，值为 profile。
	KindProfile Kind = "profile"
	// KindProject 是项目，值为 project。
	KindProject Kind = "project"
	// KindEpisodic 是情景，值为 episodic。
	KindEpisodic Kind = "episodic"
	// KindSemantic 是偏好，值为 semantic。
	KindSemantic Kind = "semantic"
)

// Item 是一条记忆。State、Evidence、When 只给本机清点的工程线，不带正文。
type Item struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Text      string    `json:"text"`
	Project   string    `json:"project,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Staging   bool      `json:"staging,omitempty"`
	Source    string    `json:"source,omitempty"`
	State     string    `json:"state,omitempty"`
	Evidence  string    `json:"evidence,omitempty"`
	When      string    `json:"when,omitempty"`
}

// MachineFact 是一次本机清点要留下的事实。ID 必须以 machine: 开头。
type MachineFact struct {
	ID       string
	Kind     string
	Project  string
	Text     string
	State    string
	Evidence string
	When     string
}

// Portrait 是已经留下的事实，按来源分开。不含推断出来的经历。
type Portrait struct {
	Device      string
	Host        string
	OS          string
	Zone        string
	Git         string
	Hours       string
	Lines       []FactLine
	Said        []string
	Noted       []string
	Preferences []string
	Updated     time.Time
}

// FactLine 是一条工程线的名字和能核对的痕迹。
type FactLine struct {
	Name     string
	State    string
	Evidence string
	When     string
}

// Store 把条目保存在内存，并刷到 memory.json。
type Store struct {
	mu    sync.Mutex
	path  string
	items []Item
}

// Open 创建 dir，权限 0o755，路径是 dir/memory.json。读入后清掉待审标记，并丢掉像密钥的条目。
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "memory.json")}
	_ = s.load()
	s.absorb()
	return s, nil
}

// load 从 memory.json 解出条目。文件不存在时返回错误，Open 会忽略。
func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.items)
}

// flush 先把 JSON 写到路径加 .tmp，权限 0o600，再改成 memory.json。
func (s *Store) flush() error {
	b, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// absorb 让旧的待审条目直接生效，并丢掉空文本和像密钥的文本。
func (s *Store) absorb() {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false
	out := s.items[:0]
	for _, it := range s.items {
		it.Text = strings.TrimSpace(it.Text)
		if it.Text == "" || secretText(it.Text) {
			changed = true
			continue
		}
		if it.Staging {
			it.Staging = false
			changed = true
		}
		out = append(out, it)
	}
	s.items = out
	if changed {
		_ = s.flush()
	}
}

// Write 留下一条。空文本或像密钥的文本不写，返回的 ID 为空。同一条再写一次不新增。
func (s *Store) Write(it Item) Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	it.Text = strings.TrimSpace(it.Text)
	if it.Text == "" || secretText(it.Text) || secretText(it.Evidence) {
		return Item{}
	}
	it.Staging = false
	if it.ID != "" {
		for i := range s.items {
			if s.items[i].ID != it.ID {
				continue
			}
			if sameFact(s.items[i], it) {
				return s.items[i]
			}
			s.items[i] = it
			s.items[i].CreatedAt = time.Now().UTC()
			_ = s.flush()
			return s.items[i]
		}
	}
	key := strings.ToLower(it.Text)
	for i := range s.items {
		if s.items[i].Kind == it.Kind && strings.ToLower(strings.TrimSpace(s.items[i].Text)) == key {
			if s.items[i].Staging {
				s.items[i].Staging = false
				_ = s.flush()
			}
			return s.items[i]
		}
	}
	if it.ID == "" {
		it.ID = time.Now().UTC().Format("20060102T150405.000000000")
	}
	if it.CreatedAt.IsZero() {
		it.CreatedAt = time.Now().UTC()
	}
	s.items = append(s.items, it)
	_ = s.flush()
	return it
}

// FileMachine 用这一轮清点替换 machine: 开头的条目。没有变化时不刷盘，返回假。
func (s *Store) FileMachine(facts []MachineFact) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := map[string]Item{}
	var rest []Item
	for _, it := range s.items {
		if strings.HasPrefix(it.ID, "machine:") {
			old[it.ID] = it
			continue
		}
		rest = append(rest, it)
	}
	var fresh []Item
	seen := map[string]bool{}
	for _, fact := range facts {
		it := Item{
			ID:       fact.ID,
			Kind:     Kind(fact.Kind),
			Text:     strings.TrimSpace(fact.Text),
			Project:  strings.TrimSpace(fact.Project),
			Source:   "machine",
			State:    strings.TrimSpace(fact.State),
			Evidence: strings.TrimSpace(fact.Evidence),
			When:     strings.TrimSpace(fact.When),
		}
		if !strings.HasPrefix(it.ID, "machine:") || it.Text == "" || secretText(it.Text) || secretText(it.Evidence) || seen[it.ID] {
			continue
		}
		seen[it.ID] = true
		if prev, ok := old[it.ID]; ok && sameFact(prev, it) {
			it.CreatedAt = prev.CreatedAt
		} else if it.CreatedAt.IsZero() {
			it.CreatedAt = time.Now().UTC()
		}
		fresh = append(fresh, it)
	}
	changed := len(fresh) != len(old)
	if !changed {
		for _, it := range fresh {
			prev, ok := old[it.ID]
			if !ok || !sameFact(prev, it) {
				changed = true
				break
			}
		}
	}
	if !changed {
		return false
	}
	s.items = append(rest, fresh...)
	_ = s.flush()
	return true
}

// Forget 删掉指定 ID。找到返回真。
func (s *Store) Forget(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.items[:0]
	ok := false
	for _, it := range s.items {
		if it.ID == id {
			ok = true
			continue
		}
		out = append(out, it)
	}
	s.items = out
	if ok {
		_ = s.flush()
	}
	return ok
}

// Search 按种类和子串筛选。结果按 CreatedAt 从新到旧。
func (s *Store) Search(q string, kind Kind, limit int) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	q = strings.ToLower(strings.TrimSpace(q))
	var out []Item
	for _, it := range s.items {
		if kind != "" && it.Kind != kind {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(it.Text), q) && !strings.Contains(strings.ToLower(it.Project), q) {
			continue
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ProfilePin 把已经留下的事实收成模型能读的短文。只写名字、时间和当事人确认过的话。
func (s *Store) ProfilePin(budget int) string {
	p := s.Portrait()
	var b strings.Builder
	write := func(line string) bool {
		if budget > 0 && b.Len()+len(line) > budget {
			return false
		}
		b.WriteString(line)
		return true
	}
	if p.Device != "" || p.Hours != "" || len(p.Lines) > 0 {
		if !write("From this machine, names and times only:\n") {
			return b.String()
		}
		if p.Device != "" && !write("- user: "+p.Device+"\n") {
			return b.String()
		}
		if p.Host != "" && !write("- host: "+p.Host+"\n") {
			return b.String()
		}
		if p.OS != "" && !write("- os: "+p.OS+"\n") {
			return b.String()
		}
		if p.Zone != "" && !write("- zone: "+p.Zone+"\n") {
			return b.String()
		}
		if p.Git != "" && !write("- git: "+p.Git+"\n") {
			return b.String()
		}
		if p.Hours != "" && !write("- weekday edits: "+p.Hours+"\n") {
			return b.String()
		}
		for _, line := range p.Lines {
			row := "- " + line.Name
			if line.State != "" {
				row += ": " + line.State
			}
			if line.Evidence != "" {
				row += "; " + line.Evidence
			}
			if line.When != "" {
				row += "; " + line.When
			}
			if !write(row + "\n") {
				return b.String()
			}
		}
	}
	if len(p.Said) > 0 {
		if !write("Said by the operator:\n") {
			return b.String()
		}
		for _, line := range p.Said {
			if !write("- " + line + "\n") {
				return b.String()
			}
		}
	}
	if len(p.Noted) > 0 {
		if !write("Noted:\n") {
			return b.String()
		}
		for _, line := range p.Noted {
			if !write("- " + line + "\n") {
				return b.String()
			}
		}
	}
	if len(p.Preferences) > 0 {
		if !write("Preferences:\n") {
			return b.String()
		}
		for _, line := range p.Preferences {
			if !write("- " + line + "\n") {
				return b.String()
			}
		}
	}
	return b.String()
}

// Portrait 把留下的条目分成设备、工作时间、工程线、当事人说过的、模型记下的和偏好。
func (s *Store) Portrait() Portrait {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out Portrait
	for _, it := range s.items {
		if it.Staging || strings.TrimSpace(it.Text) == "" {
			continue
		}
		kept := true
		switch {
		case it.ID == "machine:device":
			out.Device = it.Text
		case it.ID == "machine:host":
			out.Host = it.Text
		case it.ID == "machine:os":
			out.OS = it.Text
		case it.ID == "machine:zone":
			out.Zone = it.Text
		case it.ID == "machine:git":
			out.Git = it.Text
		case it.ID == "machine:hours" || (it.Source == "machine" && it.Project == "work"):
			out.Hours = it.Text
		case it.Kind == KindProject:
			out.Lines = append(out.Lines, FactLine{Name: it.Text, State: it.State, Evidence: it.Evidence, When: it.When})
		case it.Kind == KindProfile && operatorSource(it.Source):
			out.Said = append(out.Said, it.Text)
		case it.Kind == KindProfile:
			out.Noted = append(out.Noted, it.Text)
		case it.Kind == KindSemantic:
			out.Preferences = append(out.Preferences, it.Text)
		default:
			kept = false
		}
		if kept && it.CreatedAt.After(out.Updated) {
			out.Updated = it.CreatedAt
		}
	}
	return out
}

// Promote 仍把 Staging 设为假。新写入已经不待审，这只给旧数据用。
func (s *Store) Promote(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Staging = false
			_ = s.flush()
			return true
		}
	}
	return false
}

// All 返回条目切片的副本。
func (s *Store) All() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Item(nil), s.items...)
	return out
}

// Path 返回 memory.json 的路径，不加锁。
func (s *Store) Path() string { return s.path }

// MachineID 给本机事实一个稳定 id。同一条线每次清点都对上同一条。
func MachineID(part, name string) string {
	var b strings.Builder
	b.WriteString("machine:")
	b.WriteString(strings.TrimSpace(part))
	b.WriteByte(':')
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch r {
		case '/', '\\', ' ', ':', '\n', '\t':
			r = '-'
		}
		if r < 32 {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func sameFact(a, b Item) bool {
	return a.Kind == b.Kind && a.Text == b.Text && a.Project == b.Project && a.Source == b.Source && a.State == b.State && a.Evidence == b.Evidence && a.When == b.When
}

func operatorSource(source string) bool {
	switch strings.TrimSpace(source) {
	case "", "user", "User confirmed in chat":
		return true
	default:
		return false
	}
}

func secretText(text string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "sk-") {
		return true
	}
	for _, part := range []string{"secret", "token", "password", "cookie", "credential", "api_key", "apikey"} {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}
