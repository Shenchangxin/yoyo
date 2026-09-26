package memory

import (
	"strings"
	"testing"
)

func TestForgetRemoves(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	it := s.Write(Item{Kind: KindEpisodic, Text: "SECRET_TOKEN_XYZ"})
	if !it.Staging {
		t.Fatal("writes must stay staging")
	}
	if !s.Forget(it.ID) {
		t.Fatal("forget")
	}
	if len(s.Search("SECRET", "", 10)) != 0 {
		t.Fatal("still present")
	}
}

func TestProfilePinSkipsStaging(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.Write(Item{Kind: KindProfile, Text: "call me Ada"})
	if pin := s.ProfilePin(200); !strings.Contains(pin, "Ada") {
		t.Fatalf("profile writes pin immediately: %q", pin)
	}
}
