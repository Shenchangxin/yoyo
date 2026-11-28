package memory

import (
	"strings"
	"testing"
)

// TestMutedStaysButIsNotUsed 锁住「别再提」：条目还在、记忆页搜得到，但不进画像、不被召回；打开之后照常。
func TestMutedStaysButIsNotUsed(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	pref := s.Keep(Item{Kind: KindSemantic, Text: "代码注释用中文", Source: "user"}, "").Item
	done := s.Keep(Item{Kind: KindEpisodic, Text: "首页改版：标题换成新的，配色改成浅色", Source: "worker"}, "").Item
	if _, used := s.ProfileUse(0); len(used) != 1 || used[0].ID != pref.ID {
		t.Fatalf("used = %+v", used)
	}
	if !s.SetMuted(pref.ID, true) || !s.SetMuted(done.ID, true) {
		t.Fatal("mute")
	}
	if text, used := s.ProfileUse(0); strings.Contains(text, "代码注释用中文") || len(used) != 0 {
		t.Fatalf("a muted line was pinned: %q %+v", text, used)
	}
	if got := s.Recall("首页改版的配色还要再调一下", 3); len(got) != 0 {
		t.Fatalf("a muted line was recalled: %+v", got)
	}
	if got := s.Search("代码注释", "", 10); len(got) != 1 || !got[0].Muted {
		t.Fatalf("the memory page must still list it: %+v", got)
	}
	s.SetMuted(pref.ID, false)
	if !strings.Contains(s.ProfilePin(0), "代码注释用中文") {
		t.Fatal("unmuted, it is used again")
	}
	if s.SetMuted("machine:host", true) || s.SetMuted("nope", true) {
		t.Fatal("machine lines and unknown ids are refused")
	}
}

// TestRevertUndoesAChangeOfMind 锁住改口的撤销：退回旧说法和旧来源，历史少一条；没有历史的不能退。
func TestRevertUndoesAChangeOfMind(t *testing.T) {
	now := friday
	s := openAt(t, &now)
	first := s.Keep(Item{Kind: KindSemantic, Text: "代码注释用中文", Source: "user"}, "").Item
	if _, ok := s.Revert(first.ID); ok {
		t.Fatal("nothing to go back to")
	}
	changed := s.Keep(Item{Kind: KindSemantic, Text: "代码注释用英文", Source: "Inferred in chat"}, first.ID)
	if changed.Action != "replaced" || len(changed.Item.History) != 1 {
		t.Fatalf("changed = %+v", changed)
	}
	back, ok := s.Revert(first.ID)
	if !ok || back.Text != "代码注释用中文" || back.Source != "user" || len(back.History) != 0 || back.ID != first.ID {
		t.Fatalf("back = %+v %v", back, ok)
	}
	if !strings.Contains(s.ProfilePin(0), "代码注释用中文") || strings.Contains(s.ProfilePin(0), "英文") {
		t.Fatalf("pin = %q", s.ProfilePin(0))
	}
}
