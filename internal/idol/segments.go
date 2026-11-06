// 本文件把模型回复拆成可朗读的 Segment：一次读完（ParseReply），或一边收增量一边吐出写完的分段（ReplyStream）。
// 方括号里的语气提示从台词和字幕里去掉，第一处留在 Cue 给语音合成和立绘。photo 和 work 标签不进台词。不调用模型。

package idol

import (
	"regexp"
	"strings"
)

var (
	// tagRE 匹配成对的语言或 photo、work 标签。标签名以字母开头，可以带连字符（zh-tw）。
	tagRE = regexp.MustCompile(`(?s)<([a-zA-Z][a-zA-Z-]{1,9})>(.*?)</([a-zA-Z][a-zA-Z-]{1,9})>`)
	// cueRE 匹配一行内的方括号语气，例如 [happily]。
	cueRE = regexp.MustCompile(`\[[A-Za-z][^\[\]\n]{0,80}\]`)
)

// Parsed 是一条模型回复拆开的结果。Photo 和 Work 从台词里拿出来，不放进 Segments。
type Parsed struct {
	Segments []Segment
	Photo    string
	Work     string
}

// ParseReply 读取成对的语言标签。photo、work 不进台词。没有任何标签时，整段去掉语气提示后变成一句，口语和字幕相同。
// spoken 和 sub 交给 lang，用来识别口语和字幕标签；两者相同时每个口语标签就是一段。
func ParseReply(text, spoken, sub string) Parsed {
	return parse(text, spoken, sub, true)
}

// parse 是 ParseReply 和 ReplyStream 共用的读法。final 为假时只算已经写完的段：
// 口语和字幕不同时，最后一句口语还在等字幕，不算；也不做「没有标签就整段算一句」的兜底。
// 写完的段只会在后面追加，不会因为后来的文字改变，所以流式吐出的段和最后读完的一致。
func parse(text, spoken, sub string, final bool) Parsed {
	text = strings.TrimSpace(text)
	spoken = lang(spoken)
	sub = lang(sub)
	var out Parsed
	if text == "" {
		return out
	}
	add := func(s Segment) {
		if s.Spoken != "" || s.Subtitle != "" {
			out.Segments = append(out.Segments, s)
		}
	}
	var pending *Segment
	for _, m := range tagRE.FindAllStringSubmatch(text, -1) {
		open, body, closing := strings.ToLower(m[1]), m[2], strings.ToLower(m[3])
		if open != closing {
			continue
		}
		switch open {
		case "photo":
			if out.Photo == "" {
				out.Photo = StripCues(body)
			}
			continue
		case "work":
			if out.Work == "" {
				out.Work = strings.TrimSpace(body)
			}
			continue
		}
		if spoken == sub {
			if open == spoken {
				add(makeSegment(body, body))
			}
			continue
		}
		switch open {
		case spoken:
			if pending != nil {
				pending.Subtitle = pending.Spoken
				add(*pending)
			}
			s := makeSegment(body, "")
			pending = &s
		case sub:
			if pending != nil {
				pending.Subtitle = StripCues(body)
				add(*pending)
				pending = nil
			} else {
				add(makeSegment(body, body))
			}
		}
	}
	if final {
		if pending != nil {
			pending.Subtitle = pending.Spoken
			add(*pending)
		}
		if len(out.Segments) == 0 {
			if clean := stripTags(text); clean != "" {
				add(makeSegment(clean, clean))
			}
		}
	}
	return out
}

// makeSegment 用口语原文和字幕原文造一段：两边都去掉语气提示，口语里的第一处语气提示留在 Cue。subRaw 为空时字幕等于口语。
func makeSegment(spokenRaw, subRaw string) Segment {
	s := Segment{Spoken: StripCues(spokenRaw), Cue: firstCue(spokenRaw)}
	if strings.TrimSpace(subRaw) != "" {
		s.Subtitle = StripCues(subRaw)
	}
	if s.Subtitle == "" {
		s.Subtitle = s.Spoken
	}
	return s
}

// firstCue 取出第一处方括号语气提示里的英文，没有就是空串。
func firstCue(s string) string {
	m := cueRE.FindString(s)
	if m == "" {
		return ""
	}
	return strings.TrimSpace(strings.Trim(m, "[]"))
}

