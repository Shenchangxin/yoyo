// 本文件锁住双语拆句、语气提示、流式分段、检索、好感、告别、记忆、抽卡送礼和搭话的规则。
// fakeBrain 只返回测试用的固定句子。由 go test 调用。

package idol

import (
	"context"
	"math/rand/v2"
	"strings"
	"testing"
	"time"
)

// TestParseBilingual 锁住 ja 口语、zh 字幕，以及 photo 标签不进台词。
func TestParseBilingual(t *testing.T) {
	got := ParseReply("<ja>こんにちは</ja><zh>你好</zh><photo>窗边</photo>", "ja", "zh")
	if len(got.Segments) != 1 || got.Segments[0].Spoken != "こんにちは" || got.Segments[0].Subtitle != "你好" {
		t.Fatalf("segments %+v", got.Segments)
	}
	if got.Photo != "窗边" {
		t.Fatalf("photo %q", got.Photo)
	}
	if SpeakLine(got.Segments) != "こんにちは" || CaptionLine(got.Segments) != "你好" {
		t.Fatalf("lines")
	}
}

// TestStripCue 锁住方括号语气提示从台词里去掉，但留在 Cue 里给语音和立绘。
func TestStripCue(t *testing.T) {
	got := ParseReply("<zh>[happily] 在的。</zh>", "zh", "zh")
	if len(got.Segments) != 1 || got.Segments[0].Spoken != "在的。" || got.Segments[0].Cue != "happily" {
		t.Fatalf("%+v", got.Segments)
	}
	if CueEmotion(got.Segments[0].Cue) != "happy" || CueEmotion("shyly") != "shy" || CueEmotion("nonsense") != "" {
		t.Fatal("cue emotion")
	}
}

// TestReplyStream 锁住流式读回复：写完一段就吐一段，双语时口语要等到字幕写完；吐出的段和最后读完的一致；被打断时不收半句。
func TestReplyStream(t *testing.T) {
	full := "<photo>海边</photo><ja>[happily] やった！</ja><zh>太好了！</zh><ja>行こう</ja><zh>走吧</zh>"
	r := NewReplyStream("ja", "zh")
	var got []Segment
	photo := ""
	for _, ch := range strings.Split(full, "") {
		for _, ev := range r.Feed(ch) {
			if ev.Photo != "" {
				photo = ev.Photo
			}
			if ev.Segment != nil {
				if ev.Index != len(got) {
					t.Fatalf("index %d after %d", ev.Index, len(got))
				}
				got = append(got, *ev.Segment)
			}
		}
	}
	if photo != "海边" || len(got) != 2 {
		t.Fatalf("photo %q segs %+v", photo, got)
	}
	final, rest := r.Finish(true)
	if len(rest) != 0 || len(final.Segments) != 2 || final.Segments[0] != got[0] || got[0].Cue != "happily" || got[0].Spoken != "やった！" {
		t.Fatalf("final %+v rest %+v", final.Segments, rest)
	}

	cut := NewReplyStream("ja", "zh")
	cut.Feed("<ja>一つ目</ja><zh>第一句</zh><ja>二つ")
	p, more := cut.Finish(false)
	if len(p.Segments) != 1 || len(more) != 0 {
		t.Fatalf("stopped %+v %+v", p.Segments, more)
	}
	same := NewReplyStream("zh", "zh")
	if evs := same.Feed("<zh>第一句</zh><zh>第二"); len(evs) != 1 || evs[0].Segment.Spoken != "第一句" {
		t.Fatalf("same-lang %+v", evs)
	}
}

