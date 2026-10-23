package video

import (
	"strings"
	"testing"
	"time"
)

func TestPlanTopicFilm(t *testing.T) {
	how := planTopicFilm(Topic{Title: "三步教程：怎么煮面", SourcePlatform: "xhs", Note: "水面要宽"})
	if how.TemplateID != "howto" || how.Duration != 45 || how.Voice != "onyx" || how.Brief != "水面要宽" {
		t.Fatalf("%+v", how)
	}
	news := planTopicFilm(Topic{Title: "今天这条热搜", SourcePlatform: "douyin", Angle: "给通勤的人"})
	if news.TemplateID != "news-take" || news.Duration != 30 || news.Voice != "fable" || news.Audience != "给通勤的人" {
		t.Fatalf("%+v", news)
	}
	story := planTopicFilm(Topic{Title: "一个反转故事"})
	if story.TemplateID != "story" || story.Duration != 45 || story.Voice != "fable" {
		t.Fatalf("%+v", story)
	}
	plain := planTopicFilm(Topic{Title: "随手记"})
	if plain.TemplateID != "ugc-hook" || plain.Voice != "nova" || plain.Audience != "会刷到这条的人" {
		t.Fatalf("%+v", plain)
	}
	why := planTopicFilm(Topic{Title: "为什么天是蓝的", SourcePlatform: "zhihu"})
	if why.TemplateID != "explainer" || why.Voice != "onyx" {
		t.Fatalf("%+v", why)
	}
}

func TestStartTopicFilmsAppliesPlan(t *testing.T) {
	e := testEngine(t)
	a, err := e.SaveTopic(Topic{Title: "今天这条热搜", SourcePlatform: "douyin", Angle: "给通勤的人"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.SaveTopic(Topic{Title: "三步教程：怎么煮面", SourcePlatform: "xhs", Note: "水面要宽"})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := e.StartTopicFilms([]string{a.ID, b.ID, b.ID, "  "})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows %d", len(rows))
	}
	if rows[0].TemplateID != "news-take" || rows[0].TargetDuration != 30 || rows[0].Aspect != "9:16" || rows[0].Voice != "fable" {
		t.Fatalf("news %+v", rows[0])
	}
	if rows[1].TemplateID != "howto" || rows[1].TargetDuration != 45 || rows[1].Voice != "onyx" || rows[1].Brief != "水面要宽" || rows[1].Storyboard.Fit != "cover" {
		t.Fatalf("howto %+v", rows[1])
	}
	for _, row := range rows {
		got := waitFilm(t, e, row.ID)
		if got.AutoPhase != "failed" || !strings.Contains(got.AutoError, "口播") {
			t.Fatalf("phase %s err %s", got.AutoPhase, got.AutoError)
		}
		topic, terr := e.GetTopic(row.TopicID)
		if terr != nil || topic.Status != TopicInProduction || topic.StudioProjectID != row.ID {
			t.Fatalf("topic %+v %v", topic, terr)
		}
	}
}

func TestStartTopicFilmsSkipsFinished(t *testing.T) {
	e := testEngine(t)
	topic, err := e.SaveTopic(Topic{Title: "已经拍过", SourcePlatform: "douyin"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.CreateStudioProject(StudioProject{Title: topic.Title, TopicID: topic.ID})
	if err != nil {
		t.Fatal(err)
	}
	p.ResultHash = "already"
	p.AutoPhase = "done"
	p.Status = StudioAssembled
	if _, err := e.saveStudio(p); err != nil {
		t.Fatal(err)
	}
	rows, err := e.StartTopicFilms([]string{topic.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != p.ID || rows[0].AutoPhase != "done" || rows[0].ResultHash != "already" {
		t.Fatalf("%+v", rows)
	}
}

func TestStartTopicFilmsRejectsEmptyAndHuge(t *testing.T) {
	e := testEngine(t)
	if _, err := e.StartTopicFilms(nil); err == nil || !strings.Contains(err.Error(), "选题") {
		t.Fatal(err)
	}
	ids := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	if _, err := e.StartTopicFilms(ids); err == nil || !strings.Contains(err.Error(), "8") {
		t.Fatal(err)
	}
}

func TestFinishAutoPacksAndClosesTopic(t *testing.T) {
	e := testEngine(t)
	topic, err := e.SaveTopic(Topic{Title: "热搜甲", SourcePlatform: "douyin"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.CreateStudioProject(StudioProject{Title: topic.Title, TopicID: topic.ID, Script: "一句口播就够了。"})
	if err != nil {
		t.Fatal(err)
	}
	h, err := e.PutBytes([]byte("mp4"))
	if err != nil {
		t.Fatal(err)
	}
	p.ResultHash = h
	p.Script = "一句口播就够了。"
	if _, err := e.saveStudio(p); err != nil {
		t.Fatal(err)
	}
	e.finishAuto(p.ID, "done", "")
	got, err := e.GetStudioProject(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AutoPhase != "done" || got.Delivery.Status != DeliveryPublishable || got.CopyText == "" {
		t.Fatalf("phase %s delivery %s copy %q", got.AutoPhase, got.Delivery.Status, got.CopyText)
	}
	linked, err := e.GetTopic(topic.ID)
	if err != nil || linked.Status != TopicDone {
		t.Fatalf("%+v %v", linked, err)
	}
}

func waitFilm(t *testing.T, e *Engine, id string) StudioProject {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last StudioProject
	for time.Now().Before(deadline) {
		p, err := e.GetStudioProject(id)
		if err == nil {
			last = p
			if p.AutoPhase == "done" || p.AutoPhase == "failed" {
				return p
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("phase %s err %s", last.AutoPhase, last.AutoError)
	return last
}
