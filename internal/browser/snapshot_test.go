package browser

import (
	"strings"
	"testing"
)

func TestNormalizeSnapshotLevel(t *testing.T) {
	if got := NormalizeSnapshotLevel("", "https://example.com"); got != SnapshotInteractive {
		t.Fatalf("remote default %s", got)
	}
	if got := NormalizeSnapshotLevel("", "http://127.0.0.1:9/index.html"); got != SnapshotFull {
		t.Fatalf("local default %s", got)
	}
	if got := NormalizeSnapshotLevel("MINIMAL", "https://example.com"); got != SnapshotMinimal {
		t.Fatalf("explicit %s", got)
	}
}

func TestFormatSnapshotTiers(t *testing.T) {
	snap := Snapshot{
		URL:      "https://example.com",
		Title:    "Example",
		Text:     "hello body",
		Actions:  "  [ref=btn_0]  #go  <button> \"Go\"",
		Elements: []Action{{Ref: "btn_0", Selector: "#go", Tag: "button", Text: "Go"}},
	}
	min := FormatSnapshot(snap, SnapshotMinimal, true)
	if !strings.Contains(min, "Interactive elements: 1") || strings.Contains(min, "hello body") {
		t.Fatalf("minimal: %s", min)
	}
	inter := FormatSnapshot(snap, SnapshotInteractive, true)
	if !strings.Contains(inter, "ref=btn_0") || strings.Contains(inter, "hello body") {
		t.Fatalf("interactive: %s", inter)
	}
	full := FormatSnapshot(snap, SnapshotFull, true)
	if !strings.Contains(full, "hello body") || !strings.Contains(full, "ref=btn_0") {
		t.Fatalf("full: %s", full)
	}
	st := FormatSnapshot(snap, SnapshotStructured, true)
	if !strings.Contains(st, `"ref": "btn_0"`) {
		t.Fatalf("structured: %s", st)
	}
}
