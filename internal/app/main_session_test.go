package app_test

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
)

// TestMainSessionStartsFresh 锁住主会话不收编已经存在的旧对话，并且同一条会一直用下去。
func TestMainSessionStartsFresh(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	old, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	main, err := a.MainSession()
	if err != nil {
		t.Fatal(err)
	}
	if main.ID == "" || main.ID == old.ID {
		t.Fatalf("main %q old %q", main.ID, old.ID)
	}
	again, err := a.MainSession()
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != main.ID {
		t.Fatalf("again %s want %s", again.ID, main.ID)
	}
	if err := a.AdoptMainSession(old.ID); err != nil {
		t.Fatal(err)
	}
	got, err := a.MainSession()
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != old.ID {
		t.Fatalf("adopted %s want %s", got.ID, old.ID)
	}
}
