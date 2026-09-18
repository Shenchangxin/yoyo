package computeruse

import "testing"

func TestAllowlistBlocksOfflist(t *testing.T) {
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Act("notepad", "click", "ok"); err == nil {
		t.Fatal("offlist")
	}
	h.Allow("notepad")
	ev, err := h.Act("notepad", "click", "ok")
	if err != nil {
		t.Fatal(err)
	}
	if ev.Display == "" {
		t.Fatal("display")
	}
	if len(h.Recording()) != 1 {
		t.Fatal("journal")
	}
}
