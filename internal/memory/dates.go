package memory

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// dateLayout 是记忆里写日子的格式，也是 ValidUntil 的格式。
const dateLayout = "2006-01-02"

var (
	zhDay      = regexp.MustCompile(`大后天|后天|明天|明日|今天|今日|昨天|昨日|前天`)
	zhWeek     = regexp.MustCompile(`(上上|上个|上|下下|下个|下|本|这个|这)?(?:周|星期|礼拜)([一二三四五六日天])`)
	zhAfter    = regexp.MustCompile(`([0-9]+|[一二两三四五六七八九十]+)\s*(天|周|个星期|星期|个礼拜|礼拜|个月)(?:以后|之后|后)`)
	zhMonthEnd = regexp.MustCompile(`(上个|上|下个|下|本|这个)?月底`)
	zhYearEnd  = regexp.MustCompile(`(明年|今年|去年)?年底`)
	zhMonthDay = regexp.MustCompile(`(?:(\d{4})年)?(\d{1,2})月(\d{1,2})[日号]`)
	isoDay     = regexp.MustCompile(`\b(\d{4})-(\d{1,2})-(\d{1,2})\b`)
	enDay      = regexp.MustCompile(`(?i)\b(today|tonight|tomorrow|yesterday)\b`)
	enWeek     = regexp.MustCompile(`(?i)\b(?:(next|this|on|coming|last)\s+)?(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b`)
	enAfter    = regexp.MustCompile(`(?i)\bin\s+(\d+)\s+(days?|weeks?|months?)\b`)
	beginsWord = regexp.MustCompile(`(?i)开始|从.{0,12}起|以后都|以后就|\bstarting\b|\bonwards\b|\bfrom now on\b`)
)

// dayWordGuard 是相对日子的第一个字前面常见的、能和它连成别的词的字。比如「目前天气」里的「前天」、「说明天气」里的「明天」。
var dayWordGuard = map[rune]string{
	'前': "目之以提从当面眼空跟向事生",
	'后': "然最之以随先背前落今身而午",
	'明': "说证表声聪光文发鲜透分英清黎",
	'今': "如至当而迄从",
}

// dateHit 是正文里认出的一个日子。relative 为真时它原本不是日期，要在原词后面补上。
type dateHit struct {
	start, end int
	day        time.Time
	relative   bool
}

// anchorDates 把正文里的相对日子换算成日期，补在原词后面的括号里，比如「下周三（2026-11-11）」。
// until 是正文里今天或以后最晚的那一天：说的是一件会过去的安排。都是过去的日子，或者说的是从某天开始的状态时为空。
// 星期几前面要有「下」「本」「这」「上」或 next、this、on、last 才换算：光写「周五」「Friday」多半是每周的习惯，比如「周报周五发」。
// 「每周五」「every Friday」也不换算。已经换算过的不再补。
func anchorDates(text string, now time.Time) (string, string) {
	today := midnight(now)
	hits := findDates(text, today)
	if len(hits) == 0 {
		return text, ""
	}
	var b strings.Builder
	last := 0
	latest := time.Time{}
	for _, hit := range hits {
		if !hit.day.Before(today) && hit.day.After(latest) {
			latest = hit.day
		}
		if !hit.relative || anchored(text[hit.end:]) {
			continue
		}
		b.WriteString(text[last:hit.end])
		stamp := hit.day.Format(dateLayout)
		if r, _ := utf8.DecodeRuneInString(text[hit.start:]); cjk(r) {
			b.WriteString("（" + stamp + "）")
		} else {
			b.WriteString(" (" + stamp + ")")
		}
		last = hit.end
	}
	b.WriteString(text[last:])
	until := ""
	if !latest.IsZero() && !beginsWord.MatchString(text) {
		until = latest.Format(dateLayout)
	}
	return b.String(), until
}

// anchored 在正文紧接着已经写了日期时为真。
func anchored(rest string) bool {
	for _, lead := range []string{"（", " (", "("} {
		if strings.HasPrefix(rest, lead) {
			tail := rest[len(lead):]
			return isoDay.MatchString(tail[:min(len(tail), 10)])
		}
	}
	return false
}

