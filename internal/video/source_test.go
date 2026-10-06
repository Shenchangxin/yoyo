package video

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestLooksLikeSeries(t *testing.T) {
	if LooksLikeSeries("林小雨在巷口等车。") {
		t.Fatal("short prose should stay on the one-episode path")
	}
	long := strings.Repeat("林小雨在巷口等车。", 280)
	if !LooksLikeSeries(long) {
		t.Fatal("long text should look like a series")
	}
	chapters := "第一章 雨\n" + strings.Repeat("她等了很久。", 50) + "\n第二章 夜\n" + strings.Repeat("他没有来。", 50)
	if !LooksLikeSeries(chapters) {
		t.Fatal("two chapters should look like a series")
	}
}

func TestIngestProposeCommitSlices(t *testing.T) {
	e := testEngine(t)
	d, err := e.CreateDrama(Drama{Title: "雨巷", Style: "3d", AspectRatio: "9:16"})
	if err != nil {
		t.Fatal(err)
	}
	ch1 := "第一章 雨巷\n" + strings.Repeat("林小雨在巷口等车。", 120)
	ch2 := "第二章 夜色\n" + strings.Repeat("车灯亮了，他没有下车。", 120)
	src, err := e.IngestSource(d.ID, ch1+"\n"+ch2, "雨巷")
	if err != nil {
		t.Fatal(err)
	}
	if src.RuneCount < 400 || len(src.Slices) < 1 {
		t.Fatalf("source %+v", src)
	}
	plan, err := e.ProposeEpisodes(d.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "draft" || len(plan.Episodes) < 2 {
		t.Fatalf("plan %+v", plan)
	}
	eps, _ := e.ListEpisodes(d.ID)
	if len(eps) != 0 {
		t.Fatal("draft plan must not create episodes")
	}
	plan, err = e.CommitEpisodes(d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "committed" {
		t.Fatal(plan.Status)
	}
	eps, err = e.ListEpisodes(d.ID)
	if err != nil || len(eps) < 2 {
		t.Fatalf("episodes %d err %v", len(eps), err)
	}
	full := ch1 + "\n" + ch2
	for _, ep := range eps {
		if ep.Content == full {
			t.Fatal("episode content must be a slice, not the whole novel")
		}
		if strings.TrimSpace(ep.Content) == "" {
			t.Fatal("empty slice")
		}
		if !strings.Contains(full, ep.Content) {
			t.Fatal("slice must come from source")
		}
	}
	if _, err := e.IngestSource(d.ID, "另一部", ""); err == nil {
		t.Fatal("committed series must reject a new ingest")
	}
}

func TestReadSourceWindow(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "窗"})
	text := strings.Repeat("甲乙丙丁戊己庚辛。", 400)
	if _, err := e.IngestSource(d.ID, text, ""); err != nil {
		t.Fatal(err)
	}
	out, err := e.ReadSource(d.ID, 0, 500, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	got := out["text"].(string)
	if utf8.RuneCountInString(got) > maxReadWindow {
		t.Fatalf("window %d", utf8.RuneCountInString(got))
	}
	if out["truncated"] != true && utf8.RuneCountInString(text) > 500 {
		t.Fatal("expected truncated")
	}
}

func TestBindSessionDramaWithoutEpisode(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "窗"})
	if err := e.BindSessionDrama("s1", d.ID); err != nil {
		t.Fatal(err)
	}
	b, ok := e.SessionBind("s1")
	if !ok || b.DramaID != d.ID || b.EpisodeID != "" {
		t.Fatalf("bind %+v", b)
	}
	if _, err := e.Tool("s1", "drama_read_plan", nil); err != nil {
		t.Fatal(err)
	}
}

func TestCommitToolRejected(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "窗"})
	_ = e.BindSessionDrama("s1", d.ID)
	if _, err := e.Tool("s1", "drama_commit_episodes", nil); err == nil {
		t.Fatal("commit must stay an operator action")
	}
}

func TestStoryboardCoverageDoesNotMarkDoneEarly(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "窗"})
	ep, err := e.CreateEpisode(d.ID, "一", strings.Repeat("对白对白对白。", 400))
	if err != nil {
		t.Fatal(err)
	}
	script := strings.Repeat("角色说：「今晚必须离开。」\n", 200)
	if err := e.SaveScript(ep.ID, script); err != nil {
		t.Fatal(err)
	}
	plan := PlanDuration(script)
	if plan.SegmentCount <= 8 {
		t.Fatalf("need a long plan, got %+v", plan)
	}
	_ = e.BindSession("s1", ep.ID)
	shots := make([]Shot, 8)
	for i := range shots {
		shots[i] = Shot{ShotNumber: i + 1, Duration: 12, Description: "镜"}
	}
	raw, err := e.Tool("s1", "drama_save_storyboards", map[string]any{
		"storyboards": shots, "replace_existing": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	if out["continue"] != true {
		t.Fatalf("coverage %+v", out)
	}
	got, _ := e.GetEpisode(ep.ID)
	if pipeStatusGo(got.Pipeline, "storyboard") == "done" {
		t.Fatal("first batch of 8 must not finish a long episode")
	}
}

func pipeStatusGo(raw, stage string) string {
	var m map[string]any
	_ = json.Unmarshal([]byte(raw), &m)
	if m == nil {
		return ""
	}
	s, _ := m[stage].(map[string]any)
	if s == nil {
		return ""
	}
	v, _ := s["status"].(string)
	return v
}

func TestBibleSaveWithoutEpisode(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "窗"})
	out, err := e.SaveBibleCharacters(d.ID, []Character{{Name: "林小雨", Appearance: "短发", FirstSeenEpisode: 1}})
	if err != nil || len(out) != 1 {
		t.Fatalf("%+v %v", out, err)
	}
	chars := e.DramaCharacters(d.ID)
	if len(chars) != 1 || chars[0].FirstSeenEpisode != 1 {
		t.Fatalf("%+v", chars)
	}
}

func TestSplitAndMergePlan(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "窗"})
	text := "第一章 上\n" + strings.Repeat("甲。", 2500) + "\n第二章 下\n" + strings.Repeat("乙。", 2500)
	if _, err := e.IngestSource(d.ID, text, ""); err != nil {
		t.Fatal(err)
	}
	plan, err := e.ProposeEpisodes(d.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Episodes) < 1 {
		t.Fatal("empty")
	}
	n := plan.Episodes[0].N
	split, err := e.SplitPlanEpisode(d.ID, n)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := e.MergePlanEpisodes(d.ID, split.Episodes[0].N)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Episodes) == 0 {
		t.Fatal("merge emptied the map")
	}
}
