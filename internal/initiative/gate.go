// Package initiative 决定要不要开口，并把感知收成一条条带证据的信号。
// 拥有：沉默、预判和反馈的计数，以及每天打断和被接受的条数。判断和记账分开：Assess 只判断，Deliver 在话真正送到之后才记下。
// 不拥有：工单、模型回合、文件正文。
package initiative

import (
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Fact 是一轮里看见的一件事。Sig 用来和上次对比。Eligible 为真才可能开口。
// Reason 是给人看的一句「为什么是这条」，为空时由送达方按种类补。Refs 是这条依据的材料，界面上可以点开。
type Fact struct {
	ID       string
	Code     string
	Class    string
	Name     string
	Detail   string
	State    string
	Sig      string
	Rank     int
	Eligible bool
	MemoryID string
	Reason   string
	Refs     []Ref
}

// Ref 是一条开口依据的材料。Kind 取 file、calendar、ticket、source、session、link、memory。
// Target 是界面打开它时用的编号或地址，没有可打开的东西时为空。
type Ref struct {
	Kind   string `json:"kind"`
	Label  string `json:"label"`
	Target string `json:"target,omitempty"`
}

// Signal 是给人看的一条感知。Speak 为真才推进主会话。Held 为真表示本来会说，这一轮按住了。
// 今天已经送到过的同一条既不 Speak 也不 Held。
type Signal struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Class    string `json:"class"`
	Name     string `json:"name"`
	Detail   string `json:"detail"`
	State    string `json:"state"`
	Why      string `json:"why,omitempty"`
	Speak    bool   `json:"speak"`
	Held     bool   `json:"held"`
	Priority int    `json:"priority"`
	MemoryID string `json:"memoryId,omitempty"`
	Reason   string `json:"reason,omitempty"`
	Refs     []Ref  `json:"refs,omitempty"`
	// Sig 是这一轮的签名。送到之后由 Deliver 记成上一次告诉人的样子。不发给界面。
	Sig string `json:"-"`
}

// Commitment 是引擎能读到的一条承诺。Until 是 YYYY-MM-DD，过了那天就不再留。
type Commitment struct {
	ID       string `json:"id"`
	Class    string `json:"class"`
	Title    string `json:"title"`
	Why      string `json:"why,omitempty"`
	Until    string `json:"until"`
	Priority int    `json:"priority"`
}

// Mute 记下某一类被按住的原因。Until 为 9999-12-31 表示先不要再提。
type Mute struct {
	Ignores int    `json:"ignores,omitempty"`
	Until   string `json:"until,omitempty"`
}

// Forever 是一直按住的那一天。要人在偏好里打开。
const Forever = "9999-12-31"

// ReachQuota 记下公开检索一天说了几次、上次是什么时候。一天几条、两条隔多久由这一轮的 Input 决定。
type ReachQuota struct {
	Day string `json:"day,omitempty"`
	N   int    `json:"n,omitempty"`
	At  string `json:"at,omitempty"`
}

// Tally 是一天里开口和人的反应。Spoken 是真正送到的条数，也就是打断了几次；Used 是标了「用上了」的条数。
// 其余是忽略、稍后、事实错了和一直按住的次数。主动性看的是 Used 和 Spoken 的比，不是 Spoken 本身。
type Tally struct {
	Spoken  int `json:"spoken,omitempty"`
	Used    int `json:"used,omitempty"`
	Ignored int `json:"ignored,omitempty"`
	Later   int `json:"later,omitempty"`
	Wrong   int `json:"wrong,omitempty"`
	Muted   int `json:"muted,omitempty"`
}

// DayTally 是某一天的计数。
type DayTally struct {
	Day string `json:"day"`
	Tally
}

// tallyDays 是计数留几天。更早的丢掉。
const tallyDays = 30

// Book 是开口的记忆。
// Seen 是变化比较的基线：第一次见到时记下；会开口的变化要等送到之后才换，没送到就一直算变化。
// Spoken 是某一类上次开口的日期。Told 是某一条上次送到的日期，同一天不再开口，也不算按住。
// Tally 按天记打断和反应。
type Book struct {
	Seen   map[string]string `json:"seen,omitempty"`
	Spoken map[string]string `json:"spoken,omitempty"`
	Told   map[string]string `json:"told,omitempty"`
	Mute   map[string]Mute   `json:"mute,omitempty"`
	Open   []Commitment      `json:"open,omitempty"`
	Reach  ReachQuota        `json:"reach,omitempty"`
	Tally  map[string]Tally  `json:"tally,omitempty"`
}

// Briefing 是这一轮的感知、要开口的判断，以及还留着的承诺。
type Briefing struct {
	Day     string       `json:"day"`
	Speak   bool         `json:"speak"`
	Signals []Signal     `json:"signals,omitempty"`
	Open    []Commitment `json:"open,omitempty"`
}

