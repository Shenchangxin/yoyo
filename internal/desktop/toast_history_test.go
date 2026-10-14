package desktop

import (
	"strings"
	"testing"
)

func TestClearToastHistoryScriptClearsYoyoOnly(t *testing.T) {
	s := clearToastHistoryScript(toastAppID)
	if !strings.Contains(s, "History.Clear('Yoyo')") {
		t.Fatalf("script should clear the Yoyo history, got %q", s)
	}
}

func TestClearToastHistoryScriptQuotesAppID(t *testing.T) {
	s := clearToastHistoryScript("it's")
	if !strings.Contains(s, "History.Clear('it''s')") {
		t.Fatalf("an apostrophe must be doubled inside the PowerShell string, got %q", s)
	}
}

func TestFocusClearsOnlyAfterAToast(t *testing.T) {
	s := &Service{}
	if s.takeToastClear() {
		t.Fatal("focus with no toast sent should not clear")
	}
	s.noteToastSent()
	if !s.takeToastClear() {
		t.Fatal("first focus after a toast should clear")
	}
	if s.takeToastClear() {
		t.Fatal("a second focus with no new toast should not clear again")
	}
}
