package video

import "testing"

func TestResolvePromptRefsLongestNameWins(t *testing.T) {
	refs := []AssetRef{
		{Name: "林", URL: "data:a"},
		{Name: "林小雨", URL: "data:b"},
		{Name: "巷口", URL: "data:c"},
	}
	out, urls := ResolvePromptRefs("夜色里 @林小雨 走过 @巷口，回头看 @林。", refs, false)
	if len(urls) != 3 {
		t.Fatalf("urls %d", len(urls))
	}
	if urls[0] != "data:a" || urls[1] != "data:b" || urls[2] != "data:c" {
		t.Fatalf("slot order %+v", urls)
	}
	if out != "夜色里 @图片2林小雨 走过 @图片3巷口，回头看 @图片1林。" {
		t.Fatalf("got %q", out)
	}
}

func TestResolvePromptRefsWan(t *testing.T) {
	refs := []AssetRef{{Name: "阿宁", URL: "u1"}}
	out, urls := ResolvePromptRefs("@阿宁推门。", refs, true)
	if len(urls) != 1 || urls[0] != "u1" {
		t.Fatalf("urls %+v", urls)
	}
	if out != "图1推门。" {
		t.Fatalf("got %q", out)
	}
}

func TestNormalizeNameStripsAlias(t *testing.T) {
	if NormalizeName("林小雨（主角）") != NormalizeName("林小雨") {
		t.Fatal("alias should collapse")
	}
	if NormalizeName("  林 小 雨  ") == "" {
		t.Fatal("empty")
	}
}

func TestPlanDuration(t *testing.T) {
	p := PlanDuration(string(make([]rune, 500)))
	if p.TargetSeconds != 60 {
		t.Fatalf("seconds %d", p.TargetSeconds)
	}
	if p.SegmentCount != 5 {
		t.Fatalf("segments %d", p.SegmentCount)
	}
	if DialogueFloor(18) < MinSegment {
		t.Fatal("floor")
	}
	if ClampSegment(3, "transition") != 8 {
		t.Fatal("transition lo")
	}
	if ClampProviderDuration(20, "volcengine") != 15 {
		t.Fatal("seedance cap")
	}
	if ClampProviderDuration(40, "aliyun") != 30 {
		t.Fatal("wan cap")
	}
}
