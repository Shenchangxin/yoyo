package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestDeliverSpeaksRecordsOnlyWhatReachedTheSession(t *testing.T) {
	dir := t.TempDir()
	a := &App{Home: &home.Dir{Root: dir}, Traces: trace.NewStore(filepath.Join(dir, "sessions"))}
	store, err := a.deskBook()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 6, 9, 30, 0, 0, time.Local)
	due := initiative.Fact{ID: "cal:1", Code: "calendar", Class: "calendar", Name: "评审", State: "due", Rank: 1, Eligible: true}
	waiting := initiative.Fact{ID: "ticket:tkt-1", Code: "ticket", Class: "ticket", Name: "写周报", State: "waiting", Rank: 2}
	brief := store.Assess(initiative.Input{Now: now, Facts: []initiative.Fact{due, waiting}})
	if got := a.deliverSpeaks("", brief, now); len(got) != 0 {
		t.Fatalf("no session, nothing sent: %#v", got)
	}
	again := store.Assess(initiative.Input{Now: now.Add(time.Minute), Facts: []initiative.Fact{due, waiting}})
	if !again.Signals[0].Speak {
		t.Fatalf("an unsent signal must still speak, %#v", again.Signals)
	}
	sent := a.deliverSpeaks("main", again, now.Add(time.Minute))
	if len(sent) != 1 || sent[0].ID != "cal:1" {
		t.Fatalf("sent = %#v", sent)
	}
	evs, err := a.Traces.Read("main")
	if err != nil || len(evs) != 1 {
		t.Fatalf("events %d %v", len(evs), err)
	}
	payload := evs[0].Payload
	if payload["kind"] != "speak" || payload["class"] != "calendar" || payload["why"] != "due" || payload["name"] != "评审" {
		t.Fatalf("payload = %#v", payload)
	}
	if text, _ := payload["text"].(string); text != "「评审」今天到期。" {
		t.Fatalf("text = %q", text)
	}
	later := store.Assess(initiative.Input{Now: now.Add(2 * time.Hour), Facts: []initiative.Fact{due, waiting}})
	if later.Speak {
		t.Fatalf("told today, stay quiet: %#v", later.Signals)
	}
	if got := a.deliverSpeaks("main", later, now.Add(2*time.Hour)); len(got) != 0 {
		t.Fatalf("nothing new to send: %#v", got)
	}
	if evs, _ := a.Traces.Read("main"); len(evs) != 1 {
		t.Fatalf("a second note was written: %d", len(evs))
	}
}

func TestDeliverSpeaksPushesHeldReachIntoTheSession(t *testing.T) {
	dir := t.TempDir()
	a := &App{Home: &home.Dir{Root: dir}, Traces: trace.NewStore(filepath.Join(dir, "sessions"))}
	store, err := a.deskBook()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 23, 22, 0, 0, 0, time.Local)
	item := initiative.Fact{
		ID: "reach:a", Code: "reach", Class: "reach", Name: "Hebbia",
		Detail: "金融研究\nhttps://example.com/hebbia", State: "match", Sig: "https://example.com/hebbia", Eligible: true,
	}
	held := store.Assess(initiative.Input{Now: now, Facts: []initiative.Fact{item}, ReachHold: true, ReachDaily: 12, ReachEvery: time.Minute})
	if held.Speak || len(held.Signals) != 1 || !held.Signals[0].Held {
		t.Fatalf("held = %#v", held)
	}
	sent := a.deliverSpeaks("main", held, now)
	if len(sent) != 1 || sent[0].ID != "reach:a" || !sent[0].Speak {
		t.Fatalf("sent = %#v", sent)
	}
	evs, err := a.Traces.Read("main")
	if err != nil || len(evs) != 1 {
		t.Fatalf("events %d %v", len(evs), err)
	}
	text, _ := evs[0].Payload["text"].(string)
	if !strings.Contains(text, "https://example.com/hebbia") {
		t.Fatalf("text = %q", text)
	}
	again := store.Assess(initiative.Input{Now: now.Add(time.Minute), Facts: []initiative.Fact{item}, ReachHold: true, ReachDaily: 12, ReachEvery: time.Minute})
	if again.Speak || briefingDelivers(again) {
		t.Fatalf("already delivered, stay quiet: %#v", again.Signals)
	}
}

