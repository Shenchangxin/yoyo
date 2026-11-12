// 练习里的两步：核对一次作答，和按评分排下一次复习。
// 核对只亮出答案，不动日程；日程只在评分时改。客观题在本地逐字批改，读不出来的作答不算答错。
// 简答和开放题由学习者自评，或者应要求交给 mastery_judge.go 的模型按意思判。这里不出题，也不管题库的增删改。

package learn

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// CheckRequest 是核对一次作答的入参。ID 是题目，Answer 是学习者的作答。
// Judge 为真时简答和开放题交给模型判，SessionID 决定用哪段对话的模型。客观题不看这两项。
type CheckRequest struct {
	ID        string
	Answer    string
	Judge     bool
	SessionID string
}

// CheckResult 是核对的结果。Correct 为 nil 表示没法判定：作答读不出来，或者这道题要学习者自评。
// Suggested 是建议的评分，答对是 good，答错是 again，没法判定时为 nil。Reveal 总是带上。
// Intervals 是四种评分各自会排出的间隔天数，键是 again、hard、good、easy，界面把它写在评分按钮上。
type CheckResult struct {
	Correct   *bool              `json:"correct"`
	Feedback  string             `json:"feedback,omitempty"`
	Reveal    Reveal             `json:"reveal"`
	Suggested *string            `json:"suggested"`
	Intervals map[string]float64 `json:"intervals"`
}

// planReview 算出按 rating 评分之后的记忆状态和到期时刻，不写任何东西。评为重来十分钟后到期；
// 其余按稳定度和目标记住率 retention 排，至少一天。距上次复习的时间按整天算。c 是这道题现在的状态，now 是评分的时刻。
func planReview(c card, rating Rating, retention float64, now time.Time) (Memory, time.Time) {
	next := NextMemory(c.Mem, elapsedWholeDays(c.Last, now), rating)
	if rating == RatingAgain {
		return next, now.Add(againDelay)
	}
	return next, dueAfter(now, next, retention, 1)
}

// previewIntervals 返回四种评分各自会排出的间隔天数。和真正评分走同一个 planReview，按钮上写的就是会发生的。
func previewIntervals(c card, retention float64, now time.Time) map[string]float64 {
	out := map[string]float64{}
	for _, rating := range []Rating{RatingAgain, RatingHard, RatingGood, RatingEasy} {
		_, due := planReview(c, rating, retention, now)
		out[rating.String()] = due.Sub(now).Hours() / 24
	}
	return out
}

// objectiveType 在 qtype 是能在本地批改的题型时为真。
func objectiveType(qtype string) bool {
	switch qtype {
	case QuestionChoice, QuestionMulti, QuestionTrueFalse, QuestionFill:
		return true
	}
	return false
}

// CheckAnswer 核对一次作答并亮出标准答案和解析。它不写复习记录，也不改日程，只把作答记在题目上。
// 客观题在本地批改。简答和开放题在 req.Judge 为真时交给模型，模型判部分对或把握不足时 Correct 为 nil。
// ctx 取消或没接模型时返回错误，什么都不记。
func (s *Store) CheckAnswer(ctx context.Context, req CheckRequest) (CheckResult, error) {
	if s == nil {
		return CheckResult{}, fmt.Errorf("learn store unavailable")
	}
	now := s.clock()
	s.mu.Lock()
	c, err := loadCard(s.db, req.ID, now)
	var pointName string
	if err == nil && c.PointID != "" {
		_ = s.db.QueryRow(`SELECT name FROM points WHERE id = ?`, c.PointID).Scan(&pointName)
	}
	retention := practiceSettings(s.db).Retention
	s.mu.Unlock()
	if err != nil {
		return CheckResult{}, err
	}
	answer := strings.TrimSpace(req.Answer)
	out := CheckResult{Reveal: c.reveal(), Intervals: previewIntervals(c, retention, now)}
	decide := func(correct bool) { out.Correct = &correct }
	switch {
	case objectiveType(c.Type):
		if correct, decided := gradeLocal(c, answer); decided {
			decide(correct)
		} else {
			out.Feedback = "That answer could not be read, so it was not graded. Compare it with the answer and rate yourself."
		}
	case !req.Judge:
	case answer == "":
		out.Feedback = "The answer was empty, so nothing was graded."
	case normalizeAnswer(c.Answer) != "" && normalizeAnswer(c.Answer) == normalizeAnswer(answer):
		decide(true)
	case strings.TrimSpace(c.Answer) == "":
		out.Feedback = "This question has no reference answer to judge against. Rate yourself."
	default:
		v, err := s.judgeAnswer(ctx, req.SessionID, storedQuestion{
			ID: c.ID, Type: c.Type, Prompt: c.Stem, Expected: c.Answer, Options: c.Options,
		}, pointName, answer)
		if err != nil {
			return CheckResult{}, err
		}
		out.Feedback = v.Feedback
		// 部分对和没法判定都不给结论，由学习者看过答案自己评。
		if v.Final && v.Result == ResultCorrect {
			decide(true)
		} else if v.Final && v.Result == ResultIncorrect {
			decide(false)
		}
	}
	if out.Correct != nil {
		suggested := RatingAgain.String()
		if *out.Correct {
			suggested = RatingGood.String()
		}
		out.Suggested = &suggested
	}
	if answer != "" {
		s.mu.Lock()
		_, err = s.db.Exec(`UPDATE questions SET last_answer = ? WHERE id = ?`, clipRunes(answer, 4000), c.ID)
		s.mu.Unlock()
		if err != nil {
			return CheckResult{}, err
		}
	}
	return out, nil
}

