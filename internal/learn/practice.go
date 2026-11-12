// 题库和间隔复习的底座：题目的公开形状、练习设置、练习队列和顶部的汇总。
// 题目只经 bank.go 的 bankInsert 进库，这里不写题目。核对作答和评分在 practice_review.go，题库管理在 practice_bank.go，
// 导入在 practice_import.go，出题在 practice_generate.go，统计在 practice_insights.go。记忆模型在 fsrs.go。
// Question 里没有标准答案，也没有解析；它们只经 Reveal 在作答之后带出。

package learn

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// QuestionMulti 是多选题。标准答案是用逗号隔开的选项字母。
	QuestionMulti = "multi"
	// QuestionTrueFalse 是判断题。选项固定是 T 和 F，标准答案是其中一个。
	QuestionTrueFalse = "true_false"
	// QuestionFill 是填空题。标准答案里用竖线隔开的每一种写法都算对。
	QuestionFill = "fill"

	// CardNew 表示这道题还没评过分，也不是错题。新题受每天上限约束。
	CardNew = "new"
	// CardLearning 表示稳定度不到一天，或者是还没评过分的错题。
	CardLearning = "learning"
	// CardReview 表示这道题已经按间隔复习。
	CardReview = "review"

	flagPracticeRetention = "practice_retention"
	flagPracticeNewPerDay = "practice_new_per_day"

	defaultNewPerDay = 10
	maxNewPerDay     = 200
	defaultQueue     = 20
	maxQueue         = 100
	todayDueLimit    = 5

	// againDelay 是评为重来之后再次出现的间隔。
	againDelay = 10 * time.Minute
)

func init() {
	registerMigration(210, "practice_review_log", practiceReviewLog)
}

// practiceReviewLog 给复习记录补上三列：这次评分排出的间隔、评分之前这道题的阶段、评分之前已经评过几次。
// 旧记录按时间先后回填后两列，来自掌握路径的错题第一次复习算 learning。没有日程的旧题补一行日程，之后读题一律内连接。
func practiceReviewLog(tx *sql.Tx) error {
	if err := addColumns(tx, "practice_events",
		"interval_days REAL NOT NULL DEFAULT 0",
		"state_before TEXT NOT NULL DEFAULT ''",
		"reps_before INTEGER NOT NULL DEFAULT 0",
	); err != nil {
		return err
	}
	return execAll(tx,
		`CREATE INDEX IF NOT EXISTS practice_events_question ON practice_events(question_id, at)`,
		`UPDATE practice_events SET reps_before = (SELECT COUNT(1) FROM practice_events p
			WHERE p.question_id = practice_events.question_id AND p.at < practice_events.at)`,
		`UPDATE practice_events SET state_before = CASE
			WHEN reps_before > 0 THEN 'review'
			WHEN question_id IN (SELECT id FROM questions WHERE mistake = 1 AND source = 'mastery') THEN 'learning'
			ELSE 'new' END WHERE state_before = ''`,
		`INSERT INTO practice_state (question_id, due_at, interval_days, ease)
			SELECT id, created_at, 1, 2.5 FROM questions WHERE id NOT IN (SELECT question_id FROM practice_state)`,
	)
}

