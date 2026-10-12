package video

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestTopicCRUDAndTrend(t *testing.T) {
	e := testEngine(t)
	got, err := e.TopicFromTrend("weibo", "春日穿搭", "120万")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != TopicInbox || got.SourcePlatform != "weibo" || got.Note == "" {
		t.Fatalf("from trend %+v", got)
	}
	got.Angle = "职场通勤怎么穿"
	got.Scores = map[string]int{"heat": 5, "feasibility": 4}
	got.Status = TopicReady
	saved, err := e.SaveTopic(got)
	if err != nil {
		t.Fatal(err)
	}
	listed := e.ListTopics(TopicReady)
	if len(listed) != 1 || listed[0].ID != saved.ID {
		t.Fatalf("list ready %+v", listed)
	}
	if _, err := e.GetTopic(saved.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.DeleteTopic(saved.ID); err != nil {
		t.Fatal(err)
	}
	if n := e.ListTopics(""); len(n) != 0 {
		t.Fatalf("want empty after delete, got %+v", n)
	}
}

func TestStudioFromTopicBindAndHistory(t *testing.T) {
	e := testEngine(t)
	topic, err := e.SaveTopic(Topic{Title: "三日护肤", Angle: "成分党避坑", Status: TopicInbox})
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.StudioFromTopic(topic.ID, "9:16")
	if err != nil {
		t.Fatal(err)
	}
	if p.TopicID != topic.ID || p.Aspect != "9:16" || p.Status != StudioDraft {
		t.Fatalf("project %+v", p)
	}
	linked, err := e.GetTopic(topic.ID)
	if err != nil {
		t.Fatal(err)
	}
	if linked.StudioProjectID != p.ID || linked.Status != TopicInProduction {
		t.Fatalf("topic link %+v", linked)
	}
	again, err := e.StudioFromTopic(topic.ID, "9:16")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != p.ID {
		t.Fatalf("reuse want %s got %s", p.ID, again.ID)
	}
	if err := e.BindStudioSession("sess-studio", p.ID); err != nil {
		t.Fatal(err)
	}
	id, ok := e.StudioSessionBind("sess-studio")
	if !ok || id != p.ID {
		t.Fatalf("bind %s %v", id, ok)
	}
	st := e.SessionProject("sess-studio")
	if fmt.Sprint(st["kind"]) != "studio" || fmt.Sprint(st["id"]) != p.ID {
		t.Fatalf("session project %+v", st)
	}
	hist, err := e.ProjectHistory()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range hist {
		if fmt.Sprint(row["kind"]) == "studio" && fmt.Sprint(row["id"]) == p.ID && fmt.Sprint(row["session_id"]) == "sess-studio" {
			found = true
		}
	}
	if !found {
		t.Fatalf("history %+v", hist)
	}
	p.Script = "开场三秒报选题。"
	p.Storyboard.Shots = []StudioShot{{Caption: "hook", Duration: 3}}
	upd, err := e.UpdateStudioProject(p)
	if err != nil {
		t.Fatal(err)
	}
	if upd.Script != p.Script || len(upd.Storyboard.Shots) != 1 {
		t.Fatalf("update %+v", upd)
	}
	if err := e.DeleteStudioProject(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.StudioSessionBind("sess-studio"); ok {
		t.Fatal("bind should clear")
	}
}

func TestStudioToolDispatcher(t *testing.T) {
	e := testEngine(t)
	raw, err := e.StudioTool("s1", "topic_save", map[string]any{"title": "热点一", "angle": "反常识"})
	if err != nil {
		t.Fatal(err)
	}
	var topic Topic
	if err := json.Unmarshal([]byte(raw), &topic); err != nil || topic.ID == "" {
		t.Fatalf("save %s %v", raw, err)
	}
	listed, err := e.StudioTool("s1", "topic_list", nil)
	if err != nil {
		t.Fatal(err)
	}
	if listed == "" || listed == "null" {
		t.Fatalf("list %s", listed)
	}
	created, err := e.StudioTool("s1", "studio_create", map[string]any{"topic_id": topic.ID, "aspect": "1:1"})
	if err != nil {
		t.Fatal(err)
	}
	var p StudioProject
	if err := json.Unmarshal([]byte(created), &p); err != nil || p.ID == "" {
		t.Fatalf("create %s %v", created, err)
	}
	id, ok := e.StudioSessionBind("s1")
	if !ok || id != p.ID {
		t.Fatalf("auto bind %s %v", id, ok)
	}
	names := StudioToolNames()
	if len(names) < 10 {
		t.Fatalf("tool names %v", names)
	}
}

func TestProviderTemplatesIncludeSearchVendors(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range ProviderTemplates() {
		if p.ServiceType == "search" {
			seen[p.Provider] = true
		}
	}
	for _, want := range []string{"search", "tavily", "brave", "serper", "bocha"} {
		if !seen[want] {
			t.Fatalf("missing search template %s in %v", want, seen)
		}
	}
}

func TestMediaRefValid(t *testing.T) {
	if (MediaRef{Kind: "video"}).Valid() {
		t.Fatal("empty hash must be invalid")
	}
	if !(MediaRef{Kind: "image", CASHash: "abc"}).Valid() {
		t.Fatal("image with hash")
	}
}

func TestPublishDraftExportMark(t *testing.T) {
	e := testEngine(t)
	h, err := e.PutBytes([]byte("fake-mp4"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.SavePublish(Publish{Platform: "xhs", Title: "测试稿", MediaHash: h})
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != PublishDraft || p.CreatorURL == "" || p.AnalyticsURL == "" {
		t.Fatalf("%+v", p)
	}
	exp, err := e.ExportPublish(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if exp.ExportPath == "" {
		t.Fatal("export path")
	}
	if _, err := os.Stat(exp.ExportPath); err != nil {
		t.Fatal(err)
	}
	topic, err := e.SaveTopic(Topic{Title: "已发选题"})
	if err != nil {
		t.Fatal(err)
	}
	p.TopicID = topic.ID
	p, err = e.SavePublish(p)
	if err != nil {
		t.Fatal(err)
	}
	marked, err := e.MarkPublish(p.ID, "posted", "https://example.com/p", "ok", 10, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if marked.Status != PublishPosted || marked.Reads != 10 || marked.Likes != 2 {
		t.Fatalf("%+v", marked)
	}
	got, err := e.GetTopic(topic.ID)
	if err != nil || got.Status != TopicDone {
		t.Fatalf("topic %+v %v", got, err)
	}
	if _, err := e.SavePublish(Publish{Platform: "myspace", Title: "x"}); err == nil {
		t.Fatal("want unknown platform")
	}
	listed := e.ListPublishes("", "xhs")
	if len(listed) != 1 || listed[0].ID != p.ID {
		t.Fatalf("list %+v", listed)
	}
}

func TestUpdateStudioStyleAndResult(t *testing.T) {
	e := testEngine(t)
	p, err := e.CreateStudioProject(StudioProject{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	p.Style = "ghibli"
	p.ResultHash = "abc123"
	got, err := e.UpdateStudioProject(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Style != "ghibli" || got.ResultHash != "abc123" {
		t.Fatalf("%+v", got)
	}
}

func TestPersonaTemplate(t *testing.T) {
	md := PersonaTemplate("测试号")
	if !strings.Contains(md, "测试号") || !strings.Contains(md, "人设") {
		t.Fatal(md)
	}
}

func TestStudioP4Tools(t *testing.T) {
	e := testEngine(t)
	md, err := e.StudioTool("", "studio_persona_template", map[string]any{"name": "口播号"})
	if err != nil || !strings.Contains(md, "口播号") {
		t.Fatalf("%s %v", md, err)
	}
	if _, err := e.StudioTool("", "studio_publish_open", map[string]any{"publish_id": "x"}); err == nil {
		t.Fatal("open must be hosted")
	}
	if _, err := e.StudioTool("", "studio_persona_create", nil); err == nil {
		t.Fatal("persona create must be hosted")
	}
	raw, err := e.StudioTool("", "studio_remotion_status", nil)
	if err != nil || !strings.Contains(raw, `"bundled":false`) {
		t.Fatalf("%s %v", raw, err)
	}
	names := strings.Join(StudioToolNames(), " ")
	for _, w := range []string{"studio_publish_draft", "studio_publish_analytics", "studio_wrap", "studio_persona_template", "studio_persona_create", "studio_remotion_install"} {
		if !strings.Contains(names, w) {
			t.Fatalf("missing %s in %s", w, names)
		}
	}
}

func TestUpdateStudioClearsShots(t *testing.T) {
	e := testEngine(t)
	p, err := e.CreateStudioProject(StudioProject{
		Title:      "t",
		Script:     "hello",
		Storyboard: StudioStoryboard{Shots: []StudioShot{{ImageHash: "abc", Duration: 2}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	p.Storyboard.Shots = []StudioShot{}
	got, err := e.UpdateStudioProject(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Storyboard.Shots) != 0 {
		t.Fatalf("want no shots, got %+v", got.Storyboard.Shots)
	}
}

func TestSeedStudioShortsFolder(t *testing.T) {
	e := testEngine(t)
	if err := e.seedStudioAssets(); err != nil {
		t.Fatal(err)
	}
	if _, err := getDoc[canvasFolderRec](e, colFolders, "studio-folder-shorts"); err != nil {
		t.Fatal(err)
	}
}