// gradeLocal 在本地批改一道客观题。decided 为假表示 answer 读不出来、标准答案本身读不出来，或者题型不是客观题；这时不能当成答错。
// 没有选项的旧选择题按填空题比对。
func gradeLocal(c card, answer string) (correct, decided bool) {
	switch c.Type {
	case QuestionChoice:
		if len(c.Options) == 0 {
			return gradeFill(c.Answer, answer)
		}
		picked, ok := resolveChoice(c.Options, answer)
		if !ok {
			return false, false
		}
		// 旧库里的标准答案可能是选项正文，先认成字母再比。
		want, ok := resolveChoice(c.Options, c.Answer)
		if !ok {
			want = strings.ToUpper(strings.TrimSpace(c.Answer))
		}
		return picked == want, true
	case QuestionMulti:
		picked, ok := parseLabels(c.Options, answer)
		if !ok {
			return false, false
		}
		want, ok := parseLabels(c.Options, c.Answer)
		if !ok {
			return false, false
		}
		return strings.Join(picked, ",") == strings.Join(want, ","), true
	case QuestionTrueFalse:
		picked, ok := readTrueFalse(c.Options, answer)
		if !ok {
			return false, false
		}
		want, ok := readTrueFalse(c.Options, c.Answer)
		if !ok {
			return false, false
		}
		return picked == want, true
	case QuestionFill:
		return gradeFill(c.Answer, answer)
	}
	return false, false
}

// gradeFill 比对填空题。expected 里用竖线隔开的每一种写法，以及整段 expected，规整后和 answer 相等就算对。
// answer 规整后为空，或者 expected 里没有任何内容时，decided 为假。
func gradeFill(expected, answer string) (correct, decided bool) {
	got := normalizeAnswer(answer)
	if got == "" {
		return false, false
	}
	seen := false
	for _, want := range append(strings.Split(expected, "|"), expected) {
		w := normalizeAnswer(want)
		if w == "" {
			continue
		}
		seen = true
		if w == got {
			return true, true
		}
	}
	return false, seen
}

// parseTrueFalse 把一段文字认成判断题的 T 或 F。认得的写法是 T、true、对、正确、是、yes、√，和 F、false、错、错误、否、no、×，不分大小写。
// 其余文字返回 false。
func parseTrueFalse(text string) (string, bool) {
	t := strings.ToLower(strings.TrimSpace(text))
	t = strings.Trim(t, " \t\r\n.。!！,，;；:：()（）\"'“”‘’")
	switch t {
	case "t", "true", "对", "正确", "是", "yes", "√":
		return "T", true
	case "f", "false", "错", "错误", "否", "no", "×":
		return "F", true
	}
	return "", false
}

