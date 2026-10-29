// 练习题用 SM-2 冷启动。Again 是 10/1440 天。客观题在这里对答案，填空和简答由学习者自己打分。

package learn

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	practiceDay   = 24 * time.Hour
	practiceQueue = 20
	againDays     = 10.0 / 1440.0
)

// PracticeState 是一道题的复习状态。
type PracticeState struct {
	IntervalDays float64
	Ease         float64
	Streak       int
	Lapses       int
	ReviewCount  int
	Due          time.Time
}

// PracticeDesk 是练习泳道顶部的两个数。
type PracticeDesk struct {
	Due      int `json:"due"`
	Reviewed int `json:"reviewed"`
}

// SchedulePractice 按评分推进间隔。rating 只能是 again、hard、good、easy。now 是这次复习的时刻。
// state 是复习前的状态。返回的间隔不超过 365 天。
func SchedulePractice(state PracticeState, rating string, now time.Time) (PracticeState, error) {
	interval := state.IntervalDays
	if interval == 0 {
		interval = 1
	}
	ease := state.Ease
	if ease == 0 {
		ease = 2.5
	}
	streak := state.Streak
	lapses := state.Lapses
	switch rating {
	case "again":
		interval, streak, lapses = againDays, 0, lapses+1
		if ease-0.2 < 1.3 {
			ease = 1.3
		} else {
			ease -= 0.2
		}
	case "hard":
		interval = interval * 1.2
		if interval < 1 {
			interval = 1
		}
		if ease-0.15 < 1.3 {
			ease = 1.3
		} else {
			ease -= 0.15
		}
		streak = 0
	case "good", "easy":
		if interval <= 1 || state.ReviewCount == 0 {
			interval = 3
		} else {
			interval = interval * ease
		}
		if rating == "easy" {
			interval *= 1.3
			ease += 0.15
			if ease > 3 {
				ease = 3
			}
		}
		streak++
	default:
		return PracticeState{}, fmt.Errorf("unknown review rating")
	}
	interval = float64(int(interval*1000+0.5)) / 1000
	if interval > 365 {
		interval = 365
	}
	state.IntervalDays = interval
	state.Ease = ease
	state.Streak = streak
	state.Lapses = lapses
	state.ReviewCount++
	state.Due = now.Add(time.Duration(interval * float64(practiceDay)))
	return state, nil
}

// Card 是练习队列里的一道题。Answer 在提交前不返回。
type Card struct {
	ID          string   `json:"id"`
	Stem        string   `json:"stem"`
	Type        string   `json:"qtype"`
	Options     []string `json:"options,omitempty"`
	Explanation string   `json:"explanation,omitempty"`
	Source      string   `json:"source,omitempty"`
	DueAt       string   `json:"due_at,omitempty"`
	Streak      int      `json:"streak,omitempty"`
}

