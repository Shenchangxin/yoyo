// 掌握路径上的题目：出题、作答、跳过、修题、讲解测评、声明会了和暂缓。
// 一条路径同时只有一道待答的题。标准答案和解析在判完之前不出库。判分时不持有库锁。

package learn

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

const (
	// QuestionChoice 是单选题。作答是选项的字母。
	QuestionChoice = "choice"
	// QuestionShort 是简答题，答案是一个词或一句话。
	QuestionShort = "short"
	// QuestionOpen 是开放题，按意思判分。
	QuestionOpen = "open"

	interactionPending = "pending"
	interactionGraded  = "graded"
	interactionSkipped = "skipped"
	interactionVoided  = "voided"
)

// Option 是选择题的一个选项。Label 是 A、B、C 这样的字母，Body 是选项正文。
type Option struct {
	Label string `json:"label"`
	Body  string `json:"body"`
}

// PublicQuestion 是给模型和界面的题目。没有标准答案，也没有解析。
type PublicQuestion struct {
	ID         string   `json:"id"`
	PointID    string   `json:"point_id"`
	Type       string   `json:"question_type"`
	Prompt     string   `json:"prompt"`
	Options    []Option `json:"options,omitempty"`
	Difficulty string   `json:"difficulty,omitempty"`
	Purpose    string   `json:"purpose,omitempty"`
	CreatedAt  string   `json:"created_at,omitempty"`
}

// storedQuestion 是库里的一道题，带标准答案。只在这个包里流转。
type storedQuestion struct {
	ID, PathID, PointID, SessionID, Status, Type, Prompt, Expected, Explanation string
	UserAnswer, Difficulty, Purpose, Result, Feedback, CreatedAt                string
	Options                                                                     []Option
	Accepted                                                                    []string
	Hinted                                                                      bool
}

// public 去掉标准答案和解析。
func (q storedQuestion) public() PublicQuestion {
	return PublicQuestion{ID: q.ID, PointID: q.PointID, Type: q.Type, Prompt: q.Prompt, Options: q.Options, Difficulty: q.Difficulty, Purpose: q.Purpose, CreatedAt: q.CreatedAt}
}

const questionCols = `id, path_id, point_id, session_id, status, question_type, prompt, expected_answer, options_json, explanation,
	user_answer, difficulty, purpose, accepted_json, hinted, result, feedback, created_at`

// scanQuestion 从一行读出题目。列顺序和 questionCols 相同。
func scanQuestion(row rower) (storedQuestion, error) {
	var q storedQuestion
	var options, accepted string
	var hinted int
	err := row.Scan(&q.ID, &q.PathID, &q.PointID, &q.SessionID, &q.Status, &q.Type, &q.Prompt, &q.Expected, &options, &q.Explanation,
		&q.UserAnswer, &q.Difficulty, &q.Purpose, &accepted, &hinted, &q.Result, &q.Feedback, &q.CreatedAt)
	if err != nil {
		return storedQuestion{}, err
	}
	q.Options = decodeOptions(options)
	_ = json.Unmarshal([]byte(accepted), &q.Accepted)
	q.Hinted = hinted != 0
	return q, nil
}

// decodeOptions 解出选项。旧库里选项是字符串数组，这里按顺序补上 A、B、C。
func decodeOptions(raw string) []Option {
	var opts []Option
	if json.Unmarshal([]byte(raw), &opts) == nil && len(opts) > 0 && opts[0].Body != "" {
		return opts
	}
	var plain []string
	if json.Unmarshal([]byte(raw), &plain) != nil {
		return nil
	}
	opts = opts[:0]
	for i, body := range plain {
		if body = strings.TrimSpace(body); body != "" && i < 26 {
			opts = append(opts, Option{Label: string(rune('A' + len(opts))), Body: body})
		}
	}
	return opts
}

