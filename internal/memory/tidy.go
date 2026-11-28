// tidy.go 让每日简报那一次模型顺手整理记忆，但只提建议，人点了「对」才改。
// 拥有：找出相近、却没有自动合并的两条（TidyPairs）；收下模型对每一对的判断（Suggest）；
// 人点「对」时并成一条或换成后来的说法，点「不对」时记住这一对以后不再问（Settle）。
// 攒够 tidyEvery 条新记下或改过的记忆才再问一次；还有没处理的建议时不问。建议和「不再问」的对放在 memory-tidy.json，不进 memory.json。
// 经历、本机清点、不带进对话的和已经过去的安排不参与。
// 不拥有：叫模型、把建议做成信里的卡（app/brief_tidy.go）。模型的判断从不直接改记忆。

package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// tidyEvery 是上次问过之后，至少要有几条新记下或改过的记忆才再问一次。
	tidyEvery = 12
	// tidyPendingMax 是最多攒着几条没处理的建议。
	tidyPendingMax = 6
	// tidyDismissedMax 是最多记住几对「不再问」，多了丢最早的。
	tidyDismissedMax = 500
	// tidyStale 是一条建议放多久没人处理就不再给。
	tidyStale = 14 * 24 * time.Hour
	// tidyTextMax 是合并后说法的字数上限。
	tidyTextMax = 500
)

const (
	// VerdictSame 是同一件事换了说法：并成一条。
	VerdictSame = "same"
	// VerdictChange 是后来改了口：只留后来的说法，以前的进历史。
	VerdictChange = "change"
	// VerdictDifferent 是两件不同的事：以后不再问这一对。
	VerdictDifferent = "different"
)

// ErrTidyGone 是这条建议已经处理过、过期了，或者其中一条已经忘掉了。
var ErrTidyGone = errors.New("memory: 这条整理建议已经处理过或过期了")

// ErrTidyChanged 是建议之后这两条又改过了，建议不再作数。
var ErrTidyChanged = errors.New("memory: 这两条记忆后来改过了，这条建议不再作数")

// Pair 是相近的两条：Old 是早一些记下或改过的，New 是晚一些的。
type Pair struct {
	Old Item `json:"old"`
	New Item `json:"new"`
}

// Verdict 是模型对一对的判断。Text 只在 same 时有用：合并后的说法。
type Verdict struct {
	Pair    Pair
	Verdict string
	Text    string
}

