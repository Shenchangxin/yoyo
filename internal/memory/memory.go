// Package memory 把记忆写在目录里的 memory.json。
// 拥有：能核对的事实直接留下，像密钥一样的文本不收。同一句换个写法合并成一条，改口时旧说法进历史，相近的旧记忆交回给写的人判断。
// 写进来的「明天」「下周三」换算成日期，说的是会过去的安排时记下到哪天为止，过了就不再当成现在的事。按词打分的检索。
// 不拥有：让人在「留下」和「忘掉」之间选。模型写进来的也直接留下，出处写清楚。不做向量检索。
package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
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
// CreatedAt 是最初记下的时间；UpdatedAt 是最近一次改写或又说了一遍的时间，从没变过时为零。
// ValidUntil 是正文说的那段安排到哪天为止，格式 2006-01-02；过了这一天它就是过去的事。Seen 是同一句说过几次，零当一次。
// Origin 是从哪学到的，比如 session:编号、ticket:编号。History 是以前的说法，最多留五条。
type Item struct {
	ID         string     `json:"id"`
	Kind       Kind       `json:"kind"`
	Text       string     `json:"text"`
	Project    string     `json:"project,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at,omitzero"`
	ValidUntil string     `json:"valid_until,omitempty"`
	Seen       int        `json:"seen,omitempty"`
	Origin     string     `json:"origin,omitempty"`
	History    []Revision `json:"history,omitempty"`
	Staging    bool       `json:"staging,omitempty"`
	Source     string     `json:"source,omitempty"`
	State      string     `json:"state,omitempty"`
	Evidence   string     `json:"evidence,omitempty"`
	When       string     `json:"when,omitempty"`
}

// Revision 是一条记忆以前的说法，At 是那个说法最后一次被确认的时间。
type Revision struct {
	Text   string    `json:"text"`
	At     time.Time `json:"at"`
	Source string    `json:"source,omitempty"`
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

// Store 把条目保存在内存，并刷到 memory.json。
type Store struct {
	mu    sync.Mutex
	path  string
	items []Item
	now   func() time.Time
}

// Open 创建 dir，权限 0o755，路径是 dir/memory.json。读入后清掉待审标记，丢掉像密钥的条目，并把几乎一样的条目并成一条。
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "memory.json")}
	_ = s.load()
	s.absorb()
	return s, nil
}

func (s *Store) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
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

// absorb 让旧的待审条目直接生效，丢掉空文本和像密钥的文本，再整理一遍。
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
	if s.tidy() {
		changed = true
	}
	if changed {
		_ = s.flush()
	}
}

// Kept 是一次写入的结果。Action 取 added、same、merged、replaced 或 refused。
// Before 是合并或换掉之前的说法。Related 是同一类里和这一条相近、却没有合并的旧记忆，最多三条：可能是改口，留给写的人判断。
type Kept struct {
	Item    Item
	Action  string
	Before  string
	Related []Item
}

// 两句话共有的词占两边都到 mergeShare，才算同一句换了写法，直接合并；共有至少两个词、并占任一边 relateShare，算相近。
// 光看字面分不出「简短」和「简洁」是一个意思、「中文」和「英文」是相反的意思，所以合并的门槛很高，其余交给写的人。
const (
	mergeShare  = 0.9
	relateShare = 0.3
	historyMax  = 5
	relatedMax  = 3
)