// Input 是一轮判断的材料。Idle 超过二十分钟时，工程线和仓库变化先按住。
// ReachDaily 和 ReachEvery 是公开检索当天的条数和两条之间的间隔。都为零时按空闲额度：一天 12 条，隔 35 分钟。
// ReachHold 为真时这一轮先不说检索，例如回合还在跑，或用户刚说完还没过十分钟。
type Input struct {
	Now        time.Time
	Welcome    bool
	Idle       time.Duration
	Facts      []Fact
	Book       Book
	ReachDaily int
	ReachEvery time.Duration
	ReachHold  bool
}

// Assess 写出这一轮该说什么。它不记账：同一条在 Deliver 之前，每一轮都还会开口。
// 第一次见到的工程线只建立基线，不开口。会开口的变化在送到之前保留原来的基线。
// 日历到期、来源错误、低电量、断网和例行结果可以在第一次就开口。同一轮里同一类只让排在最前的一条开口。
func Assess(in Input) (Briefing, Book) {
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	day := now.Format("2006-01-02")
	book := cloneBook(in.Book)
	for id, at := range book.Told {
		if at < day {
			delete(book.Told, id)
		}
	}
	facts := append([]Fact(nil), in.Facts...)
	sortFacts(facts)
	idleHold := in.Idle >= 20*time.Minute
	claimed := map[string]bool{}
	reachClaimed := false
	var signals []Signal
	for _, fact := range facts {
		if strings.TrimSpace(fact.ID) == "" || strings.TrimSpace(fact.Name) == "" {
			continue
		}
		sig := fact.Sig
		if sig == "" {
			sig = fact.State
		}
		prev, seen := book.Seen[fact.ID]
		changed := seen && prev != sig
		pending := fact.Eligible && changeCode(fact.Code)
		if fact.Code != "reach" && (!seen || !pending) {
			book.Seen[fact.ID] = sig
		}
		want := false
		why := ""
		if fact.Eligible {
			switch {
			case changeCode(fact.Code):
				want = changed
				why = "changed"
			case fact.Code == "reach":
				want = !seen || changed
				why = "match"
			default:
				want = book.Told[fact.ID] != day
				why = fact.State
			}
		}
		speak := want
		held := false
		if speak && muted(book, fact, day) {
			speak = false
			held = true
		}
		// 例行的每一次运行各说一次：人自己设的例行，有话才回，靠按住和忽略来收，不按「一类一天一次」挡。
		if speak && fact.Class != "reach" && ((fact.Code != "routine" && book.Spoken[fact.Class] == day) || claimed[fact.Class]) {
			speak = false
			held = true
		}
		if speak && fact.Class == "reach" && (in.ReachHold || reachClaimed || !reachRoom(book.Reach, now, in.ReachDaily, in.ReachEvery)) {
			speak = false
			held = true
		}
		if speak && idleHold && (fact.Class == "line" || fact.Class == "git") {
			speak = false
			held = true
		}
		if !want && !awareness(fact.Code) {
			continue
		}
		if want && !speak {
			held = true
		}
		limit := 120
		switch fact.Code {
		case "reach":
			limit = 900
		case "routine":
			limit = 600
		}
		signals = append(signals, Signal{
			ID:       fact.ID,
			Code:     fact.Code,
			Class:    fact.Class,
			Name:     clip(fact.Name, 80),
			Detail:   clip(fact.Detail, limit),
			State:    fact.State,
			Why:      why,
			Speak:    speak,
			Held:     held,
			Priority: fact.Rank,
			MemoryID: fact.MemoryID,
			Reason:   clip(strings.TrimSpace(fact.Reason), 120),
			Refs:     cleanRefs(fact.Refs),
			Sig:      sig,
		})
		if speak {
			if fact.Class == "reach" {
				reachClaimed = true
			} else {
				claimed[fact.Class] = true
			}
		}
	}
	if in.Welcome && !anySpeak(signals) && book.Spoken["welcome"] != day {
		signals = append(signals, Signal{
			ID: "welcome", Code: "welcome", Class: "welcome", Name: "welcome", Why: "welcome", Speak: true,
		})
	}
	book.Open = mergeOpen(book.Open, signals, day)
	speak := anySpeak(signals)
	return Briefing{Day: day, Speak: speak, Signals: signals, Open: book.Open}, book
}

