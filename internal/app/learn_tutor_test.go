// 锁住学习对话里导师这一层：学习工具一开场就在工具清单里，模式在调用时检查，出题会收轮且不带答案，
// 卡片上判完的题只向导师通报一次，判分用的是学习者的原话而不是模型编的答案。

package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/learn"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
)

// learnDesk 打开一个应用，建一段绑了路径的学习对话，并返回挂好学习工具的工具集。
func learnDesk(t *testing.T) (*App, SessionMeta, string, map[string]string, *runtime.WorkspaceTools) {
	t.Helper()
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	sess, err := a.NewSessionOn(t.TempDir(), session.ChannelLearn)
	if err != nil {
		t.Fatal(err)
	}
	view, err := a.Learn.CreatePath(learn.CreatePathRequest{Goal: "学会导数"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Learn.ReplaceOutline(view.ID, []learn.OutlineModule{{Name: "基础", Points: []learn.OutlinePoint{
		{Name: "极限", Kind: learn.KindMemory}, {Name: "连续", Kind: learn.KindConcept},
	}}}, learn.OutlineOptions{}); err != nil {
		t.Fatal(err)
	}
	meta, err := a.SetSessionLearn(sess.ID, LearnBinding{PathID: view.ID})
	if err != nil {
		t.Fatal(err)
	}
	full, _ := a.Learn.Path(view.ID)
	ids := map[string]string{}
	for _, p := range full.Modules[0].Points {
		ids[p.Name] = p.ID
	}
	tools := &runtime.WorkspaceTools{SessionID: sess.ID, ChatOverlay: true}
	a.attachLearn(tools)
	return a, meta, view.ID, ids, tools
}

// callLearn 调一个学习工具，并把成功的结果解成 map。
func callLearn(t *testing.T, tools *runtime.WorkspaceTools, name string, args map[string]any) (map[string]any, runtime.ToolResult) {
	t.Helper()
	extra, ok := tools.Extra[name]
	if !ok {
		t.Fatalf("tool %s is not mounted", name)
	}
	raw, _ := json.Marshal(args)
	res := extra.Call(string(raw))
	out := map[string]any{}
	if res.Err == nil {
		if err := json.Unmarshal([]byte(res.Content), &out); err != nil {
			t.Fatalf("%s returned non-JSON: %s", name, res.Content)
		}
	}
	return out, res
}

// TestLearnToolsAdvertisedUpFront 确认绑了路径的学习对话开场的工具清单里就有学习工具，不用先搜。
func TestLearnToolsAdvertisedUpFront(t *testing.T) {
	_, _, _, _, tools := learnDesk(t)
	listed := map[string]bool{}
	for _, spec := range runtime.AllToolJSON(tools) {
		name, _ := spec.Function["name"].(string)
		listed[name] = true
	}
	for _, name := range []string{"learn_status", "learn_mode", "learn_profile", "learn_build", "learn_revise", "learn_quiz", "learn_grade", "learn_assess", "learn_skip", "learn_repair", "learn_defer", "learn_journal"} {
		if !listed[name] {
			t.Fatalf("%s is mounted but not in the opening tool list", name)
		}
	}
	if listed["learn_search"] {
		t.Fatal("library tools must stay off a chat with no library bound")
	}
}

// TestLearnToolsStayOffAgentChat 确认会话模块里的普通对话不挂学习工具。
func TestLearnToolsStayOffAgentChat(t *testing.T) {
	a, _, _, _, _ := learnDesk(t)
	plain, err := a.NewSession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tools := &runtime.WorkspaceTools{SessionID: plain.ID, ChatOverlay: true}
	a.attachLearn(tools)
	if len(tools.Extra) != 0 {
		t.Fatalf("agent chat got learn tools: %v", tools.Extra)
	}
}

// TestQuizEndsTurnAndHidesKey 确认模式在调用时检查，出题收轮，返回里没有标准答案，第二次出题只重现原来那道。
func TestQuizEndsTurnAndHidesKey(t *testing.T) {
	a, _, path, ids, tools := learnDesk(t)
	quiz := map[string]any{
		"point_id": ids["极限"], "question": "极限是什么", "expected_answer": "B", "explanation": "secret-why",
		"options": []map[string]any{{"label": "A", "body": "斜率"}, {"label": "B", "body": "趋近的值"}},
		"path_id": "someone-else",
	}
	if _, res := callLearn(t, tools, "learn_quiz", quiz); res.Err == nil || !strings.Contains(res.Err.Error(), "learn_mode") {
		t.Fatalf("outline mode must refuse a quiz and say how to switch: %+v", res)
	}
	if out, res := callLearn(t, tools, "learn_mode", map[string]any{"mode": "study"}); res.Err != nil || out["mode"] != "study" || res.Halt {
		t.Fatalf("%v %+v", out, res)
	}
	out, res := callLearn(t, tools, "learn_quiz", quiz)
	if res.Err != nil || !res.Halt || out["status"] != "posed" {
		t.Fatalf("a posed question must end the turn: %v %+v", out, res)
	}
	if strings.Contains(res.Content, "secret-why") || strings.Contains(res.Content, "expected") {
		t.Fatalf("the key leaked: %s", res.Content)
	}
	again, res := callLearn(t, tools, "learn_quiz", map[string]any{"point_id": ids["极限"], "question": "另一题", "expected_answer": "x"})
	if res.Err != nil || !res.Halt || again["status"] != "already_pending" {
		t.Fatalf("%v %+v", again, res)
	}
	view, _ := a.Learn.Path(path)
	if view.Next.Action != learn.ActionAnswer || view.Next.Question.Prompt != "极限是什么" {
		t.Fatalf("%+v", view.Next)
	}
	// 复习模式只回头检验已掌握的点。
	if err := a.Learn.SkipQuestion(path); err != nil {
		t.Fatal(err)
	}
	if err := a.Learn.SetMode(path, learn.ModeReview); err != nil {
		t.Fatal(err)
	}
	if _, res := callLearn(t, tools, "learn_quiz", quiz); res.Err == nil || !strings.Contains(res.Err.Error(), "not mastered") {
		t.Fatalf("review must refuse an unmastered point: %+v", res)
	}
}

// TestGradeUsesLearnerWords 确认聊天里作答时判的是学习者的原话。模型编的答案不在原话里就不用它。
func TestGradeUsesLearnerWords(t *testing.T) {
	a, _, path, ids, tools := learnDesk(t)
	if err := a.Learn.SetMode(path, learn.ModeStudy); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.Learn.PoseQuiz(learn.PoseRequest{PathID: path, PointID: ids["极限"], Prompt: "问", Expected: "趋近的值"}); err != nil {
		t.Fatal(err)
	}
	a.Learn.Complete = func(_ context.Context, req learn.CompleteRequest) (string, error) {
		if strings.Contains(req.User, "<answer>我不知道") {
			return `{"result":"incorrect","confidence":0.95,"feedback":"还没答到"}`, nil
		}
		return `{"result":"correct","confidence":0.95,"feedback":"对"}`, nil
	}
	tools.OperatorVoice = "我不知道，大概是斜率？"
	out, res := callLearn(t, tools, "learn_grade", map[string]any{"answer": "趋近的值"})
	if res.Err != nil || out["result"] != learn.ResultIncorrect {
		t.Fatalf("the model must not answer for the learner: %v %+v", out, res)
	}
	if brief := a.Learn.TakeTutorBrief(path); brief.Graded != nil {
		t.Fatal("a grade the tutor made itself is not reported back to it")
	}
	if got := learnerWords("斜率", "我觉得是 斜率 吧"); got != "斜率" {
		t.Fatalf("a quoted part of the message is accepted: %q", got)
	}
	if got := learnerWords("", "整句话"); got != "整句话" {
		t.Fatalf("%q", got)
	}
}

// TestAssessReadsLearnerMessage 确认讲解测评读的是学习者这一轮的原话，记忆型知识点不能走讲解。
func TestAssessReadsLearnerMessage(t *testing.T) {
	a, _, path, ids, tools := learnDesk(t)
	if err := a.Learn.SetMode(path, learn.ModeStudy); err != nil {
		t.Fatal(err)
	}
	var seen string
	a.Learn.Complete = func(_ context.Context, req learn.CompleteRequest) (string, error) {
		seen = req.User
		return `{"result":"correct","confidence":0.9,"feedback":"讲清楚了","gaps":[]}`, nil
	}
	tools.OperatorVoice = "连续就是这一点的极限等于函数值，所以图像不断开"
	out, res := callLearn(t, tools, "learn_assess", map[string]any{"point_id": ids["连续"], "note": "要提到极限"})
	if res.Err != nil || out["mastered"] != true {
		t.Fatalf("%v %+v", out, res)
	}
	if !strings.Contains(seen, "图像不断开") || !strings.Contains(seen, "要提到极限") {
		t.Fatalf("the grader must read the learner's own message: %s", seen)
	}
	if _, res := callLearn(t, tools, "learn_assess", map[string]any{"point_id": ids["极限"]}); res.Err == nil {
		t.Fatal("a memory point must not take an explanation check")
	}
}

// TestTurnBriefReportsCardGradeOnce 确认卡片上判完的题在下一轮的简报里出现一次，简报带着模式和下一步。
func TestTurnBriefReportsCardGradeOnce(t *testing.T) {
	a, meta, path, ids, _ := learnDesk(t)
	if brief := a.learnTurnBrief(meta); !strings.Contains(brief, `"mode":"outline"`) || !strings.Contains(brief, "Mode is outline") || !strings.Contains(brief, `"intake_needed":true`) {
		t.Fatalf("%s", brief)
	}
	if err := a.Learn.SetMode(path, learn.ModeStudy); err != nil {
		t.Fatal(err)
	}
	q, _, err := a.Learn.PoseQuiz(learn.PoseRequest{PathID: path, PointID: ids["极限"], Prompt: "问", Expected: "a"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := a.LearnCall("learn.path.answer", map[string]any{"id": path, "question_id": q.ID, "answer": "A", "session": meta.ID})
	if err != nil {
		t.Fatal(err)
	}
	if res, ok := out.(learn.GradeResult); !ok || !res.Correct || res.Expected != "a" {
		t.Fatalf("%+v", out)
	}
	brief := a.learnTurnBrief(meta)
	if !strings.Contains(brief, `"graded"`) || !strings.Contains(brief, "Do not call learn_grade") || !strings.Contains(brief, `"result":"correct"`) {
		t.Fatalf("%s", brief)
	}
	if strings.Contains(brief, `"expected"`) {
		t.Fatalf("the brief does not need the key: %s", brief)
	}
	if again := a.learnTurnBrief(meta); strings.Contains(again, `"graded"`) {
		t.Fatalf("a card grade is reported once: %s", again)
	}
	if a.learnTurnBrief(SessionMeta{}) != "" {
		t.Fatal("a chat with no path gets no brief")
	}
}

// TestLearnLanePromptPins 确认学习对话换上导师的开场引导，绑了路径才带路径守则，并去掉巡视工作区的片段。
func TestLearnLanePromptPins(t *testing.T) {
	frags := []artifact.PromptFragment{{Slot: "bootstrap", Text: "First inspect the workspace."}, {Slot: "execution", Text: "keep"}}
	loop, got := applyLearnLanePrompt(artifact.LoopPreset{Bootstrap: "old"}, frags, SessionMeta{PathID: "p", LibraryID: "l"})
	text := runtime.AssembleIdentity(loop, got)
	for _, want := range []string{"tutoring session", "## Tutoring", "## Mastery path", "## Sources", "keep"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in\n%s", want, text)
		}
	}
	if strings.Contains(text, "inspect the workspace.") || strings.Contains(text, "old") {
		t.Fatalf("workspace bootstrap survived:\n%s", text)
	}
	_, plain := applyLearnLanePrompt(artifact.LoopPreset{}, nil, SessionMeta{})
	if text := runtime.AssembleIdentity(artifact.LoopPreset{}, plain); strings.Contains(text, "## Mastery path") || strings.Contains(text, "## Sources") {
		t.Fatalf("an unbound chat carries only the tutoring pin:\n%s", text)
	}
}

// TestLearnCallMasteryRoundTrip 确认界面用的方法能建路径、写大纲、改档案，未知方法报错。
func TestLearnCallMasteryRoundTrip(t *testing.T) {
	a, _, _, _, _ := learnDesk(t)
	out, err := a.LearnCall("learn.path.create", map[string]any{"goal": "学会积分", "sources": []any{map[string]any{"kind": "library", "ref": "lib-1", "label": "讲义"}}})
	if err != nil {
		t.Fatal(err)
	}
	view := out.(learn.PathView)
	if len(view.Sources) != 1 || view.Mode != learn.ModeOutline {
		t.Fatalf("%+v", view)
	}
	modules := []any{map[string]any{"name": "章", "points": []any{map[string]any{"name": "换元", "kind": "procedure"}}}}
	if _, err := a.LearnCall("learn.path.outline", map[string]any{"id": view.ID, "modules": modules}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.LearnCall("learn.path.profile", map[string]any{"id": view.ID, "target_level": "能做题"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.LearnCall("learn.path.retention", map[string]any{"id": view.ID, "retention": 0.85}); err != nil {
		t.Fatal(err)
	}
	got, err := a.Learn.Path(view.ID)
	if err != nil || got.Counts.Total != 1 || got.Profile.TargetLevel != "能做题" || got.Profile.Teaching != learn.TeachProbe || got.Retention != 0.85 {
		t.Fatalf("%+v %v", got, err)
	}
	list, err := a.LearnCall("learn.paths", nil)
	if err != nil || len(list.([]learn.PathSummary)) != 2 {
		t.Fatalf("%v %v", list, err)
	}
	if _, err := a.LearnCall("learn.nope", nil); err == nil {
		t.Fatal("unknown method accepted")
	}
}