// TestRankCJK 锁住中文按二元词检索：整句中文不再被当成一个词，相关的那条排在前面。
func TestRankCJK(t *testing.T) {
	docs := []string{"用户养了一只叫小白的猫", "用户在东京上班", "用户喜欢喝咖啡"}
	hits := rank("你家的猫还好吗？小白最近怎么样", docs)
	if len(hits) == 0 || hits[0].index != 0 || hits[0].score < memoryMinScore {
		t.Fatalf("hits %+v", hits)
	}
	if got := rank("今天天气", docs); len(got) != 0 && got[0].score >= memoryMinScore {
		t.Fatalf("unrelated scored %+v", got)
	}
	idol := Idol{ID: "a", Facts: strings.Repeat("填充文字。", 900) + "\n# 生日\n她的生日是三月十四日，喜欢草莓蛋糕。"}
	if lore := retrieve(idol, nil, "你的生日是哪天？"); !strings.Contains(lore, "三月十四日") {
		t.Fatalf("lore %q", lore)
	}
}

// TestAffinityRecoveryAndCooldown 锁住好感的数值：低于常值每天回 10，冷却从 0 开始、回到 10 结束，礼物在冷却中最多加 3 且不会续上冷却。
func TestAffinityRecoveryAndCooldown(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	st := &State{}
	if a := st.affinityOf("a", now); a.Score != 50 || a.Tier != TierTrust {
		t.Fatalf("default %+v", a)
	}
	st.setAffinity(Affinity{IdolID: "a", Score: 30, At: now})
	if a := st.affinityOf("a", now.Add(36*time.Hour)); a.Score != 45 {
		t.Fatalf("recovered %v", a.Score)
	}
	if a := st.affinityOf("a", now.Add(30*24*time.Hour)); a.Score != 50 {
		t.Fatalf("capped at normal %v", a.Score)
	}
	st.setAffinity(Affinity{IdolID: "b", Score: 4, At: now})
	j := Judged{Level: 0, OutOfLine: true}
	if !st.cools("b", j, now) {
		t.Fatal("should cool")
	}
	_, after := st.applyJudged("b", j, now)
	if after.Score != 0 || !after.InCooldown || after.CooldownUntil.Sub(now) != 24*time.Hour {
		t.Fatalf("cooldown %+v", after)
	}
	_, gifted := st.receiveGift("b", "rare", true, now)
	if gifted.Score != 3 || !gifted.InCooldown || !gifted.CooldownUntil.Before(after.CooldownUntil) {
		t.Fatalf("gift in cooldown %+v", gifted)
	}
	if !st.affinityOf("b", now.Add(16*time.Hour)).Cooling(now.Add(16 * time.Hour)) {
		t.Fatal("still cooling")
	}
	if st.affinityOf("b", now.Add(17*time.Hour)).InCooldown {
		t.Fatal("thawed at 10")
	}
}

// TestDailyGainCap 锁住一天里说话最多加 6 分，想操纵好感的那句不加分，礼物不受上限。
func TestDailyGainCap(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	st := &State{}
	for i := 0; i < 5; i++ {
		st.applyJudged("a", Judged{Level: 4}, now)
	}
	if a := st.affinityOf("a", now); a.Score != 56 {
		t.Fatalf("capped %v", a.Score)
	}
	if _, a := st.applyJudged("a", Judged{Level: 4}, now.Add(24*time.Hour)); a.Score != 58 {
		t.Fatalf("next day %v", a.Score)
	}
	if (Judged{Level: 4, Manipulation: true}).Step() != 0 || (Judged{Level: 0, Manipulation: true}).Step() != -5 {
		t.Fatal("manipulation")
	}
	if _, a := st.receiveGift("a", "rare", false, now.Add(24*time.Hour)); a.Score != 65 || a.Tier != TierClose {
		t.Fatalf("gift %+v", a)
	}
}

