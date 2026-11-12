// 题库的管理：列表、单题详情、录入和修改、删除、标记、标签统计，以及给导师用的检索。
// 新题只经 bank.go 的 bankInsert 写入，校验在 practice_draft.go。这里不排复习：改题干或答案不动日程。
// 列表里的题没有标准答案；详情是学习者自己翻开一道题看，带答案。

package learn

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	defaultBankPage = 50
	maxBankPage     = 200
	maxBankEvents   = 100
	maxTutorHits    = 20
)

// BankQuery 是题库列表的筛选、排序和分页。Query 是要出现在题干、选项或标签里的词。
// Bookmarked 和 Suspended 为 nil 时不筛那一项。Sort 是 recent（最新录入在前，默认）、oldest 或 due（最早到期在前，暂停的排最后）。
// Limit 小于等于 0 时取 50，最多 200。Offset 小于 0 时按 0 算。
type BankQuery struct {
	Query        string `json:"query"`
	Tag          string `json:"tag"`
	Source       string `json:"source"`
	MistakesOnly bool   `json:"mistakes_only"`
	Bookmarked   *bool  `json:"bookmarked"`
	Suspended    *bool  `json:"suspended"`
	Sort         string `json:"sort"`
	Limit        int    `json:"limit"`
	Offset       int    `json:"offset"`
}

// QuestionPage 是题库列表的一页。Total 是符合筛选条件的总数，不受分页影响。
type QuestionPage struct {
	Items []Question `json:"items"`
	Total int        `json:"total"`
}

// ListQuestions 按 query 列出题库里的题，不带标准答案。query 的用法见 BankQuery，Sort 不认得时返回错误。
func (s *Store) ListQuestions(query BankQuery) (QuestionPage, error) {
	if s == nil {
		return QuestionPage{}, fmt.Errorf("learn store unavailable")
	}
	order := `q.created_at DESC, q.rowid DESC`
	switch strings.ToLower(strings.TrimSpace(query.Sort)) {
	case "", "recent":
	case "oldest":
		order = `q.created_at ASC, q.rowid ASC`
	case "due":
		order = `st.suspended ASC, st.due_at ASC, q.rowid ASC`
	default:
		return QuestionPage{}, fmt.Errorf("sort must be recent, oldest, or due")
	}
	limit, offset := query.Limit, query.Offset
	if limit <= 0 {
		limit = defaultBankPage
	}
	if limit > maxBankPage {
		limit = maxBankPage
	}
	if offset < 0 {
		offset = 0
	}
	filter := cardFilter{
		Query: query.Query, Tag: query.Tag, Source: query.Source, MistakesOnly: query.MistakesOnly,
		Bookmarked: query.Bookmarked, Suspended: query.Suspended,
	}
	where, args := filter.where()
	now := s.clock()
	s.mu.Lock()
	defer s.mu.Unlock()
	page := QuestionPage{}
	if err := s.db.QueryRow(`SELECT COUNT(1)`+cardFrom+`WHERE `+where, args...).Scan(&page.Total); err != nil {
		return QuestionPage{}, err
	}
	cards, err := listCards(s.db, now, `SELECT `+cardCols+cardFrom+`WHERE `+where+` ORDER BY `+order+` LIMIT ? OFFSET ?`,
		append(append([]any(nil), args...), limit, offset)...)
	if err != nil {
		return QuestionPage{}, err
	}
	page.Items = publicCards(cards)
	return page, nil
}

// QuestionEvent 是一条复习记录。Correct 为 nil 表示那一次没有对错的结论。
type QuestionEvent struct {
	At      string `json:"at"`
	Rating  string `json:"rating"`
	Correct *bool  `json:"correct"`
	Answer  string `json:"answer"`
}

// QuestionDetail 是一道题的详情：题目、标准答案和解析，以及复习记录，新的在前。
type QuestionDetail struct {
	Question Question        `json:"question"`
	Reveal   Reveal          `json:"reveal"`
	Events   []QuestionEvent `json:"events"`
}

