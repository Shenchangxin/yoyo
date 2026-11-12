// 锁住掌握路径的不变量：答案不外泄、三次独立答对才过门、提示不计分、没法判定不动分、前置门控、
// 改大纲保住证据、到期复习与遗忘、声明要经核实、作废后重算，以及证据回放和一路做下来的结果相同。

package learn

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// clockAt 给 s 装上一只可以拨动的钟，返回拨钟的函数。
func clockAt(s *Store, start time.Time) func(d time.Duration) {
	cur := start
	s.Now = func() time.Time { return cur }
	return func(d time.Duration) { cur = cur.Add(d) }
}

// judgeWith 给 s 装上一个假的判分模型。reply 按用户提示里的内容给出回答。
func judgeWith(s *Store, reply func(req CompleteRequest) string) *int {
	calls := 0
	s.Complete = func(_ context.Context, req CompleteRequest) (string, error) {
		calls++
		return reply(req), nil
	}
	return &calls
}

// strictJudge 装上一个把对不上原文的作答一律判错的假模型。
func strictJudge(s *Store) {
	judgeWith(s, func(CompleteRequest) string { return `{"result":"incorrect","confidence":0.95,"feedback":"不对"}` })
}

// newPath 建一条路径并写入大纲，返回路径 id 和按名字查知识点 id 的表。
func newPath(t *testing.T, s *Store, modules []OutlineModule) (string, map[string]string) {
	t.Helper()
	view, err := s.CreatePath(CreatePathRequest{Goal: "学会测试"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReplaceOutline(view.ID, modules, OutlineOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMode(view.ID, ModeStudy); err != nil {
		t.Fatal(err)
	}
	return view.ID, pointIDs(t, s, view.ID)
}

func pointIDs(t *testing.T, s *Store, pathID string) map[string]string {
	t.Helper()
	view, err := s.Path(pathID)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, m := range view.Modules {
		for _, p := range m.Points {
			ids[p.Name] = p.ID
		}
	}
	return ids
}

func pointOf(t *testing.T, s *Store, pathID, pointID string) PointView {
	t.Helper()
	view, err := s.Path(pathID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range view.Modules {
		for _, p := range m.Points {
			if p.ID == pointID {
				return p
			}
		}
	}
	t.Fatalf("point %s missing", pointID)
	return PointView{}
}

// ask 出一道简答题并用 answer 作答，返回判定。
func ask(t *testing.T, s *Store, pathID, pointID, expected, answer string) GradeResult {
	t.Helper()
	if _, _, err := s.PoseQuiz(PoseRequest{PathID: pathID, PointID: pointID, Prompt: "问", Expected: expected, Type: QuestionShort}); err != nil {
		t.Fatal(err)
	}
	res, err := s.Answer(context.Background(), AnswerRequest{PathID: pathID, Answer: answer})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func memoryOutline() []OutlineModule {
	return []OutlineModule{{Name: "基础", Points: []OutlinePoint{{Name: "极限", Kind: KindMemory}, {Name: "导数", Kind: KindMemory}}}}
}

// TestQuizHidesExpectedAnswer 确认出题的返回值和地图里的下一步都不带标准答案和解析。
func TestQuizHidesExpectedAnswer(t *testing.T) {
	s := openTemp(t)
	path, ids := newPath(t, s, memoryOutline())
	pub, already, err := s.PoseQuiz(PoseRequest{PathID: path, PointID: ids["极限"], Prompt: "极限是什么", Expected: "secret-answer", Explanation: "secret-why"})
	if err != nil || already {
		t.Fatal(already, err)
	}
	view, err := s.Path(path)
	if err != nil || view.Next == nil || view.Next.Action != ActionAnswer || view.Next.Question == nil {
		t.Fatalf("%+v %v", view.Next, err)
	}
	for _, v := range []any{pub, view} {
		raw, _ := json.Marshal(v)
		if strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "expected") {
			t.Fatal(string(raw))
		}
	}
	again, already, err := s.PoseQuiz(PoseRequest{PathID: path, PointID: ids["导数"], Prompt: "第二题", Expected: "x"})
	if err != nil || !already || again.ID != pub.ID {
		t.Fatalf("a second question must re-present the pending one: %+v %v %v", again, already, err)
	}
	// 题卡读到的状态在判完之前也不带答案，判完之后才带。
	state, err := s.QuestionState(path, pub.ID)
	if err != nil || state.Status != "pending" || state.Expected != "" || state.Explanation != "" {
		t.Fatalf("%+v %v", state, err)
	}
	if _, err := s.Answer(context.Background(), AnswerRequest{PathID: path, Answer: "secret-answer"}); err != nil {
		t.Fatal(err)
	}
	state, err = s.QuestionState(path, pub.ID)
	if err != nil || state.Status != "graded" || state.Result != ResultCorrect || state.Expected != "secret-answer" || state.Explanation != "secret-why" || state.Answer != "secret-answer" {
		t.Fatalf("%+v %v", state, err)
	}
	if _, err := s.QuestionState(path, "q-missing"); err == nil {
		t.Fatal("an unknown question must not resolve")
	}
}

// TestGateNeedsThreeIndependentCorrect 确认一次答对封顶 0.5、两次 0.8，第三次才过门并排上复习。
func TestGateNeedsThreeIndependentCorrect(t *testing.T) {
	s := openTemp(t)
	clockAt(s, time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC))
	path, ids := newPath(t, s, memoryOutline())
	id := ids["极限"]
	for i, want := range []float64{0.5, 0.8} {
		res := ask(t, s, path, id, "paris", "Paris")
		if !res.Correct || !res.Final || res.Mastered || res.Mastery != want || res.Status != StatusLearning {
			t.Fatalf("attempt %d: %+v", i+1, res)
		}
		if res.Expected != "paris" {
			t.Fatalf("a final grade reveals the key: %+v", res)
		}
	}
	res := ask(t, s, path, id, "paris", "paris.")
	if !res.Mastered || res.Status != StatusMastered || res.Mastery < quantitativeGate {
		t.Fatalf("%+v", res)
	}
	p := pointOf(t, s, path, id)
	if p.Source != "system" || p.DueAt == "" {
		t.Fatalf("%+v", p)
	}
	// 间隔取整到天：记忆类的初始稳定度是 2.3 天，第一次复习排在两个整天之后。
	due := parseTime(p.DueAt)
	if got := due.Sub(s.clock()).Hours() / 24; got != 2 {
		t.Fatalf("first review in %.3f days, want 2 whole days for a stability of %.3f", got, fsrsWeights[2])
	}
	if res.Next == nil || res.Next.PointID != ids["导数"] || res.Next.Action != ActionProbe {
		t.Fatalf("next %+v", res.Next)
	}
}

// TestHintedAnswerDoesNotCount 确认看过提示才答对的题不计入掌握分。
func TestHintedAnswerDoesNotCount(t *testing.T) {
	s := openTemp(t)
	path, ids := newPath(t, s, memoryOutline())
	if _, _, err := s.PoseQuiz(PoseRequest{PathID: path, PointID: ids["极限"], Prompt: "问", Expected: "a", Hinted: true}); err != nil {
		t.Fatal(err)
	}
	res, err := s.Answer(context.Background(), AnswerRequest{PathID: path, Answer: "a"})
	if err != nil || !res.Correct || res.Mastery != 0 || res.Status != StatusLearning {
		t.Fatalf("%+v %v", res, err)
	}
}

// TestChoiceShuffleAndUnreadable 确认选项打乱后正确项的字母跟着变，认不出选了哪项时不计分，题目继续等。
func TestChoiceShuffleAndUnreadable(t *testing.T) {
	s := openTemp(t)
	path, ids := newPath(t, s, memoryOutline())
	opts := []Option{{Label: "A", Body: "红"}, {Label: "B", Body: "绿"}, {Label: "C", Body: "蓝"}, {Label: "D", Body: "黄"}}
	pub, _, err := s.PoseQuiz(PoseRequest{PathID: path, PointID: ids["极限"], Prompt: "天空的颜色", Expected: "C", Options: opts})
	if err != nil || pub.Type != QuestionChoice || len(pub.Options) != 4 {
		t.Fatalf("%+v %v", pub, err)
	}
	correct := ""
	for _, o := range pub.Options {
		if o.Body == "蓝" {
			correct = o.Label
		}
	}
	res, err := s.Answer(context.Background(), AnswerRequest{PathID: path, Answer: "为什么不是绿？"})
	if err != nil || res.Final || res.Result != ResultUngraded || res.Mastery != 0 {
		t.Fatalf("an unreadable pick must not be graded: %+v %v", res, err)
	}
	view, _ := s.Path(path)
	if view.Next.Action != ActionAnswer {
		t.Fatalf("question must still wait: %+v", view.Next)
	}
	res, err = s.Answer(context.Background(), AnswerRequest{PathID: path, Answer: "我选 " + correct})
	if err != nil || !res.Correct || !res.Final {
		t.Fatalf("%+v %v", res, err)
	}
	if _, _, err := s.PoseQuiz(PoseRequest{PathID: path, PointID: ids["极限"], Prompt: "问", Expected: "E", Options: opts}); err == nil {
		t.Fatal("an expected answer outside the options must be refused")
	}
	if _, _, err := s.PoseQuiz(PoseRequest{PathID: path, PointID: ids["极限"], Prompt: "问", Expected: "A", Options: []Option{{Body: "x"}, {Body: "X"}}}); err == nil {
		t.Fatal("duplicate option bodies must be refused")
	}
}

// TestJudgeGradesByMeaning 确认对不上原文的简答交给模型判，把握不足时不计分，没接模型时返回错误而不是判错。
func TestJudgeGradesByMeaning(t *testing.T) {
	s := openTemp(t)
	path, ids := newPath(t, s, memoryOutline())
	id := ids["极限"]
	if _, _, err := s.PoseQuiz(PoseRequest{PathID: path, PointID: id, Prompt: "问", Expected: "函数在某点附近趋近的值"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Answer(context.Background(), AnswerRequest{PathID: path, Answer: "靠近时趋向的数"}); err == nil {
		t.Fatal("without a model a free answer cannot be graded")
	}
	calls := judgeWith(s, func(req CompleteRequest) string {
		switch {
		case strings.Contains(req.User, "不确定"):
			return `{"result":"correct","confidence":0.5,"feedback":"再说清楚一点"}`
		case strings.Contains(req.User, "一部分"):
			return "```json\n{\"result\":\"partial\",\"confidence\":0.9,\"feedback\":\"少了附近\"}\n```"
		default:
			return `{"result":"correct","confidence":0.95,"feedback":"对"}`
		}
	})
	res, err := s.Answer(context.Background(), AnswerRequest{PathID: path, Answer: "不确定是不是趋近"})
	if err != nil || res.Final || res.Result != ResultUngraded {
		t.Fatalf("low confidence must stay ungraded: %+v %v", res, err)
	}
	res, err = s.Answer(context.Background(), AnswerRequest{PathID: path, Answer: "只说对一部分"})
	if err != nil || !res.Final || res.Result != ResultPartial || res.Correct || res.Mastery != 0.5 {
		t.Fatalf("%+v %v", res, err)
	}
	if *calls != 2 {
		t.Fatalf("judge calls %d", *calls)
	}
	if _, _, err := s.PoseQuiz(PoseRequest{PathID: path, PointID: id, Prompt: "问", Expected: "趋近的值"}); err != nil {
		t.Fatal(err)
	}
	if res, err := s.Answer(context.Background(), AnswerRequest{PathID: path, Answer: " 趋近的值。"}); err != nil || !res.Correct || *calls != 2 {
		t.Fatalf("an exact match must not call the model: %+v %v calls=%d", res, err, *calls)
	}
	// 答对一部分的题进了题库，标成错题，一天后到期。
	var mistake int
	var due string
	if err := s.db.QueryRow(`SELECT q.mistake, st.due_at FROM questions q JOIN practice_state st ON st.question_id = q.id WHERE q.source = 'mastery'`).Scan(&mistake, &due); err != nil {
		t.Fatal(err)
	}
	if mistake != 1 || parseTime(due).Sub(s.clock()) < 23*time.Hour {
		t.Fatalf("mistake=%d due=%s", mistake, due)
	}
}

// TestQualitativeGate 确认概念型知识点靠讲解过门，答题正确率不算；步骤和记忆型不能走讲解。
func TestQualitativeGate(t *testing.T) {
	s := openTemp(t)
	path, ids := newPath(t, s, []OutlineModule{{Name: "章", Points: []OutlinePoint{{Name: "连续性", Kind: KindConcept}, {Name: "求导步骤", Kind: KindProcedure}}}})
	concept := ids["连续性"]
	for i := 0; i < 3; i++ {
		if res := ask(t, s, path, concept, "a", "a"); res.Mastered {
			t.Fatal("quiz accuracy must not open a qualitative gate")
		}
	}
	if view, _ := s.Path(path); view.Next.Action != ActionAssess || view.Next.PointID != concept {
		t.Fatalf("%+v", view.Next)
	}
	if _, err := s.Assess(context.Background(), AssessRequest{PathID: path, PointID: ids["求导步骤"], Explanation: "先这样再那样，一步一步来"}); err == nil {
		t.Fatal("a procedure point must not take an explanation check")
	}
	if _, err := s.Assess(context.Background(), AssessRequest{PathID: path, PointID: concept, Explanation: "嗯"}); err == nil {
		t.Fatal("an empty explanation must be refused before the model is asked")
	}
	judgeWith(s, func(req CompleteRequest) string {
		if strings.Contains(req.User, "没断开") {
			return `{"result":"correct","confidence":0.9,"feedback":"讲清楚了","gaps":[]}`
		}
		return `{"result":"partial","confidence":0.9,"feedback":"缺了极限","gaps":["没提到极限等于函数值"]}`
	})
	res, err := s.Assess(context.Background(), AssessRequest{PathID: path, PointID: concept, Explanation: "就是图像连着的意思吧大概"})
	if err != nil || !res.Final || res.Mastered || len(res.Gaps) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	res, err = s.Assess(context.Background(), AssessRequest{PathID: path, PointID: concept, Explanation: "在这一点的极限等于函数值，所以图像在这里没断开"})
	if err != nil || !res.Mastered || res.Status != StatusMastered {
		t.Fatalf("%+v %v", res, err)
	}
	if p := pointOf(t, s, path, concept); p.Mastery != 1 || p.Gate != "qualitative" || p.DueAt == "" {
		t.Fatalf("%+v", p)
	}
}

// TestClaimIsVerified 确认学习者的声明让路径往前走但标成 learner，很快排一次复习，核实时答错就撤回。
func TestClaimIsVerified(t *testing.T) {
	s := openTemp(t)
	strictJudge(s)
	advance := clockAt(s, time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC))
	path, ids := newPath(t, s, memoryOutline())
	id := ids["极限"]
	if err := s.ClaimMastery(path, id, true); err != nil {
		t.Fatal(err)
	}
	p := pointOf(t, s, path, id)
	if p.Status != StatusMastered || p.Source != "learner" || p.Mastery != 0 || p.DueAt == "" {
		t.Fatalf("%+v", p)
	}
	view, _ := s.Path(path)
	if view.Next.PointID != ids["导数"] {
		t.Fatalf("a claim must advance routing: %+v", view.Next)
	}
	advance(3 * 24 * time.Hour)
	view, _ = s.Path(path)
	if view.Next.Action != ActionReview || view.Next.PointID != id || view.Due != 1 {
		t.Fatalf("the claim must come up for review: %+v due=%d", view.Next, view.Due)
	}
	if res := ask(t, s, path, id, "a", "b"); res.Mastered || res.Status != StatusLearning {
		t.Fatalf("a failed check must withdraw the claim: %+v", res)
	}
	if p := pointOf(t, s, path, id); p.Source != "" {
		t.Fatalf("%+v", p)
	}
	if err := s.ClaimMastery(path, id, true); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimMastery(path, id, false); err != nil {
		t.Fatal(err)
	}
	if p := pointOf(t, s, path, id); p.Status == StatusMastered {
		t.Fatalf("withdrawn claim still mastered: %+v", p)
	}
}

// TestPrerequisiteGate 确认下一步跳过前置没过门的知识点，前置成环或指向不存在的点时拒绝。
func TestPrerequisiteGate(t *testing.T) {
	s := openTemp(t)
	outline := []OutlineModule{{Name: "章", Points: []OutlinePoint{
		{Name: "积分", Kind: KindMemory, Prereqs: []string{"导数"}},
		{Name: "导数", Kind: KindMemory, Prereqs: []string{"lim"}},
		{Name: "极限", Kind: KindMemory, Ref: "lim"},
	}}}
	path, ids := newPath(t, s, outline)
	view, _ := s.Path(path)
	if view.Next.PointID != ids["极限"] {
		t.Fatalf("the first ready point is 极限: %+v", view.Next)
	}
	blockedCount := 0
	for _, p := range view.Modules[0].Points {
		if p.Blocked {
			blockedCount++
		}
	}
	if blockedCount != 2 {
		t.Fatalf("blocked %d", blockedCount)
	}
	if err := s.ClaimMastery(path, ids["极限"], true); err != nil {
		t.Fatal(err)
	}
	if view, _ = s.Path(path); view.Next.PointID != ids["导数"] {
		t.Fatalf("%+v", view.Next)
	}
	cyc := []OutlineModule{{Name: "章", Points: []OutlinePoint{
		{Name: "甲", Kind: KindMemory, Prereqs: []string{"乙"}},
		{Name: "乙", Kind: KindMemory, Prereqs: []string{"甲"}},
	}}}
	if _, err := s.ReplaceOutline(path, cyc, OutlineOptions{}); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("cycle accepted: %v", err)
	}
	bad := []OutlineModule{{Name: "章", Points: []OutlinePoint{{Name: "甲", Kind: KindMemory, Prereqs: []string{"不存在"}}}}}
	if _, err := s.ReplaceOutline(path, bad, OutlineOptions{}); err == nil {
		t.Fatal("unknown prerequisite accepted")
	}
	if got := pointIDs(t, s, path); len(got) != 3 {
		t.Fatalf("a refused outline must leave the map alone: %v", got)
	}
}

// TestOutlineMergeKeepsEvidence 确认改大纲时留下来的知识点保住 id 和分数，换了位置也一样；删掉的连证据一起清掉。
func TestOutlineMergeKeepsEvidence(t *testing.T) {
	s := openTemp(t)
	strictJudge(s)
	path, ids := newPath(t, s, memoryOutline())
	ask(t, s, path, ids["极限"], "a", "a")
	ask(t, s, path, ids["导数"], "a", "b")
	next := []OutlineModule{
		{Name: "新章", Points: []OutlinePoint{{Name: "积分", Kind: KindProcedure}}},
		{Name: "基础", Points: []OutlinePoint{{Name: "极限", Kind: KindMemory}}},
	}
	change, err := s.ReplaceOutline(path, next, OutlineOptions{})
	if err != nil || change.Kept != 1 || change.Added != 1 || change.Removed != 1 {
		t.Fatalf("%+v %v", change, err)
	}
	after := pointIDs(t, s, path)
	if after["极限"] != ids["极限"] {
		t.Fatal("a surviving point must keep its id")
	}
	if p := pointOf(t, s, path, ids["极限"]); p.Mastery != 0.5 || p.Status != StatusLearning {
		t.Fatalf("evidence lost: %+v", p)
	}
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM evidence WHERE point_id = ?`, ids["导数"]).Scan(&n)
	if n != 0 {
		t.Fatal("a removed point must take its evidence with it")
	}
	// 追加不动现有的章。
	if _, err := s.ReplaceOutline(path, []OutlineModule{{Name: "基础", Points: []OutlinePoint{{Name: "连续", Kind: KindConcept}}}}, OutlineOptions{Append: true}); err != nil {
		t.Fatal(err)
	}
	view, _ := s.Path(path)
	if len(view.Modules) != 2 || view.Modules[1].Total != 2 || view.Counts.Total != 3 {
		t.Fatalf("%+v", view.Modules)
	}
	// 已掌握的点在 KeepMastered 下不能删。
	if err := s.ClaimMastery(path, ids["极限"], true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReviseOutline(path, OutlineEdit{Remove: []string{ids["极限"]}}, true); err == nil {
		t.Fatal("a mastered point must not be removed by the tutor")
	}
	if _, err := s.ReviseOutline(path, OutlineEdit{
		Rename: []OutlineRename{{ID: ids["极限"], Name: "极限的定义"}},
		Add:    []OutlineAdd{{Module: "应用", Name: "最值", Kind: "design", Prereqs: []string{"极限的定义"}}},
	}, true); err != nil {
		t.Fatal(err)
	}
	after = pointIDs(t, s, path)
	if after["极限的定义"] != ids["极限"] || after["最值"] == "" {
		t.Fatalf("%v", after)
	}
}

// TestReviewAndLapse 确认到期的复习排在新内容前面，按时答对把下次间隔拉长，答错记一次遗忘并退回学习中。
func TestReviewAndLapse(t *testing.T) {
	s := openTemp(t)
	strictJudge(s)
	advance := clockAt(s, time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC))
	path, ids := newPath(t, s, memoryOutline())
	id := ids["极限"]
	for i := 0; i < 3; i++ {
		ask(t, s, path, id, "a", "a")
	}
	first := parseTime(pointOf(t, s, path, id).DueAt).Sub(s.clock())
	if view, _ := s.Path(path); view.Next.Action == ActionReview || view.Due != 0 {
		t.Fatalf("nothing is due yet: %+v", view.Next)
	}
	advance(first + time.Hour)
	view, _ := s.Path(path)
	if view.Next.Action != ActionReview || view.Next.PointID != id || len(view.Reviews) != 1 || !view.Reviews[0].Due {
		t.Fatalf("%+v %+v", view.Next, view.Reviews)
	}
	// 间隔取整到天，所以到期时的记住率在目标附近，不一定正好低于它。
	if r := view.Reviews[0].Retrievability; r < 0.85 || r > 0.93 {
		t.Fatalf("a due item should sit near the 0.9 target: %v", r)
	}
	res := ask(t, s, path, id, "a", "a")
	if !res.Mastered {
		t.Fatalf("%+v", res)
	}
	second := parseTime(pointOf(t, s, path, id).DueAt).Sub(s.clock())
	if second <= first*2 {
		t.Fatalf("a timely success must stretch the interval: %v then %v", first, second)
	}
	advance(second + time.Hour)
	res = ask(t, s, path, id, "a", "wrong")
	if res.Mastered || res.Status != StatusLearning {
		t.Fatalf("a failed review falls back to learning: %+v", res)
	}
	detail, err := s.PointDetail(path, id)
	if err != nil || detail.Lapses != 1 || detail.Reps != 3 || len(detail.Evidence) != 5 {
		t.Fatalf("%+v %v", detail, err)
	}
	if detail.Evidence[0].Result != ResultIncorrect || detail.Evidence[0].Expected != "a" {
		t.Fatalf("newest evidence first, with the key: %+v", detail.Evidence[0])
	}
	if view, _ := s.Path(path); view.Next.Action != ActionPractice || view.Next.PointID != id {
		t.Fatalf("%+v", view.Next)
	}
}

// TestRetentionRescalesDue 确认提高目标记住率会把到期时间提前，而且不算一次复习。
func TestRetentionRescalesDue(t *testing.T) {
	s := openTemp(t)
	clockAt(s, time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC))
	path, ids := newPath(t, s, memoryOutline())
	id := ids["极限"]
	for i := 0; i < 3; i++ {
		ask(t, s, path, id, "a", "a")
	}
	before := parseTime(pointOf(t, s, path, id).DueAt)
	if err := s.SetRetention(path, 0.95); err != nil {
		t.Fatal(err)
	}
	after := parseTime(pointOf(t, s, path, id).DueAt)
	if !after.Before(before) {
		t.Fatalf("due %v should move before %v", after, before)
	}
	detail, _ := s.PointDetail(path, id)
	if detail.Reps != 1 {
		t.Fatalf("a retention change is not a review: reps %d", detail.Reps)
	}
	if view, _ := s.Path(path); view.Retention != 0.95 {
		t.Fatalf("retention %v", view.Retention)
	}
}

// TestRepairVoidsAndRekeys 确认作废一次作答后掌握分重算、题库里的错题撤掉；改正标准答案后按新答案重判。
func TestRepairVoidsAndRekeys(t *testing.T) {
	s := openTemp(t)
	strictJudge(s)
	path, ids := newPath(t, s, memoryOutline())
	id := ids["极限"]
	ask(t, s, path, id, "a", "a")
	bad := ask(t, s, path, id, "wrong-key", "b")
	if bad.Correct || bad.Mastery >= 0.5 {
		t.Fatalf("%+v", bad)
	}
	res, err := s.RepairQuestion(context.Background(), RepairRequest{PathID: path, QuestionID: bad.QuestionID, Action: "correct", Expected: "b"})
	if err != nil || !res.Correct || res.Mastery != 0.8 {
		t.Fatalf("re-keyed answer must count as correct: %+v %v", res, err)
	}
	var banked int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM questions WHERE source = 'mastery'`).Scan(&banked)
	if banked != 0 {
		t.Fatal("a corrected miss must leave the bank")
	}
	res, err = s.RepairQuestion(context.Background(), RepairRequest{PathID: path, QuestionID: bad.QuestionID, Action: "void"})
	if err != nil || res.Mastery != 0.5 {
		t.Fatalf("voided attempt must drop out of the score: %+v %v", res, err)
	}
	if _, err := s.RepairQuestion(context.Background(), RepairRequest{PathID: path, QuestionID: bad.QuestionID, Action: "void"}); err == nil {
		t.Fatal("a voided question cannot be repaired again")
	}
}

// TestSkipDeferAndBrief 确认跳过不计分，暂缓不等于掌握且排到最后，判定只向导师通报一次。
func TestSkipDeferAndBrief(t *testing.T) {
	s := openTemp(t)
	path, ids := newPath(t, s, memoryOutline())
	if _, _, err := s.PoseQuiz(PoseRequest{PathID: path, PointID: ids["极限"], Prompt: "问", Expected: "a"}); err != nil {
		t.Fatal(err)
	}
	if brief := s.TakeTutorBrief(path); brief.Pending == nil || brief.Graded != nil {
		t.Fatalf("%+v", brief)
	}
	if err := s.SkipQuestion(path); err != nil {
		t.Fatal(err)
	}
	if err := s.SkipQuestion(path); err == nil {
		t.Fatal("nothing left to skip")
	}
	if p := pointOf(t, s, path, ids["极限"]); p.Status != StatusNew || p.Attempts != 0 {
		t.Fatalf("a skipped question leaves no trace on the score: %+v", p)
	}
	if err := s.DeferPoint(path, ids["极限"], true); err != nil {
		t.Fatal(err)
	}
	view, _ := s.Path(path)
	if view.Next.PointID != ids["导数"] || view.Counts.Mastered != 0 {
		t.Fatalf("%+v", view.Next)
	}
	res := ask(t, s, path, ids["导数"], "a", "a")
	brief := s.TakeTutorBrief(path)
	if brief.Graded == nil || brief.Graded.QuestionID != res.QuestionID || !brief.Graded.Correct {
		t.Fatalf("%+v", brief)
	}
	if again := s.TakeTutorBrief(path); again.Graded != nil {
		t.Fatal("a grade is reported to the tutor once")
	}
	if err := s.ClaimMastery(path, ids["导数"], true); err != nil {
		t.Fatal(err)
	}
	if view, _ = s.Path(path); view.Next.PointID != ids["极限"] || view.Complete {
		t.Fatalf("only the deferred point remains: %+v", view.Next)
	}
	dup, err := s.Answer(context.Background(), AnswerRequest{PathID: path, QuestionID: res.QuestionID, Answer: "zzz"})
	if err != nil || !dup.Replayed || !dup.Correct {
		t.Fatalf("a repeated submit returns the stored verdict: %+v %v", dup, err)
	}
	if _, err := s.Answer(context.Background(), AnswerRequest{PathID: path, Answer: "a"}); err == nil {
		t.Fatal("no question is waiting")
	}
}

// TestReplayMatchesLive 确认把每个知识点的证据从头回放，得到的行和一路做下来的完全一样。
func TestReplayMatchesLive(t *testing.T) {
	s := openTemp(t)
	strictJudge(s)
	advance := clockAt(s, time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC))
	path, ids := newPath(t, s, memoryOutline())
	for i, answer := range []string{"a", "b", "a", "a", "a", "b", "a"} {
		ask(t, s, path, ids["极限"], "a", answer)
		advance(time.Duration(i+1) * 26 * time.Hour)
	}
	_ = s.ClaimMastery(path, ids["导数"], true)
	advance(48 * time.Hour)
	ask(t, s, path, ids["导数"], "a", "a")
	snapshot := func() string {
		rows, err := s.db.Query(`SELECT id, mastery, qualitative, mastery_source, claimed, status, attempts_json, stability, difficulty, reps, lapses, last_review_at, due_at, mastered_at FROM points WHERE path_id = ? ORDER BY id`, path)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var b strings.Builder
		for rows.Next() {
			var id, source, status, attempts, last, due, masteredAt string
			var mastery, stability, difficulty float64
			var qualitative, claimed, reps, lapses int
			if err := rows.Scan(&id, &mastery, &qualitative, &source, &claimed, &status, &attempts, &stability, &difficulty, &reps, &lapses, &last, &due, &masteredAt); err != nil {
				t.Fatal(err)
			}
			fmt.Fprintln(&b, id, mastery, qualitative, source, claimed, status, attempts, stability, difficulty, reps, lapses, last, due, masteredAt)
		}
		return b.String()
	}
	live := snapshot()
	if _, err := s.db.Exec(`UPDATE points SET mastery = 0, status = 'new', stability = 0, due_at = '', attempts_json = '[]' WHERE path_id = ?`, path); err != nil {
		t.Fatal(err)
	}
	if err := rebuildPath(s.db, path); err != nil {
		t.Fatal(err)
	}
	if replayed := snapshot(); replayed != live {
		t.Fatalf("replay drifted from live state\nlive:\n%s\nreplay:\n%s", live, replayed)
	}
}

// TestPathLifecycle 确认没有大纲时不能离开大纲模式，重置清掉证据留下大纲，删除不留孤儿行。
func TestPathLifecycle(t *testing.T) {
	s := openTemp(t)
	strictJudge(s)
	var seen []Change
	done := make(chan struct{}, 16)
	s.OnChange = func(c Change) {
		seen = append(seen, c)
		done <- struct{}{}
	}
	view, err := s.CreatePath(CreatePathRequest{Goal: "三个月内读懂微积分。然后做题", Sources: []PathSource{{Kind: "library", Ref: "lib-1", Label: "讲义"}}})
	if err != nil || view.Name != "三个月内读懂微积分" || view.Mode != ModeOutline || len(view.Sources) != 1 || !view.IntakeNeeded {
		t.Fatalf("%+v %v", view, err)
	}
	<-done
	if view.Next == nil || view.Next.Action != ActionOutline {
		t.Fatalf("%+v", view.Next)
	}
	if err := s.SetMode(view.ID, ModeStudy); err == nil {
		t.Fatal("study needs an outline")
	}
	if _, err := s.CreatePath(CreatePathRequest{}); err == nil {
		t.Fatal("a path needs a goal or a name")
	}
	if _, err := s.ReplaceOutline(view.ID, memoryOutline(), OutlineOptions{}); err != nil {
		t.Fatal(err)
	}
	ids := pointIDs(t, s, view.ID)
	teach := TeachFirst
	prior := "学过代数"
	if err := s.SetProfile(view.ID, ProfilePatch{Teaching: &teach, PriorKnowledge: &prior}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Path(view.ID)
	if got.Profile.Teaching != TeachFirst || got.IntakeNeeded || got.Next.Action != ActionTeach {
		t.Fatalf("%+v %+v", got.Profile, got.Next)
	}
	bad := "lecture"
	if err := s.SetProfile(view.ID, ProfilePatch{Teaching: &bad}); err == nil {
		t.Fatal("unknown teaching accepted")
	}
	if err := s.SetMode(view.ID, ModeStudy); err != nil {
		t.Fatal(err)
	}
	ask(t, s, view.ID, ids["极限"], "a", "b")
	if err := s.ResetPath(view.ID); err != nil {
		t.Fatal(err)
	}
	if p := pointOf(t, s, view.ID, ids["极限"]); p.Status != StatusNew || p.Attempts != 0 {
		t.Fatalf("%+v", p)
	}
	if list := s.Paths(); len(list) != 1 || list[0].Counts.Total != 2 {
		t.Fatalf("%+v", list)
	}
	if err := s.DeletePath(view.ID); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"paths", "modules", "points", "evidence", "interactions", "path_sources"} {
		var n int
		_ = s.db.QueryRow(`SELECT COUNT(1) FROM ` + table).Scan(&n)
		if n != 0 {
			t.Fatalf("%s still has %d rows", table, n)
		}
	}
	if len(seen) == 0 || seen[0].Scope != "path" || seen[0].ID != view.ID {
		t.Fatalf("%+v", seen)
	}
}

// TestDraftOutlineFromModel 确认起草的大纲丢掉指向后面的前置关系，能直接写入；没接模型时返回 ErrNoModel。
func TestDraftOutlineFromModel(t *testing.T) {
	s := openTemp(t)
	view, err := s.CreatePath(CreatePathRequest{Goal: "学会导数"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DraftOutline(context.Background(), DraftRequest{PathID: view.ID}); err != ErrNoModel {
		t.Fatalf("want ErrNoModel, got %v", err)
	}
	judgeWith(s, func(req CompleteRequest) string {
		if req.Purpose != "outline" || !strings.Contains(req.User, "学会导数") || !strings.Contains(req.User, "Simplified Chinese") {
			t.Fatalf("unexpected request: %+v", req)
		}
		return `说明：{"description":"从极限走到求导","modules":[
			{"name":"极限","objective":"能说出极限是什么","knowledge_points":[
				{"name":"极限的定义","type":"concept","prerequisites":["求导法则"]},
				{"name":"","type":"memory"}]},
			{"name":"求导","objective":"会求导","knowledge_points":[
				{"name":"求导法则","type":"procedure","prerequisites":["极限的定义","不存在"]},
				{"name":"奇怪类型","type":"essay"}]}]}`
	})
	draft, err := s.DraftOutline(context.Background(), DraftRequest{PathID: view.ID})
	if err != nil || len(draft.Modules) != 2 || draft.Description == "" {
		t.Fatalf("%+v %v", draft, err)
	}
	if len(draft.Modules[0].Points) != 1 || len(draft.Modules[0].Points[0].Prereqs) != 0 {
		t.Fatalf("forward prerequisite kept: %+v", draft.Modules[0].Points)
	}
	rule := draft.Modules[1].Points[0]
	if rule.Kind != KindProcedure || len(rule.Prereqs) != 1 || rule.Prereqs[0] != "极限的定义" || draft.Modules[1].Points[1].Kind != KindConcept {
		t.Fatalf("%+v", draft.Modules[1].Points)
	}
	if _, err := s.ReplaceOutline(view.ID, draft.Modules, OutlineOptions{}); err != nil {
		t.Fatalf("a draft must be writable as is: %v", err)
	}
}