// readTrueFalse 读出判断题的作答。先按 parseTrueFalse 认；认不出时，text 和某个选项的正文完全相同也算选了那一项。
// options 是这道题的选项，可以为空。
func readTrueFalse(options []Option, text string) (string, bool) {
	if v, ok := parseTrueFalse(text); ok {
		return v, true
	}
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return "", false
	}
	for _, o := range options {
		if strings.ToLower(o.Body) == lower {
			return parseTrueFalse(o.Label)
		}
	}
	return "", false
}

// parseLabels 从 text 里读出选了哪几个选项，按 options 的顺序返回字母。
// 接受用逗号、分号、顿号或空白隔开的字母，连写的字母（AC），以及用竖线隔开的选项正文。
// 读不出来、一项都没选，或者带问号时返回 false：读不出不等于答错。
func parseLabels(options []Option, text string) ([]string, bool) {
	text = strings.TrimSpace(text)
	if text == "" || len(options) == 0 {
		return nil, false
	}
	picked := map[string]bool{}
	collect := func() ([]string, bool) {
		out := make([]string, 0, len(picked))
		for _, o := range options {
			if picked[strings.ToUpper(o.Label)] {
				out = append(out, o.Label)
			}
		}
		return out, len(out) > 0
	}
	if strings.Contains(text, "|") {
		for _, part := range strings.Split(text, "|") {
			if strings.TrimSpace(part) == "" {
				continue
			}
			label, ok := resolveChoice(options, part)
			if !ok {
				return nil, false
			}
			picked[strings.ToUpper(label)] = true
		}
		return collect()
	}
	// 单独一个字母，或者和某个选项正文完全相同。
	if label, ok := resolveChoice(options, text); ok {
		return []string{label}, true
	}
	if strings.ContainsAny(text, "?？") {
		return nil, false
	}
	core := strings.Trim(text, " \t\r\n.。:：!！()（）[]【】\"'“”‘’")
	core = choiceLead.ReplaceAllString(core, "")
	known := map[string]bool{}
	letters := true
	for _, o := range options {
		label := strings.ToUpper(o.Label)
		known[label] = true
		if utf8.RuneCountInString(label) != 1 {
			letters = false
		}
	}
	tokens := strings.FieldsFunc(core, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(",，;；、/+&和与及", r)
	})
	for _, token := range tokens {
		upper := strings.ToUpper(token)
		if known[upper] {
			picked[upper] = true
			continue
		}
		if label, ok := resolveChoice(options, token); ok {
			picked[strings.ToUpper(label)] = true
			continue
		}
		// 连写的字母：每个字符都得是选项字母，而且不能重复，免得把一个英文单词读成作答。
		if !letters {
			return nil, false
		}
		for _, r := range upper {
			label := string(r)
			if !known[label] || picked[label] {
				return nil, false
			}
			picked[label] = true
		}
	}
	return collect()
}

// ReviewRequest 是一次评分的入参。Rating 是 again、hard、good 或 easy。Answer 是这次的作答，可空。
// Correct 是界面核对出的对错，nil 表示没有结论。RequestID 是幂等键，同一个键只排一次日程；为空时不去重。
type ReviewRequest struct {
	ID        string
	Rating    string
	Answer    string
	Correct   *bool
	RequestID string
}

// ReviewResult 是一次评分的结果。Question 是评分之后的题目和日程，IntervalDays 是这次排出的间隔天数。
type ReviewResult struct {
	Question     Question `json:"question"`
	IntervalDays float64  `json:"interval_days"`
	Reveal       Reveal   `json:"reveal"`
}

// ReviewQuestion 按评分推进记忆状态并排下一次复习，同时写一条复习记录。
// 评为重来十分钟后再出现并记一次遗忘；其余评分按稳定度和目标记住率排间隔，至少一天。距上次复习的时间按整天算。
// 答错的题只能评为重来：req.Correct 为假，或者客观题带着的 Answer 在本地批改为错，而评分高于重来时返回错误。
// 答错或评为重来的题标成错题。req.RequestID 已经用过时返回当时排出的间隔和题目现在的状态，不再排一次。
func (s *Store) ReviewQuestion(req ReviewRequest) (ReviewResult, error) {
	if s == nil {
		return ReviewResult{}, fmt.Errorf("learn store unavailable")
	}
	rating, ok := ParseRating(strings.ToLower(strings.TrimSpace(req.Rating)))
	if !ok {
		return ReviewResult{}, fmt.Errorf("rating must be again, hard, good, or easy")
	}
	s.mu.Lock()
	out, replayed, err := s.review(req, rating)
	s.mu.Unlock()
	if err != nil {
		return ReviewResult{}, err
	}
	if !replayed {
		s.notify("practice", out.Question.ID, "reviewed")
	}
	return out, nil
}