// QuestionDetail 读出一道题的详情，带标准答案。复习记录最多带最近 100 条。id 不存在时返回 question not found。
func (s *Store) QuestionDetail(id string) (QuestionDetail, error) {
	if s == nil {
		return QuestionDetail{}, fmt.Errorf("learn store unavailable")
	}
	now := s.clock()
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := loadCard(s.db, id, now)
	if err != nil {
		return QuestionDetail{}, err
	}
	out := QuestionDetail{Question: c.Question, Reveal: c.reveal(), Events: []QuestionEvent{}}
	rows, err := s.db.Query(`SELECT at, rating, correct, answer FROM practice_events WHERE question_id = ? ORDER BY at DESC, rowid DESC LIMIT ?`,
		id, maxBankEvents)
	if err != nil {
		return QuestionDetail{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var e QuestionEvent
		var correct int
		if err := rows.Scan(&e.At, &e.Rating, &correct, &e.Answer); err != nil {
			return QuestionDetail{}, err
		}
		if correct >= 0 {
			v := correct != 0
			e.Correct = &v
		}
		out.Events = append(out.Events, e)
	}
	return out, rows.Err()
}

// QuestionInput 是录入或修改一道题的入参。ID 为空是新建，来源记成 manual。
// 修改时指针为 nil 的项保持原样；新建时 nil 当作空。
type QuestionInput struct {
	ID          string
	Stem        *string
	Type        *string
	Options     *[]Option
	Answer      *string
	Explanation *string
	Difficulty  *string
	Tags        *[]string
}

// SaveQuestion 录入一道新题，或修改一道已有的题，返回保存之后的题目。内容按 normalizeDraft 校验。
// 修改不动日程和复习记录。只改题干、解析、难度或标签时不重新校验选项和答案，旧库里形状不全的题也能改这几项。
// in 的用法见 QuestionInput。
func (s *Store) SaveQuestion(in QuestionInput) (Question, error) {
	if s == nil {
		return Question{}, fmt.Errorf("learn store unavailable")
	}
	now := s.clock()
	s.mu.Lock()
	id, kind, err := s.saveQuestion(in, now)
	var c card
	if err == nil {
		c, err = loadCard(s.db, id, now)
	}
	s.mu.Unlock()
	if err != nil {
		return Question{}, err
	}
	s.notify("practice", id, kind)
	return c.Question, nil
}

// saveQuestion 在持有 s.mu 时写入或修改一道题。in 的用法见 QuestionInput，now 是写入时刻。kind 是 created 或 updated。
func (s *Store) saveQuestion(in QuestionInput, now time.Time) (id, kind string, err error) {
	id = strings.TrimSpace(in.ID)
	var draft DraftQuestion
	if id != "" {
		c, err := loadCard(s.db, id, now)
		if err != nil {
			return "", "", err
		}
		draft = DraftQuestion{Stem: c.Stem, Type: c.Type, Options: c.Options, Answer: c.Answer, Explanation: c.Explanation, Difficulty: c.Difficulty, Tags: c.Tags}
	}
	if in.Stem != nil {
		draft.Stem = *in.Stem
	}
	if in.Type != nil {
		draft.Type = *in.Type
	}
	if in.Options != nil {
		draft.Options = *in.Options
	}
	if in.Answer != nil {
		draft.Answer = *in.Answer
	}
	if in.Explanation != nil {
		draft.Explanation = *in.Explanation
	}
	if in.Difficulty != nil {
		draft.Difficulty = *in.Difficulty
	}
	if in.Tags != nil {
		draft.Tags = *in.Tags
	}
	at := formatTime(now)
	if id != "" && in.Type == nil && in.Options == nil && in.Answer == nil {
		stem := clipRunes(strings.TrimSpace(draft.Stem), 8000)
		if stem == "" {
			return "", "", fmt.Errorf("the question text is empty")
		}
		tags, _ := json.Marshal(cleanTags(draft.Tags))
		_, err := s.db.Exec(`UPDATE questions SET stem = ?, explanation = ?, difficulty = ?, tags_json = ?, updated_at = ? WHERE id = ?`,
			stem, clipRunes(strings.TrimSpace(draft.Explanation), 8000), normalizeDifficulty(draft.Difficulty), string(tags), at, id)
		return id, "updated", err
	}
	norm, err := normalizeDraft(draft)
	if err != nil {
		return "", "", err
	}
	if id == "" {
		err = s.inTx(func(tx *sql.Tx) error {
			var insertErr error
			id, _, insertErr = bankInsert(tx, bankEntryOf(norm, "manual", ""), now)
			return insertErr
		})
		return id, "created", err
	}
	options, _ := json.Marshal(norm.Options)
	tags, _ := json.Marshal(norm.Tags)
	_, err = s.db.Exec(`UPDATE questions SET stem = ?, qtype = ?, options_json = ?, answer = ?, explanation = ?, difficulty = ?, tags_json = ?, updated_at = ? WHERE id = ?`,
		norm.Stem, norm.Type, string(options), norm.Answer, norm.Explanation, norm.Difficulty, string(tags), at, id)
	return id, "updated", err
}

// DeleteQuestions 删除点名的题，连同它们的日程和复习记录，在一个事务里完成。返回真正删掉的题数。
// ids 里的空白项和不存在的题跳过；一个有效的 id 都没有时返回错误。
func (s *Store) DeleteQuestions(ids []string) (int, error) {
	if s == nil {
		return 0, fmt.Errorf("learn store unavailable")
	}
	ids = cleanIDs(ids, len(ids))
	if len(ids) == 0 {
		return 0, fmt.Errorf("no question ids given")
	}
	deleted := 0
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		for _, id := range ids {
			if err := execArgs(tx, id,
				`DELETE FROM practice_events WHERE question_id = ?`,
				`DELETE FROM practice_state WHERE question_id = ?`,
			); err != nil {
				return err
			}
			res, err := tx.Exec(`DELETE FROM questions WHERE id = ?`, id)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n > 0 {
				deleted++
			}
		}
		return nil
	})
	s.mu.Unlock()
	if err != nil {
		return 0, err
	}
	if deleted > 0 {
		s.notify("practice", "", "deleted")
	}
	return deleted, nil
}