func TestDeliverSpeaksSkipsMutedAndCappedReach(t *testing.T) {
	dir := t.TempDir()
	a := &App{Home: &home.Dir{Root: dir}, Traces: trace.NewStore(filepath.Join(dir, "sessions"))}
	store, err := a.deskBook()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 23, 22, 0, 0, 0, time.Local)
	if !a.DeskFeedback("reach", "mute", "") {
		t.Fatal("mute")
	}
	muted := initiative.Fact{
		ID: "reach:m", Code: "reach", Class: "reach", Name: "静音",
		Detail: "https://example.com/muted", State: "match", Sig: "https://example.com/muted", Eligible: true,
	}
	held := store.Assess(initiative.Input{Now: now, Facts: []initiative.Fact{muted}, ReachHold: true, ReachDaily: 12, ReachEvery: time.Minute})
	if got := a.deliverSpeaks("main", held, now); len(got) != 0 {
		t.Fatalf("muted reach was sent: %#v", got)
	}
	if !a.DeskFeedback("reach", "unmute", "") {
		t.Fatal("unmute")
	}
	var prior []initiative.Signal
	for i := 0; i < 12; i++ {
		prior = append(prior, initiative.Signal{
			ID: "reach:old" + string(rune('a'+i)), Code: "reach", Class: "reach", Speak: true, Sig: "https://example.com/old",
		})
	}
	store.Deliver(prior, now)
	extra := initiative.Fact{
		ID: "reach:c", Code: "reach", Class: "reach", Name: "超额",
		Detail: "https://example.com/capped", State: "match", Sig: "https://example.com/capped", Eligible: true,
	}
	capped := store.Assess(initiative.Input{Now: now.Add(time.Hour), Facts: []initiative.Fact{extra}, ReachHold: true, ReachDaily: 12, ReachEvery: time.Minute})
	if capped.Speak || len(capped.Signals) != 1 || !capped.Signals[0].Held {
		t.Fatalf("capped = %#v", capped)
	}
	if got := a.deliverSpeaks("main", capped, now.Add(time.Hour)); len(got) != 0 {
		t.Fatalf("over the daily cap: %#v", got)
	}
	if evs, _ := a.Traces.Read("main"); len(evs) != 0 {
		t.Fatalf("notes written: %d", len(evs))
	}
}

func TestDeskFeedbackMutesAndListsTheClass(t *testing.T) {
	dir := t.TempDir()
	a := &App{Home: &home.Dir{Root: dir}}
	if a.DeskFeedback("reach", "bogus", "") {
		t.Fatal("unknown kinds are refused")
	}
	if !a.DeskFeedback("reach", "mute", "") {
		t.Fatal("mute")
	}
	a.deskOnce.Do(func() {})
	state := a.DeskState()
	if len(state.Mutes) != 1 || state.Mutes[0].Class != "reach" || state.Mutes[0].Until != initiative.Forever {
		t.Fatalf("mutes = %#v", state.Mutes)
	}
	if !a.DeskFeedback("reach", "unmute", "") {
		t.Fatal("unmute")
	}
	if state := a.DeskState(); len(state.Mutes) != 0 {
		t.Fatalf("still muted = %#v", state.Mutes)
	}
}

func TestSpeakTextUsesOnlyTheSignal(t *testing.T) {
	cases := []struct {
		signal initiative.Signal
		want   string
	}{
		{initiative.Signal{Code: "calendar", Name: "报销", State: "overdue"}, "「报销」已经过了那一天。"},
		{initiative.Signal{Code: "line", Name: "yoyo", State: "stalled", Detail: "blank.txt"}, "「yoyo」跟上一次看见的时候不一样了：停在一段空文件，blank.txt。"},
		{initiative.Signal{Code: "channel", Name: "GitHub", State: "error"}, "GitHub 处于错误，从这里进来的工作会漏。"},
		{initiative.Signal{Code: "reach", Name: "标题", Detail: "便条\nhttps://example.com"}, "便条\nhttps://example.com"},
		{initiative.Signal{Code: "net", Name: "network", State: "offline"}, "网络断了。"},
	}
	for _, c := range cases {
		if got := speakText(c.signal); got != c.want {
			t.Fatalf("%s: %q != %q", c.signal.Code, got, c.want)
		}
	}
}