// Keep 留下一条记忆。空文本和像密钥的文本不收。正文里的「明天」「下周三」先换算成日期，说的是会过去的安排时记下 ValidUntil。
// replaces 非空时是改口：用这一条的说法换掉那一条，编号和最初记下的时间不变，旧说法进历史。
// it 带着已有的编号时换掉那一条。同一句再说一遍只记次数；几乎一样的说法并成一条，用新的写法；只是相近的不合并，放进 Related。经历之间不合并。
func (s *Store) Keep(it Item, replaces string) Kept {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock()
	it.Text = strings.TrimSpace(it.Text)
	if it.Text == "" || secretText(it.Text) || secretText(it.Evidence) {
		return Kept{Action: "refused"}
	}
	it.Staging = false
	if !machineItem(it) {
		text, until := anchorDates(it.Text, now)
		it.Text = text
		if it.ValidUntil == "" {
			it.ValidUntil = until
		}
	}
	stamp := now.UTC()
	if replaces = strings.TrimSpace(replaces); replaces != "" {
		return s.replace(replaces, it, stamp)
	}
	if it.ID != "" {
		if i := s.index(it.ID); i >= 0 {
			return s.rewrite(i, it, stamp)
		}
	}
	key := factKey(it.Text)
	for i, cur := range s.items {
		if cur.Kind == it.Kind && !machineItem(cur) && factKey(cur.Text) == key {
			s.items[i] = reconfirm(cur, it, stamp)
			_ = s.flush()
			return Kept{Item: s.items[i], Action: "same"}
		}
	}
	type near struct {
		at    int
		share float64
	}
	var related []near
	if it.Kind != KindEpisodic {
		terms := simTerms(it.Text)
		for i, cur := range s.items {
			if cur.Kind != it.Kind || machineItem(cur) {
				continue
			}
			shared, inNew, inOld := overlap(terms, simTerms(cur.Text))
			if min(inNew, inOld) >= mergeShare {
				s.items[i] = merge(cur, it, stamp)
				_ = s.flush()
				return Kept{Item: s.items[i], Action: "merged", Before: cur.Text}
			}
			if shared >= 2 && max(inNew, inOld) >= relateShare {
				related = append(related, near{at: i, share: max(inNew, inOld)})
			}
		}
	}
	if it.ID == "" {
		it.ID = s.freshID(stamp)
	}
	if it.CreatedAt.IsZero() {
		it.CreatedAt = stamp
	}
	it.UpdatedAt = time.Time{}
	s.items = append(s.items, it)
	_ = s.flush()
	sort.SliceStable(related, func(i, j int) bool { return related[i].share > related[j].share })
	out := Kept{Item: it, Action: "added"}
	for _, r := range related[:min(len(related), relatedMax)] {
		out.Related = append(out.Related, s.items[r.at])
	}
	return out
}

// Write 留下一条，返回留下的那一条。空文本或像密钥的文本不写，返回的 ID 为空。同一条再写一次不新增。
func (s *Store) Write(it Item) Item {
	return s.Keep(it, "").Item
}

// replace 用 it 的说法换掉 id 那一条。换上之后和同一类别的条目完全一样时，那一条重复的去掉。调用时要已持有锁。
func (s *Store) replace(id string, it Item, now time.Time) Kept {
	i := s.index(id)
	if i < 0 || machineItem(s.items[i]) {
		return Kept{Action: "refused"}
	}
	cur := s.items[i]
	next := cur
	next.Text = it.Text
	if it.Kind != "" {
		next.Kind = it.Kind
	}
	if it.Project != "" {
		next.Project = it.Project
	}
	if it.Origin != "" {
		next.Origin = it.Origin
	}
	next.Source = it.Source
	next.ValidUntil = it.ValidUntil
	next.UpdatedAt = now
	if factKey(cur.Text) != factKey(next.Text) {
		next.History = pushRevision(cur.History, Revision{Text: cur.Text, At: updatedOf(cur), Source: cur.Source})
	}
	key := factKey(next.Text)
	out := s.items[:0]
	for j, other := range s.items {
		switch {
		case j == i:
			out = append(out, next)
		case other.Kind == next.Kind && !machineItem(other) && factKey(other.Text) == key:
		default:
			out = append(out, other)
		}
	}
	s.items = out
	_ = s.flush()
	return Kept{Item: next, Action: "replaced", Before: cur.Text}
}

// rewrite 用带着同一个编号的 it 换掉第 i 条。最初记下的时间、次数和出处留着；正文变了时旧说法进历史，经历除外。调用时要已持有锁。
func (s *Store) rewrite(i int, it Item, now time.Time) Kept {
	cur := s.items[i]
	if sameFact(cur, it) && cur.ValidUntil == it.ValidUntil {
		return Kept{Item: cur, Action: "same"}
	}
	it.CreatedAt = cur.CreatedAt
	if it.CreatedAt.IsZero() {
		it.CreatedAt = now
	}
	it.UpdatedAt = now
	it.Seen = cur.Seen
	it.History = cur.History
	if it.Kind != KindEpisodic && factKey(cur.Text) != factKey(it.Text) {
		it.History = pushRevision(cur.History, Revision{Text: cur.Text, At: updatedOf(cur), Source: cur.Source})
	}
	if it.Origin == "" {
		it.Origin = cur.Origin
	}
	s.items[i] = it
	_ = s.flush()
	return Kept{Item: it, Action: "replaced", Before: cur.Text}
}

