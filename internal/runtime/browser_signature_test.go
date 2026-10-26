// 同类浏览器未命中共享一个签名，无关错误不进同一组。

package runtime

import "testing"

// TestBrowserMissesShareSignatureAcrossSelectors 确认不同选择器的 browser: no match 共享签名，covered 与 hidden 分开，无关错误不进组。
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
