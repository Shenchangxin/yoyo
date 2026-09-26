package browser

import "testing"

func TestViewIdleHasNoChrome(t *testing.T) {
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	v := h.View()
	if v.Live {
		t.Fatal("idle view must not spawn Chromium")
	}
	if v.Lane != "isolated" {
		t.Fatalf("lane %q", v.Lane)
	}
	if v.Screenshot != "" {
		t.Fatal("no frame yet")
	}
}

func TestViewKeepsJournal(t *testing.T) {
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h.record("open", "https://example.com")
	h.record("click", "#go")
	v := h.View()
	if len(v.Log) != 2 {
		t.Fatalf("log %d", len(v.Log))
	}
	if v.Log[0]["op"] != "open" || v.Log[1]["op"] != "click" {
		t.Fatalf("%+v", v.Log)
	}
}
