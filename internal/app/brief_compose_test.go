package app

import (
	"strings"
	"testing"
)

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
