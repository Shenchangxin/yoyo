// 锁住练习的不变量：队列先到期后新题且守住每天上限，重来十分钟后回来，答错不能评高，同一个幂等键只排一次，
// 客观题在本地批改且读不出不算错，核对不动日程，导入先预览再提交，出题只给校验过的草稿，掌握路径的错题能在这里练。

package learn

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

// saveQ 录入一道题并返回它。d 是题目的内容。
func saveQ(t *testing.T, s *Store, d DraftQuestion) Question {
	t.Helper()
	q, err := s.SaveQuestion(QuestionInput{
		Stem: &d.Stem, Type: &d.Type, Options: &d.Options, Answer: &d.Answer,
		Explanation: &d.Explanation, Difficulty: &d.Difficulty, Tags: &d.Tags,
	})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

// shortQ 录入一道简答题。stem 是题干，答案固定是「答」。
func shortQ(t *testing.T, s *Store, stem string) Question {
	t.Helper()
	return saveQ(t, s, DraftQuestion{Stem: stem, Type: QuestionShort, Answer: "答"})
}

// choiceQ 录入一道四选一，正确项是 B。
func choiceQ(t *testing.T, s *Store, stem string) Question {
	t.Helper()
	return saveQ(t, s, DraftQuestion{
		Stem: stem, Type: QuestionChoice, Answer: "B", Explanation: "因为乙",
		Options: []Option{{Label: "A", Body: "甲"}, {Label: "B", Body: "乙"}, {Label: "C", Body: "丙"}, {Label: "D", Body: "丁"}},
	})
}

// rate 给一道题评分，失败就中止。key 是幂等键，可空。
func rate(t *testing.T, s *Store, id, rating, key string) ReviewResult {
	t.Helper()
	res, err := s.ReviewQuestion(ReviewRequest{ID: id, Rating: rating, RequestID: key})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// queueIDs 返回现在练习队列里的题目 id。
func queueIDs(t *testing.T, s *Store, req QueueRequest) []string {
	t.Helper()
	list, err := s.PracticeQueue(req)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(list))
	for _, q := range list {
		ids = append(ids, q.ID)
	}
	return ids
}

// eventCount 返回一道题的复习记录条数。
func eventCount(t *testing.T, s *Store, id string) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM practice_events WHERE question_id = ?`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestPracticeQueueDueThenNewWithinCap 确认队列先出到期的题再出新题，新题每天不超过上限，评过的题到期前不出现。
func TestPracticeQueueDueThenNewWithinCap(t *testing.T) {
	s := openTemp(t)
	advance := clockAt(s, time.Date(2026, 3, 2, 9, 0, 0, 0, time.Local))
	two := 2
	if _, err := s.SetPracticeSettings(nil, &two); err != nil {
		t.Fatal(err)
	}
	a, b, c := shortQ(t, s, "一"), shortQ(t, s, "二"), shortQ(t, s, "三")
	if a.State != CardNew || a.Retrievability != 0 {
		t.Fatalf("fresh question %+v", a)
	}
	if got := queueIDs(t, s, QueueRequest{}); len(got) != 2 || got[0] != a.ID || got[1] != b.ID {
		t.Fatalf("queue %v, want the two oldest new cards", got)
	}
	rate(t, s, a.ID, "good", "")
	if got := queueIDs(t, s, QueueRequest{}); len(got) != 1 || got[0] != b.ID {
		t.Fatalf("queue after one new card %v, want one more new card", got)
	}
	rate(t, s, b.ID, "good", "")
	if got := queueIDs(t, s, QueueRequest{}); len(got) != 0 {
		t.Fatalf("queue %v, want empty: the daily cap is used up", got)
	}
	sum, err := s.PracticeSummary()
	if err != nil || sum.New != 0 || sum.Due != 0 || sum.ReviewedToday != 2 || sum.Total != 3 || sum.StreakDays != 1 || sum.NextDueAt == "" {
		t.Fatalf("summary %+v %v", sum, err)
	}
	advance(10 * 24 * time.Hour)
	got := queueIDs(t, s, QueueRequest{})
	if len(got) != 3 || got[2] != c.ID {
		t.Fatalf("queue ten days later %v, want two due cards then the new one", got)
	}
	// 点名练习不看到期和上限，按给出的顺序。
	if got := queueIDs(t, s, QueueRequest{IDs: []string{c.ID, a.ID}}); len(got) != 2 || got[0] != c.ID || got[1] != a.ID {
		t.Fatalf("named queue %v", got)
	}
	off := true
	if err := s.FlagQuestions([]string{c.ID}, QuestionFlags{Suspended: &off}); err != nil {
		t.Fatal(err)
	}
	if got := queueIDs(t, s, QueueRequest{}); len(got) != 2 {
		t.Fatalf("queue with a suspended card %v", got)
	}
}

// TestPracticeAgainComesBackInTenMinutes 确认评为重来十分钟后到期、记一次遗忘并标成错题；之后评为记得至少排到一天后。
func TestPracticeAgainComesBackInTenMinutes(t *testing.T) {
	s := openTemp(t)
	start := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	advance := clockAt(s, start)
	q := shortQ(t, s, "题")
	res := rate(t, s, q.ID, "again", "")
	if got := parseTime(res.Question.DueAt); !got.Equal(start.Add(10 * time.Minute)) {
		t.Fatalf("again due %v, want ten minutes later", got)
	}
	if math.Abs(res.IntervalDays-10.0/1440.0) > 1e-9 || res.Question.Lapses != 1 || !res.Question.Mistake || res.Question.State != CardLearning {
		t.Fatalf("again result %+v", res)
	}
	if res.Reveal.Answer != "答" {
		t.Fatalf("review must reveal the answer, got %+v", res.Reveal)
	}
	if got := queueIDs(t, s, QueueRequest{}); len(got) != 0 {
		t.Fatalf("queue right after again %v, want empty", got)
	}
	advance(10 * time.Minute)
	if got := queueIDs(t, s, QueueRequest{}); len(got) != 1 {
		t.Fatalf("queue ten minutes later %v, want the card back", got)
	}
	res = rate(t, s, q.ID, "good", "")
	if res.IntervalDays < 1 || res.Question.Lapses != 1 {
		t.Fatalf("good after again %+v, want at least a day", res)
	}
	easy := rate(t, s, shortQ(t, s, "另一题").ID, "easy", "")
	good := rate(t, s, shortQ(t, s, "第三题").ID, "good", "")
	if easy.IntervalDays <= good.IntervalDays {
		t.Fatalf("easy %v must schedule further out than good %v", easy.IntervalDays, good.IntervalDays)
	}
	if _, err := s.ReviewQuestion(ReviewRequest{ID: q.ID, Rating: "fine"}); err == nil {
		t.Fatal("an unknown rating must be refused")
	}
}

// TestPracticePreviewMatchesReview 确认核对时给出的四个间隔就是评分会排出的间隔：重来十分钟，越有把握排得越远。
func TestPracticePreviewMatchesReview(t *testing.T) {
	s := openTemp(t)
	advance := clockAt(s, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	q := shortQ(t, s, "题")
	rate(t, s, q.ID, "good", "")
	advance(5 * 24 * time.Hour)
	res, err := s.CheckAnswer(context.Background(), CheckRequest{ID: q.ID})
	if err != nil || len(res.Intervals) != 4 {
		t.Fatalf("check %+v %v", res, err)
	}
	in := res.Intervals
	if math.Abs(in["again"]-10.0/1440.0) > 1e-9 || in["hard"] < 1 || in["hard"] > in["good"] || in["good"] >= in["easy"] {
		t.Fatalf("intervals %+v", in)
	}
	if got := rate(t, s, q.ID, "good", ""); math.Abs(got.IntervalDays-in["good"]) > 1e-9 {
		t.Fatalf("review scheduled %v days, the button said %v", got.IntervalDays, in["good"])
	}
}

// TestPracticeWrongAnswerOnlyAgain 确认答错的题只能评为重来：界面说错，或者客观题的作答在本地批改为错，评高都被拒绝。
func TestPracticeWrongAnswerOnlyAgain(t *testing.T) {
	s := openTemp(t)
	clockAt(s, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	c := choiceQ(t, s, "选哪个")
	if _, err := s.ReviewQuestion(ReviewRequest{ID: c.ID, Rating: "good", Answer: "A"}); err == nil {
		t.Fatal("a wrong choice rated good must be refused")
	}
	// 界面谎报答对也没用，引擎自己再批一遍。
	yes := true
	if _, err := s.ReviewQuestion(ReviewRequest{ID: c.ID, Rating: "easy", Answer: "丙", Correct: &yes}); err == nil {
		t.Fatal("the engine must regrade an objective answer")
	}
	if eventCount(t, s, c.ID) != 0 {
		t.Fatal("a refused review must not be logged")
	}
	res, err := s.ReviewQuestion(ReviewRequest{ID: c.ID, Rating: "again", Answer: "A"})
	if err != nil || !res.Question.Mistake {
		t.Fatalf("again on a wrong answer %+v %v", res, err)
	}
	no := false
	sq := shortQ(t, s, "简答")
	if _, err := s.ReviewQuestion(ReviewRequest{ID: sq.ID, Rating: "hard", Correct: &no}); err == nil {
		t.Fatal("a self-reported miss rated hard must be refused")
	}
	detail, err := s.QuestionDetail(c.ID)
	if err != nil || len(detail.Events) != 1 || detail.Events[0].Correct == nil || *detail.Events[0].Correct || detail.Events[0].Answer != "A" {
		t.Fatalf("detail %+v %v", detail, err)
	}
}

// TestPracticeReviewIsIdempotent 确认同一个幂等键只排一次日程，换一道题用同一个键被拒绝。
func TestPracticeReviewIsIdempotent(t *testing.T) {
	s := openTemp(t)
	advance := clockAt(s, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	q := shortQ(t, s, "题")
	first := rate(t, s, q.ID, "good", "req-1")
	advance(time.Hour)
	again := rate(t, s, q.ID, "good", "req-1")
	if eventCount(t, s, q.ID) != 1 {
		t.Fatalf("events %d, want 1", eventCount(t, s, q.ID))
	}
	if again.IntervalDays != first.IntervalDays || again.Question.DueAt != first.Question.DueAt || again.Question.Reps != 1 {
		t.Fatalf("replayed review %+v, first %+v", again, first)
	}
	other := shortQ(t, s, "另一题")
	if _, err := s.ReviewQuestion(ReviewRequest{ID: other.ID, Rating: "good", RequestID: "req-1"}); err == nil {
		t.Fatal("a request id used for another question must be refused")
	}
}

// TestPracticeCheckGradesObjectiveLocally 确认四种客观题在本地批改，读不出来的作答不给结论，核对不写复习记录也不改日程。
func TestPracticeCheckGradesObjectiveLocally(t *testing.T) {
	s := openTemp(t)
	clockAt(s, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	s.Complete = func(context.Context, CompleteRequest) (string, error) {
		t.Fatal("objective questions must not reach the model")
		return "", nil
	}
	choice := choiceQ(t, s, "选哪个")
	multi := saveQ(t, s, DraftQuestion{
		Stem: "选两个", Type: QuestionMulti, Answer: "A,C",
		Options: []Option{{Label: "A", Body: "甲"}, {Label: "B", Body: "乙"}, {Label: "C", Body: "丙"}},
	})
	tf := saveQ(t, s, DraftQuestion{Stem: "地球是圆的", Type: QuestionTrueFalse, Answer: "对"})
	fill := saveQ(t, s, DraftQuestion{Stem: "一加一等于 ____", Type: QuestionFill, Answer: "2|二"})
	if len(tf.Options) != 2 || tf.Options[0].Label != "T" || tf.Options[0].Body != "正确" {
		t.Fatalf("true_false options %+v", tf.Options)
	}
	yes, no := true, false
	cases := []struct {
		id     string
		answer string
		want   *bool
	}{
		{choice.ID, "B", &yes}, {choice.ID, "乙", &yes}, {choice.ID, "选 b。", &yes}, {choice.ID, "A", &no}, {choice.ID, "是 B 吗？", nil}, {choice.ID, "", nil},
		{multi.ID, "A,C", &yes}, {multi.ID, "CA", &yes}, {multi.ID, "c、a", &yes}, {multi.ID, "甲|丙", &yes}, {multi.ID, "A", &no}, {multi.ID, "ABC", &no}, {multi.ID, "不知道", nil},
		{tf.ID, "T", &yes}, {tf.ID, "正确", &yes}, {tf.ID, "错", &no}, {tf.ID, "也许", nil},
		{fill.ID, "2", &yes}, {fill.ID, " 二 ", &yes}, {fill.ID, "3", &no}, {fill.ID, "", nil},
	}
	for _, tc := range cases {
		res, err := s.CheckAnswer(context.Background(), CheckRequest{ID: tc.id, Answer: tc.answer, Judge: true})
		if err != nil {
			t.Fatalf("check %q: %v", tc.answer, err)
		}
		switch {
		case tc.want == nil && res.Correct != nil:
			t.Fatalf("answer %q graded %v, want no verdict", tc.answer, *res.Correct)
		case tc.want != nil && (res.Correct == nil || *res.Correct != *tc.want):
			t.Fatalf("answer %q graded %v, want %v", tc.answer, res.Correct, *tc.want)
		}
		if (res.Suggested == nil) != (tc.want == nil) {
			t.Fatalf("answer %q suggested %v", tc.answer, res.Suggested)
		}
		if tc.want != nil && *tc.want != (*res.Suggested == "good") {
			t.Fatalf("answer %q suggested %s", tc.answer, *res.Suggested)
		}
		if res.Reveal.Answer == "" {
			t.Fatalf("check must reveal the answer, got %+v", res.Reveal)
		}
	}
	if eventCount(t, s, choice.ID) != 0 {
		t.Fatal("checking must not write a review")
	}
	after, err := s.QuestionDetail(choice.ID)
	if err != nil || after.Question.State != CardNew || after.Question.Reps != 0 {
		t.Fatalf("checking changed the schedule: %+v %v", after.Question, err)
	}
}

// TestPracticeCheckJudgesOnRequest 确认简答题默认自评；要求判分时交给模型，判对给结论，部分对不给结论，没接模型时报错。
func TestPracticeCheckJudgesOnRequest(t *testing.T) {
	s := openTemp(t)
	q := saveQ(t, s, DraftQuestion{Stem: "什么是导数", Type: QuestionShort, Answer: "瞬时变化率"})
	res, err := s.CheckAnswer(context.Background(), CheckRequest{ID: q.ID, Answer: "斜率"})
	if err != nil || res.Correct != nil || res.Reveal.Answer != "瞬时变化率" {
		t.Fatalf("self-rated check %+v %v", res, err)
	}
	if _, err := s.CheckAnswer(context.Background(), CheckRequest{ID: q.ID, Answer: "斜率", Judge: true}); err == nil {
		t.Fatal("judging without a model must fail")
	}
	reply := `{"result":"correct","confidence":0.95,"feedback":"对"}`
	calls := judgeWith(s, func(req CompleteRequest) string {
		if !strings.Contains(req.User, "斜率") && !strings.Contains(req.User, "变化的快慢") {
			t.Fatalf("the judge did not get the learner's answer: %s", req.User)
		}
		return reply
	})
	res, err = s.CheckAnswer(context.Background(), CheckRequest{ID: q.ID, Answer: "斜率", Judge: true})
	if err != nil || res.Correct == nil || !*res.Correct || *calls != 1 {
		t.Fatalf("judged check %+v %v calls=%d", res, err, *calls)
	}
	reply = `{"result":"partial","confidence":0.9,"feedback":"差一点"}`
	res, err = s.CheckAnswer(context.Background(), CheckRequest{ID: q.ID, Answer: "变化的快慢", Judge: true})
	if err != nil || res.Correct != nil || res.Feedback == "" {
		t.Fatalf("a partial answer must be left to the learner: %+v %v", res, err)
	}
	// 和标准答案一字不差时不问模型。
	res, err = s.CheckAnswer(context.Background(), CheckRequest{ID: q.ID, Answer: " 瞬时变化率 ", Judge: true})
	if err != nil || res.Correct == nil || !*res.Correct || *calls != 2 {
		t.Fatalf("exact match %+v %v calls=%d", res, err, *calls)
	}
}

// TestPracticeBankSaveListFlagDelete 确认题库的录入校验、修改不动日程、筛选排序分页、标记、标签统计和删除。
func TestPracticeBankSaveListFlagDelete(t *testing.T) {
	s := openTemp(t)
	advance := clockAt(s, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	stem, kind := "坏题", QuestionChoice
	opts := []Option{{Body: "甲"}, {Body: "乙"}}
	wrong := "戊"
	if _, err := s.SaveQuestion(QuestionInput{Stem: &stem, Type: &kind, Options: &opts, Answer: &wrong}); err == nil {
		t.Fatal("a choice answer that names no option must be refused")
	}
	a := saveQ(t, s, DraftQuestion{Stem: "链式法则是什么", Type: QuestionShort, Answer: "外导乘内导", Tags: []string{"#微积分", "导数"}})
	advance(time.Minute)
	b := choiceQ(t, s, "选哪个")
	advance(time.Minute)
	c := saveQ(t, s, DraftQuestion{Stem: "矩阵的秩", Type: QuestionShort, Answer: "线性无关的行数", Tags: []string{"线代"}})
	if len(a.Tags) != 2 || a.Tags[0] != "微积分" || a.Source != "manual" {
		t.Fatalf("saved question %+v", a)
	}
	rate(t, s, a.ID, "good", "")
	renamed := "链式法则怎么用"
	edited, err := s.SaveQuestion(QuestionInput{ID: a.ID, Stem: &renamed})
	if err != nil || edited.Stem != renamed || edited.Reps != 1 || edited.State == CardNew || len(edited.Tags) != 2 {
		t.Fatalf("edit must keep the schedule and the tags: %+v %v", edited, err)
	}
	page, err := s.ListQuestions(BankQuery{})
	if err != nil || page.Total != 3 || page.Items[0].ID != c.ID {
		t.Fatalf("default list %+v %v", page, err)
	}
	if raw, _ := json.Marshal(page); strings.Contains(string(raw), "外导乘内导") || strings.Contains(string(raw), "因为乙") {
		t.Fatalf("a list must not carry answers: %s", raw)
	}
	page, _ = s.ListQuestions(BankQuery{Sort: "oldest", Limit: 1, Offset: 1})
	if page.Total != 3 || len(page.Items) != 1 || page.Items[0].ID != b.ID {
		t.Fatalf("paged list %+v", page)
	}
	page, _ = s.ListQuestions(BankQuery{Query: "链式"})
	if page.Total != 1 || page.Items[0].ID != a.ID {
		t.Fatalf("search %+v", page)
	}
	page, _ = s.ListQuestions(BankQuery{Tag: "导数"})
	if page.Total != 1 {
		t.Fatalf("tag filter %+v", page)
	}
	if _, err := s.ListQuestions(BankQuery{Sort: "random"}); err == nil {
		t.Fatal("an unknown sort must be refused")
	}
	on := true
	if err := s.FlagQuestions([]string{b.ID, "missing"}, QuestionFlags{Bookmarked: &on, Mistake: &on}); err != nil {
		t.Fatal(err)
	}
	page, _ = s.ListQuestions(BankQuery{Bookmarked: &on, MistakesOnly: true})
	if page.Total != 1 || page.Items[0].ID != b.ID || page.Items[0].State != CardLearning {
		t.Fatalf("flag filter %+v", page)
	}
	if err := s.FlagQuestions([]string{"missing"}, QuestionFlags{Bookmarked: &on}); err == nil {
		t.Fatal("flagging nothing must fail")
	}
	tags, err := s.QuestionTags()
	if err != nil || len(tags) != 3 || tags[0].Count != 1 {
		t.Fatalf("tags %+v %v", tags, err)
	}
	hits, total, err := s.SearchBank("", false, 0)
	if err != nil || total != 3 {
		t.Fatalf("tutor search %v %d", err, total)
	}
	for _, h := range hits {
		if h.ID == c.ID && (h.Attempted || h.Answer != "") {
			t.Fatalf("an unattempted question must not show its answer to the tutor: %+v", h)
		}
		if h.ID == a.ID && (!h.Attempted || h.Answer == "") {
			t.Fatalf("an attempted question should carry its answer: %+v", h)
		}
	}
	n, err := s.DeleteQuestions([]string{a.ID, "missing"})
	if err != nil || n != 1 || eventCount(t, s, a.ID) != 0 {
		t.Fatalf("delete %d %v", n, err)
	}
	if _, err := s.QuestionDetail(a.ID); err == nil {
		t.Fatal("a deleted question must be gone")
	}
}

// TestPracticeImportPreviewThenCommit 确认导入先预览：逐行校验并标出重复，提交只写合格的行，同一个令牌只写一次。
func TestPracticeImportPreviewThenCommit(t *testing.T) {
	s := openTemp(t)
	clockAt(s, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	csv := "题目,题型,A,B,C,D,答案,解析,标签\n" +
		"一加一等于几,单选,1,2,3,4,B,数一数,算术；入门\n" +
		"答案对不上,单选,1,2,3,4,E,,\n" +
		"一加一等于几,单选,1,2,3,4,B,数一数,算术\n" +
		"\n" +
		"地球是平的,判断,,,,,错,,\n"
	pre, err := s.PreviewImport(ImportRequest{Text: csv})
	if err != nil {
		t.Fatal(err)
	}
	if pre.Valid != 2 || pre.Invalid != 1 || pre.Duplicates != 1 || len(pre.Rows) != 4 || pre.Token == "" {
		t.Fatalf("preview %+v", pre)
	}
	if pre.Rows[1].Error == "" || pre.Rows[1].Line != 3 || !pre.Rows[2].Duplicate || pre.Rows[3].Question.Answer != "F" {
		t.Fatalf("preview rows %+v", pre.Rows)
	}
	page, _ := s.ListQuestions(BankQuery{})
	if page.Total != 0 {
		t.Fatal("a preview must not write questions")
	}
	res, err := s.CommitImport(pre.Token, []string{"导入"}, false)
	if err != nil || res.Imported != 2 || res.Skipped != 0 {
		t.Fatalf("commit %+v %v", res, err)
	}
	again, err := s.CommitImport(pre.Token, nil, false)
	if err != nil || again != res {
		t.Fatalf("second commit %+v %v, want the first result", again, err)
	}
	page, _ = s.ListQuestions(BankQuery{Sort: "oldest"})
	if page.Total != 2 || page.Items[0].Source != "import" || len(page.Items[0].Tags) != 3 || page.Items[0].Options[1].Body != "2" {
		t.Fatalf("imported questions %+v", page)
	}
	pre, err = s.PreviewImport(ImportRequest{Text: csv})
	if err != nil || pre.Valid != 0 || pre.Duplicates != 3 {
		t.Fatalf("second preview %+v %v, want everything flagged as duplicate", pre, err)
	}
	if _, err := s.CommitImport("imp-missing", nil, false); err == nil {
		t.Fatal("an unknown token must be refused")
	}
	body := `{"questions":[{"question":"光速是多少","type":"fill","answer":["3e8","三十万公里每秒"]},{"stem":"","answer":"x"},"oops"]}`
	pre, err = s.PreviewImport(ImportRequest{Text: body})
	if err != nil || pre.Valid != 1 || pre.Invalid != 2 || pre.Rows[0].Question.Answer != "3e8|三十万公里每秒" {
		t.Fatalf("json preview %+v %v", pre, err)
	}
	res, err = s.CommitImport(pre.Token, nil, true)
	if err != nil || res.Imported != 1 {
		t.Fatalf("json commit %+v %v", res, err)
	}
	page, _ = s.ListQuestions(BankQuery{MistakesOnly: true})
	if page.Total != 1 || page.Items[0].State != CardLearning {
		t.Fatalf("imported as mistakes %+v", page)
	}
	if _, err := s.PreviewImport(ImportRequest{Text: "答案,解析\nx,y\n"}); err == nil {
		t.Fatal("a file without a question column must be refused")
	}
}

// TestPracticeImportFlagsQuestionsAlreadyWritten 确认题库里已经有题干和题型都相同的题时，导入预览把那一行标成重复，题型不同的不算。
func TestPracticeImportFlagsQuestionsAlreadyWritten(t *testing.T) {
	s := openTemp(t)
	saveQ(t, s, DraftQuestion{Stem: "水的化学式", Type: QuestionFill, Answer: "H2O"})
	pre, err := s.PreviewImport(ImportRequest{Text: "question,type,answer\n水的化学式,fill,H2O\n水的化学式,short,H2O\n"})
	if err != nil || pre.Duplicates != 1 || pre.Valid != 1 || !pre.Rows[0].Duplicate || pre.Rows[1].Duplicate {
		t.Fatalf("preview %+v %v", pre, err)
	}
	res, err := s.CommitImport(pre.Token, nil, false)
	if err != nil || res.Imported != 1 {
		t.Fatalf("commit %+v %v", res, err)
	}
}

// TestPracticeGenerateValidatesAndShuffles 确认出题只返回校验过的草稿且不落库，答案跟着打乱后的选项走；存进题库时同样的题不重复。
func TestPracticeGenerateValidatesAndShuffles(t *testing.T) {
	s := openTemp(t)
	if _, err := s.GenerateQuestions(context.Background(), GenerateRequest{Topic: "链式法则"}); err == nil {
		t.Fatal("generating without a model must fail")
	}
	s.Complete = func(_ context.Context, req CompleteRequest) (string, error) {
		if !strings.Contains(req.User, "<topic>") || !strings.Contains(req.User, "链式法则") {
			t.Fatalf("prompt lacks the topic: %s", req.User)
		}
		return `{"questions":[
			{"stem":"y=f(g(x)) 的导数是？","qtype":"choice","options":["f'(x)g'(x)","f'(g(x))g'(x)","f(g'(x))","f'(g(x))+g'(x)"],"answer":"f'(g(x))g'(x)。","explanation":"外导乘内导","difficulty":"medium"},
			{"stem":"答案不在选项里","qtype":"choice","options":["一","二"],"answer":"三"},
			{"stem":"链式法则只适用于多项式","qtype":"true_false","answer":"F","explanation":"任何可导的复合函数都行"},
			{"stem":"不允许的题型","qtype":"open","answer":"要点"}
		]}`, nil
	}
	drafts, err := s.GenerateQuestions(context.Background(), GenerateRequest{Topic: "链式法则", Count: 5, Types: []string{"choice", "判断"}})
	if err != nil || len(drafts) != 2 {
		t.Fatalf("drafts %+v %v", drafts, err)
	}
	first := drafts[0]
	if first.Type != QuestionChoice || len(first.Options) != 4 {
		t.Fatalf("first draft %+v", first)
	}
	for _, o := range first.Options {
		if (o.Label == first.Answer) != (o.Body == "f'(g(x))g'(x)") {
			t.Fatalf("the answer %q must follow its option through the shuffle: %+v", first.Answer, first.Options)
		}
	}
	page, _ := s.ListQuestions(BankQuery{})
	if page.Total != 0 {
		t.Fatal("generating must not save anything")
	}
	added, skipped, err := s.AddQuestions(drafts, []string{"生成"}, "sess-1")
	if err != nil || added != 2 || skipped != 0 {
		t.Fatalf("add %d %d %v", added, skipped, err)
	}
	added, skipped, err = s.AddQuestions(drafts, nil, "")
	if err != nil || added != 0 || skipped != 2 {
		t.Fatalf("second add %d %d %v", added, skipped, err)
	}
	bad := []DraftQuestion{drafts[0], {Stem: "坏题", Type: QuestionChoice, Options: []Option{{Body: "一"}, {Body: "二"}}, Answer: "三"}}
	if _, _, err := s.AddQuestions(bad, nil, ""); err == nil || !strings.Contains(err.Error(), "question 2") {
		t.Fatalf("a bad draft must fail the batch and say which: %v", err)
	}
	page, _ = s.ListQuestions(BankQuery{Source: "generated"})
	if page.Total != 2 || page.Items[0].SourceID != "sess-1" {
		t.Fatalf("generated questions %+v", page)
	}
}

// TestPracticeInsightsCountDays 确认统计按本地日计数：复习量、重来、新题、记住率只看评过的题、到期预报把过期的算在今天。
func TestPracticeInsightsCountDays(t *testing.T) {
	s := openTemp(t)
	advance := clockAt(s, time.Date(2026, 3, 2, 9, 0, 0, 0, time.Local))
	a, b := shortQ(t, s, "一"), shortQ(t, s, "二")
	rate(t, s, a.ID, "good", "")
	rate(t, s, b.ID, "again", "")
	advance(24 * time.Hour)
	rate(t, s, b.ID, "good", "")
	advance(30 * 24 * time.Hour)
	in, err := s.PracticeInsights(45)
	if err != nil || in.Days != 90 || len(in.Daily) != 90 || len(in.Forecast) != 7 {
		t.Fatalf("insights %+v %v", in, err)
	}
	first, second := in.Daily[90-32], in.Daily[90-31]
	if first.Date != "2026-03-02" || first.Reviews != 2 || first.Again != 1 || first.NewCards != 2 || first.Mistakes != 1 {
		t.Fatalf("first day %+v", first)
	}
	if second.Reviews != 1 || second.NewCards != 0 {
		t.Fatalf("second day %+v", second)
	}
	if in.Retention.Reviews != 1 || in.Retention.Rate != 1 {
		t.Fatalf("retention %+v, want only the review of a seen card", in.Retention)
	}
	if in.Forecast[0].Due != 2 {
		t.Fatalf("forecast %+v, want both overdue cards counted today", in.Forecast)
	}
	if len(in.Sources) != 1 || in.Sources[0].Source != "manual" || in.Sources[0].Count != 2 {
		t.Fatalf("sources %+v", in.Sources)
	}
	sum, _ := s.PracticeSummary()
	if sum.StreakDays != 0 || sum.Due != 2 {
		t.Fatalf("summary a month later %+v, want the streak broken", sum)
	}
}

// TestPracticeRetentionReschedules 确认调低目标记住率把到期时间往后推，调回去日程也回去，而且不算一次复习。
func TestPracticeRetentionReschedules(t *testing.T) {
	s := openTemp(t)
	clockAt(s, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	q := shortQ(t, s, "题")
	base := rate(t, s, q.ID, "easy", "").Question.DueAt
	low := 0.8
	set, err := s.SetPracticeSettings(&low, nil)
	if err != nil || set.Retention != 0.8 || set.NewPerDay != defaultNewPerDay {
		t.Fatalf("settings %+v %v", set, err)
	}
	later, _ := s.QuestionDetail(q.ID)
	if !parseTime(later.Question.DueAt).After(parseTime(base)) || later.Question.Reps != 1 {
		t.Fatalf("lower retention must push the due date out: %s -> %s", base, later.Question.DueAt)
	}
	back := 0.9
	if _, err := s.SetPracticeSettings(&back, nil); err != nil {
		t.Fatal(err)
	}
	restored, _ := s.QuestionDetail(q.ID)
	if restored.Question.DueAt != base || eventCount(t, s, q.ID) != 1 {
		t.Fatalf("restoring retention must restore the schedule: %s want %s", restored.Question.DueAt, base)
	}
}

// TestMasteryMissIsPracticable 确认掌握路径上答错的题进了题库，带着选项，一天后到期，能在练习里批改。
func TestMasteryMissIsPracticable(t *testing.T) {
	s := openTemp(t)
	advance := clockAt(s, time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))
	pathID, ids := newPath(t, s, memoryOutline())
	posed, _, err := s.PoseQuiz(PoseRequest{
		PathID: pathID, PointID: ids["极限"], Type: QuestionChoice, Prompt: "极限是什么", Expected: "趋近的值", Explanation: "看趋势",
		Options: []Option{{Label: "A", Body: "函数值"}, {Label: "B", Body: "趋近的值"}, {Label: "C", Body: "导数"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	wrongLabel, rightLabel := "", ""
	for _, o := range posed.Options {
		if o.Body == "趋近的值" {
			rightLabel = o.Label
		} else {
			wrongLabel = o.Label
		}
	}
	if _, err := s.Answer(context.Background(), AnswerRequest{PathID: pathID, QuestionID: posed.ID, Answer: wrongLabel}); err != nil {
		t.Fatal(err)
	}
	page, err := s.ListQuestions(BankQuery{MistakesOnly: true})
	if err != nil || page.Total != 1 {
		t.Fatalf("bank after a miss %+v %v", page, err)
	}
	q := page.Items[0]
	if q.Source != "mastery" || q.PathID != pathID || q.PointID != ids["极限"] || len(q.Options) != 3 || q.Options[0].Body == "" || q.State != CardLearning {
		t.Fatalf("banked miss %+v", q)
	}
	if got := queueIDs(t, s, QueueRequest{}); len(got) != 0 {
		t.Fatalf("queue %v, want the miss to wait a day", got)
	}
	advance(25 * time.Hour)
	if got := queueIDs(t, s, QueueRequest{}); len(got) != 1 || got[0] != q.ID {
		t.Fatalf("queue a day later %v", got)
	}
	res, err := s.CheckAnswer(context.Background(), CheckRequest{ID: q.ID, Answer: rightLabel})
	if err != nil || res.Correct == nil || !*res.Correct || res.Reveal.Explanation != "看趋势" {
		t.Fatalf("check on the banked miss %+v %v", res, err)
	}
	today, err := s.TodayView(s.clock())
	if err != nil || today.Practice.Due != 1 || len(today.Due) != 1 || today.Due[0].ID != q.ID {
		t.Fatalf("today %+v %v", today, err)
	}
}

// TestPracticeReadsLegacyRows 确认第一版留下的题还能练：选项是字符串数组，答案写的是选项正文。
func TestPracticeReadsLegacyRows(t *testing.T) {
	s := openTemp(t)
	now := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	clockAt(s, now)
	if _, err := s.db.Exec(`INSERT INTO questions (id, stem, qtype, options_json, answer, explanation, source, created_at)
		VALUES ('bank-old', '旧题', 'choice', '["甲","乙","丙"]', '乙', '', 'import', ?)`, formatTime(now.Add(-time.Hour))); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO practice_state (question_id, due_at, interval_days, ease) VALUES ('bank-old', ?, 1, 2.5)`,
		formatTime(now.Add(-time.Hour))); err != nil {
		t.Fatal(err)
	}
	list, err := s.PracticeQueue(QueueRequest{})
	if err != nil || len(list) != 1 || len(list[0].Options) != 3 || list[0].Options[1].Label != "B" || list[0].Options[1].Body != "乙" {
		t.Fatalf("legacy queue %+v %v", list, err)
	}
	res, err := s.CheckAnswer(context.Background(), CheckRequest{ID: "bank-old", Answer: "B"})
	if err != nil || res.Correct == nil || !*res.Correct {
		t.Fatalf("legacy check %+v %v", res, err)
	}
	if out := rate(t, s, "bank-old", "good", ""); out.IntervalDays < 1 {
		t.Fatalf("legacy review %+v", out)
	}
}
