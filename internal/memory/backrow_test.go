// 本文件锁住记忆的后排：很久没动的退到后排，不进画像、不召回，但还搜得到；当事人亲口说的不退；用到了回到前排；前排有上限。

package memory

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestColdMemoriesMoveToTheBackRow(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	said := s.Keep(Item{Kind: KindProfile, Text: "我住在杭州", Source: "user"}, "").Item
	noted := s.Keep(Item{Kind: KindSemantic, Text: "喜欢深色主题", Source: "worker"}, "").Item
	ep := s.Keep(Item{Kind: KindEpisodic, Text: "周报模板放在共享盘第三个文件夹", Source: "worker"}, "").Item
	if pin := s.ProfilePin(0); !strings.Contains(pin, said.Text) || !strings.Contains(pin, noted.Text) {
		t.Fatalf("fresh pin = %q", pin)
	}
	if got := s.Recall("周报模板放在哪个文件夹", 3); len(got) != 1 || got[0].ID != ep.ID {
		t.Fatalf("fresh recall = %+v", got)
	}

	now = friday.Add(coldAfter + 24*time.Hour)
	back := s.BackRow()
	if !back[noted.ID] || !back[ep.ID] || back[said.ID] {
		t.Fatalf("back row = %v", back)
	}
	pin := s.ProfilePin(0)
	if !strings.Contains(pin, said.Text) || strings.Contains(pin, noted.Text) {
		t.Fatalf("what the person said stays, cold notes leave the pin: %q", pin)
	}
	if got := s.Recall("周报模板放在哪个文件夹", 3); len(got) != 0 {
		t.Fatalf("the back row is not recalled: %+v", got)
	}
	if got := s.Search("周报模板", "", 5); len(got) != 1 || got[0].ID != ep.ID {
		t.Fatalf("the back row is still searchable: %+v", got)
	}

	if !s.MarkUsed([]string{ep.ID}) {
		t.Fatal("marking a use writes it down")
	}
	if s.MarkUsed([]string{ep.ID}) {
		t.Fatal("a second use on the same day is not written again")
	}
	if got := s.Recall("周报模板放在哪个文件夹", 3); len(got) != 1 {
		t.Fatalf("a used memory is back in front: %+v", got)
	}
	for _, it := range s.All() {
		if it.ID == ep.ID && !it.Touched().Equal(ep.Touched()) {
			t.Fatalf("a use is not an edit: %+v", it)
		}
	}
}

func TestFrontRowHasACap(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	var ids []string
	for i := 0; i < frontMax+3; i++ {
		now = now.Add(time.Minute)
		ids = append(ids, s.Keep(Item{Kind: KindEpisodic, Text: fmt.Sprintf("第 %d 张工单收尾 x%d", i, i), Source: "worker"}, "").Item.ID)
	}
	back := s.BackRow()
	if len(back) != 3 || !back[ids[0]] || !back[ids[1]] || !back[ids[2]] || back[ids[3]] {
		t.Fatalf("the three oldest go to the back row: %d %v", len(back), back)
	}
	if n := len(s.All()); n != frontMax+3 {
		t.Fatalf("nothing is deleted: %d", n)
	}
}
