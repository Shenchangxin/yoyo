package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/im"
)

// TestIMCodesSurviveRestartAndHandover 用两个共用同一个 home 的 App 模拟「重启」和「daemon 与桌面换手」：
// 一边发的短编号和记下的引用编号，另一边重读磁盘后都认得出；一边作废的号，另一边也不再认；
// 同一个东西在另一边再要号时拿到的是同一个号，不会两边各发一个。
func TestIMCodesSurviveRestartAndHandover(t *testing.T) {
	root := t.TempDir()
	first := &App{Home: &home.Dir{Root: root}}
	second := &App{Home: &home.Dir{Root: root}}

	code := first.imIssue(imCode{Kind: "ticket", Target: "tkt-1"})
	if _, err := os.Stat(filepath.Join(root, "im", "codes.json")); err != nil {
		t.Fatalf("codes.json not written: %v", err)
	}
	if _, c, ok := second.imLookup(code, "ticket"); !ok || c.Target != "tkt-1" {
		t.Fatalf("other process lookup = %+v %v", c, ok)
	}
	if again := second.imIssue(imCode{Kind: "ticket", Target: "tkt-1"}); again != code {
		t.Fatalf("same ticket got a second code: %q vs %q", again, code)
	}

	first.imRemember("lark-1", "pm-7", "回 /答 "+code+" 你的回答")
	if kind, got := second.imQuotedCode(im.Inbound{AccountID: "lark-1", QuoteID: "pm-7"}); kind != "ticket" || got != code {
		t.Fatalf("quote after handover = %q %q", kind, got)
	}

	// 最近一条短讯的号也跟着走：/有用 不带号时认的是它。
	speak := first.imIssue(imCode{Kind: "speak", Target: "s1", Class: "cal"})
	if got, _, ok := second.imLookup("", "speak"); !ok || got != speak {
		t.Fatalf("last speak = %q %v, want %q", got, ok, speak)
	}

	second.imForget(code)
	if _, _, ok := first.imLookup(code, "ticket"); ok {
		t.Fatal("a code forgotten in one process must be gone in the other")
	}

	// 坏文件不让发号失败：照常发，写回时盖成好的。
	if err := os.WriteFile(filepath.Join(root, "im", "codes.json"), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	fresh := first.imIssue(imCode{Kind: "approval", Target: "o9"})
	if _, _, ok := second.imLookup(fresh, "approval"); !ok {
		t.Fatal("a broken codes file must be rewritten on the next issue")
	}
}

// TestIMCodesWithoutHome 确认没有 home 时只用内存，照常发号、查号。
func TestIMCodesWithoutHome(t *testing.T) {
	a := &App{}
	code := a.imIssue(imCode{Kind: "approval", Target: "o1"})
	if _, c, ok := a.imLookup(code, "approval"); !ok || c.Target != "o1" {
		t.Fatalf("in-memory lookup = %+v %v", c, ok)
	}
}