// reconfirm 是同一句又说了一遍：次数加一，更新时间换成现在。来源更可信时换成新的来源，正文不动。
func reconfirm(cur, it Item, now time.Time) Item {
	cur.Seen = seenOf(cur) + 1
	cur.UpdatedAt = now
	if trust(it.Source) > trust(cur.Source) {
		cur.Source = it.Source
	}
	if cur.Origin == "" {
		cur.Origin = it.Origin
	}
	if cur.Project == "" {
		cur.Project = it.Project
	}
	if it.ValidUntil != "" {
		cur.ValidUntil = it.ValidUntil
	}
	return cur
}

// merge 把几乎一样的新说法并进 cur：用新的写法，旧写法进历史，次数加一。模型和工人记下的不改当事人自己的话。
func merge(cur, it Item, now time.Time) Item {
	cur.Seen = seenOf(cur) + 1
	cur.UpdatedAt = now
	if trust(it.Source) >= trust(cur.Source) {
		if factKey(cur.Text) != factKey(it.Text) {
			cur.History = pushRevision(cur.History, Revision{Text: cur.Text, At: updatedOf(cur), Source: cur.Source})
		}
		cur.Text = it.Text
		cur.Source = it.Source
		cur.ValidUntil = it.ValidUntil
	}
	if cur.Origin == "" {
		cur.Origin = it.Origin
	}
	return cur
}

// tidy 把几乎一样的条目并成一条：同一类，不是本机清点，也不是经历。调用时要已持有锁。返回有没有改动。
func (s *Store) tidy() bool {
	changed := false
	for i := 0; i < len(s.items); i++ {
		a := s.items[i]
		if machineItem(a) || a.Kind == KindEpisodic {
			continue
		}
		terms := simTerms(a.Text)
		for j := i + 1; j < len(s.items); {
			b := s.items[j]
			same := false
			if b.Kind == a.Kind && !machineItem(b) {
				same = factKey(a.Text) == factKey(b.Text)
				if !same {
					_, x, y := overlap(terms, simTerms(b.Text))
					same = min(x, y) >= mergeShare
				}
			}
			if !same {
				j++
				continue
			}
			a = fold(a, b)
			terms = simTerms(a.Text)
			s.items = append(s.items[:j], s.items[j+1:]...)
			changed = true
		}
		s.items[i] = a
	}
	return changed
}

// fold 把 b 并进 a：来源更可信的说法留下，一样可信时留新的；另一个说法进历史。编号用 a 的，最初记下的时间取早的，次数相加。
func fold(a, b Item) Item {
	win, lose := a, b
	if trust(b.Source) > trust(a.Source) || (trust(b.Source) == trust(a.Source) && updatedOf(b).After(updatedOf(a))) {
		win, lose = b, a
	}
	out := win
	out.ID = a.ID
	out.CreatedAt = a.CreatedAt
	if !b.CreatedAt.IsZero() && (out.CreatedAt.IsZero() || b.CreatedAt.Before(out.CreatedAt)) {
		out.CreatedAt = b.CreatedAt
	}
	if latest := maxTime(updatedOf(a), updatedOf(b)); latest.After(out.CreatedAt) {
		out.UpdatedAt = latest
	}
	out.Seen = seenOf(a) + seenOf(b)
	history := append(append([]Revision(nil), lose.History...), win.History...)
	if factKey(lose.Text) != factKey(win.Text) {
		history = append(history, Revision{Text: lose.Text, At: updatedOf(lose), Source: lose.Source})
	}
	out.History = pushRevision(nil, history...)
	if out.Origin == "" {
		out.Origin = lose.Origin
	}
	return out
}

// pushRevision 把以前的说法接在历史后面，只留最近 historyMax 条。
func pushRevision(history []Revision, more ...Revision) []Revision {
	out := append(append([]Revision(nil), history...), more...)
	if len(out) > historyMax {
		out = out[len(out)-historyMax:]
	}
	return out
}

// freshID 用时间生成编号；同一刻已经有了时加序号。调用时要已持有锁。
func (s *Store) freshID(now time.Time) string {
	base := now.Format("20060102T150405.000000000")
	id := base
	for n := 2; s.index(id) >= 0; n++ {
		id = base + "-" + strconv.Itoa(n)
	}
	return id
}

// index 返回编号为 id 的条目位置。没有时返回 -1。调用时要已持有锁。
func (s *Store) index(id string) int {
	for i := range s.items {
		if s.items[i].ID == id {
			return i
		}
	}
	return -1
}

// trust 是来源的可信程度：当事人自己说的或亲手改的最高，没写来源的旧数据其次，模型和工人记下的最低。
func trust(source string) int {
	switch strings.TrimSpace(source) {
	case "user", "User confirmed in chat":
		return 3
	case "":
		return 2
	default:
		return 1
	}
}

