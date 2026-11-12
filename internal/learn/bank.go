// 往题库里放一道题的唯一入口。掌握路径的错题、课本的随堂题、导入和手工录入都走这里，题目的形状和去重规则只有一份。
// 这里不排复习，只写题目和它的初始日程。

package learn

import (
	"encoding/json"
	"strings"
	"time"
)

// bankEntry 是要放进题库的一道题。OriginKey 非空时用来去重，同一个来源的同一道题只进一次。
// Mistake 为真表示这道题是答错才进来的。选择题的 Answer 是选项字母，多选用逗号隔开。
type bankEntry struct {
	Stem        string
	Type        string
	Options     []Option
	Answer      string
	Explanation string
	Difficulty  string
	Tags        []string
	Source      string
	SourceID    string
	OriginKey   string
	PathID      string
	PointID     string
	LibraryID   string
	DocumentID  string
	Locator     int
	Mistake     bool
	LastAnswer  string
}

// bankInsert 写入一道题和它的初始日程。OriginKey 已经在库里时不写，added 为假，id 是原来那道题。
// 错题一天后到期。其余的题立刻可练，在第一次评分之前算新题。now 是写入时刻。
func bankInsert(q querier, e bankEntry, now time.Time) (id string, added bool, err error) {
	if key := strings.TrimSpace(e.OriginKey); key != "" {
		var existing string
		if scanErr := q.QueryRow(`SELECT id FROM questions WHERE origin_key = ?`, key).Scan(&existing); scanErr == nil {
			return existing, false, nil
		}
	}
	options := e.Options
	if options == nil {
		options = []Option{}
	}
	tags := e.Tags
	if tags == nil {
		tags = []string{}
	}
	optionsJSON, _ := json.Marshal(options)
	tagsJSON, _ := json.Marshal(tags)
	source := strings.TrimSpace(e.Source)
	if source == "" {
		source = "manual"
	}
	qtype := strings.ToLower(strings.TrimSpace(e.Type))
	if qtype == "" {
		qtype = QuestionShort
	}
	id = newKey("bank")
	at := formatTime(now)
	firstWrong := ""
	due := now
	lapses := 0
	if e.Mistake {
		firstWrong = at
		due = now.Add(24 * time.Hour)
		lapses = 1
	}
	_, err = q.Exec(`INSERT INTO questions
		(id, stem, qtype, options_json, answer, explanation, source, source_id, origin_key, difficulty, tags_json,
		 path_id, point_id, library_id, document_id, locator, mistake, first_wrong_at, last_answer, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, clipRunes(strings.TrimSpace(e.Stem), 8000), qtype, string(optionsJSON), clipRunes(strings.TrimSpace(e.Answer), 4000),
		clipRunes(strings.TrimSpace(e.Explanation), 8000), source, e.SourceID, strings.TrimSpace(e.OriginKey), e.Difficulty, string(tagsJSON),
		e.PathID, e.PointID, e.LibraryID, e.DocumentID, e.Locator, boolInt(e.Mistake), firstWrong, clipRunes(e.LastAnswer, 4000), at, at)
	if err != nil {
		return "", false, err
	}
	_, err = q.Exec(`INSERT INTO practice_state (question_id, due_at, interval_days, ease, lapses, review_count) VALUES (?, ?, 1, 2.5, ?, 0)`,
		id, formatTime(due), lapses)
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

// bankRemoveOrigin 把某个来源键对应的题连同日程和复习记录一起删掉。没有这道题时什么都不做。
func bankRemoveOrigin(q querier, originKey string) error {
	return execArgs(q, originKey,
		`DELETE FROM practice_events WHERE question_id IN (SELECT id FROM questions WHERE origin_key = ?)`,
		`DELETE FROM practice_state WHERE question_id IN (SELECT id FROM questions WHERE origin_key = ?)`,
		`DELETE FROM questions WHERE origin_key = ?`,
	)
}
