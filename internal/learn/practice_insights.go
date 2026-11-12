// 练习的统计：最近若干天每天的复习量、记住率、未来七天的到期预报和各来源的题数。
// 只读复习记录和日程，不改任何东西。日子按本地日历算。

package learn

import (
	"fmt"
	"time"
)

const (
	forecastDays = 7
	dayLayout    = "2006-01-02"
)

// InsightDay 是一个本地日的统计。Reviews 是评分次数，Again 是其中评为重来的次数，
// NewCards 是这一天第一次评分的新题数，Mistakes 是这一天第一次答错的题数。
type InsightDay struct {
	Date     string `json:"date"`
	Reviews  int    `json:"reviews"`
	Again    int    `json:"again"`
	NewCards int    `json:"new_cards"`
	Mistakes int    `json:"mistakes"`
}

// InsightRetention 是记住率。Reviews 是统计范围内对评过分的题的复习次数，Rate 是其中没评为重来的比例；Reviews 为 0 时 Rate 是 0。
type InsightRetention struct {
	Rate    float64 `json:"rate"`
	Reviews int     `json:"reviews"`
}

// ForecastDay 是某个本地日到期的题数。今天这一项包括已经过期还没复习的题。
type ForecastDay struct {
	Date string `json:"date"`
	Due  int    `json:"due"`
}

// SourceCount 是一个来源和它的题数。
type SourceCount struct {
	Source string `json:"source"`
	Count  int    `json:"count"`
}

// Insights 是练习的统计。Daily 从最早的一天排到今天，共 Days 项。Forecast 从今天起共 7 项。
type Insights struct {
	Days      int              `json:"days"`
	Daily     []InsightDay     `json:"daily"`
	Retention InsightRetention `json:"retention"`
	Forecast  []ForecastDay    `json:"forecast"`
	Sources   []SourceCount    `json:"sources"`
}

// insightWindow 把天数收成 7、30 或 90。days 小于等于 0 时取 30，其余取不小于它的最近一档，超过 90 取 90。
func insightWindow(days int) int {
	switch {
	case days <= 0:
		return 30
	case days <= 7:
		return 7
	case days <= 30:
		return 30
	}
	return 90
}

// PracticeInsights 返回最近 days 天的统计。days 先交给 insightWindow。暂停的题和新题不进到期预报。
func (s *Store) PracticeInsights(days int) (Insights, error) {
	if s == nil {
		return Insights{}, fmt.Errorf("learn store unavailable")
	}
	days = insightWindow(days)
	now := s.clock()
	today := localDay(now)
	start := today.AddDate(0, 0, -(days - 1))
	out := Insights{Days: days, Daily: make([]InsightDay, days), Forecast: make([]ForecastDay, forecastDays), Sources: []SourceCount{}}
	index := map[string]int{}
	for i := range out.Daily {
		date := start.AddDate(0, 0, i).Format(dayLayout)
		out.Daily[i].Date = date
		index[date] = i
	}
	// dayOf 返回一个时刻落在 Daily 的第几项。落在范围之外时 ok 为假。
	dayOf := func(at string) (int, bool) {
		t := parseTime(at)
		if t.IsZero() {
			return 0, false
		}
		i, ok := index[localDay(t).Format(dayLayout)]
		return i, ok
	}
	since := formatTime(start)

	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT at, rating, state_before, reps_before FROM practice_events WHERE at >= ?`, since)
	if err != nil {
		return Insights{}, err
	}
	kept := 0
	for rows.Next() {
		var at, rating, state string
		var reps int
		if err := rows.Scan(&at, &rating, &state, &reps); err != nil {
			rows.Close()
			return Insights{}, err
		}
		i, ok := dayOf(at)
		if !ok {
			continue
		}
		again := rating == RatingAgain.String()
		out.Daily[i].Reviews++
		if again {
			out.Daily[i].Again++
		}
		if state == CardNew {
			out.Daily[i].NewCards++
		}
		// 记住率只看以前评过分的题：第一次见的题谈不上记没记住。
		if reps > 0 {
			out.Retention.Reviews++
			if !again {
				kept++
			}
		}
	}
	rows.Close()
	if out.Retention.Reviews > 0 {
		out.Retention.Rate = float64(kept) / float64(out.Retention.Reviews)
	}

	rows, err = s.db.Query(`SELECT first_wrong_at FROM questions WHERE first_wrong_at >= ?`, since)
	if err != nil {
		return Insights{}, err
	}
	for rows.Next() {
		var at string
		if err := rows.Scan(&at); err != nil {
			rows.Close()
			return Insights{}, err
		}
		if i, ok := dayOf(at); ok {
			out.Daily[i].Mistakes++
		}
	}
	rows.Close()

	for i := range out.Forecast {
		out.Forecast[i].Date = today.AddDate(0, 0, i).Format(dayLayout)
	}
	rows, err = s.db.Query(`SELECT st.due_at`+cardFrom+`WHERE st.suspended = 0 AND st.due_at < ? AND `+notNewSQL,
		formatTime(today.AddDate(0, 0, forecastDays)))
	if err != nil {
		return Insights{}, err
	}
	for rows.Next() {
		var due string
		if err := rows.Scan(&due); err != nil {
			rows.Close()
			return Insights{}, err
		}
		out.Forecast[forecastSlot(parseTime(due), today)].Due++
	}
	rows.Close()

	rows, err = s.db.Query(`SELECT CASE WHEN source = '' THEN 'manual' ELSE source END AS src, COUNT(1) FROM questions GROUP BY src ORDER BY COUNT(1) DESC, src`)
	if err != nil {
		return Insights{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var sc SourceCount
		if err := rows.Scan(&sc.Source, &sc.Count); err != nil {
			return Insights{}, err
		}
		out.Sources = append(out.Sources, sc)
	}
	return out, rows.Err()
}

// forecastSlot 返回到期时刻 due 落在预报的第几项。today 是今天的本地零点。早于明天零点的都算今天，超出预报范围的算最后一天。
func forecastSlot(due, today time.Time) int {
	for i := 0; i < forecastDays-1; i++ {
		if due.Before(today.AddDate(0, 0, i+1)) {
			return i
		}
	}
	return forecastDays - 1
}
