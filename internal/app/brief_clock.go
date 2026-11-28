// brief_clock.go 在 Go 里按设置里的时刻写晨间和晚间简报，并推进主会话。界面不再到点生成，只在人点的时候写那一档。
// 跨过设定时刻之后两小时内都会写：电脑睡过了头、或者应用晚开，醒来还在窗口里就补一封，并在信里写明晚了多久。
// 没写成时隔十分钟再试，不每分钟叫一次模型。写成的信经过开口门记账，当天的打断数加一；
// 材料里没有值得写的事时只留档，不放芯片，也不算打断。

package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/initiative"
)

// briefWindow 是跨过设定时刻之后还会写这一档的时长。
const briefWindow = 2 * time.Hour

// briefRetry 是没写成之后隔多久再试。
const briefRetry = 10 * time.Minute

// briefLate 是晚过这么久才在信里写明晚了。
const briefLate = 10 * time.Minute

// briefTry 是一档简报最近一次去写的样子。Busy 为真表示模型还在写。
type briefTry struct {
	At   time.Time
	Busy bool
}

// briefSlotAt 是一档简报和它在一天里的分钟数。
type briefSlotAt struct {
	Slot   string
	Minute int
}

// briefDue 是此刻该有、还在窗口里的一档。At 是设定的时刻，Day 是那一档属于的日子。
type briefDue struct {
	Slot string
	Day  string
	At   time.Time
}

// BriefSlots 读出开着的几档。off 表示关掉；空或写错时用默认的 08:30 和 21:30。
func (d DeskConfig) BriefSlots() []briefSlotAt {
	var out []briefSlotAt
	for _, slot := range []struct {
		name, value string
		fallback    int
	}{
		{"morning", d.BriefMorning, 8*60 + 30},
		{"evening", d.BriefEvening, 21*60 + 30},
	} {
		v := strings.TrimSpace(slot.value)
		if strings.EqualFold(v, "off") {
			continue
		}
		minute, ok := clockMinutes(v)
		if !ok {
			minute = slot.fallback
		}
		out = append(out, briefSlotAt{Slot: slot.name, Minute: minute})
	}
	return out
}

// dueBriefs 列出 now 落在窗口里的几档。设定时刻在昨天、窗口跨过午夜的也算，日子记成昨天。
func dueBriefs(d DeskConfig, now time.Time) []briefDue {
	now = now.In(time.Local)
	var out []briefDue
	for _, slot := range d.BriefSlots() {
		for back := 0; back <= 1; back++ {
			base := time.Date(now.Year(), now.Month(), now.Day()-back, 0, 0, 0, 0, time.Local)
			at := base.Add(time.Duration(slot.Minute) * time.Minute)
			if now.Before(at) || !now.Before(at.Add(briefWindow)) {
				continue
			}
			out = append(out, briefDue{Slot: slot.Slot, Day: base.Format("2006-01-02"), At: at})
			break
		}
	}
	return out
}

// briefTick 看这一分钟有没有该写的简报。写的那一下放到后台，不挡住开口短讯的送达。
func (a *App) briefTick(now time.Time) {
	if a == nil || a.Home == nil || a.IsPaused() {
		return
	}
	for _, due := range dueBriefs(a.Config.Desk, now) {
		if a.briefRecorded(due.Slot, due.Day) {
			continue
		}
		key := due.Slot + ":" + due.Day
		if !a.claimBrief(key, now) {
			continue
		}
		go func(due briefDue, key string) {
			a.writeBriefAt(due, time.Now())
			a.releaseBrief(key)
		}(due, key)
	}
}

// claimBrief 在这一档现在可以去写时为真，并记成正在写。还在写、或者上次没写成还没过 briefRetry 时为假。
func (a *App) claimBrief(key string, now time.Time) bool {
	a.clockMu.Lock()
	defer a.clockMu.Unlock()
	if a.clock == nil {
		a.clock = map[string]briefTry{}
	}
	prev, seen := a.clock[key]
	if seen && (prev.Busy || now.Sub(prev.At) < briefRetry) {
		return false
	}
	a.clock[key] = briefTry{At: now, Busy: true}
	for k, v := range a.clock {
		if now.Sub(v.At) > 48*time.Hour {
			delete(a.clock, k)
		}
	}
	return true
}

func (a *App) releaseBrief(key string) {
	a.clockMu.Lock()
	defer a.clockMu.Unlock()
	if try, ok := a.clock[key]; ok {
		try.Busy = false
		a.clock[key] = try
	}
}

