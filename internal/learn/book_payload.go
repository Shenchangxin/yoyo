// 块的载荷：每种块存什么形状，模型的回答怎么校验和整理成这个形状，以及给界面之前去掉什么。
// 这里不读写库，也不调用模型。随堂题的答案和解析留在存库的形状里，公开的形状没有这两项。

package learn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	maxBlockBodyRunes = 20000
	maxSubsections    = 6
	maxTimelineEvents = 12
	maxFlashCards     = 10
	maxQuizQuestions  = 6
)

// bookText 是正文块和读者笔记的载荷。旧库迁移来的正文也是这个形状。
type bookText struct {
	Body string `json:"body"`
}

// bookSection 是讲解块的载荷。
type bookSection struct {
	Intro       string           `json:"intro"`
	Subsections []bookSubsection `json:"subsections"`
	KeyTakeaway string           `json:"key_takeaway"`
}

// bookSubsection 是讲解块里的一个小节。
type bookSubsection struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
}

// bookCallout 是提示框的载荷。Variant 是 key_idea、pitfall、tip 或 summary。
type bookCallout struct {
	Variant string `json:"variant"`
	Body    string `json:"body"`
}

// bookCode 是代码块的载荷。旧库迁移来的代码块没有说明。
type bookCode struct {
	Language    string `json:"language"`
	Code        string `json:"code"`
	Explanation string `json:"explanation"`
}

// bookTimeline 是时间线的载荷。
type bookTimeline struct {
	Events []bookEvent `json:"events"`
}

