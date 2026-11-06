// 本文件拥有 state.json 和 blobs/ 的读写、当前对话和每日任务。
// Open 在读完状态后灌预设、迁移旧格式。View 给一份深拷贝；Update 在拷贝上改，回调成功才换上并整份写回，失败时库里的那份不变。不调用模型。

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

const (
	// maxBlob 是一个 blob 最大多少字节。照片和头像存之前会压到远小于这个数。
	maxBlob = 8 << 20
	// maxDraws 是最多攒几次抽奖：一周的量。
	maxDraws = 7
)

// errf 用一句英文错误文本造 error。这些句子是返回给调用方的字面量，前端按它们挑文案。
func errf(msg string) error { return errors.New(msg) }

// newID 返回 8 字节随机数的十六进制，用作偶像、对话、记录和图片的 id。
func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Store 把 State 放在 dir/state.json，图片放在 dir/blobs。mu 盖住 state 和写盘。
type Store struct {
	mu    sync.Mutex
	dir   string
	path  string
	state State
}

// Open 创建 blobs 目录并读 state.json。文件不存在或只有空白时用默认状态。
// 文件坏了时先改名成 state.json.broken-<时间> 留底，再试写盘时留下的 state.json.tmp，都不行就从默认状态开始，不让整个偶像模块打不开。
// 随后灌预设，并把旧格式的好感迁移到现在的刻度。
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, "blobs"), 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, path: filepath.Join(dir, "state.json")}
	b, err := os.ReadFile(s.path)
	switch {
	case err != nil && !os.IsNotExist(err):
		return nil, err
	case err != nil || len(strings.TrimSpace(string(b))) == 0:
		s.state = defaultState()
	default:
		if jerr := json.Unmarshal(b, &s.state); jerr != nil {
			s.state = s.recover(b)
		}
	}
	if err := s.ensurePresets(); err != nil {
		return nil, err
	}
	if s.state.AffinityRev < affinityRevision {
		s.mu.Lock()
		s.state.migrateAffinity(time.Now())
		err := s.saveLocked()
		s.mu.Unlock()
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// recover 在 state.json 读不出时留底，并尽量从写盘的临时文件恢复。raw 是读不出的原文。
func (s *Store) recover(raw []byte) State {
	_ = os.WriteFile(fmt.Sprintf("%s.broken-%s", s.path, time.Now().Format("20060102-150405")), raw, 0o644)
	if tmp, err := os.ReadFile(s.path + ".tmp"); err == nil {
		var st State
		if json.Unmarshal(tmp, &st) == nil {
			return st
		}
	}
	return defaultState()
}

// defaultState 是一份新桌面。ReachOut 和 WorkEnabled 都为真。
func defaultState() State {
	return State{Services: Services{ReachOut: true, WorkEnabled: true}, AffinityRev: affinityRevision}
}

// saveLocked 把 state 写成临时文件再改名。调用方须已持有 mu。
func (s *Store) saveLocked() error {
	b, err := json.Marshal(s.state)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// View 返回状态的一份深拷贝，调用方改它不会改到库里的那份。
func (s *Store) View() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.clone()
}

// Update 在状态的拷贝上执行 fn。fn 返回错误时什么也不变；成功时换上拷贝并写盘。
func (s *Store) Update(fn func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.state.clone()
	if err := fn(&next); err != nil {
		return err
	}
	prev := s.state
	s.state = next
	if err := s.saveLocked(); err != nil {
		s.state = prev
		return err
	}
	return nil
}

// clone 深拷贝一份状态。各个切片和切片里的切片都换成新的底层数组。
func (st State) clone() State {
	out := st
	out.Idols = make([]Idol, len(st.Idols))
	for i, it := range st.Idols {
		it.Outfits = append([]Outfit(nil), it.Outfits...)
		it.LikedGifts = append([]string(nil), it.LikedGifts...)
		out.Idols[i] = it
	}
	out.Circles = make([]Circle, len(st.Circles))
	for i, c := range st.Circles {
		c.Members = append([]string(nil), c.Members...)
		out.Circles[i] = c
	}
	out.Conversations = make([]Conversation, len(st.Conversations))
	for i, c := range st.Conversations {
		recs := make([]Record, len(c.Records))
		for j, r := range c.Records {
			r.Segments = append([]Segment(nil), r.Segments...)
			recs[j] = r
		}
		c.Records = recs
		out.Conversations[i] = c
	}
	out.Memories = append([]Memory(nil), st.Memories...)
	out.Affinity = append([]Affinity(nil), st.Affinity...)
	out.Inventory = append([]Stock(nil), st.Inventory...)
	out.Lore = append([]Lore(nil), st.Lore...)
	out.Calls = append([]CallLog(nil), st.Calls...)
	out.Reaches = append([]ReachMark(nil), st.Reaches...)
	out.Achievements = append([]string(nil), st.Achievements...)
	return out
}

// SaveBlob 把图片字节写成 blobs 下的新 id。空字节和超过 maxBlob 的拒绝。
func (s *Store) SaveBlob(b []byte) (string, error) {
	if len(b) == 0 {
		return "", errf("empty blob")
	}
	if len(b) > maxBlob {
		return "", errf("image too large")
	}
	id := newID()
	if err := os.WriteFile(filepath.Join(s.dir, "blobs", id), b, 0o644); err != nil {
		return "", err
	}
	return id, nil
}

// ReadBlob 按 id 读 blobs 里的文件。id 含 .. 或斜杠时拒绝。
func (s *Store) ReadBlob(id string) ([]byte, error) {
	if !safeBlobID(id) {
		return nil, errf("bad blob")
	}
	return os.ReadFile(filepath.Join(s.dir, "blobs", strings.TrimSpace(id)))
}

// removeBlob 删掉一个 blob 文件。预设头像（preset- 开头）留着，换一位同名偶像还能用。
func (s *Store) removeBlob(id string) {
	if !safeBlobID(id) || strings.HasPrefix(id, "preset-") {
		return
	}
	_ = os.Remove(filepath.Join(s.dir, "blobs", strings.TrimSpace(id)))
}

// RemoveBlobs 删掉这些 blob 文件，比如删除一段对话时它里面的照片。
func (s *Store) RemoveBlobs(ids []string) {
	for _, id := range ids {
		s.removeBlob(id)
	}
}

// safeBlobID 判断 blob id 能不能安全地拼进路径。
func safeBlobID(id string) bool {
	id = strings.TrimSpace(id)
	return id != "" && !strings.Contains(id, "..") && !strings.ContainsAny(id, `/\:`)
}

// DeleteIdol 删掉一位偶像和她的一切：对话、她那里记下的记忆、好感、材料、通话和先开口的记录，以及只属于她的图片。
// 她正被放在桌面上时改放桌宠 Yoyo（如果还在）。找不到时返回 idol not found。
func (s *Store) DeleteIdol(id string) error {
	var blobs []string
	err := s.Update(func(st *State) error {
		gone, ok := st.idol(id)
		if !ok {
			return errf("idol not found")
		}
		blobs = append(blobs, gone.Portrait)
		for _, o := range gone.Outfits {
			blobs = append(blobs, o.Blob)
		}
		for _, c := range st.Conversations {
			if c.IdolID != id {
				continue
			}
			for _, r := range c.Records {
				if r.PhotoID != "" {
					blobs = append(blobs, r.PhotoID)
				}
			}
		}
		set := map[string]bool{id: true}
		var idols []Idol
		for _, it := range st.Idols {
			if it.ID != id {
				idols = append(idols, it)
			}
		}
		st.Idols = idols
		st.Conversations = filterConversations(st.Conversations, set)
		st.Memories = relinkMemories(filterMemories(st.Memories, set))
		st.Affinity = filterAffinity(st.Affinity, set)
		st.Lore = filterLore(st.Lore, set)
		st.Calls = filterCalls(st.Calls, set)
		st.Reaches = filterReaches(st.Reaches, set)
		for i := range st.Circles {
			var keep []string
			for _, m := range st.Circles[i].Members {
				if m != id {
					keep = append(keep, m)
				}
			}
			st.Circles[i].Members = keep
		}
		if gone.Pinned {
			for i := range st.Idols {
				st.Idols[i].Pinned = st.Idols[i].ID == petIdolID
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, b := range blobs {
		s.removeBlob(b)
	}
	return nil
}

// relinkMemories 让取代者已经不在的记忆重新成立，免得它们永远被一条不存在的记忆盖住。
func relinkMemories(mems []Memory) []Memory {
	have := map[string]bool{}
	for _, m := range mems {
		have[m.ID] = true
	}
	for i := range mems {
		if mems[i].SupersededBy != "" && !have[mems[i].SupersededBy] {
			mems[i].SupersededBy = ""
		}
	}
	return mems
}

// DeleteMemory 删掉一条记忆，取代关系照 deleteMemory 接上。找不到时返回 memory not found。
func (s *Store) DeleteMemory(id string) error {
	return s.Update(func(st *State) error {
		if !st.deleteMemory(id) {
			return errf("memory not found")
		}
		return nil
	})
}

// idol 按 id 在这份状态里找偶像。
func (st *State) idol(id string) (Idol, bool) {
	for _, it := range st.Idols {
		if it.ID == id {
			return it, true
		}
	}
	return Idol{}, false
}

// putIdol 按 id 替换已有偶像，没有则追加。
func (st *State) putIdol(id Idol) {
	for i := range st.Idols {
		if st.Idols[i].ID == id.ID {
			st.Idols[i] = id
			return
		}
	}
	st.Idols = append(st.Idols, id)
}

// record 按 id 找一条记录，返回指向状态里那条的指针，找不到时返回 nil。
func (st *State) record(id string) *Record {
	for i := range st.Conversations {
		for j := range st.Conversations[i].Records {
			if st.Conversations[i].Records[j].ID == id {
				return &st.Conversations[i].Records[j]
			}
		}
	}
	return nil
}

// ensureCurrent 返回这位偶像当前的对话。没有就 forceNew 一段。
func (st *State) ensureCurrent(idolID string, now time.Time) Conversation {
	for _, c := range st.Conversations {
		if c.IdolID == idolID && c.Current {
			return c
		}
	}
	return st.forceNew(idolID, now)
}

// forceNew 把这位偶像当前对话的 Current 清掉，那段还没有任何记录就删掉，然后追加一段新的当前对话。
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

// appendRecord 把一条记录接到指定对话末尾。找不到对话就什么也不做。
func (st *State) appendRecord(conID string, rec Record) {
	for i := range st.Conversations {
		if st.Conversations[i].ID == conID {
			st.Conversations[i].Records = append(st.Conversations[i].Records, rec)
			return
		}
	}
}

// currentOf 返回这位偶像 Current 为真的对话。
func currentOf(st *State, idolID string) (Conversation, bool) {
	for _, c := range st.Conversations {
		if c.IdolID == idolID && c.Current {
			return c, true
		}
	}
	return Conversation{}, false
}

// conversation 按对话 id 查找。
func (st *State) conversation(id string) (Conversation, bool) {
	for _, c := range st.Conversations {
		if c.ID == id {
			return c, true
		}
	}
	return Conversation{}, false
}

// replaceConversation 按 id 换掉已有对话。找不到就什么也不做。
func (st *State) replaceConversation(c Conversation) {
	for i := range st.Conversations {
		if st.Conversations[i].ID == c.ID {
			st.Conversations[i] = c
			return
		}
	}
}

// Pin 只钉选这一位偶像。id 为空时全部取消钉选。找不到该 id 时返回错误。
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
		if id != "" && !found {
			return errf("idol not found")
		}
		return nil
	})
}

// SetWorkSession 写下这位偶像当前的工作会话 id。
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

// AttachWork 把工作会话 id 和错误写回指定记录。找不到记录也返回 nil。
func (s *Store) AttachWork(recordID, sessionID, workErr string) error {
	return s.Update(func(st *State) error {
		if rec := st.record(recordID); rec != nil {
			rec.WorkSession = sessionID
			rec.WorkError = workErr
		}
		return nil
	})
}

// Truncate 把指定记录的 Segments 截到 keep 条：通话里她被打断时，只留用户听到的那几段。keep 小于 1 时至少留 1 段，免得留下空回复。
func (s *Store) Truncate(recordID string, keep int) error {
	return s.Update(func(st *State) error {
		rec := st.record(recordID)
		if rec == nil {
			return errf("record not found")
		}
		if keep < 1 {
			keep = 1
		}
		if keep < len(rec.Segments) {
			rec.Segments = rec.Segments[:keep]
		}
		return nil
	})
}

// PhotoState 返回一条记录的照片 id 和状态（pending、failed 或空）。找不到时 ok 为假。
func (s *Store) PhotoState(recordID string) (photoID, state string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec := s.state.record(recordID); rec != nil {
		return rec.PhotoID, rec.PhotoState, true
	}
	return "", "", false
}

// pinnedOf 返回第一位钉选且未隐藏的偶像。
func pinnedOf(st State) (Idol, bool) {
	for _, it := range st.Idols {
		if it.Pinned && !it.Hidden {
			return it, true
		}
	}
	return Idol{}, false
}

// Location 返回用户卡片上的时区。
func (u UserCard) Location() *time.Location { return idolTZ(u) }

// BirthdayToday 判断用户时区的今天是不是生日。生日必须是 MM-DD。非闰年的 02-29 按 02-28。
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

// leap 判断公历闰年。
func leap(y int) bool {
	return y%4 == 0 && (y%100 != 0 || y%400 == 0)
}

// Night 在用户时区的 22 点到次日 8 点之前返回真。她不在这段时间先开口。
func (u UserCard) Night(now time.Time) bool {
	h := now.In(u.Location()).Hour()
	return h >= nightStartHour || h < nightEndHour
}

// day 返回用户时区的日期，形如 2006-01-02。
func (st *State) day(now time.Time) string {
	return now.In(st.User.Location()).Format("2006-01-02")
}

// ensureDaily 在跨过用户时区的一天时补抽奖次数（每过一天一次，最多攒 maxDraws 次），并换成当天的空任务。同一天重复调用什么也不做。
func (st *State) ensureDaily(now time.Time) {
	day := st.day(now)
	if st.DrawDay == day {
		return
	}
	gained := 1
	if prev, err := time.ParseInLocation("2006-01-02", st.DrawDay, st.User.Location()); err == nil {
		if cur, err := time.ParseInLocation("2006-01-02", day, st.User.Location()); err == nil && cur.After(prev) {
			gained = int(cur.Sub(prev).Hours()/24 + 0.5)
		}
	}
	if gained < 1 {
		gained = 1
	}
	if st.Draws < maxDraws {
		st.Draws += gained
		if st.Draws > maxDraws {
			st.Draws = maxDraws
		}
	}
	st.DrawDay = day
	st.Tasks = DayMark{Day: day}
}

// markTask 记下 talk、call 或 gift。每种当天第一次完成时 BonusDraws 加一。
func (st *State) markTask(now time.Time, kind string) {
	st.ensureDaily(now)
	switch kind {
	case "talk":
		st.Tasks.Talk = true
		if !st.Tasks.TalkRewarded {
			st.BonusDraws++
			st.Tasks.TalkRewarded = true
		}
	case "call":
		st.Tasks.Call = true
		if !st.Tasks.CallRewarded {
			st.BonusDraws++
			st.Tasks.CallRewarded = true
		}
	case "gift":
		st.Tasks.Gift = true
		if !st.Tasks.GiftRewarded {
			st.BonusDraws++
			st.Tasks.GiftRewarded = true
		}
	}
}