// review 在持有 s.mu 时完成一次评分。rating 是已经解析过的评分，req 的其余项见 ReviewRequest。
// replayed 为真表示这个幂等键已经用过，什么都没写。
func (s *Store) review(req ReviewRequest, rating Rating) (out ReviewResult, replayed bool, err error) {
	now := s.clock()
	requestID := strings.TrimSpace(req.RequestID)
	if requestID != "" {
		var questionID string
		var interval float64
		if scanErr := s.db.QueryRow(`SELECT question_id, interval_days FROM practice_events WHERE request_id = ?`, requestID).
			Scan(&questionID, &interval); scanErr == nil {
			if questionID != req.ID {
				return ReviewResult{}, false, fmt.Errorf("request_id was already used for another question")
			}
			c, err := loadCard(s.db, questionID, now)
			if err != nil {
				return ReviewResult{}, false, err
			}
			return ReviewResult{Question: c.Question, IntervalDays: interval, Reveal: c.reveal()}, true, nil
		}
	}
	c, err := loadCard(s.db, req.ID, now)
	if err != nil {
		return ReviewResult{}, false, err
	}
	answer := strings.TrimSpace(req.Answer)
	wrong := req.Correct != nil && !*req.Correct
	right := req.Correct != nil && *req.Correct
	if objectiveType(c.Type) && answer != "" {
		// 客观题的对错由引擎自己再批一遍，不只听界面传来的结论。
		if correct, decided := gradeLocal(c, answer); decided {
			wrong, right = wrong || !correct, correct
		}
	}
	if wrong && rating != RatingAgain {
		return ReviewResult{}, false, fmt.Errorf("a wrong answer can only be rated again")
	}
	elapsed := elapsedWholeDays(c.Last, now)
	next, due := planReview(c, rating, practiceSettings(s.db).Retention, now)
	lapses, streak := c.Lapses, c.Streak+1
	if rating == RatingAgain {
		lapses, streak = lapses+1, 0
	}
	interval := due.Sub(now).Hours() / 24
	// 复习记录里的对错：1 是对，0 是错，-1 是没有结论。任何一方说错就记成错。
	correctCol := -1
	switch {
	case wrong:
		correctCol = 0
	case right:
		correctCol = 1
	}
	at := formatTime(now)
	err = s.inTx(func(tx *sql.Tx) error {
		// interval_days 一并写上，还在读旧列的代码看到的间隔不会过时。
		if _, err := tx.Exec(`UPDATE practice_state SET stability = ?, difficulty = ?, due_at = ?, last_review_at = ?,
			review_count = review_count + 1, lapses = ?, streak = ?, interval_days = ? WHERE question_id = ?`,
			next.Stability, next.Difficulty, formatTime(due), at, lapses, streak, interval, c.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO practice_events
			(id, question_id, rating, at, request_id, answer, correct, elapsed_days, stability, difficulty, interval_days, state_before, reps_before)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			newKey("pev"), c.ID, rating.String(), at, requestID, clipRunes(answer, 4000), correctCol, elapsed,
			next.Stability, next.Difficulty, interval, c.State, c.Reps); err != nil {
			return err
		}
		if wrong || rating == RatingAgain {
			if _, err := tx.Exec(`UPDATE questions SET mistake = 1, first_wrong_at = CASE WHEN first_wrong_at = '' THEN ? ELSE first_wrong_at END WHERE id = ?`,
				at, c.ID); err != nil {
				return err
			}
		}
		if answer != "" {
			if _, err := tx.Exec(`UPDATE questions SET last_answer = ? WHERE id = ?`, clipRunes(answer, 4000), c.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ReviewResult{}, false, err
	}
	c, err = loadCard(s.db, c.ID, now)
	if err != nil {
		return ReviewResult{}, false, err
	}
	return ReviewResult{Question: c.Question, IntervalDays: interval, Reveal: c.reveal()}, false, nil
}
