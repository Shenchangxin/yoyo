// 锁住练习在应用这一层的接线：界面用的方法能走完录入、排队、核对、评分一圈且队列不带答案；
// 导师的题库工具一开场就在清单里，能存题，但看不到学习者没做过的题的答案。

package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/learn"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// TestLearnCallPracticeRoundTrip 确认界面用的方法能录入一道题、取队列、核对、评分，并读到汇总和设置。
func TestLearnCallPracticeRoundTrip(t *testing.T) {
	a, _, _, _, _ := learnDesk(t)
	saved, err := a.LearnCall("learn.questions.save", map[string]any{
		"stem": "一加一等于几", "qtype": "choice", "answer": "二", "explanation": "数一数",
		"options": []any{map[string]any{"body": "一"}, map[string]any{"body": "二"}, map[string]any{"body": "三"}},
		"tags":    []any{"算术"},
	})
	if err != nil {
		t.Fatal(err)
	}
	q := saved.(learn.Question)
	if q.Type != learn.QuestionChoice || len(q.Options) != 3 || q.Options[1].Label != "B" || q.State != learn.CardNew {
		t.Fatalf("saved %+v", q)
	}
	queue, err := a.LearnCall("learn.practice.queue", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(queue)
	if !strings.Contains(string(raw), q.ID) || strings.Contains(string(raw), "数一数") || strings.Contains(string(raw), `"answer"`) {
		t.Fatalf("the queue must list the question without its key: %s", raw)
	}
	checked, err := a.LearnCall("learn.practice.check", map[string]any{"id": q.ID, "answer": "B"})
	if err != nil {
		t.Fatal(err)
	}
	check := checked.(learn.CheckResult)
	if check.Correct == nil || !*check.Correct || check.Reveal.Answer != "B" || check.Suggested == nil || *check.Suggested != "good" {
		t.Fatalf("check %+v", check)
	}
	if _, err := a.LearnCall("learn.practice.review", map[string]any{"id": q.ID, "rating": "good", "answer": "A", "request_id": "r-0"}); err == nil {
		t.Fatal("a wrong answer rated good must be refused")
	}
	reviewed, err := a.LearnCall("learn.practice.review", map[string]any{"id": q.ID, "rating": "good", "answer": "B", "correct": true, "request_id": "r-1"})
	if err != nil {
		t.Fatal(err)
	}
	if res := reviewed.(learn.ReviewResult); res.IntervalDays < 1 || res.Question.Reps != 1 {
		t.Fatalf("review %+v", res)
	}
	sum, err := a.LearnCall("learn.practice.summary", nil)
	if err != nil || sum.(learn.PracticeSummary).ReviewedToday != 1 || sum.(learn.PracticeSummary).Total != 1 {
		t.Fatalf("summary %+v %v", sum, err)
	}
	set, err := a.LearnCall("learn.practice.settings", map[string]any{"new_per_day": float64(3)})
	if err != nil || set.(learn.PracticeSettings).NewPerDay != 3 || set.(learn.PracticeSettings).Retention != learn.DefaultRetention {
		t.Fatalf("settings %+v %v", set, err)
	}
	read, err := a.LearnCall("learn.practice.settings", nil)
	if err != nil || read.(learn.PracticeSettings).NewPerDay != 3 {
		t.Fatalf("settings read %+v %v", read, err)
	}
	if _, err := a.LearnCall("learn.questions.flag", map[string]any{"ids": []any{q.ID}, "bookmarked": true}); err != nil {
		t.Fatal(err)
	}
	page, err := a.LearnCall("learn.questions", map[string]any{"bookmarked": true})
	if err != nil || page.(learn.QuestionPage).Total != 1 {
		t.Fatalf("list %+v %v", page, err)
	}
	detail, err := a.LearnCall("learn.question", map[string]any{"id": q.ID})
	if err != nil || len(detail.(learn.QuestionDetail).Events) != 1 || detail.(learn.QuestionDetail).Reveal.Explanation != "数一数" {
		t.Fatalf("detail %+v %v", detail, err)
	}
	pre, err := a.LearnCall("learn.questions.import.preview", map[string]any{"text": "question,answer\n水的化学式,H2O\n"})
	if err != nil || pre.(learn.ImportPreview).Valid != 1 {
		t.Fatalf("preview %+v %v", pre, err)
	}
	done, err := a.LearnCall("learn.questions.import.commit", map[string]any{"token": pre.(learn.ImportPreview).Token})
	if err != nil || done.(learn.ImportResult).Imported != 1 {
		t.Fatalf("commit %+v %v", done, err)
	}
	today, err := a.LearnCall("learn.today", nil)
	if err != nil || today.(learn.Today).Practice.Total != 2 {
		t.Fatalf("today %+v %v", today, err)
	}
	if _, err := a.LearnCall("learn.questions.delete", map[string]any{"ids": []any{q.ID}}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.LearnCall("learn.question", map[string]any{"id": q.ID}); err == nil {
		t.Fatal("a deleted question must be gone")
	}
}

// TestBankToolsHideUnattemptedAnswers 确认题库工具在开场清单里；导师存的题进了题库，但学习者做过之前搜不到它的答案。
func TestBankToolsHideUnattemptedAnswers(t *testing.T) {
	a, _, _, _, tools := learnDesk(t)
	listed := map[string]bool{}
	for _, spec := range runtime.AllToolJSON(tools) {
		name, _ := spec.Function["name"].(string)
		listed[name] = true
	}
	if !listed["learn_bank_search"] || !listed["learn_bank_add"] {
		t.Fatal("bank tools are mounted but not in the opening tool list")
	}
	bad, res := callLearn(t, tools, "learn_bank_add", map[string]any{"questions": []any{
		map[string]any{"stem": "好题", "answer": "对"},
		map[string]any{"stem": "坏题", "qtype": "choice", "options": []any{map[string]any{"body": "一"}, map[string]any{"body": "二"}}, "answer": "三"},
	}})
	if res.Err == nil || !strings.Contains(res.Err.Error(), "question 2") {
		t.Fatalf("a malformed question must fail the batch: %v %v", bad, res.Err)
	}
	out, res := callLearn(t, tools, "learn_bank_add", map[string]any{
		"questions": []any{map[string]any{
			"stem": "导数的几何意义是什么", "qtype": "choice", "answer": "切线斜率", "explanation": "割线的极限",
			"options": []any{map[string]any{"body": "面积"}, map[string]any{"body": "切线斜率"}, map[string]any{"body": "弧长"}},
		}},
		"tags": []any{"导数"},
	})
	if res.Err != nil || out["added"].(float64) != 1 {
		t.Fatalf("add %v %v", out, res.Err)
	}
	out, res = callLearn(t, tools, "learn_bank_search", map[string]any{"query": "几何意义"})
	if res.Err != nil || out["total"].(float64) != 1 || out["untrusted"] != true {
		t.Fatalf("search %v %v", out, res.Err)
	}
	if strings.Contains(res.Content, "割线的极限") || strings.Contains(res.Content, `"answer"`) {
		t.Fatalf("the tutor must not see the key of an unattempted question: %s", res.Content)
	}
	page, err := a.Learn.ListQuestions(learn.BankQuery{})
	if err != nil || page.Total != 1 || page.Items[0].Source != "generated" || page.Items[0].SourceID != tools.SessionID {
		t.Fatalf("bank %+v %v", page, err)
	}
	if _, err := a.Learn.ReviewQuestion(learn.ReviewRequest{ID: page.Items[0].ID, Rating: "again", Answer: "A"}); err != nil {
		t.Fatal(err)
	}
	_, res = callLearn(t, tools, "learn_bank_search", map[string]any{"mistakes_only": true})
	if res.Err != nil || !strings.Contains(res.Content, "割线的极限") || !strings.Contains(res.Content, `"last_answer":"A"`) {
		t.Fatalf("an attempted question should carry its answer and the learner's last answer: %s %v", res.Content, res.Err)
	}
}
