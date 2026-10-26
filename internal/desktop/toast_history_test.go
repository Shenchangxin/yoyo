package desktop

import (
	"strings"
	"testing"
)

// TestClearToastHistoryScriptClearsYoyoOnly 锁住脚本调用 History.Clear，参数是 Yoyo。
func TestClearToastHistoryScriptClearsYoyoOnly(t *testing.T) {
	s := clearToastHistoryScript(toastAppID)
	if !strings.Contains(s, "History.Clear('Yoyo')") {
		t.Fatalf("script should clear the Yoyo history, got %q", s)
	}
}

// TestClearToastHistoryScriptQuotesAppID 锁住 AppUserModelID 里的单引号在 PowerShell 字符串中被加倍。
func TestClearToastHistoryScriptQuotesAppID(t *testing.T) {
	s := clearToastHistoryScript("it's")
	if !strings.Contains(s, "History.Clear('it''s')") {
		t.Fatalf("an apostrophe must be doubled inside the PowerShell string, got %q", s)
	}
}

// TestFocusClearsOnlyAfterAToast 锁住没发过通知时不清理，发过一次后第一次取走为真，再取一次为假。
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
