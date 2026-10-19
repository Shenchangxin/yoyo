package profile

import "testing"

func TestSeedsAndFilter(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	list := s.List()
	if len(list) < 3 {
		t.Fatalf("seeds %d", len(list))
	}
	r, err := s.Get("researcher")
	if err != nil {
		t.Fatal(err)
	}
	got := FilterTools([]string{"read_file", "web_search", "reach_status", "review_page", "shell", "computer_act", "ask_user"}, r)
	joined := stringsJoin(got)
	if !contains(got, "web_search") || !contains(got, "ask_user") || !contains(got, "reach_status") {
		t.Fatalf("researcher lost read tools: %s", joined)
	}
	if contains(got, "shell") || contains(got, "computer_act") {
		t.Fatalf("researcher exceeded grants: %s", joined)
	}
	sec, err := s.Get("secretary")
	if err != nil {
		t.Fatal(err)
	}
	got = FilterTools([]string{"read_file", "web_search", "web_fetch", "reach_status", "ask_user"}, sec)
	if contains(got, "web_search") || contains(got, "web_fetch") || contains(got, "reach_status") {
		t.Fatalf("secretary kept research tools: %v", got)
	}
}

func TestMCPAllow(t *testing.T) {
	if !MCPAllowed(nil, "anything") {
		t.Fatal("empty allow means all")
	}
	if !MCPAllowed([]string{"github"}, "github__create_issue") {
		t.Fatal("prefix")
	}
	if MCPAllowed([]string{"github"}, "slack__post") {
		t.Fatal("denied")
	}
}

func contains(have []string, n string) bool {
	for _, x := range have {
		if x == n {
			return true
		}
	}
	return false
}

func stringsJoin(s []string) string {
	out := ""
	for i, x := range s {
		if i > 0 {
			out += " "
		}
		out += x
	}
	return out
}