// writeBriefAt 写这一档并推进主会话。写下了、或者早就写下了时返回真；模型还在写或这一次没写成时返回假。
func (a *App) writeBriefAt(due briefDue, now time.Time) bool {
	if a.briefRecorded(due.Slot, due.Day) {
		return true
	}
	meta, err := a.MainSession()
	if err != nil || strings.TrimSpace(meta.ID) == "" {
		return false
	}
	got := a.ComposeToday(due.Slot)
	if got.Pending || (!got.Woven && !got.Quiet) {
		return false
	}
	zh := a.briefChinese()
	note := DeskNote{
		SessionID: meta.ID,
		Kind:      "brief",
		Slot:      due.Slot,
		Day:       due.Day,
		Chip:      briefChip(due.Slot, due.At, zh),
	}
	cards := got.Cards
	// 等人点头的记忆整理建议排在简报的卡后面（brief_tidy.go）。只有它们时也推进，不算安静。
	tidy := a.tidyCards(zh)
	if (got.Quiet || len(cards) == 0) && len(tidy) == 0 {
		cards = []BriefCard{a.quietCard(zh)}
		note.Quiet = true
		note.Text = pick(zh, "今天没有值得写的事。", "Nothing worth writing today.")
	} else {
		note.Text = briefPushText(due.Slot, zh)
		cards = append(append([]BriefCard(nil), cards...), tidy...)
		if late := now.Sub(due.At); late >= briefLate && strings.TrimSpace(cards[0].Mark) == "" {
			cards[0].Mark = lateMark(due.At, late, zh)
		}
	}
	body, err := json.Marshal(struct {
		Cards  []BriefCard       `json:"cards"`
		Picked map[string]string `json:"picked"`
		Woven  bool              `json:"woven"`
	}{cards, map[string]string{}, true})
	if err != nil {
		return false
	}
	note.Brief = string(body)
	if !a.DeskPush(note) {
		return false
	}
	if !note.Quiet {
		if store, err := a.deskBook(); err == nil {
			store.Deliver([]initiative.Signal{{
				ID: "brief:" + due.Slot + ":" + due.Day, Code: "brief", Class: "brief", Name: note.Chip, Speak: true,
			}}, now)
		}
		// IM 里点不了「对」「不对」，记忆整理卡只在桌面的简报里出现。
		a.imBrief(note.Chip, withoutTidy(cards))
	}
	return true
}

// quietCard 是没有值得写的事时的那一张。邮箱和日历没连上时写明没有核对。
func (a *App) quietCard(zh bool) BriefCard {
	var unchecked []string
	a.bookMu.Lock()
	for _, s := range a.deskSeen.Signals {
		if s.State != "unchecked" {
			continue
		}
		switch s.Code {
		case "mail":
			unchecked = append(unchecked, pick(zh, "邮箱", "mail"))
		case "calendar":
			unchecked = append(unchecked, pick(zh, "日历", "calendar"))
		}
	}
	a.bookMu.Unlock()
	card := BriefCard{
		ID:      "quiet",
		Layout:  "plain",
		Kicker:  pick(zh, "简报", "Brief"),
		Title:   pick(zh, "今天没有值得写的事", "Nothing worth writing today"),
		Lede:    pick(zh, "日程、任务、记忆和桌面里都没有要你先处理的。", "Nothing in your calendar, tasks, memory or desktop needs you first."),
		Items:   []BriefBit{},
		Choices: []BriefBit{},
	}
	if len(unchecked) > 0 {
		card.Note = pick(zh, "没有核对："+strings.Join(unchecked, "、")+"还没连上。", "Not checked: "+strings.Join(unchecked, ", ")+" is not connected.")
	}
	return card
}

func (a *App) briefChinese() bool {
	return !strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.Config.Locale)), "en")
}

func briefChip(slot string, at time.Time, zh bool) string {
	if zh {
		name := "晨间简报"
		if slot == "evening" {
			name = "晚间简报"
		}
		return fmt.Sprintf("%s · %d月%d日", name, int(at.Month()), at.Day())
	}
	name := "Morning brief"
	if slot == "evening" {
		name = "Evening brief"
	}
	return name + " · " + at.Format("Jan 2")
}

func briefPushText(slot string, zh bool) string {
	if slot == "evening" {
		return pick(zh, "晚间简报写好了。", "Your evening brief is ready.")
	}
	return pick(zh, "晨间简报写好了。", "Your morning brief is ready.")
}

func lateMark(at time.Time, late time.Duration, zh bool) string {
	minutes := int(late.Round(time.Minute) / time.Minute)
	if zh {
		return fmt.Sprintf("比设定的 %s 晚了 %d 分钟：那时这台电脑没醒着或应用没开。", at.Format("15:04"), minutes)
	}
	return fmt.Sprintf("%d minutes later than the set %s: the computer was asleep or the app was closed.", minutes, at.Format("15:04"))
}

func pick(zh bool, a, b string) string {
	if zh {
		return a
	}
	return b
}