// Deliver 记下 signals 里已经送到的话。那一类今天开过口，那一条今天说过，当天的打断数加一。
// 会开口的变化换成送到时的签名，公开检索记进当天的配额。没有 Speak 的条目不记。
func Deliver(book Book, signals []Signal, now time.Time) Book {
	if now.IsZero() {
		now = time.Now()
	}
	day := now.Format("2006-01-02")
	book = cloneBook(book)
	for _, signal := range signals {
		if !signal.Speak || strings.TrimSpace(signal.ID) == "" {
			continue
		}
		if signal.Class != "" {
			book.Spoken[signal.Class] = day
		}
		book.Told[signal.ID] = day
		if signal.Sig != "" {
			book.Seen[signal.ID] = signal.Sig
		}
		if signal.Code == "reach" {
			book.Reach = markReach(book.Reach, now)
		}
		book = bump(book, day, func(t *Tally) { t.Spoken++ })
	}
	return book
}

// Feedback 记下人对某一类的反应，并记进当天的计数。wrong 返回要忘掉的记忆编号，没有则返回空串。
// kind 取 used、ignore、later、wrong、mute、unmute。mute 一直按住，unmute 打开。
func Feedback(book Book, className, memoryID, kind, day string) (Book, string) {
	className = strings.TrimSpace(className)
	if className == "" {
		return book, ""
	}
	book = cloneBook(book)
	book = countFeedback(book, kind, day)
	switch kind {
	case "used":
		delete(book.Mute, className)
		book.Open = dropClass(book.Open, className)
	case "ignore":
		mute := book.Mute[className]
		mute.Ignores++
		if mute.Ignores >= 3 {
			mute.Until = Forever
		}
		book.Mute[className] = mute
		book.Open = dropClass(book.Open, className)
	case "later":
		mute := book.Mute[className]
		if mute.Until < day {
			mute.Until = day
		}
		book.Mute[className] = mute
		book.Open = dropClass(book.Open, className)
	case "mute":
		mute := book.Mute[className]
		mute.Until = Forever
		book.Mute[className] = mute
		book.Open = dropClass(book.Open, className)
	case "unmute":
		delete(book.Mute, className)
	case "wrong":
		mute := book.Mute[className]
		mute.Until = Forever
		book.Mute[className] = mute
		book.Open = dropClass(book.Open, className)
		return book, strings.TrimSpace(memoryID)
	}
	return book, ""
}

// Quiet 列出 day 这一天仍被按住的类别。
func Quiet(book Book, day string) map[string]Mute {
	out := map[string]Mute{}
	for name, mute := range book.Mute {
		if muteOn(mute, day) {
			out[name] = mute
		}
	}
	return out
}

// Recent 按日期从新到旧列出最近 n 天有计数的日子。n 小于 1 时按 7。
func Recent(book Book, n int) []DayTally {
	if n < 1 {
		n = 7
	}
	days := make([]string, 0, len(book.Tally))
	for day := range book.Tally {
		days = append(days, day)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(days)))
	if len(days) > n {
		days = days[:n]
	}
	out := make([]DayTally, 0, len(days))
	for _, day := range days {
		out = append(out, DayTally{Day: day, Tally: book.Tally[day]})
	}
	return out
}

// Accepted 是这几天被接受的比例：用上的条数除以送到的条数。一条都没送到时 ok 为假。
func Accepted(days []DayTally) (rate float64, ok bool) {
	spoken, used := 0, 0
	for _, d := range days {
		spoken += d.Spoken
		used += d.Used
	}
	if spoken == 0 {
		return 0, false
	}
	return float64(used) / float64(spoken), true
}

// NoteSchedule 把一条人写过的日程记成承诺。它不开口。
func NoteSchedule(book Book, id, title, day string) Book {
	id = strings.TrimSpace(id)
	title = clip(strings.TrimSpace(title), 80)
	if id == "" || title == "" {
		return book
	}
	if scrub(title) == "" {
		title = id
	}
	next := dropID(book.Open, "schedule:"+id)
	next = append(next, Commitment{
		ID: "schedule:" + id, Class: "schedule", Title: title, Why: "scheduled", Until: day, Priority: 3,
	})
	book.Open = next
	return book
}

// AllowSchedule 在编排台明确关掉时不跑日程。空模式仍算人写过的日程，可以跑。
func AllowSchedule(mode string) bool {
	return strings.TrimSpace(mode) != "off"
}

// changeCode 在这种感知只因为变化才开口时为真。
func changeCode(code string) bool {
	switch code {
	case "line", "git", "poi":
		return true
	default:
		return false
	}
}

func cloneBook(in Book) Book {
	out := Book{
		Seen:   make(map[string]string, len(in.Seen)),
		Spoken: make(map[string]string, len(in.Spoken)),
		Told:   make(map[string]string, len(in.Told)),
		Mute:   make(map[string]Mute, len(in.Mute)),
		Open:   append([]Commitment(nil), in.Open...),
		Reach:  in.Reach,
		Tally:  make(map[string]Tally, len(in.Tally)),
	}
	for k, v := range in.Seen {
		out.Seen[k] = v
	}
	for k, v := range in.Spoken {
		out.Spoken[k] = v
	}
	for k, v := range in.Told {
		out.Told[k] = v
	}
	for k, v := range in.Mute {
		out.Mute[k] = v
	}
	for k, v := range in.Tally {
		out.Tally[k] = v
	}
	return out
}