// bookEvent 是时间线上的一件事。
type bookEvent struct {
	Date        string `json:"date"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// bookFlashCards 是记忆卡片的载荷。
type bookFlashCards struct {
	Cards []bookCard `json:"cards"`
}

// bookCard 是一张记忆卡片。
type bookCard struct {
	Front string `json:"front"`
	Back  string `json:"back"`
	Hint  string `json:"hint"`
}

// bookQuiz 是随堂题存库的载荷，带答案和解析。
type bookQuiz struct {
	Questions []bookQuizQuestion `json:"questions"`
}

// bookQuizQuestion 是存库的一道随堂题。选择题的 Answer 是选项字母，简答题的 Answer 是期望的答案。
type bookQuizQuestion struct {
	ID          string   `json:"id"`
	Question    string   `json:"question"`
	Type        string   `json:"qtype"`
	Options     []Option `json:"options"`
	Answer      string   `json:"answer"`
	Explanation string   `json:"explanation"`
}

// bookQuizPublic 是给界面的随堂题载荷。
type bookQuizPublic struct {
	Questions []bookQuestionPublic `json:"questions"`
}

// bookQuestionPublic 是给界面的一道随堂题，没有答案和解析。
type bookQuestionPublic struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Type     string   `json:"qtype"`
	Options  []Option `json:"options"`
}

// blockParams 是排章节时给一块定下的要求，存在 params_json 里。Focus 是这一块要讲什么，Variant 只对提示框有意义。
type blockParams struct {
	Focus   string `json:"focus,omitempty"`
	Variant string `json:"variant,omitempty"`
}

// decodeParams 解出一块的要求。解不开时返回空的要求。
func decodeParams(raw string) blockParams {
	var p blockParams
	_ = json.Unmarshal([]byte(raw), &p)
	return p
}

// storedPayload 把库里的载荷解成 kind 对应的形状，列表为空时是空数组。认不出的 kind 原样返回解出的对象。
// raw 解不开时返回这种块的空载荷。
func storedPayload(kind, raw string) any {
	data := []byte(raw)
	switch kind {
	case BlockText, BlockUserNote:
		var p bookText
		_ = json.Unmarshal(data, &p)
		return p
	case BlockSection:
		p := bookSection{}
		_ = json.Unmarshal(data, &p)
		if p.Subsections == nil {
			p.Subsections = []bookSubsection{}
		}
		return p
	case BlockCallout:
		var p bookCallout
		_ = json.Unmarshal(data, &p)
		p.Variant = normalizeVariant(p.Variant, "key_idea")
		return p
	case BlockCode:
		var p bookCode
		_ = json.Unmarshal(data, &p)
		return p
	case BlockTimeline:
		p := bookTimeline{}
		_ = json.Unmarshal(data, &p)
		if p.Events == nil {
			p.Events = []bookEvent{}
		}
		return p
	case BlockFlashCards:
		p := bookFlashCards{}
		_ = json.Unmarshal(data, &p)
		if p.Cards == nil {
			p.Cards = []bookCard{}
		}
		return p
	case BlockQuiz:
		return storedQuiz(raw)
	default:
		out := map[string]any{}
		_ = json.Unmarshal(data, &out)
		return out
	}
}

// storedQuiz 把库里的随堂题载荷解出来，带答案。题目和选项为空时是空数组。
func storedQuiz(raw string) bookQuiz {
	p := bookQuiz{}
	_ = json.Unmarshal([]byte(raw), &p)
	if p.Questions == nil {
		p.Questions = []bookQuizQuestion{}
	}
	for i := range p.Questions {
		if p.Questions[i].Options == nil {
			p.Questions[i].Options = []Option{}
		}
	}
	return p
}

// publicPayload 返回可以交给界面和导师的载荷。随堂题去掉答案和解析，其余和存库的形状相同。
func publicPayload(kind, raw string) any {
	if kind != BlockQuiz {
		return storedPayload(kind, raw)
	}
	quiz := storedQuiz(raw)
	out := bookQuizPublic{Questions: make([]bookQuestionPublic, 0, len(quiz.Questions))}
	for _, q := range quiz.Questions {
		out.Questions = append(out.Questions, bookQuestionPublic{ID: q.ID, Question: q.Question, Type: q.Type, Options: q.Options})
	}
	return out
}

// normalizeVariant 把提示框的种类收成四种之一。认不出时返回 fallback。
func normalizeVariant(variant, fallback string) string {
	switch v := strings.ToLower(strings.TrimSpace(variant)); v {
	case "key_idea", "pitfall", "tip", "summary":
		return v
	default:
		return fallback
	}
}

// cleanBody 去掉正文两端的空白并截到上限。
func cleanBody(s string) string {
	return clipRunes(strings.TrimSpace(s), maxBlockBodyRunes)
}

// normalizePayload 校验模型为一块写的回答并整理成存库的形状。kind 是块的种类，raw 是回答里的 JSON，
// params 是排章节时定下的要求，提示框没写种类时用它的 Variant。回答不合要求时返回一句说明原因的错误，调用方据此重试。
func normalizePayload(kind string, raw []byte, params blockParams) (any, error) {
	switch kind {
	case BlockText:
		var p bookText
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, shapeError(kind, err)
		}
		if p.Body = cleanBody(p.Body); p.Body == "" {
			return nil, fmt.Errorf("body is empty")
		}
		return p, nil
	case BlockSection:
		return normalizeSection(raw)
	case BlockCallout:
		var p bookCallout
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, shapeError(kind, err)
		}
		if p.Body = cleanBody(p.Body); p.Body == "" {
			return nil, fmt.Errorf("body is empty")
		}
		p.Variant = normalizeVariant(p.Variant, normalizeVariant(params.Variant, "key_idea"))
		return p, nil
	case BlockCode:
		var p bookCode
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, shapeError(kind, err)
		}
		p.Code = clipRunes(strings.Trim(p.Code, "\r\n"), maxBlockBodyRunes)
		if strings.TrimSpace(p.Code) == "" {
			return nil, fmt.Errorf("code is empty")
		}
		p.Language = clipRunes(strings.ToLower(strings.TrimSpace(p.Language)), 32)
		p.Explanation = cleanBody(p.Explanation)
		return p, nil
	case BlockTimeline:
		return normalizeTimeline(raw)
	case BlockFlashCards:
		return normalizeFlashCards(raw)
	case BlockQuiz:
		return normalizeQuiz(raw)
	default:
		return nil, fmt.Errorf("a %s block is not written by the model", kind)
	}
}

// shapeError 把解 JSON 的错误说成模型看得懂的一句。kind 是块的种类。
func shapeError(kind string, err error) error {
	return fmt.Errorf("the reply does not have the %s shape (%v)", kind, err)
}

// listField 取出回答里名叫 field 的数组。模型有时直接回一个数组，这时整份回答就是那个数组。
func listField(raw []byte, field string) ([]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	var list []json.RawMessage
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return nil, err
		}
		return list, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &obj); err != nil {
		return nil, err
	}
	if item, ok := obj[field]; ok {
		if err := json.Unmarshal(item, &list); err != nil {
			return nil, fmt.Errorf("%s must be an array", field)
		}
	}
	return list, nil
}

// normalizeSection 整理讲解块：去掉没有正文的小节，最多留 6 个，至少要有一个。
func normalizeSection(raw []byte) (any, error) {
	var p bookSection
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, shapeError(BlockSection, err)
	}
	out := bookSection{Intro: cleanBody(p.Intro), Subsections: []bookSubsection{}, KeyTakeaway: clipRunes(strings.TrimSpace(p.KeyTakeaway), 1000)}
	for _, sub := range p.Subsections {
		body := cleanBody(sub.Body)
		if body == "" || len(out.Subsections) >= maxSubsections {
			continue
		}
		out.Subsections = append(out.Subsections, bookSubsection{Heading: clipRunes(strings.TrimSpace(sub.Heading), 200), Body: body})
	}
	if len(out.Subsections) == 0 {
		return nil, fmt.Errorf("subsections needs at least one item with a body")
	}
	return out, nil
}

// normalizeTimeline 整理时间线：去掉既没标题也没说明的事件，最多留 12 件，至少要有 2 件。日期写成数字的也认。
func normalizeTimeline(raw []byte) (any, error) {
	items, err := listField(raw, "events")
	if err != nil {
		return nil, shapeError(BlockTimeline, err)
	}
	out := bookTimeline{Events: []bookEvent{}}
	for _, item := range items {
		var e struct {
			Date        any    `json:"date"`
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if json.Unmarshal(item, &e) != nil {
			continue
		}
		ev := bookEvent{
			Date: clipRunes(strings.TrimSpace(looseString(e.Date)), 80), Title: clipRunes(strings.TrimSpace(e.Title), 200),
			Description: clipRunes(strings.TrimSpace(e.Description), 2000),
		}
		if ev.Title == "" && ev.Description == "" || len(out.Events) >= maxTimelineEvents {
			continue
		}
		out.Events = append(out.Events, ev)
	}
	if len(out.Events) < 2 {
		return nil, fmt.Errorf("events needs at least two items with a title or a description")
	}
	return out, nil
}

// normalizeFlashCards 整理记忆卡片：去掉缺正面或背面的，最多留 10 张，至少要有 2 张。
func normalizeFlashCards(raw []byte) (any, error) {
	items, err := listField(raw, "cards")
	if err != nil {
		return nil, shapeError(BlockFlashCards, err)
	}
	out := bookFlashCards{Cards: []bookCard{}}
	for _, item := range items {
		var c bookCard
		if json.Unmarshal(item, &c) != nil {
			continue
		}
		c = bookCard{Front: clipRunes(strings.TrimSpace(c.Front), 600), Back: clipRunes(strings.TrimSpace(c.Back), 2000), Hint: clipRunes(strings.TrimSpace(c.Hint), 300)}
		if c.Front == "" || c.Back == "" || len(out.Cards) >= maxFlashCards {
			continue
		}
		out.Cards = append(out.Cards, c)
	}
	if len(out.Cards) < 2 {
		return nil, fmt.Errorf("cards needs at least two items with both front and back")
	}
	return out, nil
}

// normalizeQuiz 整理随堂题：去掉没有题干、没有答案或答案对不上选项的题，给每道题一个固定的 id，
// 把选择题的答案认成选项字母并打乱选项。最多留 6 道，至少要有 1 道。
func normalizeQuiz(raw []byte) (any, error) {
	items, err := listField(raw, "questions")
	if err != nil {
		return nil, shapeError(BlockQuiz, err)
	}
	out := bookQuiz{Questions: []bookQuizQuestion{}}
	for _, item := range items {
		var q struct {
			Question    string            `json:"question"`
			Type        string            `json:"qtype"`
			Options     []json.RawMessage `json:"options"`
			Answer      any               `json:"answer"`
			Explanation string            `json:"explanation"`
		}
		if json.Unmarshal(item, &q) != nil || len(out.Questions) >= maxQuizQuestions {
			continue
		}
		stem := clipRunes(strings.TrimSpace(q.Question), 4000)
		answer := strings.TrimSpace(looseString(q.Answer))
		if stem == "" || answer == "" {
			continue
		}
		question := bookQuizQuestion{
			ID: newKey("bq"), Question: stem, Type: QuestionShort, Options: []Option{},
			Answer: clipRunes(answer, 2000), Explanation: clipRunes(strings.TrimSpace(q.Explanation), 4000),
		}
		options := quizOptions(q.Options)
		if strings.EqualFold(strings.TrimSpace(q.Type), QuestionChoice) || len(options) >= 2 {
			// 选择题的答案必须落在某个选项上，否则读者永远选不对，这道题不要。
			label, ok := resolveChoice(options, answer)
			if len(options) < 2 || !ok {
				continue
			}
			question.Type = QuestionChoice
			question.Options, question.Answer = shuffleOptions(options, label, question.ID)
		}
		out.Questions = append(out.Questions, question)
	}
	if len(out.Questions) == 0 {
		return nil, fmt.Errorf("questions needs at least one item with a question and an answer; a choice answer must be the exact text of one option")
	}
	return out, nil
}

// quizOptions 把模型给的选项整理成从 A 起的选项。选项可以是字符串，也可以是带 body 或 text 的对象。空的和重复的去掉，最多留 8 个。
func quizOptions(raw []json.RawMessage) []Option {
	out := []Option{}
	seen := map[string]bool{}
	for _, item := range raw {
		var body string
		if json.Unmarshal(item, &body) != nil {
			var obj struct {
				Body string `json:"body"`
				Text string `json:"text"`
			}
			if json.Unmarshal(item, &obj) != nil {
				continue
			}
			body = obj.Body
			if body == "" {
				body = obj.Text
			}
		}
		body = clipRunes(strings.TrimSpace(body), 1000)
		key := strings.ToLower(body)
		if body == "" || seen[key] || len(out) >= 8 {
			continue
		}
		seen[key] = true
		out = append(out, Option{Label: string(rune('A' + len(out))), Body: body})
	}
	return out
}

// encodePayload 把整理好的载荷写成存库的 JSON 文本。
func encodePayload(payload any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