// pendingQuestion 读出这条路径上待答的那道题。没有时返回 nil。
func pendingQuestion(q querier, pathID string) (*storedQuestion, error) {
	row := q.QueryRow(`SELECT `+questionCols+` FROM interactions WHERE path_id = ? AND status = ? ORDER BY created_at LIMIT 1`, pathID, interactionPending)
	sq, err := scanQuestion(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sq, nil
}

// PoseRequest 是出一道题的入参。Expected 不能空。选择题的 Expected 是选项字母，或和某个选项正文完全相同。
// Purpose 是 probe、practice 或 review，空着时按知识点状态取。Hinted 为真表示出题前已经给过这道题的提示，答对也不计入掌握分。
type PoseRequest struct {
	PathID      string
	PointID     string
	SessionID   string
	Type        string
	Prompt      string
	Expected    string
	Options     []Option
	Accepted    []string
	Explanation string
	Difficulty  string
	Purpose     string
	Hinted      bool
}

// PoseQuiz 留下一道待答的题。已经有待答的题时不新出，返回原来那道，already 为真。
// 选择题的选项在存库前打乱并重排字母，避免正确项总在同一个位置。
func (s *Store) PoseQuiz(req PoseRequest) (question PublicQuestion, already bool, err error) {
	if s == nil {
		return PublicQuestion{}, false, fmt.Errorf("learn store unavailable")
	}
	prompt := strings.TrimSpace(req.Prompt)
	expected := strings.TrimSpace(req.Expected)
	if prompt == "" || expected == "" || strings.TrimSpace(req.PointID) == "" {
		return PublicQuestion{}, false, fmt.Errorf("a question needs point_id, prompt, and expected_answer")
	}
	qtype := strings.ToLower(strings.TrimSpace(req.Type))
	options := cleanOptions(req.Options)
	if qtype == "" {
		qtype = QuestionShort
		if len(options) > 0 {
			qtype = QuestionChoice
		}
	}
	switch qtype {
	case QuestionChoice, QuestionShort, QuestionOpen:
	default:
		return PublicQuestion{}, false, fmt.Errorf("question_type must be choice, short, or open")
	}
	id := newKey("q")
	if qtype == QuestionChoice {
		if len(options) < 2 || len(options) > 8 {
			return PublicQuestion{}, false, fmt.Errorf("a choice question needs two to eight options, each with a body")
		}
		seen := map[string]bool{}
		for _, o := range options {
			key := strings.ToLower(o.Body)
			if seen[key] {
				return PublicQuestion{}, false, fmt.Errorf("two options have the same body; give each option a distinct body")
			}
			seen[key] = true
		}
		label, ok := resolveChoice(options, expected)
		if !ok {
			return PublicQuestion{}, false, fmt.Errorf("expected_answer must be an option label such as A, or exactly match one option body")
		}
		options, expected = shuffleOptions(options, label, id)
	} else {
		options = nil
	}
	difficulty := strings.ToLower(strings.TrimSpace(req.Difficulty))
	switch difficulty {
	case "easy", "medium", "hard":
	default:
		difficulty = ""
	}
	accepted := make([]string, 0, len(req.Accepted))
	for _, a := range req.Accepted {
		if a = strings.TrimSpace(a); a != "" && len(accepted) < 8 {
			accepted = append(accepted, a)
		}
	}
	optionsJSON, _ := json.Marshal(options)
	acceptedJSON, _ := json.Marshal(accepted)
	now := s.clock().Format(time.RFC3339Nano)

	s.mu.Lock()
	var status string
	if err := s.db.QueryRow(`SELECT status FROM points WHERE id = ? AND path_id = ?`, req.PointID, req.PathID).Scan(&status); err != nil {
		s.mu.Unlock()
		return PublicQuestion{}, false, fmt.Errorf("point not found on this path; read the map again for current ids")
	}
	pending, err := pendingQuestion(s.db, req.PathID)
	if err != nil {
		s.mu.Unlock()
		return PublicQuestion{}, false, err
	}
	if pending != nil {
		s.mu.Unlock()
		return pending.public(), true, nil
	}
	purpose := strings.ToLower(strings.TrimSpace(req.Purpose))
	switch purpose {
	case "probe", "practice", "review":
	default:
		switch status {
		case StatusMastered:
			purpose = "review"
		case StatusLearning:
			purpose = "practice"
		default:
			purpose = "probe"
		}
	}
	_, err = s.db.Exec(`INSERT INTO interactions
		(id, path_id, point_id, session_id, status, question_type, prompt, expected_answer, options_json, explanation, difficulty, purpose, accepted_json, hinted, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.PathID, req.PointID, req.SessionID, interactionPending, qtype, clipRunes(prompt, 4000), clipRunes(expected, 2000),
		string(optionsJSON), clipRunes(strings.TrimSpace(req.Explanation), 4000), difficulty, purpose, string(acceptedJSON), boolInt(req.Hinted), now)
	s.mu.Unlock()
	if err != nil {
		return PublicQuestion{}, false, err
	}
	s.notify("path", req.PathID, "question")
	return PublicQuestion{ID: id, PointID: req.PointID, Type: qtype, Prompt: prompt, Options: options, Difficulty: difficulty, Purpose: purpose, CreatedAt: now}, false, nil
}

// cleanOptions 去掉正文为空的选项，并去掉正文两端的空白。
func cleanOptions(in []Option) []Option {
	out := make([]Option, 0, len(in))
	for _, o := range in {
		body := strings.TrimSpace(o.Body)
		if body == "" {
			continue
		}
		out = append(out, Option{Label: strings.ToUpper(strings.TrimSpace(o.Label)), Body: body})
	}
	for i := range out {
		if out[i].Label == "" {
			out[i].Label = string(rune('A' + i))
		}
	}
	return out
}

// shuffleOptions 打乱选项并从 A 起重排字母。correct 是打乱前正确项的字母，返回打乱后的选项和正确项的新字母。
// seed 通常是题目 id，同一道题每次打乱的结果相同。
func shuffleOptions(options []Option, correct, seed string) ([]Option, string) {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))
	out := append([]Option(nil), options...)
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	label := ""
	for i := range out {
		next := string(rune('A' + i))
		if out[i].Label == correct {
			label = next
		}
		out[i].Label = next
	}
	return out, label
}

var choiceLead = regexp.MustCompile(`(?i)^(?:我?选(?:择)?|答案(?:是|为)?|应该是|是|选项|option|answer(?:\s+is)?|i\s+(?:pick|choose)|it'?s)\s*`)

// resolveChoice 从 text 里认出选的是哪个选项。只接受单独的字母，或和某个选项正文完全相同的文字。
// 带问号的话、解释性的长句都认不出，返回 false：认不出不等于答错。
func resolveChoice(options []Option, text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	lower := strings.ToLower(text)
	for _, o := range options {
		if strings.ToLower(o.Body) == lower {
			return o.Label, true
		}
	}
	if strings.ContainsAny(text, "?？") {
		return "", false
	}
	core := strings.Trim(text, " \t\r\n.。:：,，;；!！()（）[]【】\"'“”‘’")
	core = choiceLead.ReplaceAllString(core, "")
	core = strings.ToUpper(strings.Trim(core, " \t.。:：()（）"))
	for _, o := range options {
		if core == o.Label {
			return o.Label, true
		}
	}
	return "", false
}

// GradeResult 是一次作答的判定。Final 为假表示没法判定，题目还在等，不加分也不扣分。
// Expected 和 Explanation 只在 Final 为真时带出，界面在判完之后才显示它们。
type GradeResult struct {
	QuestionID  string    `json:"question_id"`
	PointID     string    `json:"point_id"`
	Result      string    `json:"result"`
	Correct     bool      `json:"correct"`
	Final       bool      `json:"final"`
	Replayed    bool      `json:"replayed,omitempty"`
	Feedback    string    `json:"feedback,omitempty"`
	Expected    string    `json:"expected,omitempty"`
	Explanation string    `json:"explanation,omitempty"`
	Mastery     float64   `json:"mastery"`
	Mastered    bool      `json:"mastered"`
	Status      string    `json:"status"`
	Gate        string    `json:"gate"`
	Next        *NextStep `json:"next,omitempty"`
}

// AnswerRequest 是学习者的一次作答。GiveUp 为真表示学习者说不会，按答错记。
// QuestionID 非空时用来核对：那道题已经判过就返回当时的结果，不重复计分。
type AnswerRequest struct {
	PathID     string
	QuestionID string
	SessionID  string
	Answer     string
	GiveUp     bool
}

// Answer 批改这条路径上待答的那道题。能直接对上的在本地判，其余交给模型按意思判。
// 模型把握不足时结果是 ungraded，题目继续等学习者说清楚。ctx 取消时返回错误，题目不动。
func (s *Store) Answer(ctx context.Context, req AnswerRequest) (GradeResult, error) {
	if s == nil {
		return GradeResult{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	pending, err := pendingQuestion(s.db, req.PathID)
	if err != nil {
		s.mu.Unlock()
		return GradeResult{}, err
	}
	if pending == nil || (req.QuestionID != "" && pending.ID != req.QuestionID) {
		if req.QuestionID == "" {
			s.mu.Unlock()
			return GradeResult{}, fmt.Errorf("no question is waiting")
		}
		// 同一道题的重复提交返回当时的判定，不重复计分。
		res, replayErr := s.replayGrade(req.PathID, req.QuestionID)
		s.mu.Unlock()
		return res, replayErr
	}
	q := *pending
	var pointName string
	_ = s.db.QueryRow(`SELECT name FROM points WHERE id = ?`, q.PointID).Scan(&pointName)
	s.mu.Unlock()

	v := verdict{Result: ResultIncorrect, Final: true}
	if !req.GiveUp {
		var decided bool
		v, decided = gradeDeterministic(q, req.Answer)
		if !decided {
			v, err = s.judgeAnswer(ctx, req.SessionID, q, pointName, req.Answer)
			if err != nil {
				return GradeResult{}, err
			}
		}
	}
	answer := strings.TrimSpace(req.Answer)
	now := s.clock()

	s.mu.Lock()
	defer s.mu.Unlock()
	cur, err := pendingQuestion(s.db, req.PathID)
	if err != nil {
		return GradeResult{}, err
	}
	if cur == nil || cur.ID != q.ID {
		return s.replayGrade(req.PathID, q.ID)
	}
	if !v.Final {
		_, _ = s.db.Exec(`UPDATE interactions SET user_answer = ? WHERE id = ?`, clipRunes(answer, 4000), q.ID)
		res := GradeResult{QuestionID: q.ID, PointID: q.PointID, Result: ResultUngraded, Feedback: v.Feedback}
		s.fillGrade(s.db, &res, req.PathID)
		return res, nil
	}
	err = s.inTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`UPDATE interactions SET status = ?, user_answer = ?, result = ?, feedback = ?, score = ?, graded_at = ?, told = 0 WHERE id = ?`,
			interactionGraded, clipRunes(answer, 4000), v.Result, clipRunes(v.Feedback, 2000), creditOf(v.Result), formatTime(now), q.ID); err != nil {
			return err
		}
		if err := appendEvidence(tx, evidenceRow{
			PathID: q.PathID, PointID: q.PointID, InteractionID: q.ID, Kind: evidenceQuiz, Result: v.Result,
			Independent: !q.Hinted, Answer: answer, Feedback: v.Feedback, SessionID: req.SessionID, At: now,
		}); err != nil {
			return err
		}
		if err := rebuildOne(tx, q.PathID, q.PointID); err != nil {
			return err
		}
		if v.Result != ResultCorrect {
			if err := bankMiss(tx, q, answer, now); err != nil {
				return err
			}
		}
		_, err := tx.Exec(`UPDATE paths SET updated_at = ? WHERE id = ?`, formatTime(now), q.PathID)
		return err
	})
	if err != nil {
		return GradeResult{}, err
	}
	res := GradeResult{
		QuestionID: q.ID, PointID: q.PointID, Result: v.Result, Correct: v.Result == ResultCorrect, Final: true,
		Feedback: v.Feedback, Expected: q.Expected, Explanation: q.Explanation,
	}
	s.fillGrade(s.db, &res, req.PathID)
	go s.notify("path", req.PathID, "graded")
	return res, nil
}

// rebuildOne 回放一个知识点。它自己读出类型和路径的目标记住率。
func rebuildOne(q querier, pathID, pointID string) error {
	var kind string
	var retention float64
	if err := q.QueryRow(`SELECT p.kind, paths.desired_retention FROM points p JOIN paths ON paths.id = p.path_id WHERE p.id = ? AND p.path_id = ?`, pointID, pathID).
		Scan(&kind, &retention); err != nil {
		return fmt.Errorf("point not found")
	}
	_, err := rebuildPoint(q, pointID, kind, ClampRetention(retention))
	return err
}

// fillGrade 往 res 里补上知识点现在的掌握分、状态和下一步。调用方要已经持有 s.mu。
func (s *Store) fillGrade(q querier, res *GradeResult, pathID string) {
	st, err := loadPath(q, pathID)
	if err != nil {
		return
	}
	for _, p := range st.Points {
		if p.ID != res.PointID {
			continue
		}
		res.Mastery, res.Status, res.Mastered = p.Mastery, p.Status, p.mastered()
		res.Gate = "quantitative"
		if qualitativeKind(p.Kind) {
			res.Gate = "qualitative"
		}
	}
	step := st.next(s.clock())
	res.Next = &step
}

// replayGrade 返回一道已经判过的题当时的结果。questionID 必须是这条路径上判过的题。调用方要已经持有 s.mu。
func (s *Store) replayGrade(pathID, questionID string) (GradeResult, error) {
	q, err := scanQuestion(s.db.QueryRow(`SELECT `+questionCols+` FROM interactions WHERE path_id = ? AND status = ? AND id = ?`,
		pathID, interactionGraded, questionID))
	if err != nil {
		return GradeResult{}, fmt.Errorf("no question is waiting")
	}
	res := GradeResult{
		QuestionID: q.ID, PointID: q.PointID, Result: q.Result, Correct: q.Result == ResultCorrect, Final: true, Replayed: true,
		Feedback: q.Feedback, Expected: q.Expected, Explanation: q.Explanation,
	}
	s.fillGrade(s.db, &res, pathID)
	return res, nil
}

// bankMiss 把答错或只答对一部分的题放进题库，标成错题，一天后到期。同一道题只进一次。
func bankMiss(q querier, sq storedQuestion, answer string, now time.Time) error {
	_, _, err := bankInsert(q, bankEntry{
		Stem: sq.Prompt, Type: sq.Type, Options: sq.Options, Answer: sq.Expected, Explanation: sq.Explanation, Difficulty: sq.Difficulty,
		Source: "mastery", SourceID: sq.ID, OriginKey: "mastery:" + sq.ID, PathID: sq.PathID, PointID: sq.PointID,
		Mistake: true, LastAnswer: answer,
	}, now)
	return err
}

// SkipQuestion 放弃这条路径上待答的那道题，不计分。没有待答的题时返回错误。
func (s *Store) SkipQuestion(pathID string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	res, err := s.db.Exec(`UPDATE interactions SET status = ?, graded_at = ? WHERE path_id = ? AND status = ?`,
		interactionSkipped, formatTime(s.clock()), pathID, interactionPending)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("no question is waiting")
	}
	s.notify("path", pathID, "skipped")
	return nil
}

// RepairRequest 是修一道题的入参。Action 是 void 或 correct。QuestionID 为空时取最近判过的那道。
// correct 需要 Expected，是这道题真正的标准答案。
type RepairRequest struct {
	PathID     string
	QuestionID string
	SessionID  string
	Action     string
	Expected   string
}

// RepairQuestion 处理出错的题。void 作废这次作答，它不再计入掌握分和日程。correct 换上正确的标准答案并重新批改当时的作答。
// 两种做法之后都重新回放这个知识点。重新批改没法判定时，这次作答也作废。
func (s *Store) RepairQuestion(ctx context.Context, req RepairRequest) (GradeResult, error) {
	if s == nil {
		return GradeResult{}, fmt.Errorf("learn store unavailable")
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "void" && action != "correct" {
		return GradeResult{}, fmt.Errorf("action must be void or correct")
	}
	expected := strings.TrimSpace(req.Expected)
	if action == "correct" && expected == "" {
		return GradeResult{}, fmt.Errorf("correct needs the true expected_answer")
	}
	s.mu.Lock()
	query := `SELECT ` + questionCols + ` FROM interactions WHERE path_id = ? AND status IN (?, ?) ORDER BY created_at DESC LIMIT 1`
	args := []any{req.PathID, interactionGraded, interactionPending}
	if req.QuestionID != "" {
		query = `SELECT ` + questionCols + ` FROM interactions WHERE path_id = ? AND status IN (?, ?) AND id = ?`
		args = append(args, req.QuestionID)
	}
	q, err := scanQuestion(s.db.QueryRow(query, args...))
	var pointName string
	if err == nil {
		_ = s.db.QueryRow(`SELECT name FROM points WHERE id = ?`, q.PointID).Scan(&pointName)
	}
	s.mu.Unlock()
	if err != nil {
		return GradeResult{}, fmt.Errorf("no question to repair")
	}
	v := verdict{}
	if action == "correct" {
		if q.Type == QuestionChoice {
			label, ok := resolveChoice(q.Options, expected)
			if !ok {
				return GradeResult{}, fmt.Errorf("expected_answer must be an option label on this question")
			}
			expected = label
		}
		q.Expected = expected
		if q.Status == interactionGraded {
			var decided bool
			v, decided = gradeDeterministic(q, q.UserAnswer)
			if !decided {
				if v, err = s.judgeAnswer(ctx, req.SessionID, q, pointName, q.UserAnswer); err != nil {
					return GradeResult{}, err
				}
			}
		}
	}
	now := s.clock()
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.inTx(func(tx *sql.Tx) error {
		void := action == "void" || (q.Status == interactionGraded && !v.Final)
		switch {
		case void:
			if _, err := tx.Exec(`UPDATE interactions SET status = ?, graded_at = ? WHERE id = ?`, interactionVoided, formatTime(now), q.ID); err != nil {
				return err
			}
			if _, err := tx.Exec(`UPDATE evidence SET voided = 1 WHERE interaction_id = ?`, q.ID); err != nil {
				return err
			}
			if err := bankRemoveOrigin(tx, "mastery:"+q.ID); err != nil {
				return err
			}
		case q.Status == interactionPending:
			if _, err := tx.Exec(`UPDATE interactions SET expected_answer = ? WHERE id = ?`, expected, q.ID); err != nil {
				return err
			}
		default:
			if _, err := tx.Exec(`UPDATE interactions SET expected_answer = ?, result = ?, feedback = ?, score = ?, told = 0 WHERE id = ?`,
				expected, v.Result, clipRunes(v.Feedback, 2000), creditOf(v.Result), q.ID); err != nil {
				return err
			}
			if _, err := tx.Exec(`UPDATE evidence SET result = ?, feedback = ? WHERE interaction_id = ?`, v.Result, clipRunes(v.Feedback, 2000), q.ID); err != nil {
				return err
			}
			if _, err := tx.Exec(`UPDATE questions SET answer = ? WHERE origin_key = ?`, expected, "mastery:"+q.ID); err != nil {
				return err
			}
			if v.Result == ResultCorrect {
				if err := bankRemoveOrigin(tx, "mastery:"+q.ID); err != nil {
					return err
				}
			} else if err := bankMiss(tx, q, q.UserAnswer, now); err != nil {
				return err
			}
		}
		return rebuildOne(tx, q.PathID, q.PointID)
	})
	if err != nil {
		return GradeResult{}, err
	}
	res := GradeResult{QuestionID: q.ID, PointID: q.PointID, Result: v.Result, Correct: v.Result == ResultCorrect, Final: v.Final, Feedback: v.Feedback}
	if action == "void" || !v.Final {
		res.Result, res.Final = "voided", false
	}
	s.fillGrade(s.db, &res, req.PathID)
	go s.notify("path", req.PathID, "repaired")
	return res, nil
}

// AssessRequest 是一次讲解测评。Explanation 是学习者自己的原话，不能空。Note 是导师补充的上下文，可空。
type AssessRequest struct {
	PathID      string
	PointID     string
	SessionID   string
	Explanation string
	Note        string
}

// AssessResult 是讲解测评的结果。Final 为假表示把握不足，什么都没记，请学习者再讲清楚。
type AssessResult struct {
	PointID  string    `json:"point_id"`
	Result   string    `json:"result"`
	Final    bool      `json:"final"`
	Feedback string    `json:"feedback,omitempty"`
	Gaps     []string  `json:"gaps,omitempty"`
	Mastered bool      `json:"mastered"`
	Status   string    `json:"status"`
	Next     *NextStep `json:"next,omitempty"`
}

// Assess 判定学习者对一个概念或设计型知识点的讲解。判定由模型按讲解原文给出，调用方不能直接传通过与否。
// 记忆和步骤型知识点不走这里。ctx 取消时返回错误，什么都不记。
func (s *Store) Assess(ctx context.Context, req AssessRequest) (AssessResult, error) {
	if s == nil {
		return AssessResult{}, fmt.Errorf("learn store unavailable")
	}
	explanation := strings.TrimSpace(req.Explanation)
	s.mu.Lock()
	var name, kind, objective string
	err := s.db.QueryRow(`SELECT p.name, p.kind, m.objective FROM points p JOIN modules m ON m.id = p.module_id WHERE p.id = ? AND p.path_id = ?`,
		req.PointID, req.PathID).Scan(&name, &kind, &objective)
	s.mu.Unlock()
	if err != nil {
		return AssessResult{}, fmt.Errorf("point not found on this path; read the map again for current ids")
	}
	if !qualitativeKind(kind) {
		return AssessResult{}, fmt.Errorf("%q is a %s point: gate it with questions, not with an explanation check", name, kind)
	}
	if len([]rune(explanation)) < 8 {
		return AssessResult{}, fmt.Errorf("there is no explanation to assess yet: ask the learner to explain %q in their own words first", name)
	}
	v, err := s.judgeExplanation(ctx, req.SessionID, name, kind, objective, explanation, req.Note)
	if err != nil {
		return AssessResult{}, err
	}
	out := AssessResult{PointID: req.PointID, Result: v.Result, Final: v.Final, Feedback: v.Feedback, Gaps: v.Gaps}
	now := s.clock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if v.Final {
		err = s.inTx(func(tx *sql.Tx) error {
			if err := appendEvidence(tx, evidenceRow{
				PathID: req.PathID, PointID: req.PointID, Kind: evidenceQualitative, Result: v.Result, Independent: true,
				Answer: explanation, Feedback: v.Feedback, SessionID: req.SessionID, At: now,
			}); err != nil {
				return err
			}
			if err := rebuildOne(tx, req.PathID, req.PointID); err != nil {
				return err
			}
			_, err := tx.Exec(`UPDATE paths SET updated_at = ? WHERE id = ?`, formatTime(now), req.PathID)
			return err
		})
		if err != nil {
			return AssessResult{}, err
		}
		go s.notify("path", req.PathID, "assessed")
	} else {
		out.Result = ResultUngraded
	}
	st, err := loadPath(s.db, req.PathID)
	if err != nil {
		return out, nil
	}
	for _, p := range st.Points {
		if p.ID == req.PointID {
			out.Mastered, out.Status = p.mastered(), p.Status
		}
	}
	step := st.next(now)
	out.Next = &step
	return out, nil
}

// ClaimMastery 记下或撤回学习者自己的声明。声明让路径往前走，但地图上标的是 learner，不冒充测评证据。
// on 为真是声明会了，为假是撤回。pathID 和 pointID 必须属于同一条路径。
func (s *Store) ClaimMastery(pathID, pointID string, on bool) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	result := "off"
	if on {
		result = "on"
	}
	now := s.clock()
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		var n int
		_ = tx.QueryRow(`SELECT COUNT(1) FROM points WHERE id = ? AND path_id = ?`, pointID, pathID).Scan(&n)
		if n == 0 {
			return fmt.Errorf("point not found")
		}
		if err := appendEvidence(tx, evidenceRow{PathID: pathID, PointID: pointID, Kind: evidenceClaim, Result: result, Independent: true, At: now}); err != nil {
			return err
		}
		if err := rebuildOne(tx, pathID, pointID); err != nil {
			return err
		}
		_, err := tx.Exec(`UPDATE paths SET updated_at = ? WHERE id = ?`, formatTime(now), pathID)
		return err
	})
	s.mu.Unlock()
	if err == nil {
		s.notify("path", pathID, "claimed")
	}
	return err
}