// Suggestion 是一条等人点头的整理建议。OldText、NewText 是提建议时两条的说法，处理时拿来核对有没有改过。
type Suggestion struct {
	ID        string    `json:"id"`
	Old       string    `json:"old"`
	New       string    `json:"new"`
	OldText   string    `json:"old_text"`
	NewText   string    `json:"new_text"`
	Verdict   string    `json:"verdict"`
	Text      string    `json:"text,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// tidyState 是 memory-tidy.json：没处理的建议、不再问的对、上次问的时间。
type tidyState struct {
	Pending   []Suggestion `json:"pending,omitempty"`
	Dismissed []string     `json:"dismissed,omitempty"`
	AskedAt   time.Time    `json:"asked_at,omitzero"`
}

func (s *Store) tidyPath() string {
	return filepath.Join(filepath.Dir(s.path), "memory-tidy.json")
}

// loadTidy 读 memory-tidy.json。没有或读不出时给空的。
func (s *Store) loadTidy() tidyState {
	var st tidyState
	if b, err := os.ReadFile(s.tidyPath()); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return st
}

// saveTidy 先写到 .tmp 再改名，权限 0o600，和 memory.json 一样。
func (s *Store) saveTidy(st tidyState) error {
	if len(st.Dismissed) > tidyDismissedMax {
		st.Dismissed = st.Dismissed[len(st.Dismissed)-tidyDismissedMax:]
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.tidyPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.tidyPath())
}

// pairKey 是一对的键，和先后无关。
func pairKey(a, b string) string {
	if b < a {
		a, b = b, a
	}
	return a + "|" + b
}

// tidyable 在这一条可以参与整理时为真。
func tidyable(it Item, now time.Time) bool {
	return !machineItem(it) && it.Kind != KindEpisodic && !it.Muted && !it.Staging && strings.TrimSpace(it.Text) != "" && !Expired(it, now)
}

// livePendingLocked 是还作数的建议：两条都还在，也没放太久。调用时要已持有锁。
func (s *Store) livePendingLocked(st tidyState, now time.Time) []Suggestion {
	var out []Suggestion
	for _, sug := range st.Pending {
		if now.Sub(sug.CreatedAt) > tidyStale || s.index(sug.Old) < 0 || s.index(sug.New) < 0 {
			continue
		}
		out = append(out, sug)
	}
	return out
}

// TidyPairs 返回这一次要问的相近的对，最相近的在前，最多 limit 对。不到时候时返回 nil：
// 还有没处理的建议，或者上次问过之后新记下、改过的不到 tidyEvery 条。第一次问不用等。
// 相近是同一类、共有至少两个词、占任一边 relateShare，但还没像到自动合并（mergeShare）。人说过「不对」的对不再给。
func (s *Store) TidyPairs(limit int) []Pair {
	if limit <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncLocked()
	now := s.clock()
	st := s.loadTidy()
	if len(s.livePendingLocked(st, now)) > 0 {
		return nil
	}
	if !st.AskedAt.IsZero() {
		fresh := 0
		for _, it := range s.items {
			if !machineItem(it) && it.Touched().After(st.AskedAt) {
				fresh++
			}
		}
		if fresh < tidyEvery {
			return nil
		}
	}
	skip := map[string]bool{}
	for _, k := range st.Dismissed {
		skip[k] = true
	}
	var pool []Item
	var terms []map[string]bool
	for _, it := range s.items {
		if tidyable(it, now) {
			pool = append(pool, it)
			terms = append(terms, simTerms(it.Text))
		}
	}
	type cand struct {
		pair  Pair
		share float64
	}
	var found []cand
	for i := range pool {
		for j := i + 1; j < len(pool); j++ {
			a, b := pool[i], pool[j]
			if a.Kind != b.Kind || skip[pairKey(a.ID, b.ID)] {
				continue
			}
			shared, x, y := overlap(terms[i], terms[j])
			if shared < 2 || max(x, y) < relateShare || min(x, y) >= mergeShare {
				continue
			}
			old, nw := a, b
			if b.Touched().Before(a.Touched()) {
				old, nw = b, a
			}
			found = append(found, cand{pair: Pair{Old: old, New: nw}, share: max(x, y)})
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].share > found[j].share })
	var out []Pair
	for _, c := range found[:min(len(found), limit)] {
		out = append(out, c.pair)
	}
	return out
}

// Suggest 记下这一次问过了，并收下模型的判断：same、change 留成等人点头的建议，different 记成以后不再问。
// 判断的那两条在问过之后改过了、或者已经没了时不收。合并后的说法像密钥时丢掉，退回用后来的说法。返回新收下的建议。
func (s *Store) Suggest(asked []Pair, got []Verdict) []Suggestion {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncLocked()
	now := s.clock()
	st := s.loadTidy()
	st.Pending = s.livePendingLocked(st, now)
	st.AskedAt = now.UTC()
	dismissed := map[string]bool{}
	for _, k := range st.Dismissed {
		dismissed[k] = true
	}
	pending := map[string]bool{}
	for _, sug := range st.Pending {
		pending[pairKey(sug.Old, sug.New)] = true
	}
	var added []Suggestion
	for _, v := range got {
		p := v.Pair
		key := pairKey(p.Old.ID, p.New.ID)
		oi, ni := s.index(p.Old.ID), s.index(p.New.ID)
		if p.Old.ID == p.New.ID || oi < 0 || ni < 0 || dismissed[key] || pending[key] ||
			factKey(s.items[oi].Text) != factKey(p.Old.Text) || factKey(s.items[ni].Text) != factKey(p.New.Text) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(v.Verdict)) {
		case VerdictDifferent:
			st.Dismissed = append(st.Dismissed, key)
			dismissed[key] = true
		case VerdictSame, VerdictChange:
			if len(st.Pending) >= tidyPendingMax {
				continue
			}
			verdict := strings.ToLower(strings.TrimSpace(v.Verdict))
			text := ""
			if verdict == VerdictSame {
				text = clipRunes(strings.TrimSpace(v.Text), tidyTextMax)
				if secretText(text) {
					text = ""
				}
			}
			sum := sha256.Sum256([]byte(key + "\n" + verdict + "\n" + now.UTC().Format(time.RFC3339Nano)))
			sug := Suggestion{
				ID:        "tidy-" + hex.EncodeToString(sum[:6]),
				Old:       p.Old.ID,
				New:       p.New.ID,
				OldText:   s.items[oi].Text,
				NewText:   s.items[ni].Text,
				Verdict:   verdict,
				Text:      text,
				CreatedAt: now.UTC(),
			}
			st.Pending = append(st.Pending, sug)
			pending[key] = true
			added = append(added, sug)
		}
	}
	_ = s.saveTidy(st)
	return added
}

// Suggestions 返回还作数的整理建议，先提的在前。
func (s *Store) Suggestions() []Suggestion {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncLocked()
	return s.livePendingLocked(s.loadTidy(), s.clock())
}

// Settle 处理一条建议。accept 为假时这一对以后不再问，记忆不动。
// accept 为真时两条并成一条，留在早的那条上（编号和最初记下的时间不变）：change 用后来的说法，same 用合并后的说法（没有时用后来的）。
// 被换掉的说法进历史，所以记忆页还能撤销；来源记成当事人确认过；次数相加。两条在建议之后改过时返回 ErrTidyChanged，不改。
func (s *Store) Settle(id string, accept bool) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.begin()()
	now := s.clock()
	st := s.loadTidy()
	live := s.livePendingLocked(st, now)
	at := -1
	for i, sug := range live {
		if sug.ID == strings.TrimSpace(id) {
			at = i
			break
		}
	}
	if at < 0 {
		return Item{}, ErrTidyGone
	}
	sug := live[at]
	st.Pending = append(live[:at:at], live[at+1:]...)
	if !accept {
		st.Dismissed = append(st.Dismissed, pairKey(sug.Old, sug.New))
		return Item{}, s.saveTidy(st)
	}
	oi, ni := s.index(sug.Old), s.index(sug.New)
	if oi < 0 || ni < 0 || factKey(s.items[oi].Text) != factKey(sug.OldText) || factKey(s.items[ni].Text) != factKey(sug.NewText) {
		_ = s.saveTidy(st)
		return Item{}, ErrTidyChanged
	}
	old, nw := s.items[oi], s.items[ni]
	stamp := now.UTC()
	text, until := nw.Text, nw.ValidUntil
	if sug.Verdict == VerdictSame && strings.TrimSpace(sug.Text) != "" {
		var anchoredUntil string
		text, anchoredUntil = anchorDates(strings.TrimSpace(sug.Text), now)
		if anchoredUntil != "" {
			until = anchoredUntil
		}
	}
	next := old
	// 后来的说法先进历史，以前的说法排在最后：撤销一步退回的是以前的说法。
	history := append(append([]Revision(nil), old.History...), nw.History...)
	if factKey(nw.Text) != factKey(text) {
		history = append(history, Revision{Text: nw.Text, At: updatedOf(nw), Source: nw.Source})
	}
	if factKey(old.Text) != factKey(text) {
		history = append(history, Revision{Text: old.Text, At: updatedOf(old), Source: old.Source})
	}
	next.History = pushRevision(nil, history...)
	next.Text = text
	next.ValidUntil = until
	next.Source = "user"
	next.UpdatedAt = stamp
	next.Seen = seenOf(old) + seenOf(nw)
	if !nw.CreatedAt.IsZero() && (next.CreatedAt.IsZero() || nw.CreatedAt.Before(next.CreatedAt)) {
		next.CreatedAt = nw.CreatedAt
	}
	if next.Project == "" {
		next.Project = nw.Project
	}
	if next.Origin == "" {
		next.Origin = nw.Origin
	}
	s.items[oi] = next
	s.items = append(s.items[:ni], s.items[ni+1:]...)
	_ = s.flush()
	// 提到被并掉那一条的其他建议不再作数。
	keep := st.Pending[:0]
	for _, other := range st.Pending {
		if other.Old != nw.ID && other.New != nw.ID {
			keep = append(keep, other)
		}
	}
	st.Pending = keep
	_ = s.saveTidy(st)
	return next, nil
}

// clipRunes 超过 n 个字时截到前 n 个。
func clipRunes(text string, n int) string {
	r := []rune(text)
	if len(r) <= n {
		return text
	}
	return string(r[:n])
}
