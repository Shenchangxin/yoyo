// desk_loop.go 在 Go 里按分钟刷新清点、判断要不要开口，并把通过开口门的话写进主会话。
// 一条话只有真正写进会话之后才记成说过，界面读到的判断和推送不会互相消耗。
// 晨间和晚间简报的时刻也在这里看（brief_clock.go），界面只在人点的时候写那一档。例行的结果经同一道门（routine.go）。
// 工单等人走交接，不在这里重复。

package app

import (
	"sort"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/reach"
)

// deskEvery 是送达循环的间隔。开口门本身按天限次，这里只决定多久看一次。
const deskEvery = time.Minute

// DeskState 是开口门此刻的样子：最近一轮判断，今天仍被按住的类别，以及最近七天打断了几次、被接受了几次。
// At 是那一轮判断的时刻。Accepted 是七天里用上的条数除以送到的条数，一条都没送到时为负。
type DeskState struct {
	Briefing initiative.Briefing   `json:"briefing"`
	Mutes    []DeskMute            `json:"mutes"`
	At       string                `json:"at,omitempty"`
	Tally    []initiative.DayTally `json:"tally"`
	Accepted float64               `json:"accepted"`
	// Lead 是这台电脑上谁在跑主动链路。不选主时为空。
	Lead DeskLead `json:"lead"`
	// Presence 是头像的全局状态：导航轨、伴侣窗和托盘读的是同一份，见 presence.go。
	Presence Presence `json:"presence"`
}

// DeskMute 是一类被按住的开口。Until 为 9999-12-31 表示要人在偏好里打开。Ignores 是标过没用的次数。
type DeskMute struct {
	Class   string `json:"class"`
	Until   string `json:"until,omitempty"`
	Ignores int    `json:"ignores,omitempty"`
}

// DeskState 返回最近一轮判断、按住的类别和谁在跑主动链路。不选主时第一次调用才开始按分钟送达；选主时由领头的进程送，见 lead.go。
func (appHost *deskHost) DeskState() DeskState {
	out := DeskState{Mutes: []DeskMute{}, Tally: []initiative.DayTally{}, Accepted: -1}
	if appHost.app == nil || appHost.app.Home == nil {
		return out
	}
	if !appHost.app.managed() {
		appHost.app.ensureDeskLoop()
	}
	briefing, at := appHost.app.seenBriefing()
	out.Briefing = briefing
	out.Lead = appHost.app.LeadState()
	out.Presence = appHost.app.Presence()
	if !at.IsZero() {
		out.At = at.UTC().Format(time.RFC3339)
	}
	store, err := appHost.app.deskBook()
	if err != nil {
		return out
	}
	for name, mute := range store.Quiet(time.Now().Format("2006-01-02")) {
		out.Mutes = append(out.Mutes, DeskMute{Class: name, Until: mute.Until, Ignores: mute.Ignores})
	}
	sort.Slice(out.Mutes, func(i, j int) bool { return out.Mutes[i].Class < out.Mutes[j].Class })
	if days := store.Recent(7); len(days) > 0 {
		out.Tally = days
		if rate, ok := initiative.Accepted(days); ok {
			out.Accepted = rate
		}
	}
	return out
}

// ensureDeskLoop 只启动一次送达循环。Close 之后不再重启。
func (appHost *deskHost) ensureDeskLoop() {
	appHost.app.deskOnce.Do(func() {
		stop := make(chan struct{})
		appHost.app.bookMu.Lock()
		appHost.app.deskStop = stop
		appHost.app.bookMu.Unlock()
		go appHost.app.deskLoop(stop)
	})
}

// stopDeskLoop 停掉送达循环。还没启动时什么都不做。
func (appHost *deskHost) stopDeskLoop() {
	if appHost.app == nil {
		return
	}
	appHost.app.bookMu.Lock()
	stop := appHost.app.deskStop
	appHost.app.deskStop = nil
	appHost.app.bookMu.Unlock()
	if stop != nil {
		close(stop)
	}
}

func (appHost *deskHost) deskLoop(stop <-chan struct{}) {
	tick := time.NewTicker(deskEvery)
	defer tick.Stop()
	for {
		appHost.app.deskTick(time.Now())
		select {
		case <-stop:
			return
		case <-tick.C:
		}
	}
}

// deskTick 刷新一轮清点和判断，再把该说的写进主会话。工作站暂停时整轮跳过。
func (appHost *deskHost) deskTick(now time.Time) {
	if appHost.app == nil || appHost.app.IsPaused() {
		return
	}
	appHost.app.briefTick(now)
	appHost.app.imFlush(now)
	snap := appHost.app.PoiRefresh()
	if snap.Initiative == nil || !briefingDelivers(*snap.Initiative) {
		return
	}
	meta, err := appHost.app.MainSession()
	if err != nil || strings.TrimSpace(meta.ID) == "" {
		return
	}
	appHost.app.deliverSpeaks(meta.ID, *snap.Initiative, now)
}

