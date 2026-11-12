// 证据和回放。知识点上的掌握分、状态和记忆日程都由证据按顺序重新算出来，不单独累加。
// 这样作废一条证据、改目标记住率或迁移旧库之后，结果和一路做下来的完全一样。

package learn

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// evidenceQuiz 是一次答题。
	evidenceQuiz = "quiz"
	// evidenceQualitative 是一次讲解测评。
	evidenceQualitative = "qualitative"
	// evidenceClaim 是学习者自己声明会了或撤回声明。result 是 on 或 off。
	evidenceClaim = "claim"
	// evidenceSeed 是迁移旧库时抄过来的记忆状态。feedback 里是 seedState 的 JSON。
	evidenceSeed = "seed"

	// ResultCorrect 是答对。
	ResultCorrect = "correct"
	// ResultPartial 是答对一部分。
	ResultPartial = "partial"
	// ResultIncorrect 是答错。
	ResultIncorrect = "incorrect"
	// ResultUngraded 是没法判定。它不进证据，不加分也不扣分。
	ResultUngraded = "ungraded"
)

// evidenceRow 是一条证据。Independent 为假表示作答前看过提示，不计入掌握分。
type evidenceRow struct {
	ID            string
	PathID        string
	PointID       string
	InteractionID string
	Kind          string
	Result        string
	Answer        string
	Feedback      string
	SessionID     string
	Independent   bool
	At            time.Time
}

// seedState 是迁移时写进 seed 证据的记忆状态。
type seedState struct {
	Stability  float64 `json:"stability"`
	Difficulty float64 `json:"difficulty"`
	Reps       int     `json:"reps"`
	Last       string  `json:"last_review_at"`
	Due        string  `json:"due_at"`
}