// Desk 统计到期题和本地日历日里已经评过的次数。now 同时用来算到期和当天起点。
func (s *Store) Desk(now time.Time) PracticeDesk {
	if s == nil {
		return PracticeDesk{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var d PracticeDesk
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UTC().Format(time.RFC3339Nano)
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM practice_state WHERE suspended = 0 AND due_at <= ?`, now.UTC().Format(time.RFC3339Nano)).Scan(&d.Due)
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM practice_events WHERE at >= ?`, start).Scan(&d.Reviewed)
	return d
}

// DueCards 返回今天到期的题，最多 20 道。未到期的不出现。
func (s *Store) DueCards(now time.Time) []Card {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT q.id, q.stem, q.qtype, q.options_json, q.source, s.due_at, s.streak
		FROM practice_state s JOIN questions q ON q.id = s.question_id
		WHERE s.suspended = 0 AND s.due_at <= ? ORDER BY s.due_at LIMIT ?`,
		now.UTC().Format(time.RFC3339Nano), practiceQueue)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Card
	for rows.Next() {
		var c Card
		var raw string
		if err := rows.Scan(&c.ID, &c.Stem, &c.Type, &raw, &c.Source, &c.DueAt, &c.Streak); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(raw), &c.Options)
		out = append(out, c)
	}
	return out
}

// ImportQuestion 写入一道练习题，并立刻排进队列。answer 是标准答案，界面在作答前不读取它。
// stem 不能空。qtype 空则是 short。
func (s *Store) ImportQuestion(stem, qtype, answer, explanation string, options []string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("learn store unavailable")
	}
	stem = strings.TrimSpace(stem)
	if stem == "" {
		return "", fmt.Errorf("stem required")
	}
	qtype = strings.ToLower(strings.TrimSpace(qtype))
	if qtype == "" {
		qtype = "short"
	}
	id := newKey("bank")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	raw, _ := json.Marshal(options)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`INSERT INTO questions (id, stem, qtype, options_json, answer, explanation, source, created_at) VALUES (?, ?, ?, ?, ?, ?, 'import', ?)`,
		id, stem, qtype, string(raw), strings.TrimSpace(answer), strings.TrimSpace(explanation), now)
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(`INSERT INTO practice_state (question_id, due_at, interval_days, ease) VALUES (?, ?, 1, 2.5)`, id, now)
	return id, err
}

// RateCard 按学习者的评分改日程。rating 传给 SchedulePractice。questionID 必须已有状态。
// now 是这次复习的时刻。
func (s *Store) RateCard(questionID, rating string, now time.Time) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rate(questionID, rating, now)
}

// GradeCard 批改客观题。qtype 是 fill 或 short 时返回 self rate，不改日程。
// questionID 必须存在。answer 和库里的标准答案比较。now 用来排下一次。
func (s *Store) GradeCard(questionID, answer string, now time.Time) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var qtype, expected string
	err := s.db.QueryRow(`SELECT qtype, answer FROM questions WHERE id = ?`, questionID).Scan(&qtype, &expected)
	if err != nil {
		return false, fmt.Errorf("question not found")
	}
	if qtype == "fill" || qtype == "short" {
		return false, fmt.Errorf("self rate")
	}
	ok := gradeAnswer(answer, expected, qtype)
	rating := "again"
	if ok {
		rating = "good"
	}
	return ok, s.rate(questionID, rating, now)
}

func (s *Store) rate(questionID, rating string, now time.Time) error {
	var interval, ease float64
	var lapses, reviews, streak int
	err := s.db.QueryRow(`SELECT interval_days, ease, lapses, review_count, streak FROM practice_state WHERE question_id = ?`, questionID).
		Scan(&interval, &ease, &lapses, &reviews, &streak)
	if err != nil {
		return fmt.Errorf("question not found")
	}
	next, err := SchedulePractice(PracticeState{
		IntervalDays: interval, Ease: ease, Streak: streak, Lapses: lapses, ReviewCount: reviews,
	}, rating, now)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE practice_state SET due_at = ?, interval_days = ?, ease = ?, lapses = ?, review_count = ?, streak = ? WHERE question_id = ?`,
		next.Due.UTC().Format(time.RFC3339Nano), next.IntervalDays, next.Ease, next.Lapses, next.ReviewCount, next.Streak, questionID)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO practice_events (id, question_id, rating, at) VALUES (?, ?, ?, ?)`,
		newKey("pev"), questionID, rating, now.UTC().Format(time.RFC3339Nano))
	return err
}

// NoteEmbed 记下一次嵌入请求，不写入对话用量。
func (s *Store) NoteEmbed() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.db.Exec(`INSERT INTO flags (k, v) VALUES ('embed_requests', '1')
		ON CONFLICT(k) DO UPDATE SET v = CAST(CAST(v AS INTEGER) + 1 AS TEXT)`)
}

// RevealCard 在作答之后返回标准答案和解析。questionID 必须存在。
func (s *Store) RevealCard(questionID string) map[string]string {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var answer, explanation string
	err := s.db.QueryRow(`SELECT answer, explanation FROM questions WHERE id = ?`, questionID).Scan(&answer, &explanation)
	if err != nil {
		return nil
	}
	return map[string]string{"answer": answer, "explanation": explanation}
}

// Questions 列出题库，不带标准答案。
func (s *Store) Questions() []Card {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT id, stem, qtype, options_json, source FROM questions ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Card
	for rows.Next() {
		var c Card
		var raw string
		if err := rows.Scan(&c.ID, &c.Stem, &c.Type, &raw, &c.Source); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(raw), &c.Options)
		out = append(out, c)
	}
	return out
}
