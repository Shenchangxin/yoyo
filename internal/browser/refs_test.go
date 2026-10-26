// 本文件锁住 ref 解析、testid 优先打分、只靠 role 对不上，以及缓存的选择器能找回描述。
// 由 go test 调用。

package browser

import "testing"

// TestParseRef 锁住 ref=、ref:、方括号和裸句柄能取出名字，CSS 和 text= 则不是 ref。
func TestParseRef(t *testing.T) {
	cases := map[string]string{
		"ref=btn_0":   "btn_0",
		"ref:inp_2":   "inp_2",
		"[ref=lnk_1]": "lnk_1",
		"btn_3":       "btn_3",
		"#opt-0":      "",
		"text=Start":  "",
		"":            "",
	}
	for in, want := range cases {
		if got := parseRef(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}

// TestMatchTargetPrefersTestID 锁住带相同 testid 的按钮得分超过阈值。
func TestMatchTargetPrefersTestID(t *testing.T) {
	want := Action{TestID: "login", Name: "Sign in", Role: "button"}
	have := []Action{
		{Ref: "btn_0", Tag: "button", Text: "Cancel", Role: "button"},
		{Ref: "btn_1", Tag: "button", Text: "Sign in", Role: "button", TestID: "login", ID: "go"},
	}
	got, why, score := matchTarget(want, have)
	if got.Ref != "btn_1" || score < matchThreshold {
		t.Fatalf("got %+v why=%s score=%d", got, why, score)
	}
	if why == "not-found" {
		t.Fatal(why)
	}
}

// TestMatchTargetRejectsWeakOverlap 锁住只有 role 相同的时候分数低于阈值。
func TestMatchTargetRejectsWeakOverlap(t *testing.T) {
	want := Action{Role: "button"}
	have := []Action{{Ref: "btn_0", Role: "button", Text: "Other"}}
	_, why, score := matchTarget(want, have)
	if score >= matchThreshold {
		t.Fatalf("role-only should miss, score=%d why=%s", score, why)
	}
}

// TestRememberedTargetFromCachedSelector 锁住 #id 和 data-testid 能找回缓存，未知 CSS 不会编出一个描述。
func TestRememberedTargetFromCachedSelector(t *testing.T) {
	h := &Host{}
	h.refs.store([]Action{{Ref: "btn_1", Selector: "#go", TestID: "login", ID: "go", Name: "Sign in"}})
	got, ok := h.rememberedTarget("#go")
	if !ok || got.TestID != "login" || got.Ref != "btn_1" {
		t.Fatalf("%+v %v", got, ok)
	}
	got, ok = h.rememberedTarget(`[data-testid="login"]`)
	if !ok || got.ID != "go" {
		t.Fatalf("testid %+v %v", got, ok)
	}
	if _, ok := h.rememberedTarget("#missing"); ok {
		t.Fatal("unknown CSS must not invent a descriptor")
	}
}

// TestRefCacheRoundTrip 锁住 store 之后能 lookup，clear 之后不能。
func TestRefCacheRoundTrip(t *testing.T) {
	var c refCache
	c.store([]Action{{Ref: "btn_0", Selector: "#a", Text: "A"}})
	got, ok := c.lookup("btn_0")
	if !ok || got.Selector != "#a" {
		t.Fatalf("%+v %v", got, ok)
	}
	c.clear()
	if _, ok := c.lookup("btn_0"); ok {
		t.Fatal("cleared cache still hit")
	}
}