// DeferPoint 暂缓或恢复一个知识点。暂缓不等于掌握，只是先学别的。暂缓时这个点上待答的题一并放弃。
// on 为真是暂缓，为假是恢复。
func (s *Store) DeferPoint(pathID, pointID string, on bool) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	now := formatTime(s.clock())
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		res, err := tx.Exec(`UPDATE points SET deferred = ? WHERE id = ? AND path_id = ?`, boolInt(on), pointID, pathID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("point not found")
		}
		if on {
			if _, err := tx.Exec(`UPDATE interactions SET status = ?, graded_at = ? WHERE path_id = ? AND point_id = ? AND status = ?`,
				interactionSkipped, now, pathID, pointID, interactionPending); err != nil {
				return err
			}
		}
		_, err = tx.Exec(`UPDATE paths SET updated_at = ? WHERE id = ?`, now, pathID)
		return err
	})
	s.mu.Unlock()
	if err == nil {
		s.notify("path", pathID, "deferred")
	}
	return err
}

// PointEvidence 是知识点详情里的一条记录。题目已经判过，所以带上题干和标准答案。
type PointEvidence struct {
	At          string `json:"at"`
	Kind        string `json:"kind"`
	Result      string `json:"result"`
	Independent bool   `json:"independent"`
	Answer      string `json:"answer,omitempty"`
	Feedback    string `json:"feedback,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	Expected    string `json:"expected,omitempty"`
}

// PointDetail 是一个知识点的详情：状态、记忆和最近的证据。
type PointDetail struct {
	Point      PointView       `json:"point"`
	Module     string          `json:"module"`
	Stability  float64         `json:"stability"`
	Difficulty float64         `json:"difficulty"`
	Reps       int             `json:"reps"`
	Lapses     int             `json:"lapses"`
	LastReview string          `json:"last_review_at,omitempty"`
	Evidence   []PointEvidence `json:"evidence"`
}

// PointDetail 读取一个知识点的详情，证据最多带最近 40 条，新的在前。迁移留下的记忆种子不列出来。
func (s *Store) PointDetail(pathID, pointID string) (PointDetail, error) {
	if s == nil {
		return PointDetail{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := loadPath(s.db, pathID)
	if err != nil {
		return PointDetail{}, err
	}
	view := st.view(s.clock())
	var out PointDetail
	found := false
	for _, m := range view.Modules {
		for _, p := range m.Points {
			if p.ID == pointID {
				out.Point, out.Module, found = p, m.Name, true
			}
		}
	}
	if !found {
		return PointDetail{}, fmt.Errorf("point not found")
	}
	for _, p := range st.Points {
		if p.ID == pointID {
			out.Stability, out.Difficulty, out.Reps, out.Lapses, out.LastReview = p.Mem.Stability, p.Mem.Difficulty, p.Reps, p.Lapses, formatTime(p.Last)
		}
	}
	rows, err := s.db.Query(`SELECT e.at, e.kind, e.result, e.independent, e.answer, e.feedback, COALESCE(i.prompt, ''), COALESCE(i.expected_answer, '')
		FROM evidence e LEFT JOIN interactions i ON i.id = e.interaction_id
		WHERE e.point_id = ? AND e.voided = 0 AND e.kind != ? ORDER BY e.seq DESC LIMIT 40`, pointID, evidenceSeed)
	if err != nil {
		return PointDetail{}, err
	}
	defer rows.Close()
	out.Evidence = []PointEvidence{}
	for rows.Next() {
		var e PointEvidence
		var independent int
		if err := rows.Scan(&e.At, &e.Kind, &e.Result, &independent, &e.Answer, &e.Feedback, &e.Prompt, &e.Expected); err != nil {
			continue
		}
		e.Independent = independent != 0
		out.Evidence = append(out.Evidence, e)
	}
	return out, nil
}

// QuestionState 是一道题现在的样子，给对话里的题卡用。Status 是 pending、graded、skipped 或 voided。
// 标准答案和解析只在判完之后带出。
type QuestionState struct {
	Question    PublicQuestion `json:"question"`
	Status      string         `json:"status"`
	Answer      string         `json:"answer,omitempty"`
	Result      string         `json:"result,omitempty"`
	Feedback    string         `json:"feedback,omitempty"`
	Expected    string         `json:"expected,omitempty"`
	Explanation string         `json:"explanation,omitempty"`
}

// QuestionState 读取一道题的状态。questionID 必须是 pathID 这条路径上的题。
func (s *Store) QuestionState(pathID, questionID string) (QuestionState, error) {
	if s == nil {
		return QuestionState{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	q, err := scanQuestion(s.db.QueryRow(`SELECT `+questionCols+` FROM interactions WHERE path_id = ? AND id = ?`, pathID, questionID))
	if err != nil {
		return QuestionState{}, fmt.Errorf("question not found")
	}
	out := QuestionState{Question: q.public(), Status: q.Status, Answer: q.UserAnswer}
	if q.Status == interactionGraded {
		out.Result, out.Feedback, out.Expected, out.Explanation = q.Result, q.Feedback, q.Expected, q.Explanation
	}
	return out, nil
}

// TutorBrief 是每一轮开头给导师的简报：刚判完还没告诉导师的题，以及待答的题。
type TutorBrief struct {
	Graded  *GradeResult    `json:"graded,omitempty"`
	Pending *PublicQuestion `json:"pending,omitempty"`
}

// TakeTutorBrief 取出这条路径上还没告诉导师的判定，并标成已告知。每条判定只带出一次。
// 学习者在卡片上作答时引擎先判完，导师靠这份简报知道结果，不用再判一遍。
func (s *Store) TakeTutorBrief(pathID string) TutorBrief {
	var out TutorBrief
	if s == nil {
		return out
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	q, err := scanQuestion(s.db.QueryRow(`SELECT `+questionCols+` FROM interactions WHERE path_id = ? AND status = ? AND told = 0 ORDER BY graded_at DESC LIMIT 1`,
		pathID, interactionGraded))
	if err == nil {
		res := GradeResult{
			QuestionID: q.ID, PointID: q.PointID, Result: q.Result, Correct: q.Result == ResultCorrect, Final: true,
			Feedback: q.Feedback, Expected: q.Expected, Explanation: q.Explanation,
		}
		s.fillGrade(s.db, &res, pathID)
		out.Graded = &res
		_, _ = s.db.Exec(`UPDATE interactions SET told = 1 WHERE path_id = ? AND status = ? AND told = 0`, pathID, interactionGraded)
	}
	if pending, err := pendingQuestion(s.db, pathID); err == nil && pending != nil {
		pub := pending.public()
		out.Pending = &pub
	}
	return out
}

// MarkTold 把这条路径上判过的题都标成导师已知。导师自己调用判分工具之后用它，避免下一轮重复通报。
func (s *Store) MarkTold(pathID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.db.Exec(`UPDATE interactions SET told = 1 WHERE path_id = ? AND status = ? AND told = 0`, pathID, interactionGraded)
}
