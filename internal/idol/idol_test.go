// 本文件锁住双语拆句、语气提示、好感冷却、记忆盖过、一轮对话的工作标记，以及抽卡送礼。
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

// TestStripCue 锁住方括号语气提示会从 zh 台词里去掉。
func TestStripCue(t *testing.T) {
	got := ParseReply("<zh>[happily] 在的。</zh>", "zh", "zh")
	if len(got.Segments) != 1 || got.Segments[0].Spoken != "在的。" {
		t.Fatalf("%+v", got.Segments)
	}
}

// TestTier 锁住 0 是 cold、50 是 trust、80 是 intimate，以及一次 -40 会落到 10 并进入冷却。
func TestTier(t *testing.T) {
	if TierOf(0) != TierCold || TierOf(50) != TierTrust || TierOf(80) != TierIntimate {
		t.Fatalf("tiers")
	}
	st := &State{}
	a := st.ApplyDelta("a", -40, time.Now(), 0)
	if a.Score != 10 || !a.Cooling(time.Now()) {
		t.Fatalf("cooldown %+v", a)
	}
}

// TestMemorySupersedeAndCircle 锁住后一句盖过前一句，同圈子的另一位偶像能看见那条记忆。
func TestMemorySupersedeAndCircle(t *testing.T) {
	st := &State{}
	idol := Idol{ID: "m", Name: "M", CircleID: "c"}
	st.remember(idol, "用户养了一只猫", time.Now())
	st.remember(idol, "用户养了一只猫，叫小白", time.Now())
	live := 0
	for _, m := range st.Memories {
		if m.SupersededBy == "" {
			live++
		}
	}
	if live != 1 {
		t.Fatalf("live memories %d", live)
	}
	other := Idol{ID: "k", Name: "K", CircleID: "c"}
	st.Idols = []Idol{idol, other}
	seen := st.visibleMemories(other)
	if len(seen) != 1 {
		t.Fatalf("shared %d", len(seen))
	}
}

// TestTurnAndWorkFlag 锁住工作通道打开时 Turn 标上 Work，并留下一条记忆。
func TestTurnAndWorkFlag(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := st.SaveIdol(Idol{Name: "小灯", SpokenLang: "zh", SubtitleLang: "zh", Sheet: "短句。", WorkEnabled: true}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	eng := &Engine{Store: st, Brain: fakeBrain{}}
	res, err := eng.Turn(context.Background(), TurnInput{IdolID: saved.ID, Text: "帮我把桌面上的文件整理一下", Work: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Work || res.Speak == "" || res.RecordID == "" {
		t.Fatalf("%+v", res)
	}
	hist := st.View().History(saved.ID)
	if hist.Current == nil || len(hist.Current.Records) != 1 {
		t.Fatalf("history %+v", hist.Current)
	}
	mems := st.View().MemoriesFor(saved.ID)
	if len(mems) != 1 || !strings.Contains(mems[0].Text, "整理") {
		t.Fatalf("mem %+v", mems)
	}
}

// TestDrawAndGift 锁住有抽数时 Draw 给出礼物或重置卡，送出喜欢的礼物后分数不低于 50。
func TestDrawAndGift(t *testing.T) {
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
	if res.Gift != nil {
		_, aff, err := st.Give("a", res.Gift.ID, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if aff.Score < 50 {
			t.Fatalf("score %d", aff.Score)
		}
	}
}

// fakeBrain 按 kind 返回固定文本，供 Turn 测试，不连真正的模型。
type fakeBrain struct{}

// Chat 在 judge 时返回带记忆和工作标记的 JSON，title 时返回「整理文件」，其余返回一句中文。
func (fakeBrain) Chat(_ context.Context, kind, _ string, _ []Msg) (string, error) {
	switch kind {
	case "judge":
		return `{"affinity":1,"memory":"用户想整理文件","emotion":"calm","work":true}`, nil
	case "title":
		return "整理文件", nil
	default:
		return "<zh>我去办。</zh>", nil
	}
}

// Paint 固定返回错误，让这一轮不保存照片。
func (fakeBrain) Paint(context.Context, string) ([]byte, error) { return nil, errf("no") }
