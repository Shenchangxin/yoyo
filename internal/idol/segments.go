package idol

import (
	"regexp"
	"strings"
)

var (
	tagRE = regexp.MustCompile(`(?s)<([a-zA-Z]{2,8})>(.*?)</([a-zA-Z]{2,8})>`)
	cueRE = regexp.MustCompile(`\[[A-Za-z][^\[\]\n]{0,80}\]`)
)

// Parsed is one model reply split into speakable segments.
type Parsed struct {
	Segments []Segment
	Photo    string
	Work     string
}

// ParseReply reads <lang>…</lang> pairs. A photo or work tag is kept out of
// the spoken line. Untagged text becomes one segment in both languages.
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

func StripCues(s string) string {
	s = cueRE.ReplaceAllString(s, "")
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func stripTags(s string) string {
	return strings.TrimSpace(tagRE.ReplaceAllString(s, "$2"))
}

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
