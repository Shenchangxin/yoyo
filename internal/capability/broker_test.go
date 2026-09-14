package capability

import (
	"path/filepath"
	"testing"
)

func TestWithinWorkspace(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "a", "b.txt")
	if !WithinWorkspace(root, inside) {
		t.Fatal("inside")
	}
	if WithinWorkspace(root, filepath.Join(root, "..", "x")) {
		t.Fatal("escape")
	}
}

func TestBrokerSessionAllow(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{}, func(Request) (Decision, error) {
		n++
		return Session, nil
	})
	req := Request{Level: Shell, SessionID: "s1"}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("asked %d times", n)
	}
}
