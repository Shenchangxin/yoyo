package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

func TestParseTodayBriefTellsQuietFromBroken(t *testing.T) {
	if cards, ok := parseTodayBrief(`今天没什么：{"cards":[]}`); !ok || len(cards) != 0 {
		t.Fatalf("quiet = %v %v", cards, ok)
	}
	if _, ok := parseTodayBrief("模型没写 JSON"); ok {
		t.Fatal("prose is not a brief")
	}
	noisy := "先想一下 {\"step\":1}\n```json\n{\"cards\":[{\"id\":\"a\",\"title\":\"先回 Ada\",\"refs\":[\"tkt-3\",\"[d1]\",\"zz\"]}]}\n```\n还有 {}"
	cards, ok := parseTodayBrief(noisy)
	if !ok || len(cards) != 1 || len(cards[0].Refs) != 3 {
		t.Fatalf("noisy = %+v %v", cards, ok)
	}
	if cards, ok := parseTodayBrief(`前言 {"x":{"y":1}} 正文 {"cards":[{"id":"b","title":"花括号}在字符串里"}]}`); !ok || len(cards) != 1 || cards[0].Title != "花括号}在字符串里" {
		t.Fatalf("balanced = %+v %v", cards, ok)
	}
	var many []string
	for i := 0; i < 9; i++ {
		many = append(many, `{"id":"c`+string(rune('0'+i))+`","title":"t"}`)
	}
	if cards, _ := parseTodayBrief(`{"cards":[` + strings.Join(many, ",") + `]}`); len(cards) != briefMaxCards {
		t.Fatalf("cap = %d", len(cards))
	}
}

func TestGroundRefsKeepsOnlyMaterialThatExists(t *testing.T) {
	m := &briefMaterial{}
	ticket := m.mark("", "ticket", "写周报\n细节", "tkt-3")
	date := m.mark("d", "date", "评审会", "calendar:acct:1")
	if ticket != "[tkt-3]" || date != "[d1]" {
		t.Fatalf("marks %s %s", ticket, date)
	}
	cards := groundRefs([]BriefCard{{ID: "a", Title: "x", Refs: []BriefRef{{ID: "tkt-3"}, {ID: "[d1]"}, {ID: "tkt-3"}, {ID: "nope"}}}}, m.refs)
	refs := cards[0].Refs
	if len(refs) != 2 || refs[0].Kind != "ticket" || refs[0].Target != "tkt-3" || refs[0].Label != "写周报" || refs[1].Kind != "date" || refs[1].Target != "" {
		t.Fatalf("refs = %+v", refs)
	}
}

func TestBriefSkipsWhatTheSystemPushed(t *testing.T) {
	for _, ev := range []trace.Event{
		{Type: trace.TypeAssistant, Source: "desk", Payload: map[string]any{"kind": "brief", "text": "早上这一份在这里。"}},
		{Type: trace.TypeAssistant, Source: "worker", Payload: map[string]any{"kind": "handoff", "text": "做完了"}},
		{Type: trace.TypeAssistant, Payload: map[string]any{"kind": "speak", "text": "今天到期"}},
	} {
		if !pushedBySystem(ev) {
			t.Fatalf("%+v should be skipped", ev)
		}
	}
	if pushedBySystem(trace.Event{Type: trace.TypeUser, Payload: map[string]any{"text": "帮我看看岗位"}}) {
		t.Fatal("what the person said stays")
	}
	var b strings.Builder
	writeMorningSaid(&b, []BriefRecord{
		{Slot: "morning", Day: "2026-11-05", Brief: `{"cards":[{"title":"昨天的"}]}`},
		{Slot: "morning", Day: "2026-11-06", Brief: `{"cards":[{"title":"先回 Ada"},{"title":"评审会前看稿"}]}`},
	}, "2026-11-06")
	if got := b.String(); !strings.Contains(got, "先回 Ada") || strings.Contains(got, "昨天的") {
		t.Fatalf("morning said = %q", got)
	}
}