// Question 是题库里一道题的公开形状，带着它的日程，不带标准答案和解析。
// State 是 new、learning 或 review。Retrievability 是现在还记得的估计概率，没评过分的题是 0。
// Options 和 Tags 没有内容时是空数组。
type Question struct {
	ID             string   `json:"id"`
	Stem           string   `json:"stem"`
	Type           string   `json:"qtype"`
	Options        []Option `json:"options"`
	Difficulty     string   `json:"difficulty"`
	Tags           []string `json:"tags"`
	Source         string   `json:"source"`
	SourceID       string   `json:"source_id"`
	PathID         string   `json:"path_id"`
	PointID        string   `json:"point_id"`
	LibraryID      string   `json:"library_id"`
	Bookmarked     bool     `json:"bookmarked"`
	Mistake        bool     `json:"mistake"`
	Suspended      bool     `json:"suspended"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	State          string   `json:"state"`
	DueAt          string   `json:"due_at"`
	Reps           int      `json:"reps"`
	Lapses         int      `json:"lapses"`
	Stability      float64  `json:"stability"`
	Retrievability float64  `json:"retrievability"`
	LastReviewAt   string   `json:"last_review_at"`
}

// Reveal 是作答之后才给出的标准答案和解析。
type Reveal struct {
	Answer      string `json:"answer"`
	Explanation string `json:"explanation"`
}

// PracticeSettings 是练习的两项设置。Retention 是目标记住率，NewPerDay 是每天最多开始几道新题。
type PracticeSettings struct {
	Retention float64 `json:"retention"`
	NewPerDay int     `json:"new_per_day"`
}

// PracticeSummary 是练习一栏顶部的数字。Due 是已经到期的非新题，New 是今天在上限之内还能开始的新题。
// ReviewedToday 是本地今天零点以来评分的次数。NextDueAt 是还没到期的题里最早的到期时刻，没有时为空串。
// StreakDays 是连续有复习的本地天数，最后一天必须是今天或昨天。暂停的题不计入 Due、New 和 NextDueAt。
type PracticeSummary struct {
	Due           int     `json:"due"`
	New           int     `json:"new"`
	ReviewedToday int     `json:"reviewed_today"`
	Total         int     `json:"total"`
	Mistakes      int     `json:"mistakes"`
	NextDueAt     string  `json:"next_due_at"`
	Retention     float64 `json:"retention"`
	NewPerDay     int     `json:"new_per_day"`
	StreakDays    int     `json:"streak_days"`
}

// card 是库里的一道题连同日程和标准答案。只在这个包里流转，出包之前取它的 Question。
type card struct {
	Question
	Answer       string
	Explanation  string
	LastAnswer   string
	FirstWrongAt string
	Mem          Memory
	Due          time.Time
	Last         time.Time
	Streak       int
}

// reveal 返回这道题的标准答案和解析。
func (c card) reveal() Reveal {
	return Reveal{Answer: c.Answer, Explanation: c.Explanation}
}

const cardCols = `q.id, q.stem, q.qtype, q.options_json, q.answer, q.explanation, q.source, q.source_id, q.difficulty, q.tags_json,
	q.path_id, q.point_id, q.library_id, q.bookmarked, q.mistake, q.first_wrong_at, q.last_answer, q.created_at, q.updated_at,
	st.due_at, st.lapses, st.review_count, st.streak, st.suspended, st.stability, st.difficulty, st.last_review_at`

const cardFrom = ` FROM questions q JOIN practice_state st ON st.question_id = q.id `

// notNewSQL 挑出不算新题的题：评过分的，或者是错题。
const notNewSQL = `NOT (st.review_count = 0 AND q.mistake = 0)`

// isNewSQL 挑出新题：没评过分，也不是错题。
const isNewSQL = `(st.review_count = 0 AND q.mistake = 0)`

// cardState 按已评次数、是否错题和稳定度给出阶段。reps 是已评次数，mistake 是错题标记，stability 是稳定度的天数。
func cardState(reps int, mistake bool, stability float64) string {
	if reps <= 0 {
		if mistake {
			return CardLearning
		}
		return CardNew
	}
	if stability < 1 {
		return CardLearning
	}
	return CardReview
}

// scanCard 从一行读出一道题。列顺序和 cardCols 相同。now 用来估计现在还记得多少。
func scanCard(row rower, now time.Time) (card, error) {
	var c card
	var options, tags, due, last string
	var bookmarked, mistake, suspended int
	err := row.Scan(&c.ID, &c.Stem, &c.Type, &options, &c.Answer, &c.Explanation, &c.Source, &c.SourceID, &c.Difficulty, &tags,
		&c.PathID, &c.PointID, &c.LibraryID, &bookmarked, &mistake, &c.FirstWrongAt, &c.LastAnswer, &c.CreatedAt, &c.UpdatedAt,
		&due, &c.Lapses, &c.Reps, &c.Streak, &suspended, &c.Mem.Stability, &c.Mem.Difficulty, &last)
	if err != nil {
		return card{}, err
	}
	c.Type = strings.ToLower(strings.TrimSpace(c.Type))
	c.Options = decodeOptions(options)
	if c.Options == nil {
		c.Options = []Option{}
	}
	_ = json.Unmarshal([]byte(tags), &c.Tags)
	if c.Tags == nil {
		c.Tags = []string{}
	}
	if c.Source == "" {
		c.Source = "manual"
	}
	if c.UpdatedAt == "" {
		c.UpdatedAt = c.CreatedAt
	}
	c.Bookmarked, c.Mistake, c.Suspended = bookmarked != 0, mistake != 0, suspended != 0
	c.Due, c.Last = parseTime(due), parseTime(last)
	c.DueAt, c.LastReviewAt = due, last
	c.Stability = c.Mem.Stability
	c.State = cardState(c.Reps, c.Mistake, c.Mem.Stability)
	if c.Reps > 0 && c.Mem.Seen() && !c.Last.IsZero() {
		c.Retrievability = Retrievability(now.Sub(c.Last).Hours()/24, c.Mem.Stability)
	}
	return c, nil
}

// loadCard 用 q 读出一道题。id 不存在时返回 question not found。now 交给 scanCard。
func loadCard(q querier, id string, now time.Time) (card, error) {
	c, err := scanCard(q.QueryRow(`SELECT `+cardCols+cardFrom+`WHERE q.id = ?`, id), now)
	if err != nil {
		return card{}, fmt.Errorf("question not found")
	}
	return c, nil
}

// listCards 执行一条以 cardCols 开头的查询并读完结果。query 是完整的语句，args 是它的参数，now 交给 scanCard。
func listCards(q querier, now time.Time, query string, args ...any) ([]card, error) {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []card
	for rows.Next() {
		c, err := scanCard(rows, now)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// publicCards 取出每道题的公开形状。cards 为空时返回空数组，不返回 nil。
func publicCards(cards []card) []Question {
	out := make([]Question, 0, len(cards))
	for _, c := range cards {
		out = append(out, c.Question)
	}
	return out
}

// cardFilter 是队列和题库列表共用的筛选条件。零值不筛。Bookmarked 和 Suspended 为 nil 时不看那一项。
type cardFilter struct {
	Query        string
	Tag          string
	Source       string
	MistakesOnly bool
	Bookmarked   *bool
	Suspended    *bool
}

// where 把筛选条件拼成 AND 连接的 SQL 片段和参数。没有条件时片段是 1 = 1。
// Query 按空白拆成最多 6 个词，每个词都要出现在题干、选项或标签里。
func (f cardFilter) where() (string, []any) {
	clauses := []string{"1 = 1"}
	var args []any
	for i, term := range strings.Fields(f.Query) {
		if i >= 6 {
			break
		}
		like := "%" + likeEscape(term) + "%"
		clauses = append(clauses, `(q.stem LIKE ? ESCAPE '\' OR q.options_json LIKE ? ESCAPE '\' OR q.tags_json LIKE ? ESCAPE '\')`)
		args = append(args, like, like, like)
	}
	if tag := strings.TrimSpace(f.Tag); tag != "" {
		// 标签存成 JSON 数组。带引号的整个标签出现在里面才算，这样不会匹配到别的标签的一部分。
		quoted, _ := json.Marshal(tag)
		clauses = append(clauses, `q.tags_json LIKE ? ESCAPE '\'`)
		args = append(args, "%"+likeEscape(string(quoted))+"%")
	}
	if source := strings.ToLower(strings.TrimSpace(f.Source)); source != "" {
		// 旧库里来源为空的题读出来算手工录入，筛选时也要对上。
		clauses = append(clauses, `(q.source = ? OR (? = 'manual' AND q.source = ''))`)
		args = append(args, source, source)
	}
	if f.MistakesOnly {
		clauses = append(clauses, `q.mistake = 1`)
	}
	if f.Bookmarked != nil {
		clauses = append(clauses, `q.bookmarked = ?`)
		args = append(args, boolInt(*f.Bookmarked))
	}
	if f.Suspended != nil {
		clauses = append(clauses, `st.suspended = ?`)
		args = append(args, boolInt(*f.Suspended))
	}
	return strings.Join(clauses, " AND "), args
}