// briefingDelivers 在这一轮有话要写进主会话时为真。开口的条目要送，按住的资讯也要送，不留在任务栏。
func briefingDelivers(b initiative.Briefing) bool {
	for _, signal := range b.Signals {
		if signal.Class == "ticket" || signal.Code == "welcome" {
			continue
		}
		if signal.Speak || (signal.Held && signal.Code == "reach") {
			return true
		}
	}
	return false
}

// deliverSpeaks 把 b 里该开口的写进 sessionID，再把真正写进去的记成已经说过。返回写进去的条目。
// 工单走交接，欢迎信走简报，都不在这里重复。写不进去的留到下一轮，不会被记成说过。
// 按住的资讯同样写进主会话：人在忙、或还没到下一条的间隔，都不再改放到任务栏。人按住的类别不送。当天额度用完后不再追加。
func (appHost *deskHost) deliverSpeaks(sessionID string, b initiative.Briefing, now time.Time) []initiative.Signal {
	store, err := appHost.app.deskBook()
	if err != nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	day := b.Day
	if day == "" {
		day = now.Format("2006-01-02")
	}
	quiet := store.Quiet(day)
	daily := reach.AllowanceFor(appHost.app.reachPace(now)).Daily
	if daily < 1 {
		daily = 12
	}
	sentCount := store.ReachSent(day)
	var sent []initiative.Signal
	for _, signal := range b.Signals {
		if signal.Class == "ticket" || signal.Code == "welcome" || !deliverable(signal, quiet) {
			continue
		}
		if !signal.Speak && sentCount >= daily {
			continue
		}
		note := speakNote(sessionID, day, signal)
		if note.Text == "" {
			continue
		}
		if !appHost.app.DeskPush(note) {
			continue
		}
		if !signal.Speak {
			signal.Speak = true
		}
		if signal.Code == "reach" {
			sentCount++
		}
		appHost.app.imSpeak(signal, note)
		sent = append(sent, signal)
	}
	if len(sent) > 0 {
		store.Deliver(sent, now)
		appHost.app.routineDelivered(sent)
	}
	return sent
}

// deliverable 在这条信号该写进主会话时为真。开口的都送。按住的只送资讯，并且人按住的类别或这一条不送。
func deliverable(signal initiative.Signal, quiet map[string]initiative.Mute) bool {
	if _, muted := quiet[signal.Class]; muted {
		return false
	}
	if _, muted := quiet[signal.ID]; muted {
		return false
	}
	if signal.Speak {
		return true
	}
	return signal.Held && signal.Code == "reach"
}

// speakNote 把一条信号收成主会话里的短讯。一条信号一天只有一个编号，重复写会被 DeskPush 挡住。
func speakNote(sessionID, day string, s initiative.Signal) DeskNote {
	note := DeskNote{
		SessionID: sessionID,
		Kind:      "speak",
		Slot:      s.ID,
		Day:       day,
		Text:      speakText(s),
		Chip:      s.Name,
		Class:     s.Class,
		Why:       s.Why,
		Name:      s.Name,
		Detail:    s.Detail,
		State:     s.State,
		MemoryID:  s.MemoryID,
		Reason:    signalReason(s),
		Refs:      s.Refs,
	}
	if s.Code == "reach" {
		note.Detail = ""
	}
	return note
}

// speakText 只用信号里的名字、状态和细节写一句。界面有这一类的说法时按界面语言重写，这一句留给会话记录和路由。
func speakText(s initiative.Signal) string {
	name := strings.TrimSpace(s.Name)
	detail := strings.TrimSpace(s.Detail)
	switch s.Code {
	case "reach":
		if detail != "" {
			return detail
		}
		return name
	case "calendar":
		if s.State == "overdue" {
			return "「" + name + "」已经过了那一天。"
		}
		return "「" + name + "」今天到期。"
	case "channel":
		if detail != "" {
			return name + " 处于错误：" + detail
		}
		return name + " 处于错误，从这里进来的工作会漏。"
	case "line":
		line := "「" + name + "」跟上一次看见的时候不一样了：" + lineWord(s.State)
		if detail != "" {
			line += "，" + detail
		}
		return line + "。"
	case "git":
		line := "仓库「" + name + "」跟上一次看见的时候不一样了"
		if detail != "" {
			line += "：分支 " + detail
		}
		switch s.State {
		case "dirty":
			line += "，有还没提交的改动"
		case "clean":
			line += "，已经干净"
		}
		return line + "。"
	case "routine":
		if s.State == "failed" {
			line := "例行「" + name + "」没跑成"
			if detail != "" {
				line += "：" + detail
			}
			return line + "。"
		}
		if detail != "" {
			return "例行「" + name + "」：\n" + detail
		}
		return "例行「" + name + "」跑完了。"
	case "power":
		return "电量 " + name + "%，没在充电。"
	case "net":
		return "网络断了。"
	}
	if detail == "" {
		return name
	}
	return name + "\n" + detail
}

func lineWord(state string) string {
	switch state {
	case "stalled":
		return "停在一段空文件"
	case "quiet":
		return "这阵子没再动"
	default:
		return "最近还在改"
	}
}
