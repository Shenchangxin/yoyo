package capability

import (
	"context"
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
	b := NewBroker(AutoPolicy{}, func(context.Context, Request) (Decision, error) {
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

func TestBrokerForceAskSkipsSession(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{}, func(context.Context, Request) (Decision, error) {
		n++
		return Session, nil
	})
	req := Request{Level: Network, SessionID: "s1", ForceAsk: true, Action: "mcp"}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("ForceAsk should skip session cache, asked %d", n)
	}
}

func TestBrokerForceAskHonorsAlways(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{}, func(context.Context, Request) (Decision, error) {
		n++
		return Always, nil
	})
	req := Request{Level: Network, SessionID: "s1", ForceAsk: true, Action: "mcp"}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("ForceAsk must still honor Always, asked %d", n)
	}
}

func TestBrokerNeverAlwaysHighRisk(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{}, func(context.Context, Request) (Decision, error) {
		n++
		return Always, nil
	})
	req := Request{Level: HighRisk, SessionID: "s1", ForceAsk: true, Action: "wasm"}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("identity Always must not stick, asked %d", n)
	}
}

func TestBrokerAskModeIgnoresAlways(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{Allow: []Level{ReadWorkspace, WriteWorkspace}}, func(context.Context, Request) (Decision, error) {
		n++
		return Once, nil
	})
	b.ApplyAuthMode("s1", AuthAsk)
	req := Request{Level: WriteWorkspace, SessionID: "s1"}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if err := b.Check(req); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("ask mode should skip always-map, asked %d", n)
	}
}

func TestBrokerFullGrantsShell(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{Allow: []Level{ReadWorkspace, WriteWorkspace}}, func(context.Context, Request) (Decision, error) {
		n++
		return Deny, nil
	})
	b.ApplyAuthMode("s1", AuthFull)
	if err := b.Check(Request{Level: Shell, SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("full access should grant shell, asked %d", n)
	}
	if err := b.Check(Request{Level: HighRisk, SessionID: "s1", ForceAsk: true}); err == nil {
		t.Fatal("identity must still ask")
	}
}

func TestBrokerAskThenDefaultRestoresAlways(t *testing.T) {
	n := 0
	b := NewBroker(AutoPolicy{Allow: []Level{WriteWorkspace}}, func(context.Context, Request) (Decision, error) {
		n++
		return Once, nil
	})
	b.ApplyAuthMode("s1", AuthAsk)
	if err := b.Check(Request{Level: WriteWorkspace, SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	b.ApplyAuthMode("s1", AuthDefault)
	if err := b.Check(Request{Level: WriteWorkspace, SessionID: "s1"}); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("default should honor always-map again, asked %d", n)
	}
}

func TestParseAuthMode(t *testing.T) {
	if ParseAuthMode("auto-edit") != AuthAutoEdit {
		t.Fatal(ParseAuthMode("auto-edit"))
	}
	if ParseAuthMode("full_access") != AuthFull {
		t.Fatal(ParseAuthMode("full_access"))
	}
	if ParseAuthMode("") != AuthDefault {
		t.Fatal(ParseAuthMode(""))
	}
}