// TestEndsAndWarns 锁住她何时警告、何时告别：最近几轮扣到 6 分先警告一次，扣到 12 分告别；通话里第二次越界挂断；开始冷却时告别。
func TestEndsAndWarns(t *testing.T) {
	hurt := &Judged{Level: 0}
	if !warns(hurt, false, 0, []float64{2}) || warns(hurt, false, 0, []float64{6}) {
		t.Fatal("warns once on the way up")
	}
	if ends(hurt, false, 0, []float64{5, 2}) != "fed_up" || ends(hurt, false, 0, []float64{2}) != "" {
		t.Fatal("fed up at 12")
	}
	rude := &Judged{Level: 0, OutOfLine: true}
	if ends(rude, false, 1, nil) != "out_of_line" || ends(rude, false, 0, nil) != "" {
		t.Fatal("out of line twice")
	}
	if ends(&Judged{Level: 2}, true, 0, nil) != "cooldown" || ends(nil, true, 0, nil) != "" {
		t.Fatal("cooldown")
	}
	con := Conversation{Records: []Record{
		{Kind: "turn", Hurt: 5, OutOfLine: false},
		{Kind: "turn", Call: "c1", Hurt: 10, OutOfLine: true},
		{Kind: "turn", Hurt: 2},
	}}
	if out, hurts := tally(con, ""); out != 0 || len(hurts) != 2 {
		t.Fatalf("typed tally %d %v", out, hurts)
	}
	if out, hurts := tally(con, "c1"); out != 1 || len(hurts) != 1 {
		t.Fatalf("call tally %d %v", out, hurts)
	}
	lines := affinityPrompt(TierClose, &Judged{Level: 4}, "", "hangs_up")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "更近") || !strings.Contains(joined, "很开心") || !strings.Contains(joined, "挂电话") {
		t.Fatalf("prompt %q", joined)
	}
	if len(affinityPrompt(TierTrust, &Judged{Level: 2}, "", "")) != 0 {
		t.Fatal("trust with no effect says nothing")
	}
}

// TestJudgeReadsHerReachOut 锁住评判和记忆读到她先开口的那句：用户只回「考砸了」时，上文里要有她问的考试。热词带上她的名字、读音表写法和称呼。
func TestJudgeReadsHerReachOut(t *testing.T) {
	recs := []Record{
		{Kind: "turn", User: "我下周考试", Segments: []Segment{{Subtitle: "加油哦"}}},
		{Kind: "reach", Segments: []Segment{{Subtitle: "考试怎么样了？"}}},
	}
	p := judgePrompt(Idol{Name: "小灯"}, UserCard{}, recs, "考砸了", false, false, false)
	if !strings.Contains(p, "用户：我下周考试\n她：加油哦\n她：考试怎么样了？") {
		t.Fatalf("context %q", p)
	}
	if herLatest(recs) != "考试怎么样了？" {
		t.Fatal("her latest words")
	}
	mem := memoryPrompt(Idol{Name: "小灯"}, UserCard{}, recs, "考砸了", []Segment{{Subtitle: "没关系"}}, nil, time.Now())
	if !strings.Contains(mem[1].Content, "小灯：考试怎么样了？\n用户（最新的一句话）：考砸了") {
		t.Fatalf("memory context %q", mem[1].Content)
	}
	st := State{Idols: []Idol{{ID: "a", Name: "樱木真乃", Readings: "真乃=まの\n# 注释\nピーちゃん ぴーちゃん"}}, User: UserCard{Address: "制作人"}}
	if got := st.HearingHint("a"); got != "樱木真乃、真乃、ピーちゃん、制作人" {
		t.Fatalf("hint %q", got)
	}
}

// TestNowPrompt 锁住「此刻」写出星期、时区（没设时区时写 UTC 偏移而不是 Local）、上次说话隔了多久和生日。
func TestNowPrompt(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 30, 0, 0, time.Local)
	got := nowPrompt(UserCard{Birthday: now.Format("01-02")}, now, talkTyping, nil, now.Add(-26*time.Hour))
	for _, want := range []string{"星期", "UTC", "隔了 26 小时", "今天是用户的生日"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "Local") {
		t.Fatalf("zone shown as Local: %q", got)
	}
	call := nowPrompt(UserCard{Timezone: "Asia/Tokyo"}, now, talkCall, nil, time.Time{})
	if !strings.Contains(call, "Asia/Tokyo") || strings.Contains(call, "上次和用户说话") || !strings.Contains(call, "打电话") {
		t.Fatalf("call %q", call)
	}
}

