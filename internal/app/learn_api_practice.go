// 练习的桌面接口：汇总、队列、核对、评分、统计、设置，以及题库的列表、详情、录入、标记、导入和出题。
// 这里只做参数转接，日程、判分和校验都在 internal/learn。形状见 docs/architecture/learn.md。

package app

import (
	"context"

	"github.com/Shenchangxin/yoyo/internal/learn"
)

// learnPracticeCall 处理 learn.practice 和 learn.question 开头的方法。ok 为假表示这个方法不归它管。
func (a *App) learnPracticeCall(method string, params map[string]any) (any, bool, error) {
	s := a.Learn
	id := strParam(params, "id")
	switch method {
	case "learn.practice.summary":
		out, err := s.PracticeSummary()
		return out, true, err
	case "learn.practice.queue":
		out, err := s.PracticeQueue(learn.QueueRequest{
			Limit: intParam(params, "limit"), Tag: strParam(params, "tag"), Source: strParam(params, "source"),
			MistakesOnly: boolParam(params, "mistakes_only"), IDs: strSlice(params, "ids"),
		})
		return out, true, err
	case "learn.practice.check":
		out, err := s.CheckAnswer(context.Background(), learn.CheckRequest{
			ID: id, Answer: strParam(params, "answer"), Judge: boolParam(params, "judge"), SessionID: strParam(params, "session"),
		})
		return out, true, err
	case "learn.practice.review":
		out, err := s.ReviewQuestion(learn.ReviewRequest{
			ID: id, Rating: strParam(params, "rating"), Answer: strParam(params, "answer"),
			Correct: boolPtrParam(params, "correct"), RequestID: strParam(params, "request_id"),
		})
		return out, true, err
	case "learn.practice.insights":
		out, err := s.PracticeInsights(intParam(params, "days"))
		return out, true, err
	case "learn.practice.settings":
		if !hasParam(params, "retention") && !hasParam(params, "new_per_day") {
			out, err := s.PracticeSettings()
			return out, true, err
		}
		var retention *float64
		var perDay *int
		if hasParam(params, "retention") {
			v := floatParam(params, "retention")
			retention = &v
		}
		if hasParam(params, "new_per_day") {
			v := intParam(params, "new_per_day")
			perDay = &v
		}
		out, err := s.SetPracticeSettings(retention, perDay)
		return out, true, err
	case "learn.questions":
		out, err := s.ListQuestions(learn.BankQuery{
			Query: strParam(params, "query"), Tag: strParam(params, "tag"), Source: strParam(params, "source"),
			MistakesOnly: boolParam(params, "mistakes_only"), Bookmarked: boolPtrParam(params, "bookmarked"),
			Suspended: boolPtrParam(params, "suspended"), Sort: strParam(params, "sort"),
			Limit: intParam(params, "limit"), Offset: intParam(params, "offset"),
		})
		return out, true, err
	case "learn.question":
		out, err := s.QuestionDetail(id)
		return out, true, err
	case "learn.questions.save":
		in, err := questionInput(params)
		if err != nil {
			return nil, true, err
		}
		out, err := s.SaveQuestion(in)
		return out, true, err
	case "learn.questions.delete":
		n, err := s.DeleteQuestions(strSlice(params, "ids"))
		return map[string]any{"ok": err == nil, "deleted": n}, true, err
	case "learn.questions.flag":
		return okMap(), true, s.FlagQuestions(strSlice(params, "ids"), learn.QuestionFlags{
			Bookmarked: boolPtrParam(params, "bookmarked"), Suspended: boolPtrParam(params, "suspended"), Mistake: boolPtrParam(params, "mistake"),
		})
	case "learn.questions.tags":
		out, err := s.QuestionTags()
		return out, true, err
	case "learn.questions.import.preview":
		out, err := s.PreviewImport(learn.ImportRequest{Path: strParam(params, "path"), Text: strParam(params, "text"), Format: strParam(params, "format")})
		return out, true, err
	case "learn.questions.import.commit":
		out, err := s.CommitImport(strParam(params, "token"), strSlice(params, "tags"), boolParam(params, "as_mistakes"))
		return out, true, err
	case "learn.questions.generate":
		items, err := s.GenerateQuestions(context.Background(), learn.GenerateRequest{
			Topic: strParam(params, "topic"), LibraryID: strParam(params, "library_id"), Count: intParam(params, "count"),
			Types: strSlice(params, "qtypes"), Difficulty: strParam(params, "difficulty"), SessionID: strParam(params, "session"),
		})
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"items": items}, true, nil
	case "learn.questions.add":
		var items []learn.DraftQuestion
		if err := decodeParam(params, "items", &items); err != nil {
			return nil, true, err
		}
		added, skipped, err := s.AddQuestions(items, strSlice(params, "tags"), strParam(params, "session"))
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"added": added, "skipped": skipped}, true, nil
	}
	return nil, false, nil
}

// boolPtrParam 从 m 里取一个可以缺席的布尔。键不存在、是 null 或类型不对时返回 nil。
func boolPtrParam(m map[string]any, key string) *bool {
	if m == nil {
		return nil
	}
	v, ok := m[key].(bool)
	if !ok {
		return nil
	}
	return &v
}

// questionInput 把 learn.questions.save 的参数收成录入用的入参。params 里没给的项留成 nil，修改时保持原样。
// options 或 tags 的形状不对时返回错误。
func questionInput(params map[string]any) (learn.QuestionInput, error) {
	in := learn.QuestionInput{ID: strParam(params, "id")}
	text := func(key string) *string {
		if !hasParam(params, key) {
			return nil
		}
		v := strParam(params, key)
		return &v
	}
	in.Stem, in.Type, in.Answer = text("stem"), text("qtype"), text("answer")
	in.Explanation, in.Difficulty = text("explanation"), text("difficulty")
	if hasParam(params, "options") {
		options := []learn.Option{}
		if err := decodeParam(params, "options", &options); err != nil {
			return in, err
		}
		in.Options = &options
	}
	if hasParam(params, "tags") {
		tags := []string{}
		if err := decodeParam(params, "tags", &tags); err != nil {
			return in, err
		}
		in.Tags = &tags
	}
	return in, nil
}