// QuestionFlags 是对一批题的标记修改。指针为 nil 的项不动。
// Suspended 为真的题不进练习队列。Mistake 改成真时，这道题还没有第一次答错的时刻就记成现在。
type QuestionFlags struct {
	Bookmarked *bool `json:"bookmarked"`
	Suspended  *bool `json:"suspended"`
	Mistake    *bool `json:"mistake"`
}

// FlagQuestions 给点名的题改收藏、暂停或错题标记，在一个事务里完成。flags 三项都是 nil 时返回错误。
// ids 里不存在的题跳过；一道都没对上时返回 question not found。
func (s *Store) FlagQuestions(ids []string, flags QuestionFlags) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	ids = cleanIDs(ids, len(ids))
	if len(ids) == 0 {
		return fmt.Errorf("no question ids given")
	}
	if flags.Bookmarked == nil && flags.Suspended == nil && flags.Mistake == nil {
		return fmt.Errorf("say what to change: bookmarked, suspended, or mistake")
	}
	at := formatTime(s.clock())
	touched := 0
	s.mu.Lock()
	err := s.inTx(func(tx *sql.Tx) error {
		for _, id := range ids {
			res, err := tx.Exec(`UPDATE questions SET updated_at = ? WHERE id = ?`, at, id)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n == 0 {
				continue
			}
			touched++
			if flags.Bookmarked != nil {
				if _, err := tx.Exec(`UPDATE questions SET bookmarked = ? WHERE id = ?`, boolInt(*flags.Bookmarked), id); err != nil {
					return err
				}
			}
			if flags.Mistake != nil {
				if _, err := tx.Exec(`UPDATE questions SET mistake = ?,
					first_wrong_at = CASE WHEN ? = 1 AND first_wrong_at = '' THEN ? ELSE first_wrong_at END WHERE id = ?`,
					boolInt(*flags.Mistake), boolInt(*flags.Mistake), at, id); err != nil {
					return err
				}
			}
			if flags.Suspended != nil {
				if _, err := tx.Exec(`UPDATE practice_state SET suspended = ? WHERE question_id = ?`, boolInt(*flags.Suspended), id); err != nil {
					return err
				}
			}
		}
		if touched == 0 {
			return fmt.Errorf("question not found")
		}
		return nil
	})
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.notify("practice", "", "flagged")
	return nil
}