// likeEscape 给 s 里的反斜杠、百分号和下划线加上反斜杠，好放进带 ESCAPE '\' 的 LIKE 模式里按字面匹配。
func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// localDay 返回 t 所在的本地日历日的零点。t 是零值时返回零值。
func localDay(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	l := t.In(time.Local)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.Local)
}

// practiceSettings 用 q 从 flags 读出练习设置。没写过或写坏了的项用默认值，读出的值夹进合法范围。
func practiceSettings(q querier) PracticeSettings {
	out := PracticeSettings{Retention: DefaultRetention, NewPerDay: defaultNewPerDay}
	var raw string
	if err := q.QueryRow(`SELECT v FROM flags WHERE k = ?`, flagPracticeRetention).Scan(&raw); err == nil {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			out.Retention = ClampRetention(v)
		}
	}
	if err := q.QueryRow(`SELECT v FROM flags WHERE k = ?`, flagPracticeNewPerDay).Scan(&raw); err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
			out.NewPerDay = clampNewPerDay(v)
		}
	}
	return out
}

// clampNewPerDay 把每天的新题上限夹进 0 到 maxNewPerDay 之间。
func clampNewPerDay(n int) int {
	if n < 0 {
		return 0
	}
	if n > maxNewPerDay {
		return maxNewPerDay
	}
	return n
}

