// 词轴。一句拆成词，改一行只重排这一行的时间。

package film

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"
)

// WordsFromCues 把每一句拆成词，并让这些词平分这一句的时间。汉字和假名一个字一个词，拉丁字母连在一起。
// cues 逐句用 Text 拆词，时间从 Start 平分到 End；End 减 Start 小于 0 时按 0。
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

// WordsFromStamps 读词级 JSON。有扁平的 words 就用它。WhisperX 写在 segments 里的词会展开，并带上句子的说话人。解不开就返回错误。
// raw 交给 json.Unmarshal。
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

// ReplaceLine 重写一行，并只把这一行原来的时间平分给新词。不用这一行的镜头保持原样。然后重算待办，输入没变的产物编号会留下。找不到这一行就原样返回。
// lineID 经 clean 后传给 lineSpan，并用来丢掉旧词、写下新词的 ID 和 LineID。
// text 传给 splitTokens；一个词都没有时改用 clean 后的整段作为唯一词。
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

// lineSpan 返回这一行词的最早开始和最晚结束。没有这一行时第三个结果是假。
// words 逐个比较 LineID，取最早的 Start 和最晚的 End。
// lineID 用来匹配同一行；没有匹配时第三个结果是假。
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

// splitTokens 把一句拆成词。汉字、假名、片假名和谚文逐字切开，拉丁字母和数字连成一块。
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

// cueLine 给第 i 句一个行编号，从 line-0 起。
func cueLine(i int) string { return "line-" + itoa(i) }

// Line 是按时间排好的一行，可以整行改写。
type Line struct {
	ID   string
	Text string
}

// LineTexts 按说话顺序把每一行的词拼回句子。
// p 按 Words 的现有顺序，LineID 变化时把上一行交给 joinTokens。
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

// joinTokens 把词拼回一句。任一词超过一个字时，和相邻词之间加空格。
// words 逐个写入，任一词的字数大于 1 时和相邻词之间加空格。
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

// ShotWords 返回挂在这一镜上的词。一镜没有词、但有时间跨度时，返回一个盖住这段时间的空词。
// p 用 Words 按 shot 的 WordIDs 取出词。
// shot 没有对上的词且 End 大于 Start 时，返回一个盖住 Start 到 End 的空词。
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

// itoa 把非负整数写成十进制。不使用 fmt。
// n 为 0 时返回 0，大于 0 时按余 10 写成十进制。
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