// bump 改 day 这一天的计数，再丢掉 tallyDays 之前的日子。book 要是已经复制过的。
func bump(book Book, day string, fn func(*Tally)) Book {
	if strings.TrimSpace(day) == "" {
		return book
	}
	if book.Tally == nil {
		book.Tally = map[string]Tally{}
	}
	t := book.Tally[day]
	fn(&t)
	book.Tally[day] = t
	if len(book.Tally) > tallyDays {
		days := make([]string, 0, len(book.Tally))
		for d := range book.Tally {
			days = append(days, d)
		}
		sort.Strings(days)
		for _, d := range days[:len(days)-tallyDays] {
			delete(book.Tally, d)
		}
	}
	return book
}

func countFeedback(book Book, kind, day string) Book {
	switch kind {
	case "used":
		return bump(book, day, func(t *Tally) { t.Used++ })
	case "ignore":
		return bump(book, day, func(t *Tally) { t.Ignored++ })
	case "later":
		return bump(book, day, func(t *Tally) { t.Later++ })
	case "wrong":
		return bump(book, day, func(t *Tally) { t.Wrong++ })
	case "mute":
		return bump(book, day, func(t *Tally) { t.Muted++ })
	}
	return book
}

func cleanRefs(in []Ref) []Ref {
	var out []Ref
	for _, ref := range in {
		ref.Kind = strings.TrimSpace(ref.Kind)
		ref.Label = clip(strings.TrimSpace(ref.Label), 80)
		ref.Target = strings.TrimSpace(ref.Target)
		if ref.Label == "" || scrub(ref.Label) == "" {
			continue
		}
		out = append(out, ref)
		if len(out) == 4 {
			break
		}
	}
	return out
}

func reachRoom(q ReachQuota, now time.Time, daily int, every time.Duration) bool {
	if daily < 1 {
		daily = 12
	}
	if every <= 0 {
		every = 35 * time.Minute
	}
	day := now.Format("2006-01-02")
	if q.Day == day && q.N >= daily {
		return false
	}
	if q.At == "" {
		return true
	}
	at, err := time.Parse(time.RFC3339, q.At)
	if err != nil {
		return true
	}
	return now.Sub(at) >= every
}

func markReach(q ReachQuota, now time.Time) ReachQuota {
	day := now.Format("2006-01-02")
	if q.Day != day {
		q.Day = day
		q.N = 0
	}
	q.N++
	q.At = now.Format(time.RFC3339)
	return q
}

func muted(book Book, fact Fact, day string) bool {
	return muteOn(book.Mute[fact.Class], day) || muteOn(book.Mute[fact.ID], day)
}

func muteOn(mute Mute, day string) bool {
	if mute.Until == "" {
		return false
	}
	return mute.Until >= day
}

func awareness(code string) bool {
	switch code {
	case "line", "git", "power", "net", "focus", "idle", "mail", "calendar", "channel", "poi", "ticket", "routine":
		return true
	default:
		return false
	}
}

func anySpeak(signals []Signal) bool {
	for _, signal := range signals {
		if signal.Speak {
			return true
		}
	}
	return false
}

func mergeOpen(prev []Commitment, signals []Signal, day string) []Commitment {
	var kept []Commitment
	for _, item := range prev {
		if item.Class == "schedule" && item.Until >= day {
			kept = append(kept, item)
		}
	}
	for _, signal := range signals {
		if !signal.Speak && !signal.Held {
			continue
		}
		kept = append(kept, Commitment{
			ID: signal.ID, Class: signal.Class, Title: signal.Name, Why: signal.Why, Until: day, Priority: signal.Priority,
		})
	}
	return kept
}

func dropClass(items []Commitment, className string) []Commitment {
	var out []Commitment
	for _, item := range items {
		if item.Class == className {
			continue
		}
		out = append(out, item)
	}
	return out
}

func dropID(items []Commitment, id string) []Commitment {
	var out []Commitment
	for _, item := range items {
		if item.ID == id {
			continue
		}
		out = append(out, item)
	}
	return out
}

func sortFacts(facts []Fact) {
	for i := 1; i < len(facts); i++ {
		item := facts[i]
		j := i
		for j > 0 && facts[j-1].Rank > item.Rank {
			facts[j] = facts[j-1]
			j--
		}
		facts[j] = item
	}
}

func clip(text string, n int) string {
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	runes := []rune(text)
	return string(runes[:n])
}

func scrub(text string) string {
	lower := strings.ToLower(text)
	for _, word := range []string{"secret", "token", "password", "cookie", "credential", "api_key", "apikey"} {
		if strings.Contains(lower, word) {
			return ""
		}
	}
	return text
}
