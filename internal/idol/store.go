package idol

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func errf(msg string) error { return errors.New(msg) }

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

type Store struct {
	mu    sync.Mutex
	dir   string
	path  string
	state State
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, "blobs"), 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, path: filepath.Join(dir, "state.json")}
	b, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		s.state = defaultState()
		return s, s.saveLocked()
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		s.state = defaultState()
		return s, nil
	}
	if err := json.Unmarshal(b, &s.state); err != nil {
		return nil, err
	}
	return s, nil
}

func defaultState() State {
	return State{Services: Services{ReachOut: true, WorkEnabled: true}}
}

func (s *Store) saveLocked() error {
	b, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) View() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, _ := json.Marshal(s.state)
	var out State
	_ = json.Unmarshal(b, &out)
	return out
}

func (s *Store) Update(fn func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.state); err != nil {
		return err
	}
	return s.saveLocked()
}

func (s *Store) SaveBlob(b []byte) (string, error) {
	if len(b) == 0 {
		return "", errf("empty blob")
	}
	if len(b) > 2<<20 {
		return "", errf("image too large")
	}
	id := newID()
	path := filepath.Join(s.dir, "blobs", id)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) ReadBlob(id string) ([]byte, error) {
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return nil, errf("bad blob")
	}
	return os.ReadFile(filepath.Join(s.dir, "blobs", id))
}

func (st *State) idol(id string) (Idol, bool) {
	for _, it := range st.Idols {
		if it.ID == id {
			return it, true
		}
	}
	return Idol{}, false
}

func (st *State) putIdol(id Idol) {
	for i := range st.Idols {
		if st.Idols[i].ID == id.ID {
			st.Idols[i] = id
			return
		}
	}
	st.Idols = append(st.Idols, id)
}

func (st *State) ensureCurrent(idolID string, now time.Time) Conversation {
	for _, c := range st.Conversations {
		if c.IdolID == idolID && c.Current {
			return c
		}
	}
	return st.forceNew(idolID, now)
}

func (st *State) forceNew(idolID string, now time.Time) Conversation {
	for i := range st.Conversations {
		if st.Conversations[i].IdolID == idolID && st.Conversations[i].Current {
			st.Conversations[i].Current = false
			if len(st.Conversations[i].Records) == 0 {
				st.Conversations = append(st.Conversations[:i], st.Conversations[i+1:]...)
				break
			}
		}
	}
	c := Conversation{ID: newID(), IdolID: idolID, Current: true, CreatedAt: now}
	st.Conversations = append(st.Conversations, c)
	return c
}

func (st *State) appendRecord(conID string, rec Record) {
	for i := range st.Conversations {
		if st.Conversations[i].ID != conID {
			continue
		}
		st.Conversations[i].Records = append(st.Conversations[i].Records, rec)
		return
	}
}

func currentOf(st *State, idolID string) (Conversation, bool) {
	for _, c := range st.Conversations {
		if c.IdolID == idolID && c.Current {
			return c, true
		}
	}
	return Conversation{}, false
}

func (st *State) conversation(id string) (Conversation, bool) {
	for _, c := range st.Conversations {
		if c.ID == id {
			return c, true
		}
	}
	return Conversation{}, false
}

func (st *State) replaceConversation(c Conversation) {
	for i := range st.Conversations {
		if st.Conversations[i].ID == c.ID {
			st.Conversations[i] = c
			return
		}
	}
}

func (s *Store) Pin(id string) error {
	return s.Update(func(st *State) error {
		found := false
		for i := range st.Idols {
			on := st.Idols[i].ID == id
			st.Idols[i].Pinned = on
			if on {
				found = true
			}
		}
		if id == "" {
			return nil
		}
		if !found {
			return errf("idol not found")
		}
		return nil
	})
}

func (s *Store) SetWorkSession(idolID, sessionID string) error {
	return s.Update(func(st *State) error {
		idol, ok := st.idol(idolID)
		if !ok {
			return errf("idol not found")
		}
		idol.WorkSessionID = sessionID
		st.putIdol(idol)
		return nil
	})
}

func (s *Store) AttachWork(recordID, sessionID, workErr string) error {
	return s.Update(func(st *State) error {
		for i := range st.Conversations {
			for j := range st.Conversations[i].Records {
				if st.Conversations[i].Records[j].ID == recordID {
					st.Conversations[i].Records[j].WorkSession = sessionID
					st.Conversations[i].Records[j].WorkError = workErr
					return nil
				}
			}
		}
		return nil
	})
}