// findDates 认出正文里所有日子，按出现先后排好，重叠的只留先出现、更长的那一个。
func findDates(text string, today time.Time) []dateHit {
	var hits []dateHit
	add := func(start, end int, day time.Time, relative bool) {
		if !day.IsZero() {
			hits = append(hits, dateHit{start: start, end: end, day: day, relative: relative})
		}
	}
	for _, m := range zhDay.FindAllStringIndex(text, -1) {
		word := text[m[0]:m[1]]
		first, _ := utf8.DecodeRuneInString(word)
		if prev, _ := utf8.DecodeLastRuneInString(text[:m[0]]); strings.ContainsRune(dayWordGuard[first], prev) {
			continue
		}
		add(m[0], m[1], today.AddDate(0, 0, zhDayOffset(word)), true)
	}
	for _, m := range zhWeek.FindAllStringSubmatchIndex(text, -1) {
		if m[2] < 0 || recurring(text[:m[0]]) {
			continue
		}
		add(m[0], m[1], weekDay(today, zhWeekday(text[m[4]:m[5]]), zhWeekShift(text[m[2]:m[3]]), false), true)
	}
	for _, m := range zhAfter.FindAllStringSubmatchIndex(text, -1) {
		if n := zhNumber(text[m[2]:m[3]]); n > 0 {
			add(m[0], m[1], afterUnit(today, n, text[m[4]:m[5]]), true)
		}
	}
	for _, m := range zhMonthEnd.FindAllStringSubmatchIndex(text, -1) {
		months := 0
		if m[2] >= 0 {
			switch {
			case strings.HasPrefix(text[m[2]:m[3]], "下"):
				months = 1
			case strings.HasPrefix(text[m[2]:m[3]], "上"):
				months = -1
			}
		}
		first := time.Date(today.Year(), today.Month()+time.Month(months)+1, 1, 0, 0, 0, 0, today.Location())
		add(m[0], m[1], first.AddDate(0, 0, -1), true)
	}
	for _, m := range zhYearEnd.FindAllStringSubmatchIndex(text, -1) {
		year := today.Year()
		if m[2] >= 0 {
			switch text[m[2]:m[3]] {
			case "明年":
				year++
			case "去年":
				year--
			}
		}
		add(m[0], m[1], time.Date(year, 12, 31, 0, 0, 0, 0, today.Location()), true)
	}
	for _, m := range zhMonthDay.FindAllStringSubmatchIndex(text, -1) {
		year := today.Year()
		if m[2] >= 0 {
			year, _ = strconv.Atoi(text[m[2]:m[3]])
		}
		month, _ := strconv.Atoi(text[m[4]:m[5]])
		day, _ := strconv.Atoi(text[m[6]:m[7]])
		add(m[0], m[1], civilDay(year, month, day, today.Location()), false)
	}
	for _, m := range isoDay.FindAllStringSubmatchIndex(text, -1) {
		year, _ := strconv.Atoi(text[m[2]:m[3]])
		month, _ := strconv.Atoi(text[m[4]:m[5]])
		day, _ := strconv.Atoi(text[m[6]:m[7]])
		add(m[0], m[1], civilDay(year, month, day, today.Location()), false)
	}
	for _, m := range enDay.FindAllStringIndex(text, -1) {
		offset := 0
		switch strings.ToLower(text[m[0]:m[1]]) {
		case "tomorrow":
			offset = 1
		case "yesterday":
			offset = -1
		}
		add(m[0], m[1], today.AddDate(0, 0, offset), true)
	}
	for _, m := range enWeek.FindAllStringSubmatchIndex(text, -1) {
		if m[2] < 0 || recurring(text[:m[0]]) {
			continue
		}
		shift, upcoming := 0, true
		switch strings.ToLower(text[m[2]:m[3]]) {
		case "next":
			shift, upcoming = 1, false
		case "last":
			shift, upcoming = -1, false
		}
		add(m[0], m[1], weekDay(today, enWeekday(strings.ToLower(text[m[4]:m[5]])), shift, upcoming), true)
	}
	for _, m := range enAfter.FindAllStringSubmatchIndex(text, -1) {
		n, _ := strconv.Atoi(text[m[2]:m[3]])
		unit := strings.ToLower(text[m[4]:m[5]])
		switch {
		case strings.HasPrefix(unit, "day"):
			unit = "天"
		case strings.HasPrefix(unit, "week"):
			unit = "周"
		default:
			unit = "个月"
		}
		if n > 0 {
			add(m[0], m[1], afterUnit(today, n, unit), true)
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].start != hits[j].start {
			return hits[i].start < hits[j].start
		}
		return hits[i].end > hits[j].end
	})
	var out []dateHit
	end := -1
	for _, hit := range hits {
		if hit.start < end {
			continue
		}
		out = append(out, hit)
		end = hit.end
	}
	return out
}