func TestQuietBriefIsKeptButNotPushed(t *testing.T) {
	dir := t.TempDir()
	a := &App{Home: &home.Dir{Root: dir}, Traces: trace.NewStore(filepath.Join(dir, "sessions"))}
	note := DeskNote{SessionID: "main", Kind: "brief", Slot: "morning", Day: "2026-11-06", Text: "今天没有要先处理的事。", Brief: `{"cards":[{"id":"quiet","title":"今天没有必须先确认的事。"}]}`, Quiet: true}
	if !a.DeskPush(note) {
		t.Fatal("quiet brief should be sealed")
	}
	if evs, _ := a.Traces.Read("main"); len(evs) != 0 {
		t.Fatalf("a quiet brief must not put a chip in the session: %d", len(evs))
	}
	if list := a.BriefList(); len(list) != 1 || list[0].Slot != "morning" {
		t.Fatalf("ledger = %+v", list)
	}
}

func TestParseTodayCardsKeepsGroundedShape(t *testing.T) {
	raw := "```json\n{\"cards\":[{\"id\":\"job\",\"title\":\"简历还差目标岗位\",\"lede\":\"你刚问过投递\",\"layout\":\"evidence\",\"items\":[{\"title\":\"缺口\",\"detail\":\"方向和城市还没定\"}],\"primary\":\"补上再筛\",\"task\":\"按我上次说的找工作，先列还缺的条件\"},{\"id\":\"\",\"title\":\"丢掉\"}]}\n```"
	cards := parseTodayCards(raw)
	if len(cards) != 1 || cards[0].ID != "job" || cards[0].Task == "" || cards[0].Layout != "evidence" {
		t.Fatalf("%+v", cards)
	}
}

func TestBriefPacketWorthIgnoresMachineOnly(t *testing.T) {
	if briefPacketWorth("称呼 Ada\n这台机器 desk Windows\n工作时间 19:30–21:58\n") {
		t.Fatal("machine")
	}
	if briefPacketWorth("感知\n- 电量 72 充电中\n") {
		t.Fatal("battery")
	}
	if !briefPacketWorth("最近说过\n本人 帮我看 golang 岗位\n") {
		t.Fatal("talk")
	}
	if !briefPacketWorth("桌面清点\n- 工程线 yoyo 最近有改\n") {
		t.Fatal("census")
	}
}

func TestParseTodayCardsKeepsEveryCard(t *testing.T) {
	var cards []string
	for i := 0; i < 6; i++ {
		cards = append(cards, `{"id":"c`+string(rune('0'+i))+`","title":"第`+string(rune('0'+i))+`张","layout":"plain"}`)
	}
	got := parseTodayCards(`{"cards":[` + strings.Join(cards, ",") + `]}`)
	if len(got) != 6 {
		t.Fatalf("len %d", len(got))
	}
}

func TestParseTodayCardsKeepsLetterStatsAndNumbers(t *testing.T) {
	raw := `{"cards":[{"id":"gap","title":"手上有的和还没有的","layout":"Letter","leftLabel":"有","rightLabel":"没有","items":[{"title":"概念","detail":"已起稿"}],"choices":[{"id":"city","title":"城市","detail":"还没定"}]},{"id":"n","title":"数得出来","layout":"stats","stats":[{"label":"工程","value":3},{"label":"空","value":""}]},{"id":"odd","title":"交给界面","layout":"poster"}]}`
	got := parseTodayCards(raw)
	if len(got) != 3 {
		t.Fatalf("%+v", got)
	}
	if got[0].Layout != "letter" || got[0].LeftLabel != "有" || len(got[0].Choices) != 1 {
		t.Fatalf("letter %+v", got[0])
	}
	if got[1].Layout != "stats" || len(got[1].Stats) != 1 || got[1].Stats[0].Value != "3" {
		t.Fatalf("stats %+v", got[1])
	}
	if got[2].Layout != "" {
		t.Fatalf("layout %q", got[2].Layout)
	}
}
