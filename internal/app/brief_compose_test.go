package app

import "testing"

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
	if !briefPacketWorth("最近说过\n本人 帮我看 golang 岗位\n") {
		t.Fatal("talk")
	}
}
