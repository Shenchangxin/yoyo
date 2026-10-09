package personal

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var relativeTime = regexp.MustCompile(`(?i)\b(?:\d+|an?|one)\s+(?:sec(?:ond)?|min(?:ute)?|hour|hr|day|week|month|year)s?\s+ago\b|\bvor\s+(?:\d+|einer?|einem)\s+(?:sekunde|minute|stunde|tag|woche|monat|jahr)(?:e|en|n)?\b`)

var numberToken = regexp.MustCompile(`\d+(?:[.,]\d+)*`)

type PageDiff struct {
	Added   []string
	Updated []string
	Removed []string
}

func pageLines(text string, limit int) []string {
	if limit <= 0 {
		limit = 2000
	}
	seen := map[string]bool{}
	var out []string
	for _, raw := range strings.Split(text, "\n") {
		line := strings.Join(strings.Fields(strings.TrimSpace(strings.TrimSuffix(raw, "\r"))), " ")
		if len(line) > 300 {
			line = line[:300]
		}
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		out = append(out, line)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func withoutRelativeTimes(text string) string {
	return relativeTime.ReplaceAllString(text, "<time>")
}

func numberless(line string) string {
	s := withoutRelativeTimes(line)
	s = numberToken.ReplaceAllString(s, "#")
	var b strings.Builder
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if unicode.IsLetter(r) && i+1 < len(rs) && rs[i+1] == 's' && (i+2 == len(rs) || !unicode.IsLetter(rs[i+2])) {
			b.WriteRune(unicode.ToLower(r))
			i++
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func pair(previous, current []string, key func(string) string) (paired, unpaired, left []string) {
	open := map[string][]string{}
	for _, line := range previous {
		k := key(line)
		open[k] = append(open[k], line)
	}
	used := map[string]int{}
	for _, line := range current {
		k := key(line)
		bucket := open[k]
		if len(bucket) == 0 {
			unpaired = append(unpaired, line)
			continue
		}
		match := bucket[0]
		open[k] = bucket[1:]
		used[match]++
		paired = append(paired, line)
	}
	for _, line := range previous {
		if used[line] > 0 {
			used[line]--
			continue
		}
		left = append(left, line)
	}
	return
}

func empty(ss []string) []string {
	if ss == nil {
		return []string{}
	}
	return ss
}

func diffPage(previous, current []string) PageDiff {
	_, unpaired, left := pair(previous, current, func(line string) string { return line })
	_, unpaired, left = pair(left, unpaired, withoutRelativeTimes)
	paired, unpaired, left := pair(left, unpaired, numberless)
	return PageDiff{Added: empty(unpaired), Updated: empty(paired), Removed: empty(left)}
}

func describePageDiff(diff PageDiff) string {
	section := func(title string, lines []string, limit int) []string {
		if len(lines) == 0 {
			return nil
		}
		out := []string{title + ":"}
		n := limit
		if n > len(lines) {
			n = len(lines)
		}
		for _, line := range lines[:n] {
			clip := line
			if len(clip) > 160 {
				clip = clip[:160]
			}
			out = append(out, "• "+clip)
		}
		if len(lines) > limit {
			out = append(out, fmt.Sprintf("+%d more", len(lines)-limit))
		}
		return out
	}
	var parts []string
	parts = append(parts, section("New", diff.Added, 8)...)
	parts = append(parts, section("Updated", diff.Updated, 4)...)
	parts = append(parts, section("Removed", diff.Removed, 4)...)
	return strings.Join(parts, "\n")
}

func countPageDiff(diff PageDiff) string {
	total := len(diff.Added) + len(diff.Updated) + len(diff.Removed)
	if total == 0 {
		return ""
	}
	var parts []string
	if len(diff.Added) > 0 {
		parts = append(parts, fmt.Sprintf("%d new", len(diff.Added)))
	}
	if len(diff.Updated) > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", len(diff.Updated)))
	}
	if len(diff.Removed) > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", len(diff.Removed)))
	}
	noun := "lines"
	if total == 1 {
		noun = "line"
	}
	return fmt.Sprintf("%d %s changed (%s)", total, noun, strings.Join(parts, ", "))
}