// StreamEvent 是流式读回复时的一件事：一段写完了（Segment 非空，Index 是它的序号），或者她写完了照片的画面描述（Photo 非空）。
type StreamEvent struct {
	Segment *Segment
	Index   int
	Photo   string
}

// ReplyStream 一边收模型的增量，一边吐出写完的分段和照片描述。同一段只吐一次。
type ReplyStream struct {
	spoken, sub string
	buf         strings.Builder
	sent        int
	photoSent   bool
}

// NewReplyStream 按这位偶像的口语和字幕语言建一个读回复的流。
func NewReplyStream(spoken, sub string) *ReplyStream {
	return &ReplyStream{spoken: spoken, sub: sub}
}

// Feed 收下一段增量，返回因此写完的段和照片描述。
func (r *ReplyStream) Feed(delta string) []StreamEvent {
	r.buf.WriteString(delta)
	return r.events(parse(r.buf.String(), r.spoken, r.sub, false))
}

// Finish 结束这条回复，返回完整的读法和还没吐出的段。flush 为假时（她被打断）只留已经写完的段，不收半句。
func (r *ReplyStream) Finish(flush bool) (Parsed, []StreamEvent) {
	p := parse(r.buf.String(), r.spoken, r.sub, flush)
	if !flush && len(p.Segments) > r.sent {
		p.Segments = p.Segments[:r.sent]
	}
	return p, r.events(p)
}

// Text 返回到目前为止收到的全部原文。
func (r *ReplyStream) Text() string { return r.buf.String() }

// events 把新写完的段和第一次出现的照片描述变成事件。
func (r *ReplyStream) events(p Parsed) []StreamEvent {
	var out []StreamEvent
	if p.Photo != "" && !r.photoSent {
		r.photoSent = true
		out = append(out, StreamEvent{Photo: p.Photo})
	}
	for i := r.sent; i < len(p.Segments); i++ {
		s := p.Segments[i]
		out = append(out, StreamEvent{Segment: &s, Index: i})
	}
	if len(p.Segments) > r.sent {
		r.sent = len(p.Segments)
	}
	return out
}

// cueMoods 把语气提示里的英文词根归到立绘的七种情绪。按顺序找，先命中的算。
var cueMoods = []struct {
	mood  string
	roots []string
}{
	{"surprised", []string{"surpris", "shock", "gasp", "startl", "astonish", "panic"}},
	{"shy", []string{"shy", "embarrass", "bashful", "blush", "fluster", "timid", "sheepish"}},
	{"angry", []string{"angr", "annoy", "sulk", "pout", "irrit", "frustrat", "huff", "indignant"}},
	{"sad", []string{"sad", "disappoint", "worr", "lonel", "tear", "sigh", "gloom", "hurt", "upset"}},
	{"thinking", []string{"think", "hesita", "ponder", "wonder", "unsure", "mumbl", "puzzl"}},
	{"happy", []string{"happ", "excit", "cheer", "laugh", "giggl", "smil", "joy", "bright", "playful", "teas", "proud", "delight"}},
	{"calm", []string{"gentl", "soft", "calm", "whisper", "sleepy", "quiet", "warm"}},
}

// CueEmotion 把语气提示归到 calm、happy、shy、surprised、sad、angry、thinking 之一；认不出时返回空串。
func CueEmotion(cue string) string {
	c := strings.ToLower(strings.TrimSpace(cue))
	if c == "" {
		return ""
	}
	for _, m := range cueMoods {
		for _, root := range m.roots {
			if strings.Contains(c, root) {
				return m.mood
			}
		}
	}
	return ""
}

// StripCues 去掉方括号语气，再把空白收成单空格。
func StripCues(s string) string {
	s = cueRE.ReplaceAllString(s, "")
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

// stripTags 去掉成对标签，只留标签里的文字，再去掉语气提示。
func stripTags(s string) string {
	return StripCues(tagRE.ReplaceAllString(s, "$2"))
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

// lang 把语言码收成小写并去掉空白。空则是 zh，长于 10 个字符则截到 10。
func lang(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "zh"
	}
	if len(s) > 10 {
		return s[:10]
	}
	return s
}
