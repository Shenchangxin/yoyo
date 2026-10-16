package video

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestStudioJobShouldSkip(t *testing.T) {
	draft := StudioProject{Status: StudioDraft, Gate: GateScript, ScriptStatus: ScriptNeedsReview}
	if StudioJobShouldSkip(draft, "") {
		t.Fatal("empty draft should still be shootable")
	}
	if !StudioJobShouldSkip(draft, "sess-1") {
		t.Fatal("bound chat must skip the timer")
	}
	approved := StudioProject{ScriptStatus: ScriptApproved, ResultHash: ""}
	if !StudioJobShouldSkip(approved, "") {
		t.Fatal("approved script must skip")
	}
	cut := StudioProject{ResultHash: "abc"}
	if !StudioJobShouldSkip(cut, "") {
		t.Fatal("assembled cut must skip")
	}
}

func TestTopicScheduledStatus(t *testing.T) {
	e := testEngine(t)
	got, err := e.SaveTopic(Topic{Title: "定时选题", Status: TopicScheduled})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != TopicScheduled {
		t.Fatalf("status %s", got.Status)
	}
	listed := e.ListTopics(TopicScheduled)
	if len(listed) != 1 || listed[0].ID != got.ID {
		t.Fatalf("list scheduled %+v", listed)
	}
	queued, err := e.SaveTopic(Topic{ID: got.ID, Title: got.Title, Status: "queued"})
	if err != nil {
		t.Fatal(err)
	}
	if queued.Status != TopicScheduled {
		t.Fatalf("queued alias %s", queued.Status)
	}
}

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

