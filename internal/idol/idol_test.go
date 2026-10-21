package idol

import (
	"context"
	"math/rand/v2"
	"strings"
	"testing"
	"time"
)

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

func TestStripCue(t *testing.T) {
	got := ParseReply("<zh>[happily] 在的。</zh>", "zh", "zh")
	if len(got.Segments) != 1 || got.Segments[0].Spoken != "在的。" {
		t.Fatalf("%+v", got.Segments)
	}
}

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

type fakeBrain struct{}

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

func (fakeBrain) Paint(context.Context, string) ([]byte, error) { return nil, errf("no") }