// Touched 是这一条最近一次记下、改过或又说了一遍的时间。
func (it Item) Touched() time.Time {
	return updatedOf(it)
}

// updatedOf 是这一条最近一次记下或改过的时间。
func updatedOf(it Item) time.Time {
	if !it.UpdatedAt.IsZero() {
		return it.UpdatedAt
	}
	return it.CreatedAt
}

// seenOf 是同一句说过几次，至少一次。
func seenOf(it Item) int {
	return max(it.Seen, 1)
}

func machineItem(it Item) bool {
	return it.Source == "machine" || strings.HasPrefix(it.ID, "machine:")
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
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
			it.CreatedAt = s.clock().UTC()
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

// factKey 是比较是不是同一句时用的写法：小写，空白收成一个空格，去掉句末标点。
func factKey(text string) string {
	key := strings.ToLower(strings.Join(strings.Fields(text), " "))
	return strings.TrimRight(key, "。.!！?？,，;；、 ")
}

// Edit 把 id 这一条改成 text，并当成当事人确认过。最初记下的时间不变，更新时间换成现在，旧说法进历史；正文里的相对日子照样换算。
// 本机清点、空文本和像密钥的文本不改，返回假。
func (s *Store) Edit(id, text string) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	text = strings.TrimSpace(text)
	if id == "" || text == "" || strings.HasPrefix(id, "machine:") || secretText(text) {
		return Item{}, false
	}
	i := s.index(id)
	if i < 0 || s.items[i].Source == "machine" {
		return Item{}, false
	}
	now := s.clock()
	cur := s.items[i]
	anchored, until := anchorDates(text, now)
	if factKey(cur.Text) != factKey(anchored) {
		cur.History = pushRevision(cur.History, Revision{Text: cur.Text, At: updatedOf(cur), Source: cur.Source})
	}
	cur.Text = anchored
	cur.ValidUntil = until
	cur.Source = "user"
	cur.Staging = false
	cur.UpdatedAt = now.UTC()
	s.items[i] = cur
	_ = s.flush()
	return cur, true
}

func nonEmpty(parts ...string) []string {
	var out []string
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
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

var (
	keyPrefix   = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:sk|pk|rk)-[a-z0-9_\-]{8,}`)
	knownKey    = regexp.MustCompile(`(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|xox[abpr]-[A-Za-z0-9-]{10,}|AKIA[0-9A-Z]{16}|AIza[0-9A-Za-z_\-]{30,}|eyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}\.`)
	assignWord  = regexp.MustCompile(`(?i)\b(?:secret|token|password|passwd|cookie|credential|api[_\- ]?key|apikey)s?\s*[:=：]\s*\S{4,}`)
	passwordIs  = regexp.MustCompile(`(?i)\bpass(?:word|wd)\s+is\s+\S{4,}`)
	assignHan   = regexp.MustCompile(`(?:密码|密钥|秘钥|口令|令牌)\s*(?:[:=：]|是|为)\s*\S{3,}`)
	envName     = regexp.MustCompile(`[A-Z0-9]*_?(?:SECRET|TOKEN|PASSWORD|PASSWD|API_?KEY|COOKIE|CREDENTIAL)S?_?[A-Z0-9_]*`)
	opaqueRun   = regexp.MustCompile(`[A-Za-z0-9+=_\-]{32,}`)
	opaqueSplit = regexp.MustCompile(`[_\-]`)
)

// secretText 在文本里像有一把钥匙时为真：常见密钥前缀、钥匙名后面跟着值、大写的钥匙变量名，或一长串带数字的乱码。
// 只是提到 token、密码这些词，或 task-、risk- 这种普通写法，不算。
func secretText(text string) bool {
	if keyPrefix.MatchString(text) || knownKey.MatchString(text) || assignWord.MatchString(text) || passwordIs.MatchString(text) || assignHan.MatchString(text) {
		return true
	}
	for _, name := range envName.FindAllString(text, -1) {
		if strings.Contains(name, "_") {
			return true
		}
	}
	for _, run := range opaqueRun.FindAllString(text, -1) {
		if !strings.ContainsFunc(run, unicode.IsDigit) || !strings.ContainsFunc(run, unicode.IsLetter) {
			continue
		}
		for _, part := range opaqueSplit.Split(run, -1) {
			if len(part) >= 20 {
				return true
			}
		}
	}
	return false
}
