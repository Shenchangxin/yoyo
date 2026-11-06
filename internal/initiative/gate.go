// Package initiative 决定要不要开口，并把感知收成一条条带证据的信号。
// 拥有：沉默、预判和反馈的计数。不拥有：工单、模型回合、文件正文。
package initiative

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Fact 是一轮里看见的一件事。Sig 用来和上次对比。Eligible 为真才可能开口。
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
}

// Signal 是给人看的一条感知。Speak 为真才推进主会话。Held 为真表示本来会说，这一轮按住了。
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

// ReachQuota 记下公开检索一天说了几次、上次是什么时候。一天最多三次，两次至少隔三小时。
type ReachQuota struct {
	Day string `json:"day,omitempty"`
	N   int    `json:"n,omitempty"`
	At  string `json:"at,omitempty"`
}

// Book 是开口的记忆。Seen 是上次签名，Spoken 是某一类上次开口的日期。
type Book struct {
	Seen   map[string]string `json:"seen,omitempty"`
	Spoken map[string]string `json:"spoken,omitempty"`
	Mute   map[string]Mute   `json:"mute,omitempty"`
	Open   []Commitment      `json:"open,omitempty"`
	Reach  ReachQuota        `json:"reach,omitempty"`
}

// Briefing 是这一轮的感知、要开口的判断，以及还留着的承诺。
type Briefing struct {
	Day     string       `json:"day"`
	Speak   bool         `json:"speak"`
	Signals []Signal     `json:"signals,omitempty"`
	Open    []Commitment `json:"open,omitempty"`
}

// Input 是一轮判断的材料。Idle 超过二十分钟时，工程线和仓库变化先按住。
type Input struct {
	Now     time.Time
	Welcome bool
	Idle    time.Duration
	Facts   []Fact
	Book    Book
}

// Assess 写出这一轮该说什么。第一次见到的工程线只建立基线，不开口。
// 日历到期、等人、卡住、来源错误、低电量和断网可以在第一次就开口。
func Assess(in Input) (Briefing, Book) {
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	day := now.Format("2006-01-02")
	book := in.Book
	if book.Seen == nil {
		book.Seen = map[string]string{}
	}
	if book.Spoken == nil {
		book.Spoken = map[string]string{}
	}
	if book.Mute == nil {
		book.Mute = map[string]Mute{}
	}
	facts := append([]Fact(nil), in.Facts...)
	sortFacts(facts)
	idleHold := in.Idle >= 20*time.Minute
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
		if fact.Code != "reach" {
			book.Seen[fact.ID] = sig
		}
		want := false
		why := ""
		if fact.Eligible {
			switch fact.Code {
			case "line", "git", "poi":
				want = changed
				why = "changed"
			case "reach":
				want = !seen || changed
				why = "match"
			default:
				want = true
				why = fact.State
			}
		}
		speak := want
		held := false
		if speak && muted(book, fact, day) {
			speak = false
			held = true
		}
		if speak && book.Spoken[fact.Class] == day && fact.Class != "reach" {
			speak = false
			held = true
		}
		if speak && fact.Class == "reach" && !reachRoom(book.Reach, now) {
			speak = false
			held = true
		}
		if speak && idleHold && (fact.Class == "line" || fact.Class == "git") {
			speak = false
			held = true
			if why == "" {
				why = "changed"
			}
		}
		if !want && !awareness(fact.Code) {
			continue
		}
		if want && !speak {
			held = true
		}
		limit := 120
		if fact.Code == "reach" {
			limit = 900
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
		})
		if speak {
			book.Spoken[fact.Class] = day
			if fact.Code == "reach" {
				book.Seen[fact.ID] = sig
				book.Reach = markReach(book.Reach, now)
			}
		}
	}
	if in.Welcome && !anySpeak(signals) && book.Spoken["welcome"] != day {
		signals = append(signals, Signal{
			ID: "welcome", Code: "welcome", Class: "welcome", Name: "welcome", Why: "welcome", Speak: true,
		})
		book.Spoken["welcome"] = day
	}
	book.Open = mergeOpen(book.Open, signals, day)
	speak := anySpeak(signals)
	return Briefing{Day: day, Speak: speak, Signals: signals, Open: book.Open}, book
}

// Feedback 记下人对某一类的反应。wrong 返回要忘掉的记忆编号，没有则返回空串。
// kind 取 used、ignore、later、wrong。
func Feedback(book Book, className, memoryID, kind, day string) (Book, string) {
	className = strings.TrimSpace(className)
	if className == "" {
		return book, ""
	}
	if book.Mute == nil {
		book.Mute = map[string]Mute{}
	}
	switch kind {
	case "used":
		delete(book.Mute, className)
		book.Open = dropClass(book.Open, className)
	case "ignore":
		mute := book.Mute[className]
		mute.Ignores++
		if mute.Ignores >= 3 {
			mute.Until = "9999-12-31"
		}
		book.Mute[className] = mute
		book.Open = dropClass(book.Open, className)
	case "later":
		mute := book.Mute[className]
		mute.Until = day
		book.Mute[className] = mute
		book.Open = dropClass(book.Open, className)
	case "wrong":
		mute := book.Mute[className]
		mute.Until = "9999-12-31"
		book.Mute[className] = mute
		book.Open = dropClass(book.Open, className)
		return book, strings.TrimSpace(memoryID)
	}
	return book, ""
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

func reachRoom(q ReachQuota, now time.Time) bool {
	day := now.Format("2006-01-02")
	if q.Day == day && q.N >= 3 {
		return false
	}
	if q.At == "" {
		return true
	}
	at, err := time.Parse(time.RFC3339, q.At)
	if err != nil {
		return true
	}
	return now.Sub(at) >= 3*time.Hour
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
	case "line", "git", "power", "net", "focus", "idle", "mail", "calendar", "channel", "poi", "ticket":
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