func (s *Store) Truncate(recordID string, keep int) error {
	return s.Update(func(st *State) error {
		for i := range st.Conversations {
			for j := range st.Conversations[i].Records {
				rec := &st.Conversations[i].Records[j]
				if rec.ID != recordID {
					continue
				}
				if keep < 0 {
					keep = 0
				}
				if keep > len(rec.Segments) {
					keep = len(rec.Segments)
				}
				rec.Segments = rec.Segments[:keep]
				return nil
			}
		}
		return errf("record not found")
	})
}

func pinnedOf(st State) (Idol, bool) {
	for _, it := range st.Idols {
		if it.Pinned && !it.Hidden {
			return it, true
		}
	}
	return Idol{}, false
}

func (st *State) remember(idol Idol, text string, now time.Time) {
	text = strings.TrimSpace(text)
	if text == "" || strings.EqualFold(text, "NONE") {
		return
	}
	if len([]rune(text)) > 80 {
		text = string([]rune(text)[:80])
	}
	mem := Memory{ID: newID(), Text: text, SourceIdol: idol.ID, CircleID: idol.CircleID, At: now}
	for i := range st.Memories {
		old := st.Memories[i]
		if old.SupersededBy != "" || old.SourceIdol != idol.ID {
			continue
		}
		if strings.Contains(old.Text, text) || strings.Contains(text, old.Text) {
			if len([]rune(old.Text)) >= 6 && len([]rune(text)) >= 6 {
				st.Memories[i].SupersededBy = mem.ID
			}
		}
	}
	st.Memories = append(st.Memories, mem)
}

func (st *State) visibleMemories(idol Idol) []Memory {
	var out []Memory
	names := map[string]string{}
	for _, it := range st.Idols {
		names[it.ID] = it.Name
	}
	_ = names
	for _, m := range st.Memories {
		if m.SupersededBy != "" {
			continue
		}
		if m.SourceIdol == idol.ID {
			out = append(out, m)
			continue
		}
		if idol.CircleID != "" && m.CircleID == idol.CircleID {
			out = append(out, m)
		}
	}
	if len(out) > 8 {
		out = out[len(out)-8:]
	}
	return out
}

func formatMemories(st *State, idol Idol) string {
	mems := st.visibleMemories(idol)
	if len(mems) == 0 {
		return ""
	}
	names := map[string]string{}
	for _, it := range st.Idols {
		names[it.ID] = it.Name
	}
	var b strings.Builder
	b.WriteString("你记得这些关于用户的事。别人告诉你的，只能用转述的口吻说：\n")
	for _, m := range mems {
		if m.SourceIdol == idol.ID || m.SourceIdol == "" {
			fmt.Fprintf(&b, "- （用户亲口对你说的）%s\n", m.Text)
			continue
		}
		name := names[m.SourceIdol]
		if name == "" {
			name = "另一位"
		}
		fmt.Fprintf(&b, "- （听%s说的，不是用户亲口告诉你的）%s\n", name, m.Text)
	}
	return b.String()
}

func (u UserCard) Location() *time.Location { return idolTZ(u) }

func (u UserCard) BirthdayToday(now time.Time) bool {
	b := strings.TrimSpace(u.Birthday)
	if len(b) != 5 || b[2] != '-' {
		return false
	}
	local := now.In(u.Location())
	today := local.Format("01-02")
	if b == "02-29" && !leap(local.Year()) {
		b = "02-28"
	}
	return today == b
}

func leap(y int) bool {
	return y%4 == 0 && (y%100 != 0 || y%400 == 0)
}

func (u UserCard) Night(now time.Time) bool {
	h := now.In(u.Location()).Hour()
	return h >= 23 || h < 7
}

func (st *State) day(now time.Time) string {
	return now.In(st.User.Location()).Format("2006-01-02")
}

func (st *State) ensureDaily(now time.Time) {
	day := st.day(now)
	if st.DrawDay == day {
		return
	}
	if st.Draws < 5 {
		st.Draws++
	}
	st.DrawDay = day
	st.Tasks = DayMark{Day: day}
}

func (st *State) markTask(now time.Time, loc *time.Location, kind string) {
	_ = loc
	st.ensureDaily(now)
	switch kind {
	case "talk":
		if !st.Tasks.Talk {
			st.Tasks.Talk = true
		}
		if st.Tasks.Talk && !st.Tasks.TalkRewarded {
			st.BonusDraws++
			st.Tasks.TalkRewarded = true
		}
	case "call":
		if !st.Tasks.Call {
			st.Tasks.Call = true
		}
		if st.Tasks.Call && !st.Tasks.CallRewarded {
			st.BonusDraws++
			st.Tasks.CallRewarded = true
		}
	case "gift":
		if !st.Tasks.Gift {
			st.Tasks.Gift = true
		}
		if st.Tasks.Gift && !st.Tasks.GiftRewarded {
			st.BonusDraws++
			st.Tasks.GiftRewarded = true
		}
	}
}
