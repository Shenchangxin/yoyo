package computeruse

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCaptureRefusesOperatorScreen(t *testing.T) {
	t.Setenv("YOYO_CU_DISPLAY", "")
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	err = h.Capture(filepath.Join(t.TempDir(), "x.png"))
	if err == nil {
		if runtime.GOOS == "windows" {
			return
		}
		t.Fatal("must refuse primary screen")
	}
	if runtime.GOOS != "windows" && !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("want refuse, got %v", err)
	}
}

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