// appendEvidence 追加一条证据。ID 为空时生成一个。At 为零值时返回错误，时间由调用方给定。
func appendEvidence(q querier, e evidenceRow) error {
	if e.ID == "" {
		e.ID = newKey("ev")
	}
	if e.At.IsZero() {
		return fmt.Errorf("evidence needs a time")
	}
	_, err := q.Exec(`INSERT INTO evidence (id, path_id, point_id, interaction_id, kind, result, independent, answer, feedback, session_id, at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.PathID, e.PointID, e.InteractionID, e.Kind, e.Result, boolInt(e.Independent),
		clipRunes(e.Answer, 4000), clipRunes(e.Feedback, 2000), e.SessionID, formatTime(e.At))
	return err
}

// pointReplay 是回放一个知识点的证据后得到的状态。
type pointReplay struct {
	Credits     []float64
	Qualitative bool
	Claimed     bool
	Seen        bool
	Mem         Memory
	Reps        int
	Lapses      int
	Last        time.Time
	Due         time.Time
	MasteredAt  time.Time
}

// assessed 在测评证据足够过门时为真。kind 是知识点类型。学习者的声明不算。
func (r pointReplay) assessed(kind string) bool {
	if qualitativeKind(kind) {
		return r.Qualitative
	}
	return ComputeMastery(r.Credits) >= quantitativeGate
}

// mastered 在测评过门或学习者声明会了时为真。
func (r pointReplay) mastered(kind string) bool { return r.Claimed || r.assessed(kind) }

// initialMemory 是知识点第一次过门时的记忆状态。类型越偏理解和设计，第一次复习隔得越久。
// 这是按类型给的先验，之后每次复习由 FSRS 自己调整。
func initialMemory(kind string) Memory {
	base := NextMemory(Memory{}, 0, RatingGood)
	switch kind {
	case KindProcedure:
		base.Stability = 3.5
	case KindConcept:
		base.Stability = 5
	case KindDesign:
		base.Stability = fsrsWeights[3]
	}
	return base
}

// ratingOf 把一次作答的结果折成复习评分。看过提示才答对的只算吃力。
func ratingOf(result string, independent bool) Rating {
	switch result {
	case ResultCorrect:
		if independent {
			return RatingGood
		}
		return RatingHard
	case ResultPartial:
		return RatingHard
	default:
		return RatingAgain
	}
}

// creditOf 把结果折成掌握分用的得分。
func creditOf(result string) float64 {
	switch result {
	case ResultCorrect:
		return 1
	case ResultPartial:
		return 0.5
	default:
		return 0
	}
}

// replayPoint 按顺序回放 rows，算出一个知识点现在的状态。kind 是知识点类型，retention 是路径的目标记住率。
// 没过门时记忆状态不动。过门那一刻取初始记忆，之后每次作答都算一次复习。
// 学习者声明会了之后，第一次独立作答答错会撤掉声明：证据比声明说了算。
func replayPoint(kind string, retention float64, rows []evidenceRow) pointReplay {
	var st pointReplay
	for _, e := range rows {
		switch e.Kind {
		case evidenceSeed:
			var seed seedState
			if json.Unmarshal([]byte(e.Feedback), &seed) == nil && seed.Stability > 0 {
				st.Mem = Memory{Stability: seed.Stability, Difficulty: clampDifficulty(seed.Difficulty)}
				st.Reps = seed.Reps
				st.Last = parseTime(seed.Last)
				st.Due = parseTime(seed.Due)
				if st.MasteredAt.IsZero() {
					st.MasteredAt = st.Last
				}
			}
			continue
		case evidenceClaim:
			st.Claimed = e.Result == "on"
			if st.Claimed && !st.Mem.Seen() {
				// 声明会了的点很快就排一次复习，用来核实。
				st.Mem = NextMemory(Memory{}, 0, RatingHard)
				st.Last = e.At
				st.Due = dueAfter(e.At, st.Mem, retention, 1)
				if st.MasteredAt.IsZero() {
					st.MasteredAt = e.At
				}
			}
			continue
		case evidenceQuiz, evidenceQualitative:
		default:
			continue
		}
		was := st.mastered(kind)
		st.Seen = true
		if e.Kind == evidenceQuiz {
			if e.Independent {
				st.Credits = append(st.Credits, creditOf(e.Result))
			}
		} else {
			st.Qualitative = e.Result == ResultCorrect
		}
		if st.Claimed && e.Result == ResultIncorrect {
			st.Claimed = false
		}
		now := st.mastered(kind)
		switch {
		case was:
			rating := ratingOf(e.Result, e.Independent)
			st.Mem = NextMemory(st.Mem, elapsedWholeDays(st.Last, e.At), rating)
			st.Reps++
			if rating == RatingAgain {
				st.Lapses++
			}
			st.Last = e.At
			st.Due = dueAfter(e.At, st.Mem, retention, 1)
		case now:
			if st.Mem.Seen() {
				st.Mem = NextMemory(st.Mem, elapsedWholeDays(st.Last, e.At), RatingGood)
			} else {
				st.Mem = initialMemory(kind)
			}
			st.Reps++
			st.Last = e.At
			st.Due = dueAfter(e.At, st.Mem, retention, 1)
			if st.MasteredAt.IsZero() {
				st.MasteredAt = e.At
			}
		}
	}
	return st
}

// loadEvidence 读出一个知识点没作废的证据，按写入顺序。
func loadEvidence(q querier, pointID string) ([]evidenceRow, error) {
	rows, err := q.Query(`SELECT id, path_id, point_id, interaction_id, kind, result, independent, answer, feedback, session_id, at
		FROM evidence WHERE point_id = ? AND voided = 0 ORDER BY seq`, pointID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []evidenceRow
	for rows.Next() {
		var e evidenceRow
		var independent int
		var at string
		if err := rows.Scan(&e.ID, &e.PathID, &e.PointID, &e.InteractionID, &e.Kind, &e.Result, &independent, &e.Answer, &e.Feedback, &e.SessionID, &at); err != nil {
			return nil, err
		}
		e.Independent = independent != 0
		e.At = parseTime(at)
		out = append(out, e)
	}
	return out, rows.Err()
}

// rebuildPoint 回放一个知识点的证据并把结果写回它的行。kind 是知识点类型，retention 是路径的目标记住率。
func rebuildPoint(q querier, pointID, kind string, retention float64) (pointReplay, error) {
	rows, err := loadEvidence(q, pointID)
	if err != nil {
		return pointReplay{}, err
	}
	st := replayPoint(kind, retention, rows)
	mastery := ComputeMastery(st.Credits)
	if qualitativeKind(kind) {
		switch {
		case st.Qualitative:
			mastery = 1
		case st.Seen:
			mastery = 0.4
		default:
			mastery = 0
		}
	}
	status := StatusNew
	source := ""
	switch {
	case st.assessed(kind):
		status, source = StatusMastered, "system"
	case st.Claimed:
		status, source = StatusMastered, "learner"
	case st.Seen:
		status = StatusLearning
	}
	credits, _ := json.Marshal(st.Credits)
	if st.Credits == nil {
		credits = []byte("[]")
	}
	_, err = q.Exec(`UPDATE points SET mastery = ?, qualitative = ?, mastery_source = ?, claimed = ?, status = ?, attempts_json = ?,
		stability = ?, difficulty = ?, reps = ?, lapses = ?, last_review_at = ?, due_at = ?, mastered_at = ? WHERE id = ?`,
		mastery, boolInt(st.Qualitative), source, boolInt(st.Claimed), status, string(credits),
		st.Mem.Stability, st.Mem.Difficulty, st.Reps, st.Lapses, formatTime(st.Last), formatTime(st.Due), formatTime(st.MasteredAt), pointID)
	return st, err
}

// rebuildPath 回放一条路径上每个知识点的证据。改了目标记住率或迁移之后用它。
func rebuildPath(q querier, pathID string) error {
	var retention float64
	if err := q.QueryRow(`SELECT desired_retention FROM paths WHERE id = ?`, pathID).Scan(&retention); err != nil {
		return fmt.Errorf("path not found")
	}
	retention = ClampRetention(retention)
	rows, err := q.Query(`SELECT id, kind FROM points WHERE path_id = ?`, pathID)
	if err != nil {
		return err
	}
	type pt struct{ id, kind string }
	var list []pt
	for rows.Next() {
		var p pt
		if err := rows.Scan(&p.id, &p.kind); err == nil {
			list = append(list, p)
		}
	}
	rows.Close()
	for _, p := range list {
		if _, err := rebuildPoint(q, p.id, p.kind, retention); err != nil {
			return err
		}
	}
	return nil
}

// boolInt 把布尔写成 SQLite 里的 0 或 1。
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
