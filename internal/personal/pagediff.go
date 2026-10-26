// 比较两次监视抓到的文本行。先按整行配对，再忽略相对时间，最后忽略数字。

package personal

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// relativeTime 匹配英语和德语里的相对时间短语。
var relativeTime = regexp.MustCompile(`(?i)\b(?:\d+|an?|one)\s+(?:sec(?:ond)?|min(?:ute)?|hour|hr|day|week|month|year)s?\s+ago\b|\bvor\s+(?:\d+|einer?|einem)\s+(?:sekunde|minute|stunde|tag|woche|monat|jahr)(?:e|en|n)?\b`)

// numberToken 匹配整数，以及用点或逗号分隔的数字。
var numberToken = regexp.MustCompile(`\d+(?:[.,]\d+)*`)

// PageDiff 是两次页面文本的新增、更新和删除行。
type PageDiff struct {
	Added   []string
	Updated []string
	Removed []string
}

// pageLines 按行去掉空白、折叠空白、单行最多 300 字节并去重。limit 不大于 0 时按 2000 行截断。
// text 按换行切开。每行去掉末尾的 \r 和首尾空白，再折叠空白。
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

// withoutRelativeTimes 把相对时间短语换成 <time>。
// text 里匹配到的相对时间短语换成 <time>。
func withoutRelativeTimes(text string) string {
	return relativeTime.ReplaceAllString(text, "<time>")
}

// numberless 先去掉相对时间，再把数字换成 #，并把字母后单独的 s 去掉后转成小写。
// line 先交给 withoutRelativeTimes，再把数字换成 #。
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

// pair 用 key 把上一份和这一份里相同键的行配对，返回配对上的当前行、未配上的当前行，以及剩下的旧行。
// previous 按 key 分组。被当前行用掉的从剩余里减去，没被用掉的留在 left。
// current 逐行用 key 去配。配上的放进 paired，配不上的放进 unpaired。
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

// empty 把 nil 切片换成空切片，非 nil 原样返回。
// ss 为 nil 时返回空切片，否则原样返回。
func empty(ss []string) []string {
	if ss == nil {
		return []string{}
	}
	return ss
}

// diffPage 先按整行配对，再按去掉相对时间的文本配对，最后按去掉数字的文本配对。未配上的当前行算新增，配上的算更新，剩下的旧行算删除。
// previous 作为第一轮的旧行，之后改用上一轮剩下的旧行。
// current 作为第一轮的当前行，之后改用上一轮没配上的当前行。
func diffPage(previous, current []string) PageDiff {
	_, unpaired, left := pair(previous, current, func(line string) string { return line })
	_, unpaired, left = pair(left, unpaired, withoutRelativeTimes)
	paired, unpaired, left := pair(left, unpaired, numberless)
	return PageDiff{Added: empty(unpaired), Updated: empty(paired), Removed: empty(left)}
}

// describePageDiff 用 New、Updated、Removed 列出差异。新增最多 8 行，更新和删除最多 4 行，每行最多 160 字节。
// diff 的 Added 最多列 8 行，Updated 和 Removed 最多列 4 行。
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

// countPageDiff 返回新增、更新、删除的行数说明。总数为 0 时返回空串。
// diff 的 Added、Updated、Removed 长度用来拼说明。三者长度都是 0 时返回空串。
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
