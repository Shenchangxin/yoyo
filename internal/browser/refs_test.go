package browser

import "testing"

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

func TestMatchTargetRejectsWeakOverlap(t *testing.T) {
	want := Action{Role: "button"}
	have := []Action{{Ref: "btn_0", Role: "button", Text: "Other"}}
	_, why, score := matchTarget(want, have)
	if score >= matchThreshold {
		t.Fatalf("role-only should miss, score=%d why=%s", score, why)
	}
}

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
