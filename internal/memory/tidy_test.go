// 本文件锁住整理建议的规则：只挑相近却没自动合并的对；模型的判断只是建议，人点了才改；点「不对」以后不再问；攒够新记忆才再问。

package memory

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// tidyFixture 记下两条相近的偏好（不会自动合并）和一条经历，返回早的那条和晚的那条。
func tidyFixture(t *testing.T, now *time.Time) (*Store, Item, Item) {
	t.Helper()
	s := openAt(t, now)
	old := s.Keep(Item{Kind: KindSemantic, Text: "代码注释用中文写", Source: "user"}, "").Item
	*now = now.Add(time.Hour)
	kept := s.Keep(Item{Kind: KindSemantic, Text: "代码注释用英文写", Source: "user"}, "")
	if kept.Action != "added" {
		t.Fatalf("the two preferences must stay apart for the person to judge: %+v", kept)
	}
	s.Keep(Item{Kind: KindEpisodic, Text: "代码注释用中文写完了一版", Source: "worker"}, "")
	return s, old, kept.Item
}

func TestTidyPairsFindsNearButUnmergedPairs(t *testing.T) {
	now := friday
	s, old, nw := tidyFixture(t, &now)
	pairs := s.TidyPairs(5)
	if len(pairs) != 1 || pairs[0].Old.ID != old.ID || pairs[0].New.ID != nw.ID {
		t.Fatalf("pairs = %+v", pairs)
	}
	if s.TidyPairs(0) != nil {
		t.Fatal("limit 0 asks for nothing")
	}
}

func TestTidySuggestionWaitsForThePerson(t *testing.T) {
	now := friday
	s, old, nw := tidyFixture(t, &now)
	pairs := s.TidyPairs(5)
	added := s.Suggest(pairs, []Verdict{{Pair: pairs[0], Verdict: VerdictChange}})
	if len(added) != 1 || added[0].Verdict != VerdictChange {
		t.Fatalf("suggestions = %+v", added)
	}
	if n := len(s.All()); n != 3 {
		t.Fatalf("a suggestion must not touch memory; items = %d", n)
	}
	if got := s.Suggestions(); len(got) != 1 || got[0].ID != added[0].ID {
		t.Fatalf("pending = %+v", got)
	}
	if s.TidyPairs(5) != nil {
		t.Fatal("do not ask again while a suggestion is waiting")
	}

	now = now.Add(time.Hour)
	merged, err := s.Settle(added[0].ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if merged.ID != old.ID || merged.Text != nw.Text || merged.Source != "user" || merged.Seen != 2 || !merged.CreatedAt.Equal(old.CreatedAt) {
		t.Fatalf("merged = %+v", merged)
	}
	for _, it := range s.All() {
		if it.ID == nw.ID {
			t.Fatal("the later item is folded into the earlier one")
		}
	}
	if len(s.Suggestions()) != 0 {
		t.Fatal("a settled suggestion is gone")
	}
	if _, err := s.Settle(added[0].ID, true); !errors.Is(err, ErrTidyGone) {
		t.Fatalf("settling twice = %v", err)
	}
	back, ok := s.Revert(old.ID)
	if !ok || back.Text != old.Text {
		t.Fatalf("one undo goes back to the earlier wording: %+v %v", back, ok)
	}
}

func TestTidySameUsesTheMergedWording(t *testing.T) {
	now := friday
	s, old, _ := tidyFixture(t, &now)
	pairs := s.TidyPairs(5)
	added := s.Suggest(pairs, []Verdict{{Pair: pairs[0], Verdict: VerdictSame, Text: "代码注释中英文都行"}})
	if len(added) != 1 || added[0].Text != "代码注释中英文都行" {
		t.Fatalf("suggestions = %+v", added)
	}
	merged, err := s.Settle(added[0].ID, true)
	if err != nil || merged.ID != old.ID || merged.Text != "代码注释中英文都行" || len(merged.History) != 2 {
		t.Fatalf("merged = %+v %v", merged, err)
	}
	if merged.History[len(merged.History)-1].Text != old.Text {
		t.Fatalf("the earlier wording is the last undo step: %+v", merged.History)
	}
}

func TestTidyNoMeansNeverAskAgain(t *testing.T) {
	now := friday
	s, _, _ := tidyFixture(t, &now)
	pairs := s.TidyPairs(5)
	added := s.Suggest(pairs, []Verdict{{Pair: pairs[0], Verdict: VerdictSame}})
	if _, err := s.Settle(added[0].ID, false); err != nil {
		t.Fatal(err)
	}
	if n := len(s.All()); n != 3 {
		t.Fatalf("declining changes nothing; items = %d", n)
	}
	if s.TidyPairs(5) != nil {
		t.Fatal("right after asking there are not enough new memories to ask again")
	}
	for i := 0; i < tidyEvery; i++ {
		now = now.Add(time.Minute)
		s.Keep(Item{Kind: KindEpisodic, Text: fmt.Sprintf("第 %d 张工单收尾", i), Source: "worker"}, "")
	}
	if got := s.TidyPairs(5); len(got) != 0 {
		t.Fatalf("a declined pair is never asked again: %+v", got)
	}
}

func TestTidyDifferentIsRememberedWithoutAsking(t *testing.T) {
	now := friday
	s, _, _ := tidyFixture(t, &now)
	pairs := s.TidyPairs(5)
	if added := s.Suggest(pairs, []Verdict{{Pair: pairs[0], Verdict: VerdictDifferent}}); len(added) != 0 {
		t.Fatalf("different is not a suggestion: %+v", added)
	}
	if len(s.Suggestions()) != 0 {
		t.Fatal("nothing waits for the person")
	}
	for i := 0; i < tidyEvery; i++ {
		now = now.Add(time.Minute)
		s.Keep(Item{Kind: KindEpisodic, Text: fmt.Sprintf("第 %d 次周会", i), Source: "worker"}, "")
	}
	if got := s.TidyPairs(5); len(got) != 0 {
		t.Fatalf("a pair judged different is not asked again: %+v", got)
	}
}

func TestTidyIgnoresStaleVerdictsAndLaterEdits(t *testing.T) {
	now := friday
	s, old, nw := tidyFixture(t, &now)
	pairs := s.TidyPairs(5)
	stale := pairs[0]
	stale.New.Text = "别的说法"
	if added := s.Suggest(pairs, []Verdict{{Pair: stale, Verdict: VerdictChange}}); len(added) != 0 {
		t.Fatalf("a verdict on wording that is no longer there is dropped: %+v", added)
	}

	now = now.Add(time.Hour)
	for i := 0; i < tidyEvery; i++ {
		now = now.Add(time.Minute)
		s.Keep(Item{Kind: KindEpisodic, Text: fmt.Sprintf("第 %d 次复盘", i), Source: "worker"}, "")
	}
	pairs = s.TidyPairs(5)
	if len(pairs) != 1 {
		t.Fatalf("the pair is asked again once enough is new: %+v", pairs)
	}
	added := s.Suggest(pairs, []Verdict{{Pair: pairs[0], Verdict: VerdictChange}})
	if len(added) != 1 {
		t.Fatalf("suggestions = %+v", added)
	}
	if _, ok := s.Edit(nw.ID, "代码注释一律用英文写"); !ok {
		t.Fatal("edit failed")
	}
	if _, err := s.Settle(added[0].ID, true); !errors.Is(err, ErrTidyChanged) {
		t.Fatalf("an edited pair is not merged: %v", err)
	}
	for _, it := range s.All() {
		if it.ID == old.ID && it.Text != old.Text {
			t.Fatalf("nothing changed: %+v", it)
		}
	}
}
