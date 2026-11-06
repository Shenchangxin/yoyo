// desk_loop.go 在 Go 里按分钟刷新清点、判断要不要开口，并把通过开口门的话写进主会话。
// 一条话只有真正写进会话之后才记成说过，界面读到的判断和推送不会互相消耗。
// 不拥有晨间和晚间简报的时刻，那一档仍由界面到点生成。工单等人走交接，不在这里重复。

package app

import (
	"sort"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/initiative"
)

// deskEvery 是送达循环的间隔。开口门本身按天限次，这里只决定多久看一次。
const deskEvery = time.Minute

// DeskState 是开口门此刻的样子：最近一轮判断，以及今天仍被按住的类别。At 是那一轮判断的时刻。
type DeskState struct {
	Briefing initiative.Briefing `json:"briefing"`
	Mutes    []DeskMute          `json:"mutes"`
	At       string              `json:"at,omitempty"`
}

// DeskMute 是一类被按住的开口。Until 为 9999-12-31 表示要人在偏好里打开。Ignores 是标过没用的次数。
type DeskMute struct {
	Class   string `json:"class"`
	Until   string `json:"until,omitempty"`
	Ignores int    `json:"ignores,omitempty"`
}

// DeskState 返回最近一轮判断和按住的类别。第一次调用时开始按分钟送达，所以只有接上界面的进程才会往主会话里说话。
func (a *App) DeskState() DeskState {
	out := DeskState{Mutes: []DeskMute{}}
	if a == nil || a.Home == nil {
		return out
	}
	a.ensureDeskLoop()
	a.bookMu.Lock()
	out.Briefing = a.deskSeen
	at := a.deskSeenAt
	a.bookMu.Unlock()
	if !at.IsZero() {
		out.At = at.UTC().Format(time.RFC3339)
	}
	store, err := a.deskBook()
	if err != nil {
		return out
	}
	for name, mute := range store.Quiet(time.Now().Format("2006-01-02")) {
		out.Mutes = append(out.Mutes, DeskMute{Class: name, Until: mute.Until, Ignores: mute.Ignores})
	}
	sort.Slice(out.Mutes, func(i, j int) bool { return out.Mutes[i].Class < out.Mutes[j].Class })
	return out
}

// ensureDeskLoop 只启动一次送达循环。Close 之后不再重启。
func (a *App) ensureDeskLoop() {
	a.deskOnce.Do(func() {
		stop := make(chan struct{})
		a.bookMu.Lock()
		a.deskStop = stop
		a.bookMu.Unlock()
		go a.deskLoop(stop)
	})
}

// stopDeskLoop 停掉送达循环。还没启动时什么都不做。
func (a *App) stopDeskLoop() {
	if a == nil {
		return
	}
	a.bookMu.Lock()
	stop := a.deskStop
	a.deskStop = nil
	a.bookMu.Unlock()
	if stop != nil {
		close(stop)
	}
}

func (a *App) deskLoop(stop <-chan struct{}) {
	tick := time.NewTicker(deskEvery)
	defer tick.Stop()
	for {
		a.deskTick(time.Now())
		select {
		case <-stop:
			return
		case <-tick.C:
		}
	}
}

// deskTick 刷新一轮清点和判断，再把该说的写进主会话。工作站暂停时整轮跳过。
func (a *App) deskTick(now time.Time) {
	if a == nil || a.IsPaused() {
		return
	}
	snap := a.PoiRefresh()
	if snap.Initiative == nil || !snap.Initiative.Speak {
		return
	}
	meta, err := a.MainSession()
	if err != nil || strings.TrimSpace(meta.ID) == "" {
		return
	}
	a.deliverSpeaks(meta.ID, *snap.Initiative, now)
}

// deliverSpeaks 把 b 里该开口的写进 sessionID，再把真正写进去的记成已经说过。返回写进去的条目。
// 工单走交接，欢迎信走简报，都不在这里重复。写不进去的留到下一轮，不会被记成说过。
func (a *App) deliverSpeaks(sessionID string, b initiative.Briefing, now time.Time) []initiative.Signal {
	store, err := a.deskBook()
	if err != nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	day := b.Day
	if day == "" {
		day = now.Format("2006-01-02")
	}
	var sent []initiative.Signal
	for _, signal := range b.Signals {
		if !signal.Speak || signal.Class == "ticket" || signal.Code == "welcome" {
			continue
		}
		note := speakNote(sessionID, day, signal)
		if note.Text == "" {
			continue
		}
		if a.DeskPush(note) {
			sent = append(sent, signal)
		}
	}
	if len(sent) > 0 {
		store.Deliver(sent, now)
	}
	return sent
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
