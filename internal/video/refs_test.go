// refs_test.go 检查 @名字 替换、人名归一和时长估算。

package video

import "testing"

// TestResolvePromptRefsLongestNameWins 确认较长的名字先匹配，URL 顺序与输入一致。
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

// TestResolvePromptRefsWan 确认 wan 为真时 @名字 换成图N，名字本身不留在句子里。
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

// TestNormalizeNameStripsAlias 确认括号别名收成同一个键，含空白的名字不会收成空串。
func TestNormalizeNameStripsAlias(t *testing.T) {
	if NormalizeName("林小雨（主角）") != NormalizeName("林小雨") {
		t.Fatal("alias should collapse")
	}
	if NormalizeName("  林 小 雨  ") == "" {
		t.Fatal("empty")
	}
}

// TestPlanDuration 确认 500 字对应 60 秒和 5 段，并检查对白下限、转场下限，以及 volcengine 的 15 秒上限和 aliyun 的 30 秒上限。
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