// TestMigrateAffinity 锁住旧的整数好感换到新分档时各档相对位置不变。
func TestMigrateAffinity(t *testing.T) {
	now := time.Now()
	st := &State{Affinity: []Affinity{{IdolID: "a", Score: 62}, {IdolID: "b", Score: 80}, {IdolID: "c", Score: 5, CooldownUntil: now.Add(time.Hour)}}}
	st.migrateAffinity(now)
	if st.Affinity[0].Tier != TierClose || st.Affinity[1].Tier != TierIntimate || !st.Affinity[2].InCooldown || st.AffinityRev != affinityRevision {
		t.Fatalf("%+v", st.Affinity)
	}
}

// TestMemorySupersedeAndCircle 锁住新记忆按编号取代旧的、一样的不重写、删掉取代者时旧的重新成立，以及同圈子的偶像能看见。
func TestMemorySupersedeAndCircle(t *testing.T) {
	now := time.Now()
	st := &State{}
	mano := Idol{ID: "m", Name: "M", CircleID: "c"}
	st.addMemories(mano, []newMemory{{Text: "用户养了一只猫"}}, nil, now)
	shown := st.holding(mano, false)
	added := st.addMemories(mano, []newMemory{{Text: "用户养了一只叫小白的猫", Supersedes: []int{1}}, {Text: "用户养了一只叫小白的猫"}}, shown, now)
	if len(added) != 1 || len(st.holding(mano, false)) != 1 || st.holding(mano, false)[0].Text != "用户养了一只叫小白的猫" {
		t.Fatalf("holding %+v", st.Memories)
	}
	other := Idol{ID: "k", Name: "K", CircleID: "c"}
	st.Idols = []Idol{mano, other}
	if len(st.holding(other, false)) != 1 || len(st.holding(other, true)) != 0 {
		t.Fatal("circle share")
	}
	if !strings.Contains(formatMemories(st, other, st.holding(other, false)), "听M说的") {
		t.Fatal("hearsay label")
	}
	if !st.deleteMemory(added[0].ID) || len(st.holding(mano, false)) != 1 || st.holding(mano, false)[0].Text != "用户养了一只猫" {
		t.Fatalf("relink %+v", st.Memories)
	}
	if ws, ok := parseMemories("好的：{\"memories\":[{\"text\":\"用户明天考试\",\"supersedes\":[]}]}"); !ok || len(ws) != 1 {
		t.Fatalf("parse %+v", ws)
	}
}

// TestDrawAndGift 锁住有抽数时 Draw 给出礼物或重置卡，送出喜欢的礼物按稀有度加分并记下她喜欢；按天补抽奖次数最多攒 7 次。
func TestDrawAndGift(t *testing.T) {
	now := time.Now()
	st := &State{Draws: 1}
	st.Idols = []Idol{{ID: "a", Name: "A", LikedGifts: []string{"tea"}}}
	rng := rand.New(rand.NewPCG(1, 2))
	res, err := st.Draw(rng)
	if err != nil {
		t.Fatal(err)
	}
	if res.Gift == nil && !res.ResetCard {
		t.Fatal("empty draw")
	}
	st.addStock("tea", 1)
	rec, aff, change, err := st.Give("a", "tea", now)
	if err != nil {
		t.Fatal(err)
	}
	if aff.Score != 56 || !rec.Liked || rec.Delta != 6 || change == nil || !change.Up {
		t.Fatalf("gift %+v %+v %+v", rec, aff, change)
	}
	if !st.GiftUnanswered("a") {
		t.Fatal("gift waits for her answer")
	}
	day := &State{DrawDay: now.AddDate(0, 0, -3).Format("2006-01-02")}
	day.ensureDaily(now)
	if day.Draws != 3 {
		t.Fatalf("draws %d", day.Draws)
	}
	day.DrawDay = now.AddDate(0, 0, -30).Format("2006-01-02")
	day.ensureDaily(now)
	if day.Draws != maxDraws {
		t.Fatalf("capped draws %d", day.Draws)
	}
}

