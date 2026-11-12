// 练习上的学习工具和导师守则。题库是全局的，不靠会话绑定，所以每段学习对话都挂着。
// 导师能在题库里找题、往题库里存题。它不能替学习者评分，也看不到学习者还没做过的题的答案。

package app

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/learn"
)

// learnPracticePin 是学习对话里关于题库的导师守则。
const learnPracticePin = `## Question bank
- The learner keeps one question bank that is reviewed on a spaced schedule. Missed questions from study land there on their own.
- learn_bank_search finds questions and the learner's past mistakes. Use it when they ask about a mistake, want to go over what they got wrong, or ask what is in the bank. It shows answers only for questions they have already attempted; do not guess the key for the rest.
- learn_bank_add saves questions for later review. Use it only when the learner asks to keep questions or to build a practice set. Write questions that test understanding, one idea each, with a checkable answer and a short explanation.
- You cannot rate a review for the learner. Reviews happen on the practice page.`

// mountPracticeTools 在学习对话上挂题库工具。
func (a *App) mountPracticeTools(m *learnMount) {
	s := a.Learn
	sessionID := m.session

	m.add("learn_bank_search", "Search the learner's question bank. query matches words in the question, its options, or its tags; leave it empty to list the most recent. mistakes_only lists questions they got wrong. Answers and the learner's last answer come back only for questions they have attempted.", true, map[string]any{
		"query":         map[string]any{"type": "string"},
		"mistakes_only": map[string]any{"type": "boolean"},
		"limit":         map[string]any{"type": "integer", "description": "How many to return. Default 8, at most 20."},
	}, nil, func(args map[string]any) (learnOut, error) {
		hits, total, err := s.SearchBank(strParam(args, "query"), boolParam(args, "mistakes_only"), intParam(args, "limit"))
		if err != nil {
			return learnOut{}, err
		}
		return learnOut{Value: map[string]any{"total": total, "questions": hits, "untrusted": true}}, nil
	})

	option := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"label": map[string]any{"type": "string"},
			"body":  map[string]any{"type": "string"},
		},
		"required": []string{"body"},
	}
	question := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"stem":        map[string]any{"type": "string", "description": "The question text. It must stand on its own."},
			"qtype":       map[string]any{"type": "string", "enum": []string{learn.QuestionChoice, learn.QuestionMulti, learn.QuestionTrueFalse, learn.QuestionFill, learn.QuestionShort, learn.QuestionOpen}},
			"options":     map[string]any{"type": "array", "items": option, "description": "For choice and multi: two to eight options."},
			"answer":      map[string]any{"type": "string", "description": "choice: the letter or exact text of the correct option. multi: letters joined by commas. true_false: T or F. fill: acceptable forms joined by |. short and open: the reference answer."},
			"explanation": map[string]any{"type": "string"},
			"difficulty":  map[string]any{"type": "string", "enum": []string{"easy", "medium", "hard"}},
			"tags":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"stem", "answer"},
	}
	m.add("learn_bank_add", "Save questions into the learner's question bank for spaced review. Use only when the learner asks to keep questions or to build a practice set. Every question is validated; if one is malformed nothing is saved and the error says which. A question already in the bank is skipped.", false, map[string]any{
		"questions": map[string]any{"type": "array", "items": question, "description": "One to twenty questions."},
		"tags":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Tags added to every question in this batch."},
	}, []string{"questions"}, func(args map[string]any) (learnOut, error) {
		var items []learn.DraftQuestion
		if err := decodeParam(args, "questions", &items); err != nil {
			return learnOut{}, fmt.Errorf("questions must be an array of question objects: %v", err)
		}
		if len(items) == 0 {
			return learnOut{}, fmt.Errorf("learn_bank_add needs at least one question")
		}
		if len(items) > 20 {
			return learnOut{}, fmt.Errorf("learn_bank_add takes at most 20 questions per call")
		}
		var tags []string
		if err := decodeParam(args, "tags", &tags); err != nil {
			tags = nil
		}
		added, skipped, err := s.AddQuestions(items, tags, sessionID)
		if err != nil {
			return learnOut{}, err
		}
		return learnOut{Value: map[string]any{"added": added, "skipped": skipped}}, nil
	})
}
