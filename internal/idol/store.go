// 本文件拥有 state.json 和 blobs/ 的读写、当前对话、记忆和每日任务。
// Open 在读完状态后调用 ensurePresets。Update 在回调成功后整份写回。不调用模型。

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

// errf 用一句英文错误文本造 error。这些句子是返回给调用方的字面量。
// msg 传给 errors.New。
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

// Open 创建 blobs 目录并读 state.json。文件不存在或只有空白时用默认状态：主动开口和工作通道都打开。随后灌预设。
// dir 用来创建 blobs 目录，并作为 state.json 的父目录。
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
	} else if len(strings.TrimSpace(string(b))) == 0 {
		s.state = defaultState()
	} else if err := json.Unmarshal(b, &s.state); err != nil {
		return nil, err
	}
	if err := s.ensurePresets(); err != nil {
		return nil, err
	}
	return s, nil
}

// defaultState 是一份新桌面。ReachOut 和 WorkEnabled 都为真。
func defaultState() State {
	return State{Services: Services{ReachOut: true, WorkEnabled: true}}
}

// saveLocked 把 state 缩进写成临时文件再改名。调用方须已持有 mu。
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

// View 返回状态的一份 JSON 往返副本，调用方改它不会改到库里的那份。
func (s *Store) View() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, _ := json.Marshal(s.state)
	var out State
	_ = json.Unmarshal(b, &out)
	return out
}

// Update 在锁内改状态。fn 返回错误则不写盘。
func (s *Store) Update(fn func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.state); err != nil {
		return err
	}
	return s.saveLocked()
}

// SaveBlob 把图片字节写成 blobs 下的新 id。空字节拒绝；大于 2 MiB 拒绝。
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

// ReadBlob 按 id 读 blobs 里的文件。id 含 .. 或斜杠时拒绝。
func (s *Store) ReadBlob(id string) ([]byte, error) {
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return nil, errf("bad blob")
	}
	return os.ReadFile(filepath.Join(s.dir, "blobs", id))
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

// ensureCurrent 返回这位偶像当前的对话。没有就 forceNew 一段。
// idolID 与对话的 IdolID 比较。Current 为真则返回该对话，否则和 now 一起传给 forceNew。
// now 在没有当前对话时传给 forceNew。
func (st *State) ensureCurrent(idolID string, now time.Time) Conversation {
	for _, c := range st.Conversations {
		if c.IdolID == idolID && c.Current {
			return c
		}
	}
	return st.forceNew(idolID, now)
}

// forceNew 把这位偶像当前对话的 Current 清掉。那段还没有任何记录就删掉。然后追加一段新的当前对话。
// idolID 用来清掉这位偶像当前对话，并写入新对话的 IdolID。
// now 写入新对话的 CreatedAt。
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
// conID 与对话的 ID 比较，命中则把 rec 追加到 Records。
// rec 追加到命中对话的 Records。
func (st *State) appendRecord(conID string, rec Record) {
	for i := range st.Conversations {
		if st.Conversations[i].ID != conID {
			continue
		}
		st.Conversations[i].Records = append(st.Conversations[i].Records, rec)
		return
	}
}

// currentOf 返回这位偶像 Current 为真的对话。
// st 的 Conversations 用来查找。
// idolID 与对话的 IdolID 比较，且 Current 为真才返回。
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
		if id == "" {
			return nil
		}
		if !found {
			return errf("idol not found")
		}
		return nil
	})
}

// SetWorkSession 写下这位偶像当前的工作会话 id。
// idolID 传给 idol。
// sessionID 写入 WorkSessionID。
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
// recordID 与记录的 ID 比较。
// sessionID 写入命中记录的 WorkSession。
// workErr 写入命中记录的 WorkError。
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

// Truncate 把指定记录的 Segments 截到 keep 条。keep 小于 0 当成 0，大于现有长度则不变。找不到记录返回错误。
// recordID 与记录的 ID 比较。找不到返回 record not found。
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

// pinnedOf 返回第一位钉选且未隐藏的偶像。
// st 的 Idols 里，返回第一位 Pinned 为真且 Hidden 为假的。
func pinnedOf(st State) (Idol, bool) {
	for _, it := range st.Idols {
		if it.Pinned && !it.Hidden {
			return it, true
		}
	}
	return Idol{}, false
}

// remember 追加一句记忆，最长 80 个字符。空白、NONE 丢掉。同一位偶像未盖过的旧句若互相包含且两边都至少 6 个字符，旧句标上 SupersededBy。
// idol 的 ID 和 CircleID 写入新记忆，并用来找同一位未盖过的旧句。
// text 去掉空白。为空或与 NONE 不区分大小写相等时直接返回。rune 数大于 80 时截到 80。
// now 写入记忆的 At。
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

// visibleMemories 返回这位偶像能用的记忆：她自己的，以及同一圈子里别人的。已盖过的跳过，最多留最后 8 条。
// idol 的 ID 和 CircleID 用来挑选未盖过的记忆。结果多于 8 条时只留最后 8 条。
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

// formatMemories 把可见记忆写成提示。别人转告的标明出处，不能说成用户亲口讲的。
// st 传给 visibleMemories，并用 Idols 建名字表。
// idol 传给 visibleMemories。SourceIdol 等于它或为空时写成用户亲口说的。
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

// Location 返回用户卡片上的时区。
func (u UserCard) Location() *time.Location { return idolTZ(u) }

// BirthdayToday 判断用户时区的今天是不是生日。生日必须是 MM-DD。非闰年的 02-29 按 02-28。
// now 换到用户时区后格式化成 01-02，与生日比较。
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
// y 能被 4 整除，且不能被 100 整除或能被 400 整除时返回 true。
func leap(y int) bool {
	return y%4 == 0 && (y%100 != 0 || y%400 == 0)
}

// Night 在用户时区的 23 点到次日 7 点之前返回真。reachDue 在这段时间不开口。
// now 换到用户时区后取 Hour。大于等于 23 或小于 7 时返回 true。
func (u UserCard) Night(now time.Time) bool {
	h := now.In(u.Location()).Hour()
	return h >= 23 || h < 7
}

// day 返回用户时区的日期，形如 2006-01-02。
// now 换到用户时区后格式化成 2006-01-02。
func (st *State) day(now time.Time) string {
	return now.In(st.User.Location()).Format("2006-01-02")
}

// ensureDaily 在日期变化时把 Draws 补到最多 5，并换成当天的空任务。同一天重复调用什么也不做。
// now 传给 day。得到的日期与 DrawDay 相同则返回。否则 Draws 小于 5 时加一，并把 DrawDay 和当天任务写成这个日期。
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

// markTask 记下 talk、call 或 gift。每种当天第一次完成时 BonusDraws 加一。
// now 传给 ensureDaily。
// loc 没有被读取，只赋给空白标识符。
// kind 为 talk、call 或 gift 时记下对应任务。当天第一次完成时 BonusDraws 加一。
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
