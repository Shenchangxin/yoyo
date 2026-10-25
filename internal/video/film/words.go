package film

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"
)

// WordsFromCues splits each cue into tokens and gives every token an equal slice of that cue.
func WordsFromCues(cues []Cue) []Word {
	var out []Word
	for i, cue := range cues {
		tokens := splitTokens(cue.Text)
		if len(tokens) == 0 {
			continue
		}
		line := cueLine(i)
		span := cue.End - cue.Start
		if span < 0 {
			span = 0
		}
		step := span / float64(len(tokens))
		for j, token := range tokens {
			start := cue.Start + step*float64(j)
			end := start + step
			if j == len(tokens)-1 {
				end = cue.End
			}
			out = append(out, Word{
				ID: line + "-" + itoa(j), LineID: line, Text: token, Speaker: clean(cue.Speaker),
				Start: start, End: end,
			})
		}
	}
	return out
}

type stampWord struct {
	Text    string  `json:"text"`
	Word    string  `json:"word"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Speaker string  `json:"speaker"`
	Line    string  `json:"line"`
}

type stampFile struct {
	Words    []stampWord `json:"words"`
	Segments []struct {
		Speaker string      `json:"speaker"`
		Words   []stampWord `json:"words"`
	} `json:"segments"`
}

// WordsFromStamps reads a word-level JSON document.
// A flat words list is used as-is. WhisperX writes segments[].words instead.
func WordsFromStamps(raw []byte) ([]Word, error) {
	var doc stampFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	items := doc.Words
	if len(items) == 0 {
		for _, seg := range doc.Segments {
			for _, item := range seg.Words {
				if clean(item.Speaker) == "" {
					item.Speaker = seg.Speaker
				}
				items = append(items, item)
			}
		}
	}
	out := make([]Word, 0, len(items))
	lineCount := map[string]int{}
	for _, item := range items {
		text := clean(item.Text)
		if text == "" {
			text = clean(item.Word)
		}
		if text == "" {
			continue
		}
		line := clean(item.Line)
		if line == "" {
			line = "line-0"
		}
		n := lineCount[line]
		lineCount[line] = n + 1
		out = append(out, Word{
			ID: line + "-" + itoa(n), LineID: line, Text: text, Speaker: clean(item.Speaker),
			Start: item.Start, End: item.End,
		})
	}
	return out, nil
}

// ReplaceLine rewrites one line and redistributes only that line's time.
// Shots and needs that do not use the line keep their asset hashes.
func ReplaceLine(p Project, lineID, text string) Project {
	lineID = clean(lineID)
	start, end, ok := lineSpan(p.Words, lineID)
	if !ok {
		return p
	}
	speaker := ""
	var kept []Word
	for _, w := range p.Words {
		if w.LineID == lineID {
			if speaker == "" {
				speaker = w.Speaker
			}
			continue
		}
		kept = append(kept, w)
	}
	tokens := splitTokens(text)
	if len(tokens) == 0 {
		tokens = []string{clean(text)}
	}
	span := end - start
	step := span / float64(len(tokens))
	fresh := make([]Word, 0, len(tokens))
	ids := make([]string, 0, len(tokens))
	for j, token := range tokens {
		if token == "" {
			continue
		}
		wstart := start + step*float64(j)
		wend := wstart + step
		if j == len(tokens)-1 {
			wend = end
		}
		id := lineID + "-" + itoa(j)
		fresh = append(fresh, Word{ID: id, LineID: lineID, Text: token, Speaker: speaker, Start: wstart, End: wend})
		ids = append(ids, id)
	}
	p.Words = append(kept, fresh...)
	sort.SliceStable(p.Words, func(i, j int) bool {
		return p.Words[i].Start < p.Words[j].Start
	})
	for i := range p.Shots {
		if p.Shots[i].LineID != lineID {
			continue
		}
		p.Shots[i].WordIDs = append([]string(nil), ids...)
	}
	return rehash(p)
}

func lineSpan(words []Word, lineID string) (float64, float64, bool) {
	start, end := 0.0, 0.0
	seen := false
	for _, w := range words {
		if w.LineID != lineID {
			continue
		}
		if !seen || w.Start < start {
			start = w.Start
		}
		if !seen || w.End > end {
			end = w.End
		}
		seen = true
	}
	return start, end, seen
}

func splitTokens(s string) []string {
	var out []string
	var latin strings.Builder
	flush := func() {
		if latin.Len() == 0 {
			return
		}
		out = append(out, latin.String())
		latin.Reset()
	}
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r):
			flush()
			out = append(out, string(r))
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'':
			latin.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return out
}

func cueLine(i int) string { return "line-" + itoa(i) }

// Line is one rewriteable sentence, in time order.
type Line struct {
	ID   string
	Text string
}

// LineTexts joins each line's tokens in the order they are spoken.
func LineTexts(p Project) []Line {
	var out []Line
	var cur string
	var buf []Word
	flush := func() {
		if cur == "" || len(buf) == 0 {
			return
		}
		out = append(out, Line{ID: cur, Text: joinTokens(buf)})
		buf = nil
	}
	for _, w := range p.Words {
		if w.LineID != cur {
			flush()
			cur = w.LineID
		}
		buf = append(buf, w)
	}
	flush()
	return out
}

func joinTokens(words []Word) string {
	var b strings.Builder
	prevLong := false
	for _, w := range words {
		long := len([]rune(w.Text)) > 1
		if b.Len() > 0 && (long || prevLong) {
			b.WriteByte(' ')
		}
		b.WriteString(w.Text)
		prevLong = long
	}
	return b.String()
}

// ShotWords returns the words anchored to a shot, or one span when the shot has none.
func ShotWords(p Project, shot Shot) []Word {
	var out []Word
	for _, id := range shot.WordIDs {
		for _, w := range p.Words {
			if w.ID == id {
				out = append(out, w)
				break
			}
		}
	}
	if len(out) == 0 && shot.End > shot.Start {
		out = []Word{{Start: shot.Start, End: shot.End}}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