// TagCount 是一个标签和带这个标签的题数。
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// QuestionTags 统计题库里每个标签的题数，题多的在前，一样多时按标签排。没有标签时返回空数组。
func (s *Store) QuestionTags() ([]TagCount, error) {
	if s == nil {
		return nil, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	rows, err := s.db.Query(`SELECT tags_json FROM questions WHERE tags_json != '[]' AND tags_json != ''`)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	counts := map[string]int{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var tags []string
		_ = json.Unmarshal([]byte(raw), &tags)
		for _, tag := range tags {
			if tag = strings.TrimSpace(tag); tag != "" {
				counts[tag]++
			}
		}
	}
	rows.Close()
	s.mu.Unlock()
	out := make([]TagCount, 0, len(counts))
	for tag, n := range counts {
		out = append(out, TagCount{Tag: tag, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Tag < out[j].Tag
	})
	return out, nil
}

// BankHit 是给导师看的一道题。Attempted 为真表示学习者评过这道题，或者它是错题；
// 只有这时才带 Answer、Explanation 和学习者上一次的作答 LastAnswer，没做过的题不向导师亮答案。
type BankHit struct {
	ID          string   `json:"id"`
	Stem        string   `json:"stem"`
	Type        string   `json:"type"`
	Options     []Option `json:"options,omitempty"`
	Tags        []string `json:"tags"`
	Mistake     bool     `json:"mistake"`
	Attempted   bool     `json:"attempted"`
	Answer      string   `json:"answer,omitempty"`
	Explanation string   `json:"explanation,omitempty"`
	LastAnswer  string   `json:"last_answer,omitempty"`
}

// SearchBank 替导师在题库里找题，最近复习过或最近录入的在前。query 是要出现在题干、选项或标签里的词，可空。
// mistakesOnly 为真时只找错题。limit 小于等于 0 时取 8，最多 20。total 是符合条件的总数。
// 标准答案和解析只随做过的题带出，见 BankHit。
func (s *Store) SearchBank(query string, mistakesOnly bool, limit int) (hits []BankHit, total int, err error) {
	if s == nil {
		return nil, 0, fmt.Errorf("learn store unavailable")
	}
	if limit <= 0 {
		limit = 8
	}
	if limit > maxTutorHits {
		limit = maxTutorHits
	}
	where, args := cardFilter{Query: query, MistakesOnly: mistakesOnly}.where()
	now := s.clock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.db.QueryRow(`SELECT COUNT(1)`+cardFrom+`WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	cards, err := listCards(s.db, now, `SELECT `+cardCols+cardFrom+`WHERE `+where+`
		ORDER BY COALESCE(NULLIF(st.last_review_at, ''), q.created_at) DESC, q.rowid DESC LIMIT ?`,
		append(append([]any(nil), args...), limit)...)
	if err != nil {
		return nil, 0, err
	}
	hits = make([]BankHit, 0, len(cards))
	for _, c := range cards {
		hit := BankHit{
			ID: c.ID, Stem: clipRunes(c.Stem, 1200), Type: c.Type, Options: c.Options, Tags: c.Tags,
			Mistake: c.Mistake, Attempted: c.Reps > 0 || c.Mistake,
		}
		if hit.Attempted {
			hit.Answer, hit.Explanation, hit.LastAnswer = clipRunes(c.Answer, 1200), clipRunes(c.Explanation, 2000), clipRunes(c.LastAnswer, 600)
		}
		hits = append(hits, hit)
	}
	return hits, total, nil
}