// recurring 在日子前面紧挨着「每」「隔」「逢」「every」「each」时为真：说的是反复的事，不是哪一天。
func recurring(before string) bool {
	tail := []rune(before)
	if len(tail) > 3 {
		tail = tail[len(tail)-3:]
	}
	if strings.ContainsAny(string(tail), "每隔逢") {
		return true
	}
	lower := strings.ToLower(strings.TrimRight(before, " "))
	return strings.HasSuffix(lower, "every") || strings.HasSuffix(lower, "each")
}

func zhDayOffset(word string) int {
	switch word {
	case "大后天":
		return 3
	case "后天":
		return 2
	case "明天", "明日":
		return 1
	case "昨天", "昨日":
		return -1
	case "前天":
		return -2
	default:
		return 0
	}
}

// zhWeekShift 把周前面的字换成相对本周的周数。
func zhWeekShift(prefix string) int {
	switch prefix {
	case "下", "下个":
		return 1
	case "下下":
		return 2
	case "上", "上个":
		return -1
	case "上上":
		return -2
	default:
		return 0
	}
}

// zhWeekday 把「一」到「日」换成周一为 1、周日为 7 的序号。认不出时返回 0。
func zhWeekday(word string) int {
	switch word {
	case "一":
		return 1
	case "二":
		return 2
	case "三":
		return 3
	case "四":
		return 4
	case "五":
		return 5
	case "六":
		return 6
	case "日", "天":
		return 7
	default:
		return 0
	}
}

func enWeekday(word string) int {
	for i, name := range []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"} {
		if name == word {
			return i + 1
		}
	}
	return 0
}

// weekDay 是本周（周一开始）往后 shift 周的第 n 天。upcoming 为真时不看 shift，取今天或以后最近的那一天。
func weekDay(today time.Time, n, shift int, upcoming bool) time.Time {
	if n < 1 || n > 7 {
		return time.Time{}
	}
	monday := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	if upcoming {
		day := monday.AddDate(0, 0, n-1)
		if day.Before(today) {
			day = day.AddDate(0, 0, 7)
		}
		return day
	}
	return monday.AddDate(0, 0, shift*7+n-1)
}

func afterUnit(today time.Time, n int, unit string) time.Time {
	switch unit {
	case "天":
		return today.AddDate(0, 0, n)
	case "周", "个星期", "星期", "个礼拜", "礼拜":
		return today.AddDate(0, 0, 7*n)
	default:
		return today.AddDate(0, n, 0)
	}
}

var zhDigits = map[rune]int{'一': 1, '二': 2, '两': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}

// zhNumber 认阿拉伯数字和一到九十九的中文数字。认不出时返回 0。
func zhNumber(word string) int {
	if n, err := strconv.Atoi(word); err == nil {
		return n
	}
	runes := []rune(word)
	switch {
	case len(runes) == 1 && runes[0] == '十':
		return 10
	case len(runes) == 1:
		return zhDigits[runes[0]]
	case len(runes) == 2 && runes[0] == '十':
		return 10 + zhDigits[runes[1]]
	case len(runes) == 2 && runes[1] == '十':
		return zhDigits[runes[0]] * 10
	case len(runes) == 3 && runes[1] == '十':
		return zhDigits[runes[0]]*10 + zhDigits[runes[2]]
	default:
		return 0
	}
}

// civilDay 是年月日对应的那一天。日子不存在时返回零值。
func civilDay(year, month, day int, loc *time.Location) time.Time {
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
	if t.Month() != time.Month(month) {
		return time.Time{}
	}
	return t
}

func midnight(now time.Time) time.Time {
	local := now.Local()
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
}

// Expired 在这一条说的是一段已经过去的安排时为真：ValidUntil 早于 now 所在的那一天。
func Expired(it Item, now time.Time) bool {
	return it.ValidUntil != "" && it.ValidUntil < midnight(now).Format(dateLayout)
}

