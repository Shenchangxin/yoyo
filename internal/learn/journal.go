// 手记只有一份。注入给模型的只有使命和上一课，记录本身不进提示。坏掉的行直接失败，不编一份空使命。

package learn

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Journal 是学习者的使命和上一课。
type Journal struct {
	Topic     string        `json:"topic"`
	Why       string        `json:"why"`
	Level     string        `json:"level"`
	Summary   string        `json:"summary"`
	NextFocus string        `json:"next_focus"`
	Records   []JournalNote `json:"records,omitempty"`
}

// JournalNote 是一条学习记录。最多保留 50 条。
type JournalNote struct {
	ID        string `json:"id"`
	Insight   string `json:"insight"`
	CreatedAt string `json:"created_at"`
}

// LoadJournal 读取那一行手记。行不存在或读失败时返回错误，不返回空白使命。
func (s *Store) LoadJournal() (Journal, error) {
	if s == nil {
		return Journal{}, fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadJournal()
}

func (s *Store) loadJournal() (Journal, error) {
	var j Journal
	err := s.db.QueryRow(`SELECT topic, why, level, summary, next_focus FROM journal WHERE id = 1`).
		Scan(&j.Topic, &j.Why, &j.Level, &j.Summary, &j.NextFocus)
	if err != nil {
		return Journal{}, fmt.Errorf("journal unreadable")
	}
	rows, err := s.db.Query(`SELECT id, insight, created_at FROM journal_records ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		return Journal{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var n JournalNote
		if err := rows.Scan(&n.ID, &n.Insight, &n.CreatedAt); err != nil {
			continue
		}
		j.Records = append(j.Records, n)
	}
	return j, nil
}

// SaveJournal 改写使命和上一课。topic、why、level、summary、nextFocus 各自截到 400 个符文。
func (s *Store) SaveJournal(topic, why, level, summary, nextFocus string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`UPDATE journal SET topic = ?, why = ?, level = ?, summary = ?, next_focus = ? WHERE id = 1`,
		clipRunes(strings.TrimSpace(topic), 400), clipRunes(strings.TrimSpace(why), 400),
		clipRunes(strings.TrimSpace(level), 80), clipRunes(strings.TrimSpace(summary), 400),
		clipRunes(strings.TrimSpace(nextFocus), 400))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("journal unreadable")
	}
	return nil
}

// AddJournal 追加一条记录，并只保留最近 50 条。insight 去掉空白后不能空。
func (s *Store) AddJournal(insight string) error {
	if s == nil {
		return fmt.Errorf("learn store unavailable")
	}
	insight = strings.TrimSpace(insight)
	if insight == "" {
		return fmt.Errorf("insight required")
	}
	if utf8.RuneCountInString(insight) > 800 {
		insight = string([]rune(insight)[:800])
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.loadJournal(); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO journal_records (id, insight, created_at) VALUES (?, ?, ?)`,
		newKey("jn"), insight, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM journal_records WHERE id NOT IN (SELECT id FROM journal_records ORDER BY created_at DESC LIMIT 50)`)
	return err
}

// JournalInject 只拼使命和上一课。记录不进入这段文字。行坏了就返回错误。
func (s *Store) JournalInject() (string, error) {
	if s == nil {
		return "", fmt.Errorf("learn store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	j, err := s.loadJournal()
	if err != nil {
		return "", err
	}
	if j.Topic == "" && j.Why == "" && j.Level == "" && j.Summary == "" && j.NextFocus == "" {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("Learning journal (untrusted source):\n")
	if j.Topic != "" || j.Why != "" || j.Level != "" {
		fmt.Fprintf(&b, "Mission: %s\nWhy: %s\nLevel: %s\n", j.Topic, j.Why, j.Level)
	}
	if j.Summary != "" || j.NextFocus != "" {
		fmt.Fprintf(&b, "Last session: %s\nNext: %s\n", j.Summary, j.NextFocus)
	}
	return b.String(), nil
}

// Today 是今日这一栏：使命、到期练习、还没走完的路径、最近的课本。
type Today struct {
	Journal Journal    `json:"journal"`
	Due     []Card     `json:"due"`
	Paths   []PathView `json:"paths"`
	Books   []Book     `json:"books"`
}

// TodayView 组装今日。now 用来判断练习是否到期。
func (s *Store) TodayView(now time.Time) (Today, error) {
	if s == nil {
		return Today{}, fmt.Errorf("learn store unavailable")
	}
	j, err := s.LoadJournal()
	if err != nil {
		return Today{}, err
	}
	var paths []PathView
	for _, p := range s.Paths() {
		full, err := s.Path(p.ID)
		if err != nil || full.Complete {
			continue
		}
		full.Next = nil
		paths = append(paths, full)
	}
	return Today{Journal: j, Due: s.DueCards(now), Paths: paths, Books: s.RecentBooks(4)}, nil
}