func TestStudioBindOneToOne(t *testing.T) {
	e := testEngine(t)
	a, err := e.CreateStudioProject(StudioProject{Title: "短片A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.CreateStudioProject(StudioProject{Title: "短片B"})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.BindStudioSession("sess-a", a.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.BindStudioSession("sess-b", b.ID); err != nil {
		t.Fatal(err)
	}
	if id, ok := e.StudioSessionBind("sess-a"); !ok || id != a.ID {
		t.Fatalf("a bind %s %v", id, ok)
	}
	if id, ok := e.StudioSessionBind("sess-b"); !ok || id != b.ID {
		t.Fatalf("b bind %s %v", id, ok)
	}
	if got := e.LatestStudioSession(a.ID); got != "sess-a" {
		t.Fatalf("latest a %s", got)
	}
	if got := e.LatestStudioSession(b.ID); got != "sess-b" {
		t.Fatalf("latest b %s", got)
	}
	if err := e.BindStudioSession("sess-a", b.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.StudioSessionBind("sess-a"); !ok {
		t.Fatal("session should stay bound")
	}
	if id, _ := e.StudioSessionBind("sess-a"); id != b.ID {
		t.Fatalf("rebind moved to b, got %s", id)
	}
	if _, ok := e.StudioSessionBind("sess-b"); ok {
		t.Fatal("short b should keep one copilot; previous session must unbind")
	}
	if got := e.LatestStudioSession(a.ID); got != "" {
		t.Fatalf("short a must not keep the stolen chat, got %s", got)
	}
	if got := e.LatestStudioSession(b.ID); got != "sess-a" {
		t.Fatalf("short b copilot %s", got)
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

func TestProviderTemplatesIncludeMusicStockAsr(t *testing.T) {
	type hit struct{ kind, vendor, name string }
	seen := map[hit]bool{}
	for _, p := range ProviderTemplates() {
		seen[hit{p.ServiceType, p.Provider, p.Name}] = true
		if p.ServiceType == "music" && strings.Contains(strings.ToLower(p.Model), "tts") {
			t.Fatalf("music template still points at a speech model: %+v", p)
		}
	}
	for _, want := range []hit{
		{"music", "minimax", "MiniMax Music"},
		{"music", "elevenlabs", "ElevenLabs Sound"},
		{"music", "stability", "Stability Audio"},
		{"music", "gemini", "Google Lyria"},
		{"music", "suno", "Suno compatible"},
		{"stock", "unsplash", "Unsplash"},
		{"stock", "pexels", "Pexels Photos"},
		{"stock", "pexels", "Pexels Videos"},
		{"stock", "pixabay", "Pixabay Photos"},
		{"stock", "pixabay", "Pixabay Videos"},
		{"stock", "giphy", "Giphy"},
		{"asr", "openai", "OpenAI Transcription"},
		{"asr", "groq", "Groq Whisper"},
		{"asr", "deepgram", "Deepgram"},
		{"asr", "assemblyai", "AssemblyAI"},
		{"asr", "azure", "Azure Transcription"},
		{"asr", "aliyun", "DashScope Paraformer"},
		{"image", "aliyun", "Qwen Image"},
		{"image", "jimeng", "Jimeng Image"},
		{"image", "kolors", "Kolors"},
		{"image", "zhipu", "CogView"},
		{"image", "tencent", "Hunyuan Image"},
		{"video", "jimeng", "Jimeng Video"},
		{"video", "vidu", "Vidu"},
		{"video", "tencent", "Hunyuan Video"},
		{"video", "zhipu", "CogVideoX"},
		{"video", "kling", "Kling Video"},
		{"tts", "aliyun", "Qwen CosyVoice"},
		{"tts", "spark", "Spark Speech"},
		{"music", "mureka", "Mureka"},
		{"asr", "volcengine", "Volcengine ASR"},
		{"asr", "spark", "Spark ASR"},
		{"asr", "tencent", "Tencent ASR"},
		{"search", "metaso", "Metaso"},
		{"storage", "aliyun", "Aliyun OSS"},
		{"storage", "tencent", "Tencent COS"},
	} {
		if !seen[want] {
			t.Fatalf("missing template %+v", want)
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

func TestStudioPipelineGatesApprovePackage(t *testing.T) {
	e := testEngine(t)
	p, err := e.CreateStudioProject(StudioProject{Title: "钩子测试", Aspect: "9:16"})
	if err != nil {
		t.Fatal(err)
	}
	if p.FurthestGate() != GateScript || p.Identity.CharacterID == "" || p.PresetID == "" {
		t.Fatalf("create %+v furthest=%s", p, p.FurthestGate())
	}
	if _, err := e.ApproveStudioScript(p.ID); err == nil {
		t.Fatal("approve empty script")
	}
	p.Script = "开头三秒报选题。中间讲一个可核验的点。结尾告诉你下一步。"
	if _, err := e.UpdateStudioProject(p); err != nil {
		t.Fatal(err)
	}
	approved, err := e.ApproveStudioScript(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if approved.ScriptStatus != ScriptApproved || approved.FurthestGate() != GateAudio {
		t.Fatalf("approved %+v gate=%s", approved.ScriptStatus, approved.FurthestGate())
	}
	if len(approved.Segments) < 2 || len(approved.Storyboard.Shots) == 0 {
		t.Fatalf("segments/shots %+v %+v", approved.Segments, approved.Storyboard.Shots)
	}
	skipped, err := e.SkipStudioAudio(approved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if skipped.FurthestGate() != GateVisuals {
		t.Fatalf("skip audio gate %s", skipped.FurthestGate())
	}
	ident, err := e.SetStudioIdentity(skipped.ID, "narrator", "paper")
	if err != nil || ident.Identity.CharacterID != "narrator" {
		t.Fatalf("identity %+v %v", ident.Identity, err)
	}
	preset, err := e.ApplyStudioPreset(ident.ID, "bilibili")
	if err != nil || preset.Aspect != "16:9" || preset.Platform != "bilibili" {
		t.Fatalf("preset %+v %v", preset, err)
	}
	tpl, err := e.ApplyStudioTemplate(preset.ID, "explainer")
	if err != nil || tpl.Tone == "" {
		t.Fatalf("template %+v %v", tpl, err)
	}
	h, err := e.PutBytes([]byte("fake-still"))
	if err != nil {
		t.Fatal(err)
	}
	shots := tpl.Storyboard.Shots
	for i := range shots {
		shots[i].ImageHash = h
		shots[i].Stale = false
	}
	tpl.Storyboard.Shots = shots
	tpl, err = e.UpdateStudioProject(tpl)
	if err != nil {
		t.Fatal(err)
	}
	if !tpl.VisualsReady() || tpl.FurthestGate() != GateCompose {
		t.Fatalf("visuals gate %s ready=%v", tpl.FurthestGate(), tpl.VisualsReady())
	}
	vid, err := e.PutBytes([]byte("fake-mp4-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	tpl.ResultHash = vid
	tpl, err = e.UpdateStudioProject(tpl)
	if err != nil {
		t.Fatal(err)
	}
	packed, err := e.PackageStudio(tpl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if packed.Delivery.Status != DeliveryPublishable || packed.CopyHash == "" || packed.FurthestGate() != GateDelivery {
		t.Fatalf("pack %+v", packed.Delivery)
	}
	if len(packed.Versions) == 0 {
		t.Fatal("expected snapshot")
	}
	n := packed.Versions[0].N
	restored, err := e.RestoreStudioVersion(packed.ID, n)
	if err != nil || restored.ResultHash != vid {
		t.Fatalf("restore %+v %v", restored.ResultHash, err)
	}
}

func TestStudioTopicExpandAndCopy(t *testing.T) {
	e := testEngine(t)
	tpc, err := e.SaveTopic(Topic{Title: "春日穿搭", Note: "热度高"})
	if err != nil {
		t.Fatal(err)
	}
	exp, err := e.ExpandTopic(tpc.ID)
	if err != nil || len(exp.Options) < 3 {
		t.Fatalf("expand %+v %v", exp.Options, err)
	}
	pick := exp.Options[1]
	got, err := e.ChooseTopicOption(tpc.ID, pick.ID)
	if err != nil || got.Angle != pick.Angle {
		t.Fatalf("choose %+v %v", got, err)
	}
	p, err := e.CreateStudioProject(StudioProject{Title: got.Title, Script: "可核验的三点。", Platform: "xhs"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := e.WriteStudioCopy(p.ID, "xhs")
	if err != nil || !strings.Contains(out.CopyText, "小红书") {
		t.Fatalf("copy %s %v", out.CopyText, err)
	}
}

func TestStudioStaleShotOnVisualEdit(t *testing.T) {
	e := testEngine(t)
	h, err := e.PutBytes([]byte("still"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.CreateStudioProject(StudioProject{
		Title:      "t",
		Storyboard: StudioStoryboard{Shots: []StudioShot{{ID: "s1", ImageHash: h, Visual: "desk", Duration: 3}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	p.Storyboard.Shots[0].Visual = "window"
	got, err := e.UpdateStudioProject(p)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Storyboard.Shots[0].Stale || got.Storyboard.Shots[0].ImageHash != h {
		t.Fatalf("want stale keep hash %+v", got.Storyboard.Shots[0])
	}
}

func TestStudioCatalogAndTools(t *testing.T) {
	e := testEngine(t)
	raw, err := e.StudioTool("", "studio_catalog", nil)
	if err != nil || !strings.Contains(raw, "douyin-shorts") {
		t.Fatalf("%s %v", raw, err)
	}
	created, err := e.StudioTool("s2", "studio_create", map[string]any{"title": "目录短片", "template_id": "howto"})
	if err != nil {
		t.Fatal(err)
	}
	var p StudioProject
	if err := json.Unmarshal([]byte(created), &p); err != nil || p.ID == "" {
		t.Fatalf("%s %v", created, err)
	}
	if p.Identity.CharacterID == "" || p.TemplateID != "howto" {
		t.Fatalf("identity %+v template %q", p.Identity, p.TemplateID)
	}
	names := strings.Join(StudioToolNames(), " ")
	for _, w := range []string{"studio_approve_script", "studio_generate_shot", "studio_package", "studio_ingest", "studio_autoclip", "studio_expand_topic"} {
		if !strings.Contains(names, w) {
			t.Fatalf("missing %s", w)
		}
	}
	pf, err := e.StudioPreflight(p.ID, GateVisuals)
	if err != nil {
		t.Fatal(err)
	}
	if pf.Gate != GateVisuals {
		t.Fatalf("%+v", pf)
	}
}

func TestParseSRTDurations(t *testing.T) {
	got := parseSRTDurations("1\n00:00:00,000 --> 00:00:02,500\nhi\n\n2\n00:00:02,500 --> 00:00:05,000\nthere\n")
	if len(got) != 2 || got[0] < 2.4 || got[0] > 2.6 || got[1] < 2.4 {
		t.Fatalf("%v", got)
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