// PracticeSettings 读出目标记住率和每天的新题上限。没写过时是 DefaultRetention 和 10。
func (s *Store) PracticeSettings() (PracticeSettings, error) {
	if s == nil {
		return PracticeSettings{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return practiceSettings(s.db), nil
}

// SetPracticeSettings 改练习设置并返回改完的值。retention 和 newPerDay 为 nil 的那一项不动。
// retention 先交给 ClampRetention，newPerDay 夹进 0 到 200。目标记住率变了时，上一次评分不是重来的题按新的目标重排到期时间，
// 到期时间只由上次复习的时刻和稳定度算出，所以改回去日程也回去。这不算一次复习。
func (s *Store) SetPracticeSettings(retention *float64, newPerDay *int) (PracticeSettings, error) {
	if s == nil {
		return PracticeSettings{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	before := practiceSettings(s.db)
	after := before
	if retention != nil {
		after.Retention = ClampRetention(*retention)
	}
	if newPerDay != nil {
		after.NewPerDay = clampNewPerDay(*newPerDay)
	}
	changed := after != before
	err := s.inTx(func(tx *sql.Tx) error {
		if err := setFlag(tx, flagPracticeRetention, strconv.FormatFloat(after.Retention, 'f', -1, 64)); err != nil {
			return err
		}
		if err := setFlag(tx, flagPracticeNewPerDay, strconv.Itoa(after.NewPerDay)); err != nil {
			return err
		}
		if after.Retention == before.Retention {
			return nil
		}
		return rescheduleCards(tx, after.Retention)
	})
	s.mu.Unlock()
	if err != nil {
		return PracticeSettings{}, err
	}
	if changed {
		s.notify("practice", "", "settings")
	}
	return after, nil
}

// setFlag 用 q 写入或改写 flags 里的一项。k 是键，v 是值。
func setFlag(q querier, k, v string) error {
	_, err := q.Exec(`INSERT INTO flags (k, v) VALUES (?, ?) ON CONFLICT(k) DO UPDATE SET v = excluded.v`, k, v)
	return err
}

// rescheduleCards 按新的目标记住率 retention 重排到期时间。只动评过分、上一次不是重来的题：streak 大于 0 正是这个意思。
// 评为重来的题留在十分钟的间隔上。
func rescheduleCards(q querier, retention float64) error {
	type row struct {
		id        string
		stability float64
		last      time.Time
	}
	rows, err := q.Query(`SELECT question_id, stability, last_review_at FROM practice_state
		WHERE review_count > 0 AND streak > 0 AND stability > 0 AND last_review_at != ''`)
	if err != nil {
		return err
	}
	var list []row
	for rows.Next() {
		var r row
		var last string
		if err := rows.Scan(&r.id, &r.stability, &last); err != nil {
			rows.Close()
			return err
		}
		if r.last = parseTime(last); !r.last.IsZero() {
			list = append(list, r)
		}
	}
	rows.Close()
	for _, r := range list {
		due := dueAfter(r.last, Memory{Stability: r.stability}, retention, 1)
		if _, err := q.Exec(`UPDATE practice_state SET due_at = ?, interval_days = ? WHERE question_id = ?`,
			formatTime(due), due.Sub(r.last).Hours()/24, r.id); err != nil {
			return err
		}
	}
	return nil
}

// QueueRequest 是取练习队列的入参。Limit 小于等于 0 时取 20，最多 100。Tag、Source 和 MistakesOnly 是筛选条件。
// IDs 非空时只练点名的这些题，按给出的顺序，不看到期时间和新题上限；这时 Limit 小于等于 0 表示全部，仍然最多 100。
type QueueRequest struct {
	Limit        int      `json:"limit"`
	Tag          string   `json:"tag"`
	Source       string   `json:"source"`
	MistakesOnly bool     `json:"mistakes_only"`
	IDs          []string `json:"ids"`
}

// PracticeQueue 返回现在该练的题，不带标准答案。先是已经到期的非新题，最早到期的在前；再是新题，最早录入的在前，
// 今天已经开始的新题加上这次给出的新题不超过每天的上限。暂停的题不出现。req 的用法见 QueueRequest。
func (s *Store) PracticeQueue(req QueueRequest) ([]Question, error) {
	if s == nil {
		return nil, fmt.Errorf("learn store unavailable")
	}
	now := s.clock()
	s.mu.Lock()
	defer s.mu.Unlock()
	cards, err := practiceQueue(s.db, req, now)
	if err != nil {
		return nil, err
	}
	return publicCards(cards), nil
}

// practiceQueue 用 q 排出练习队列。req 的用法见 QueueRequest。now 用来判断到期和今天的起点。
func practiceQueue(q querier, req QueueRequest, now time.Time) ([]card, error) {
	filter := cardFilter{Tag: req.Tag, Source: req.Source, MistakesOnly: req.MistakesOnly}
	where, args := filter.where()
	limit := req.Limit
	if ids := cleanIDs(req.IDs, maxQueue); len(ids) > 0 {
		if limit <= 0 || limit > maxQueue {
			limit = maxQueue
		}
		marks := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
		idArgs := make([]any, 0, len(ids)+len(args))
		for _, id := range ids {
			idArgs = append(idArgs, id)
		}
		found, err := listCards(q, now, `SELECT `+cardCols+cardFrom+`WHERE st.suspended = 0 AND q.id IN (`+marks+`) AND `+where,
			append(idArgs, args...)...)
		if err != nil {
			return nil, err
		}
		byID := map[string]card{}
		for _, c := range found {
			byID[c.ID] = c
		}
		out := make([]card, 0, len(found))
		for _, id := range ids {
			if c, ok := byID[id]; ok && len(out) < limit {
				out = append(out, c)
			}
		}
		return out, nil
	}
	if limit <= 0 {
		limit = defaultQueue
	}
	if limit > maxQueue {
		limit = maxQueue
	}
	out, err := dueCards(q, filter, now, limit)
	if err != nil {
		return nil, err
	}
	room := newRoomToday(q, now)
	if rest := limit - len(out); rest < room {
		room = rest
	}
	if room <= 0 {
		return out, nil
	}
	fresh, err := listCards(q, now, `SELECT `+cardCols+cardFrom+`WHERE st.suspended = 0 AND `+isNewSQL+` AND `+where+`
		ORDER BY q.created_at ASC, q.rowid ASC LIMIT ?`, append(args, room)...)
	if err != nil {
		return nil, err
	}
	return append(out, fresh...), nil
}

// dueCards 用 q 读出已经到期的非新题，最早到期的在前，最多 limit 道。暂停的题不出现。f 是附加的筛选条件，now 是判断到期的时刻。
func dueCards(q querier, f cardFilter, now time.Time, limit int) ([]card, error) {
	where, args := f.where()
	args = append([]any{formatTime(now)}, args...)
	return listCards(q, now, `SELECT `+cardCols+cardFrom+`WHERE st.suspended = 0 AND st.due_at <= ? AND `+notNewSQL+` AND `+where+`
		ORDER BY st.due_at ASC, q.rowid ASC LIMIT ?`, append(args, limit)...)
}

// newRoomToday 用 q 算出今天在上限之内还能开始几道新题。now 所在的本地日里已经开始的新题从上限里扣掉，结果不小于 0。
func newRoomToday(q querier, now time.Time) int {
	var started int
	_ = q.QueryRow(`SELECT COUNT(1) FROM practice_events WHERE state_before = ? AND at >= ?`, CardNew, formatTime(localDay(now))).Scan(&started)
	room := practiceSettings(q).NewPerDay - started
	if room < 0 {
		return 0
	}
	return room
}

// cleanIDs 去掉 ids 里的空白项和重复项，保持顺序，最多留 max 个。
func cleanIDs(ids []string, max int) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] || len(out) >= max {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// PracticeSummary 返回练习一栏顶部的数字。各项的含义见 PracticeSummary 类型。
func (s *Store) PracticeSummary() (PracticeSummary, error) {
	if s == nil {
		return PracticeSummary{}, fmt.Errorf("learn store unavailable")
	}
	now := s.clock()
	s.mu.Lock()
	defer s.mu.Unlock()
	return practiceSummary(s.db, now)
}

// practiceSummary 用 q 算出汇总。now 用来判断到期、今天的起点和连续天数。
func practiceSummary(q querier, now time.Time) (PracticeSummary, error) {
	settings := practiceSettings(q)
	out := PracticeSummary{Retention: settings.Retention, NewPerDay: settings.NewPerDay}
	at := formatTime(now)
	var fresh int
	var next sql.NullString
	for _, step := range []struct {
		dst   any
		query string
		args  []any
	}{
		{&out.Due, `SELECT COUNT(1)` + cardFrom + `WHERE st.suspended = 0 AND st.due_at <= ? AND ` + notNewSQL, []any{at}},
		{&fresh, `SELECT COUNT(1)` + cardFrom + `WHERE st.suspended = 0 AND ` + isNewSQL, nil},
		{&out.ReviewedToday, `SELECT COUNT(1) FROM practice_events WHERE at >= ?`, []any{formatTime(localDay(now))}},
		{&out.Total, `SELECT COUNT(1) FROM questions`, nil},
		{&out.Mistakes, `SELECT COUNT(1) FROM questions WHERE mistake = 1`, nil},
		{&next, `SELECT MIN(st.due_at)` + cardFrom + `WHERE st.suspended = 0 AND st.due_at > ? AND ` + notNewSQL, []any{at}},
	} {
		if err := q.QueryRow(step.query, step.args...).Scan(step.dst); err != nil {
			return PracticeSummary{}, err
		}
	}
	out.NextDueAt = next.String
	out.New = newRoomToday(q, now)
	if fresh < out.New {
		out.New = fresh
	}
	streak, err := reviewStreak(q, now)
	if err != nil {
		return PracticeSummary{}, err
	}
	out.StreakDays = streak
	return out, nil
}

// reviewStreak 用 q 数出连续有复习的本地天数。最近一次复习早于 now 的前一天时是 0。
func reviewStreak(q querier, now time.Time) (int, error) {
	rows, err := q.Query(`SELECT at FROM practice_events ORDER BY at DESC`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	today := localDay(now)
	streak := 0
	var want time.Time
	for rows.Next() {
		var at string
		if err := rows.Scan(&at); err != nil {
			return 0, err
		}
		day := localDay(parseTime(at))
		if day.IsZero() {
			continue
		}
		// 钟被往回拨过时会有落在将来的记录，算作今天。
		if day.After(today) {
			day = today
		}
		switch {
		case streak == 0:
			if day.Before(today.AddDate(0, 0, -1)) {
				return 0, nil
			}
			streak, want = 1, day.AddDate(0, 0, -1)
		case day.After(want):
			// 和上一条记录是同一天。
		case day.Equal(want):
			streak++
			want = day.AddDate(0, 0, -1)
		default:
			return streak, nil
		}
	}
	return streak, rows.Err()
}

// practiceToday 返回学习计划里练习那一块：汇总，和最多 limit 道已经到期的题，不带标准答案。now 是判断到期的时刻。
func (s *Store) practiceToday(now time.Time, limit int) (PracticeSummary, []Question, error) {
	now = now.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	summary, err := practiceSummary(s.db, now)
	if err != nil {
		return PracticeSummary{}, nil, err
	}
	due, err := dueCards(s.db, cardFilter{}, now, limit)
	if err != nil {
		return PracticeSummary{}, nil, err
	}
	return summary, publicCards(due), nil
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
