// 本文件锁住简报里的记忆整理：只认材料里给过的编号；建议卡写清以前和后来的说法、按钮是「对」「不对」；IM 里不放整理卡。

package app

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/memory"
)

func TestParseTidyKeepsOnlyKnownPairs(t *testing.T) {
	pairs := map[string]memory.Pair{
		"t1": {Old: memory.Item{ID: "a"}, New: memory.Item{ID: "b"}},
		"t2": {Old: memory.Item{ID: "c"}, New: memory.Item{ID: "d"}},
	}
	text := "好的\n```json\n" + `{"cards":[],"memory":[{"pair":"[t1]","verdict":"same","text":"合并"},{"pair":"t1","verdict":"change"},{"pair":"t9","verdict":"same"},{"pair":"t2","verdict":"different"}]}` + "\n```"
	got := parseTidy(text, pairs)
	if len(got) != 2 {
		t.Fatalf("verdicts = %+v", got)
	}
	if got[0].Pair.Old.ID != "a" || got[0].Verdict != "same" || got[0].Text != "合并" {
		t.Fatalf("first = %+v", got[0])
	}
	if got[1].Pair.Old.ID != "c" || got[1].Verdict != "different" {
		t.Fatalf("second = %+v", got[1])
	}
	if parseTidy(`{"cards":[]}`, pairs) != nil {
		t.Fatal("no memory array means no verdicts")
	}
}

func TestTidyCardAsksYesOrNo(t *testing.T) {
	same := tidyCard(memory.Suggestion{ID: "tidy-1", Old: "a", New: "b", OldText: "注释用中文", NewText: "注释写中文", Verdict: memory.VerdictSame, Text: "代码注释用中文"}, true)
	if same.ID != "tidy-1" || same.Tidy != "tidy-1" || same.Primary != "对" || same.Secondary != "不对" || same.Task != "" || same.Alt != "" {
		t.Fatalf("card = %+v", same)
	}
	if !strings.Contains(same.Lede, "代码注释用中文") || len(same.Items) != 2 || same.Items[0].Detail != "注释用中文" || same.Items[1].Detail != "注释写中文" {
		t.Fatalf("same card shows the merged wording and both originals: %+v", same)
	}
	if len(same.Refs) != 2 || same.Refs[0].Target != "a" || same.Refs[1].Target != "b" {
		t.Fatalf("refs open the two memories: %+v", same.Refs)
	}
	change := tidyCard(memory.Suggestion{ID: "tidy-2", OldText: "注释用中文", NewText: "注释用英文", Verdict: memory.VerdictChange}, false)
	if change.Primary != "Yes" || !strings.Contains(change.Title, "changed your mind") {
		t.Fatalf("change card = %+v", change)
	}
}

func TestIMBriefLeavesOutTidyCards(t *testing.T) {
	cards := []BriefCard{{ID: "c1", Title: "先交周报"}, tidyCard(memory.Suggestion{ID: "tidy-1", Verdict: memory.VerdictChange}, true)}
	if got := withoutTidy(cards); len(got) != 1 || got[0].ID != "c1" {
		t.Fatalf("im cards = %+v", got)
	}
	if text := briefIMText("晨间简报", withoutTidy(cards), true); strings.Contains(text, "改了主意") {
		t.Fatalf("im text = %q", text)
	}
}
