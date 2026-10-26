// 本文件把模型回复拆成可朗读的 Segment，并去掉方括号里的语气提示。
// photo 和 work 标签不进台词。不调用模型。

package idol

import (
	"regexp"
	"strings"
)

var (
	// tagRE 匹配成对的语言或 photo、work 标签，标签名 2 到 8 个字母。
	tagRE = regexp.MustCompile(`(?s)<([a-zA-Z]{2,8})>(.*?)</([a-zA-Z]{2,8})>`)
	// cueRE 匹配一行内的方括号语气，例如 [happily]。
	cueRE = regexp.MustCompile(`\[[A-Za-z][^\[\]\n]{0,80}\]`)
)

// Parsed 是一条模型回复拆开的结果。Photo 和 Work 从台词里拿出来，不放进 Segments。
type Parsed struct {
	Segments []Segment
	Photo    string
	Work     string
}

// ParseReply 读取成对的语言标签。photo、work 不进台词。没有标签时，整段去掉语气提示后变成一句，口语和字幕相同。
func ParseReply(text, spoken, sub string) Parsed {
	text = strings.TrimSpace(text)
	spoken = lang(spoken)
	sub = lang(sub)
	var out Parsed
	if text == "" {
		return out
	}
	matches := tagRE.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		clean := StripCues(text)
		if clean != "" {
			out.Segments = []Segment{{Spoken: clean, Subtitle: clean}}
		}
		return out
	}
	var pendingSpoken string
	for _, m := range matches {
		open, body, close := strings.ToLower(m[1]), strings.TrimSpace(m[2]), strings.ToLower(m[3])
		if open != close {
			continue
		}
		switch open {
		case "photo":
			if out.Photo == "" {
				out.Photo = StripCues(body)
			}
		case "work":
			if out.Work == "" {
				out.Work = strings.TrimSpace(body)
			}
		default:
			if open == spoken || (spoken == sub && open == sub) {
				if pendingSpoken != "" && open == spoken {
					out.Segments = append(out.Segments, Segment{Spoken: StripCues(pendingSpoken), Subtitle: StripCues(pendingSpoken)})
				}
				pendingSpoken = body
				if spoken == sub {
					out.Segments = append(out.Segments, Segment{Spoken: StripCues(body), Subtitle: StripCues(body)})
					pendingSpoken = ""
				}
				continue
			}
			if open == sub && pendingSpoken != "" {
				out.Segments = append(out.Segments, Segment{Spoken: StripCues(pendingSpoken), Subtitle: StripCues(body)})
				pendingSpoken = ""
				continue
			}
			if open == sub && pendingSpoken == "" {
				out.Segments = append(out.Segments, Segment{Spoken: StripCues(body), Subtitle: StripCues(body)})
			}
		}
	}
	if pendingSpoken != "" {
		out.Segments = append(out.Segments, Segment{Spoken: StripCues(pendingSpoken), Subtitle: StripCues(pendingSpoken)})
	}
	if len(out.Segments) == 0 {
		clean := StripCues(stripTags(text))
		if clean != "" {
			out.Segments = []Segment{{Spoken: clean, Subtitle: clean}}
		}
	}
	return out
}

// StripCues 去掉方括号语气，再把空白收成单空格。
func StripCues(s string) string {
	s = cueRE.ReplaceAllString(s, "")
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

// stripTags 去掉成对标签，只留标签里的文字。
func stripTags(s string) string {
	return strings.TrimSpace(tagRE.ReplaceAllString(s, "$2"))
}

// SpeakLine 把各段 Spoken 用空格接成一行，给语音用。
func SpeakLine(segs []Segment) string {
	var b strings.Builder
	for _, s := range segs {
		if strings.TrimSpace(s.Spoken) == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(strings.TrimSpace(s.Spoken))
	}
	return b.String()
}

// CaptionLine 把各段字幕用换行接起来。某段字幕为空时用该段口语。
func CaptionLine(segs []Segment) string {
	var b strings.Builder
	for _, s := range segs {
		line := s.Subtitle
		if strings.TrimSpace(line) == "" {
			line = s.Spoken
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimSpace(line))
	}
	return b.String()
}

// lang 把语言码收成小写并去掉空白。空则是 zh，长于 8 个字符则截到 8。
func lang(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "zh"
	}
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