var (
	zhPastWeek  = regexp.MustCompile(`(上上|上)(?:周|个星期|星期|个礼拜|礼拜)`)
	zhThisWeek  = regexp.MustCompile(`(?:这|本)(?:周|个星期|星期|个礼拜|礼拜)`)
	zhPastMonth = regexp.MustCompile(`上个?月`)
	zhThisMonth = regexp.MustCompile(`这个月|本月`)
	recentWord  = regexp.MustCompile(`(?i)最近|这几天|这两天|近几天|前几天|\b(?:recently|lately|these days)\b`)
	enPastWeek  = regexp.MustCompile(`(?i)\blast week\b`)
	enThisWeek  = regexp.MustCompile(`(?i)\bthis week\b`)
	enPastMonth = regexp.MustCompile(`(?i)\blast month\b`)
	enThisMonth = regexp.MustCompile(`(?i)\bthis month\b`)
	vagueWord   = regexp.MustCompile(`(?i)下个?月|月初|月中|周末|过几天|过两天|改天|回头|\bnext month\b|\bthis weekend\b|\bsoon\b`)
)

// weekdayRunes 跟在「周」后面时，说的是哪一天，不是一整周。
const weekdayRunes = "一二三四五六日天"

// window 是查询问到的一段日子，两头都算。
type window struct{ from, to time.Time }

// holds 在这一条落在这段日子里时为真：最近一次记下或改过在这几天，或者正文写着这几天里的某个日期。
func (w window) holds(it Item) bool {
	if day := midnight(updatedOf(it)); !day.Before(w.from) && !day.After(w.to) {
		return true
	}
	for _, m := range isoDay.FindAllStringSubmatch(it.Text, -1) {
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		if d := civilDay(year, month, day, w.from.Location()); !d.IsZero() && !d.Before(w.from) && !d.After(w.to) {
			return true
		}
	}
	return false
}

// queryWindow 认出查询问的是过去哪几天：昨天、上周三、上周、这个月、最近，或者写明的日期。将来的日子不算，今天算。
// 说到好几处时取把它们都包住的那一段。
func queryWindow(q string, now time.Time) (window, bool) {
	today := midnight(now)
	monday := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	var w window
	ok := false
	widen := func(from, to time.Time) {
		if to.After(today) {
			to = today
		}
		if from.After(to) {
			return
		}
		if !ok || from.Before(w.from) {
			w.from = from
		}
		if !ok || to.After(w.to) {
			w.to = to
		}
		ok = true
	}
	for _, hit := range findDates(q, today) {
		if !hit.day.After(today) {
			widen(hit.day, hit.day)
		}
	}
	for _, m := range zhPastWeek.FindAllStringSubmatchIndex(q, -1) {
		if next, _ := utf8.DecodeRuneInString(q[m[1]:]); strings.ContainsRune(weekdayRunes, next) {
			continue
		}
		weeks := 1
		if q[m[2]:m[3]] == "上上" {
			weeks = 2
		}
		from := monday.AddDate(0, 0, -7*weeks)
		widen(from, from.AddDate(0, 0, 6))
	}
	if enPastWeek.MatchString(q) {
		widen(monday.AddDate(0, 0, -7), monday.AddDate(0, 0, -1))
	}
	for _, m := range zhThisWeek.FindAllStringIndex(q, -1) {
		if next, _ := utf8.DecodeRuneInString(q[m[1]:]); !strings.ContainsRune(weekdayRunes, next) {
			widen(monday, today)
		}
	}
	if enThisWeek.MatchString(q) {
		widen(monday, today)
	}
	if zhPastMonth.MatchString(q) || enPastMonth.MatchString(q) {
		widen(first.AddDate(0, -1, 0), first.AddDate(0, 0, -1))
	}
	if zhThisMonth.MatchString(q) || enThisMonth.MatchString(q) {
		widen(first, today)
	}
	if recentWord.MatchString(q) {
		widen(today.AddDate(0, 0, -7), today)
	}
	return w, ok
}

// Vague 返回正文里换算不成日子的说法，比如「下个月」「周末」。已经能换算的（「下个月底」）和反复的（「每周末」）不算。
func Vague(text string) []string {
	hits := findDates(text, midnight(time.Now()))
	var out []string
	seen := map[string]bool{}
	for _, m := range vagueWord.FindAllStringIndex(text, -1) {
		covered := false
		for _, hit := range hits {
			if m[0] < hit.end && hit.start < m[1] {
				covered = true
				break
			}
		}
		word := text[m[0]:m[1]]
		if covered || recurring(text[:m[0]]) || seen[word] {
			continue
		}
		seen[word] = true
		out = append(out, word)
	}
	return out
}
