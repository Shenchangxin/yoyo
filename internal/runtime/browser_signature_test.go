package runtime

import "testing"

// Session 3b9f1bbb660e4236: every browser_click miss named a different
// selector, so the repeat detector saw six different errors and never stopped.
// Misses of one kind must share a signature; unrelated errors must not.
func TestBrowserMissesShareSignatureAcrossSelectors(t *testing.T) {
	a := errorSignature("ERROR: browser: selector #opts button:nth-child(2) matched no element\nvisible actions now (use one of these selectors):\n  #opt-0 ...")
	c := errorSignature("ERROR: browser: selector #opt-1 matched no element\nno visible interactive elements on this page")
	if a != "browser: no match" || a != c {
		t.Fatalf("no-match misses must group: %q vs %q", a, c)
	}
	covered := errorSignature("ERROR: browser: #x is covered by div#cover, so a real click would not reach it")
	if covered != "browser: covered" {
		t.Fatalf("covered: %q", covered)
	}
	if hidden := errorSignature("ERROR: browser: #x is not visible"); hidden != "browser: hidden" {
		t.Fatalf("hidden: %q", hidden)
	}
	if got := errorSignature("ERROR: exit status 1 something else"); got == "browser: no match" {
		t.Fatalf("unrelated error must not join the browser group: %q", got)
	}
}
