// 把旧库里记在知识点行上的作答和复习日程抄成证据，再回放一遍。只跑一次。
// 旧的 repetitions 表留着不删，之后不再读写。

package learn

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func init() {
	registerMigration(101, "mastery_v2_evidence", masteryBackfillEvidence)
}

// legacyIntervals 是旧版按类型写死的复习间隔，单位是天。迁移时用它估计旧日程对应的稳定度。
var legacyIntervals = map[string][]int{
	KindMemory:    {0, 1, 3, 7, 14, 30, 60},
	KindConcept:   {3, 7, 14, 30},
	KindProcedure: {3, 7, 14},
	KindDesign:    {14, 28},
}

// masteryBackfillEvidence 把每个旧知识点的作答、讲解通过、自我声明和复习日程写成证据，然后回放。
// 旧作答没有时间，按顺序排在路径更新时间之前，每条隔一分钟。旧日程抄成记忆种子，到期时间不变。
func masteryBackfillEvidence(tx *sql.Tx) error {
	type legacy struct {
		id, pathID, kind, attempts, source, updated string
		qualitative                                 int
	}
	rows, err := tx.Query(`SELECT p.id, p.path_id, p.kind, p.attempts_json, p.mastery_source, p.qualitative, paths.updated_at
		FROM points p JOIN paths ON paths.id = p.path_id`)
	if err != nil {
		return err
	}
	var list []legacy
	for rows.Next() {
		var l legacy
		if err := rows.Scan(&l.id, &l.pathID, &l.kind, &l.attempts, &l.source, &l.qualitative, &l.updated); err != nil {
			rows.Close()
			return err
		}
		list = append(list, l)
	}
	rows.Close()
	type schedule struct {
		index int
		next  string
	}
	schedules := map[string]schedule{}
	rows, err = tx.Query(`SELECT point_id, interval_index, next_review_at FROM repetitions`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		var sc schedule
		if err := rows.Scan(&id, &sc.index, &sc.next); err != nil {
			rows.Close()
			return err
		}
		schedules[id] = sc
	}
	rows.Close()

	paths := map[string]bool{}
	for _, l := range list {
		var done int
		_ = tx.QueryRow(`SELECT COUNT(1) FROM evidence WHERE point_id = ?`, l.id).Scan(&done)
		if done > 0 {
			continue
		}
		base := parseTime(l.updated)
		if base.IsZero() {
			base = time.Now().UTC()
		}
		var scores []float64
		_ = json.Unmarshal([]byte(l.attempts), &scores)
		at := base.Add(-time.Duration(len(scores)+3) * time.Minute)
		add := func(kind, result, feedback string) error {
			at = at.Add(time.Minute)
			return appendEvidence(tx, evidenceRow{
				ID: fmt.Sprintf("ev-mig-%s-%d", l.id, at.UnixNano()), PathID: l.pathID, PointID: l.id,
				Kind: kind, Result: result, Independent: true, Feedback: feedback, At: at,
			})
		}
		for _, score := range scores {
			result := ResultIncorrect
			switch {
			case score >= 1:
				result = ResultCorrect
			case score > 0:
				result = ResultPartial
			}
			if err := add(evidenceQuiz, result, ""); err != nil {
				return err
			}
		}
		if l.qualitative != 0 {
			if err := add(evidenceQualitative, ResultCorrect, ""); err != nil {
				return err
			}
		}
		if l.source == "learner" {
			if err := add(evidenceClaim, "on", ""); err != nil {
				return err
			}
		}
		if sc, ok := schedules[l.id]; ok {
			seq := legacyIntervals[normalizeKind(l.kind)]
			if sc.index < 0 {
				sc.index = 0
			}
			if sc.index >= len(seq) {
				sc.index = len(seq) - 1
			}
			days := float64(seq[sc.index])
			if days < 1 {
				days = 1
			}
			due := parseTime(sc.next)
			seed := seedState{
				Stability: days, Difficulty: initialMemory(l.kind).Difficulty, Reps: sc.index + 1,
				Due: formatTime(due), Last: formatTime(due.Add(-time.Duration(days * 24 * float64(time.Hour)))),
			}
			raw, _ := json.Marshal(seed)
			if err := add(evidenceSeed, "", string(raw)); err != nil {
				return err
			}
		}
		paths[l.pathID] = true
	}
	// 没有任何证据的路径也回放一遍，把 status 列写成和旧分数一致的值。
	for _, l := range list {
		paths[l.pathID] = true
	}
	for id := range paths {
		if err := rebuildPath(tx, id); err != nil {
			return err
		}
	}
	return nil
}