// TestReachSchedule 锁住她先开口的时间：信任档在上次说话后 20 到 36 小时之间，夜里不开口，开口过一次就等用户回话。
func TestReachSchedule(t *testing.T) {
	loc := time.UTC
	contact := time.Date(2026, 11, 5, 10, 0, 0, 0, loc)
	st := &State{Services: Services{ReachOut: true}, User: UserCard{Timezone: "UTC"}}
	idol := Idol{ID: "a", Name: "A"}
	st.Idols = []Idol{idol}
	st.Conversations = []Conversation{{ID: "c", IdolID: "a", Current: true, Records: []Record{{Kind: "turn", At: contact, User: "hi"}}}}
	if reachDue(st, idol, contact.Add(19*time.Hour)) {
		t.Fatal("too early")
	}
	if !reachDue(st, idol, time.Date(2026, 11, 7, 10, 0, 0, 0, loc)) {
		t.Fatal("due after 48h in the day")
	}
	if reachDue(st, idol, time.Date(2026, 11, 7, 23, 30, 0, 0, loc)) {
		t.Fatal("not at night")
	}
	st.Conversations[0].Records = append(st.Conversations[0].Records, Record{Kind: "reach", At: contact.Add(30 * time.Hour)})
	if reachDue(st, idol, time.Date(2026, 11, 7, 12, 0, 0, 0, loc)) {
		t.Fatal("waits for an answer")
	}
	if got := pastNight(st.User, time.Date(2026, 11, 6, 23, 0, 0, 0, loc), 0); got.Hour() != 8 || got.Day() != 7 {
		t.Fatalf("pushed to morning %v", got)
	}
}

// fakeBrain 按 kind 返回固定文本，供 Turn 测试，不连真正的模型。
type fakeBrain struct{}

// Chat 在 judge 时返回定级，memory 时返回一条记忆，title 时返回「整理文件」，其余返回一句中文。
func (fakeBrain) Chat(_ context.Context, kind, _ string, _ []Msg) (string, error) {
	switch kind {
	case "judge":
		return `{"level":3,"emotion":"happy","work":true}`, nil
	case "memory":
		return `{"memories":[{"text":"用户的桌面很乱","supersedes":[]}]}`, nil
	case "title":
		return "整理文件", nil
	default:
		return "<work>整理桌面上的文件</work><zh>我去办。</zh>", nil
	}
}

// Paint 固定返回错误，让这一轮不保存照片。
func (fakeBrain) Paint(context.Context, string) ([]byte, error) { return nil, errf("no") }

// TestTurnAndWorkFlag 锁住工作通道打开时 Turn 标上 Work 并带上她写的指令、好感上升，回合之后写下一条记忆。
func TestTurnAndWorkFlag(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	saved, err := st.SaveIdol(Idol{Name: "小灯", SpokenLang: "zh", SubtitleLang: "zh", Sheet: "短句。", WorkEnabled: true}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	eng := &Engine{Store: st, Brain: fakeBrain{}}
	res, err := eng.Turn(context.Background(), TurnInput{IdolID: saved.ID, Text: "帮我把桌面上的文件整理一下"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Work || res.WorkInstruction != "整理桌面上的文件" || res.Speak != "我去办。" || res.Change == nil || !res.Change.Up {
		t.Fatalf("%+v", res)
	}
	hist := st.View().History(saved.ID)
	if hist.Current == nil || len(hist.Current.Records) != 1 || hist.Current.Records[0].WorkText != "整理桌面上的文件" {
		t.Fatalf("history %+v", hist.Current)
	}
	mems := st.View().MemoriesFor(saved.ID)
	if len(mems) != 1 || !strings.Contains(mems[0].Text, "桌面") {
		t.Fatalf("mem %+v", mems)
	}
}
