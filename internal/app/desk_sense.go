package app

import (
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/poi"
)

// assessDesk 把这一轮清点、编排台和本机状态收成要不要开口的判断。记录写不成时 ok 为假，调用方就不要假装已经判断过。
func (a *App) assessDesk(snap poi.Snapshot) (initiative.Briefing, bool) {
	if a == nil || a.Home == nil {
		return initiative.Briefing{}, false
	}
	store, err := initiative.Open(a.Home.Memory())
	if err != nil {
		return initiative.Briefing{}, false
	}
	facts, idle := initiative.LocalFacts(poi.DefaultRoots())
	facts = append(facts, a.reachFacts(store.Seen(), time.Now())...)
	facts = append(facts, lineFacts(snap.Census)...)
	facts = append(facts, poiFacts(snap)...)
	facts = append(facts, a.connectorFacts()...)
	facts = append(facts, a.orchestratorFacts(time.Now())...)
	return store.Assess(initiative.Input{Now: time.Now(), Idle: idle, Facts: facts}), true
}

// DeskFeedback 记下人对某一类开口的反应。wrong 会忘掉对应的记忆。
// className 是这一类的名字。kind 取 used、ignore、later、wrong。memoryID 只在 wrong 时使用。
func (a *App) DeskFeedback(className, kind, memoryID string) bool {
	if a == nil || a.Home == nil {
		return false
	}
	store, err := initiative.Open(a.Home.Memory())
	if err != nil {
		return false
	}
	forget := store.Feedback(strings.TrimSpace(className), strings.TrimSpace(memoryID), strings.TrimSpace(kind), time.Now().Format("2006-01-02"))
	if forget != "" {
		a.MemoryForget(forget)
	}
	return true
}

func (a *App) noteSchedule(id, title string) {
	if a == nil || a.Home == nil {
		return
	}
	store, err := initiative.Open(a.Home.Memory())
	if err != nil {
		return
	}
	store.NoteSchedule(id, title, time.Now().Format("2006-01-02"))
}

func (a *App) scheduleAllowed() bool {
	if a == nil {
		return true
	}
	a.deskMu.Lock()
	rt := a.orch
	a.deskMu.Unlock()
	if rt == nil {
		return true
	}
	return initiative.AllowSchedule(rt.View().Mode)
}

func lineFacts(c poi.Census) []initiative.Fact {
	var facts []initiative.Fact
	for _, line := range c.Lines {
		name := strings.TrimSpace(line.Name)
		if name == "" || scrubSense(name) == "" {
			continue
		}
		evidence := scrubSense(line.Evidence)
		facts = append(facts, initiative.Fact{
			ID: "line:" + strings.ToLower(name), Code: "line", Class: "line", Name: name,
			Detail: evidence, State: line.State, Sig: line.State + "|" + evidence, Rank: 5, Eligible: true,
		})
	}
	return facts
}

func poiFacts(snap poi.Snapshot) []initiative.Fact {
	var facts []initiative.Fact
	for _, name := range append(append([]string{}, snap.Added...), snap.Dropped...) {
		name = strings.TrimSpace(name)
		if name == "" || scrubSense(name) == "" || len(facts) >= 6 {
			continue
		}
		facts = append(facts, initiative.Fact{
			ID: "poi:" + strings.ToLower(name), Code: "poi", Class: "poi", Name: name,
			State: "changed", Sig: name, Rank: 7,
		})
	}
	return facts
}

func (a *App) connectorFacts() []initiative.Fact {
	mail, calendar := false, false
	if a != nil && a.Connectors != nil {
		for _, acct := range a.Connectors.List() {
			switch acct.Kind {
			case connector.KindMail:
				mail = true
			case connector.KindCalendar:
				calendar = true
			}
		}
	}
	return []initiative.Fact{
		linkFact("mail", mail),
		linkFact("calendar", calendar),
	}
}

func linkFact(code string, linked bool) initiative.Fact {
	state := "unchecked"
	if linked {
		state = "linked"
	}
	return initiative.Fact{ID: code, Code: code, Class: code, Name: code, State: state, Sig: state, Rank: 6}
}

func (a *App) orchestratorFacts(now time.Time) []initiative.Fact {
	if a == nil {
		return nil
	}
	a.deskMu.Lock()
	rt := a.orch
	a.deskMu.Unlock()
	if rt == nil {
		return nil
	}
	view := rt.View()
	var facts []initiative.Fact
	for _, row := range view.Queued {
		if row.Source != "calendar" || strings.TrimSpace(row.Due) == "" {
			continue
		}
		delta, ok := dayDelta(row.Due, now)
		if !ok || delta > 14 || delta < -14 {
			continue
		}
		state := "soon"
		eligible := false
		rank := 2
		if delta < 0 {
			state = "overdue"
			eligible = true
			rank = 0
		} else if delta == 0 {
			state = "due"
			eligible = true
			rank = 1
		} else {
			continue
		}
		name := scrubSense(row.Text)
		if name == "" {
			name = row.ID
		}
		facts = append(facts, initiative.Fact{
			ID: "cal:" + row.ID, Code: "calendar", Class: "calendar", Name: name,
			State: state, Sig: state + "|" + row.Due, Rank: rank, Eligible: eligible,
		})
	}
	for _, ticket := range view.Tickets {
		if ticket.Stage == "ended" {
			continue
		}
		state := ""
		rank := 3
		if ticket.Person {
			state = "waiting"
			rank = 2
		} else if ticket.Action == "blocked" || ticket.Action == "failed" {
			state = ticket.Action
		}
		if state == "" {
			continue
		}
		name := scrubSense(ticket.Title)
		if name == "" {
			name = ticket.ID
		}
		facts = append(facts, initiative.Fact{
			ID: "ticket:" + ticket.ID, Code: "ticket", Class: "ticket", Name: name,
			State: state, Sig: state, Rank: rank, Eligible: true,
		})
	}
	for _, src := range view.Sources {
		name := scrubSense(src.Name)
		if name == "" {
			continue
		}
		detail := scrubSense(src.Detail)
		state := src.State
		eligible := state == "error"
		if state != "error" && state != "ok" {
			continue
		}
		facts = append(facts, initiative.Fact{
			ID: "channel:" + strings.ToLower(name), Code: "channel", Class: "source", Name: name,
			Detail: detail, State: state, Sig: state, Rank: 3, Eligible: eligible,
		})
	}
	return facts
}

func dayDelta(due string, now time.Time) (int, bool) {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(due))
	if err != nil {
		t, err = time.ParseInLocation("2006-01-02", strings.TrimSpace(due), now.Location())
		if err != nil {
			return 0, false
		}
	}
	t = t.In(now.Location())
	a := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	b := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
	return int(b.Sub(a).Hours() / 24), true
}

func scrubSense(text string) string {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)
	for _, word := range []string{"secret", "token", "password", "cookie", "credential", "api_key", "apikey"} {
		if strings.Contains(lower, word) {
			return ""
		}
	}
	return text
}
