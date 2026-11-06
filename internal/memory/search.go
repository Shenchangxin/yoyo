package memory

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Search 按种类和查询筛选。查询为空时按最近一次记下或改过的时间从新到旧。
// 查询不空时按词打分：少见的词比常见的词重，整句对上再加分；经历越旧分越低，过期的安排打折，说过多次的略高。
// 一个词都对不上的不返回。分数相同时新的在前。
func (s *Store) Search(q string, kind Kind, limit int) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	var docs []Item
	for _, it := range s.items {
		if kind == "" || it.Kind == kind {
			docs = append(docs, it)
		}
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		sort.SliceStable(docs, func(i, j int) bool { return updatedOf(docs[i]).After(updatedOf(docs[j])) })
		return clipItems(docs, limit)
	}
	var out []Item
	for _, hit := range rank(docs, q, s.clock(), true) {
		out = append(out, hit.item)
	}
	return clipItems(out, limit)
}

// recallRunes 是召回时看的那一段话的长度。要问的事多半写在开头，后面贴的日志和代码只会带来碰巧对上的词。
const recallRunes = 240

// maxTerms 是一次打分最多用的词数。
const maxTerms = 64

// askCue 认出一句话是在问事情。只有在问的时候，「昨天」「上周」才当成要找哪几天的记忆；「今天帮我写周报」不算。
var askCue = regexp.MustCompile(`(?i)[?？]|什么|哪|吗|是不是|有没有|记得|多少|几个|几次|谁|怎么样|\b(what|when|which|who|did|was|were|remember|how many)\b`)

// Recall 是这句话可能用得上、又不在常驻画像里的记忆：经历、项目和过期的安排。本机清点不进来。
// 只看这句话的开头。至少对上两个不算常用词的词，或者整句对上，才算；问的是过去哪几天时，那几天的记忆也算。
// 几乎一样的几条只带第一条。最多 limit 条。
func (s *Store) Recall(q string, limit int) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	q = strings.ToLower(strings.TrimSpace(q))
	if runes := []rune(q); len(runes) > recallRunes {
		q = string(runes[:recallRunes])
	}
	if q == "" {
		return nil
	}
	now := s.clock()
	var docs []Item
	for _, it := range s.items {
		if it.Staging || machineItem(it) || strings.TrimSpace(it.Text) == "" {
			continue
		}
		if (it.Kind == KindProfile || it.Kind == KindSemantic) && !Expired(it, now) {
			continue
		}
		docs = append(docs, it)
	}
	var out []Item
	var picked []map[string]bool
	for _, hit := range rank(docs, q, now, askCue.MatchString(q)) {
		if hit.matched < 2 && !hit.phrase {
			continue
		}
		terms := simTerms(hit.item.Text)
		if nearAny(terms, picked) {
			continue
		}
		picked = append(picked, terms)
		out = append(out, hit.item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// nearAny 在 terms 和已经挑出的某一条几乎一样时为真：共有的词超过两边合起来的六成。
func nearAny(terms map[string]bool, picked []map[string]bool) bool {
	for _, other := range picked {
		shared, _, _ := overlap(terms, other)
		if union := len(terms) + len(other) - shared; union > 0 && float64(shared) > 0.6*float64(union) {
			return true
		}
	}
	return false
}

// groundShare 是一句记忆里至少要有多少词出自当事人自己写的话，才算他说的。
const groundShare = 0.6

// Grounded 在 text 的词大半出自 said 时为真：这句记忆是当事人自己写过的话，而不是转述工人报告、网页或邮件里的内容。
func Grounded(text string, said []string) bool {
	terms := simTerms(text)
	if len(terms) == 0 {
		return false
	}
	heard := map[string]bool{}
	for _, line := range said {
		for term := range simTerms(line) {
			heard[term] = true
		}
	}
	hit := 0
	for term := range terms {
		if heard[term] {
			hit++
		}
	}
	return float64(hit) >= groundShare*float64(len(terms))
}

// ranked 是一条打过分的记忆。matched 是对上的、不算常用词的不同词数。
type ranked struct {
	item    Item
	score   float64
	matched int
	phrase  bool
}

// windowBonus 让落在查询所问那几天里的记忆排在最前，同在那几天里的再按词排。
const windowBonus = 100

// rank 用 BM25 给 docs 打分，再乘上 weight。词的稀有程度只在 docs 里算。
// useWindow 为真且查询问的是过去哪几天时，那几天里的记忆就算一个词都没对上也留下，并排在最前。
func rank(docs []Item, q string, now time.Time, useWindow bool) []ranked {
	terms := queryTerms(q)
	win, hasWin := window{}, false
	if useWindow {
		win, hasWin = queryWindow(q, now)
	}
	if (len(terms) == 0 && !hasWin) || len(docs) == 0 {
		return nil
	}
	if len(terms) > maxTerms {
		terms = terms[:maxTerms]
	}
	hays := make([]string, len(docs))
	total := 0
	for i, it := range docs {
		hays[i] = strings.ToLower(it.Text + "\n" + it.Project)
		total += utf8.RuneCountInString(hays[i])
	}
	avg := math.Max(float64(total)/float64(len(docs)), 1)
	n := float64(len(docs))
	idf := make(map[string]float64, len(terms))
	for _, term := range terms {
		df := 0
		for _, hay := range hays {
			if strings.Contains(hay, term) {
				df++
			}
		}
		idf[term] = math.Log(1 + (n-float64(df)+0.5)/(float64(df)+0.5))
	}
	const k1, b = 1.2, 0.75
	var out []ranked
	for i, it := range docs {
		hay := hays[i]
		length := float64(utf8.RuneCountInString(hay))
		score := 0.0
		matched := 0
		for _, term := range terms {
			tf := float64(strings.Count(hay, term))
			if tf == 0 {
				continue
			}
			score += idf[term] * tf * (k1 + 1) / (tf + k1*(1-b+b*length/avg))
			if !stopTerm(term) {
				matched++
			}
		}
		inWindow := hasWin && win.holds(it)
		if score == 0 && !inWindow {
			continue
		}
		phrase := len(terms) > 1 && strings.Contains(hay, q)
		if phrase {
			for _, term := range terms {
				score += idf[term]
			}
		}
		score *= weight(it, now)
		if inWindow {
			score += windowBonus
			matched = max(matched, 2)
		}
		out = append(out, ranked{item: it, score: score, matched: matched, phrase: phrase})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return updatedOf(out[i].item).After(updatedOf(out[j].item))
	})
	return out
}

// weight 是打分之外的倍数：经历按年纪衰减，九十天减半，最低留一半；过期的安排打六折；说过多次的每多一次加一成，最多五成。
func weight(it Item, now time.Time) float64 {
	w := 1.0
	if it.Kind == KindEpisodic {
		if age := now.Sub(updatedOf(it)).Hours() / 24; age > 0 {
			w *= 0.5 + 0.5*math.Pow(0.5, age/90)
		}
	}
	if Expired(it, now) {
		w *= 0.6
	}
	if extra := seenOf(it) - 1; extra > 0 {
		w *= 1 + 0.1*float64(min(extra, 5))
	}
	return w
}

func clipItems(items []Item, limit int) []Item {
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return append([]Item(nil), items...)
}

// queryTerms 把查询切成词。拉丁字母和数字按连续一段算，至少两个字符；中文、日文、韩文按相邻两个字，单独一个字时就用这一个字。
func queryTerms(q string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(term string) {
		if term != "" && !seen[term] {
			seen[term] = true
			out = append(out, term)
		}
	}
	var word, han []rune
	flushWord := func() {
		if len(word) >= 2 || (len(word) == 1 && unicode.IsDigit(word[0])) {
			add(string(word))
		}
		word = word[:0]
	}
	flushHan := func() {
		if len(han) == 1 {
			add(string(han))
		}
		for i := 0; i+1 < len(han); i++ {
			add(string(han[i : i+2]))
		}
		han = han[:0]
	}
	for _, r := range q {
		switch {
		case cjk(r):
			flushWord()
			han = append(han, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			flushHan()
			word = append(word, unicode.ToLower(r))
		default:
			flushWord()
			flushHan()
		}
	}
	flushWord()
	flushHan()
	return out
}

func cjk(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r)
}

// stopTerms 是几乎每句话都有、对上了也说明不了相关的词。它们照样参与打分，只是不算进 matched。
var stopTerms = map[string]bool{}

func init() {
	for _, term := range strings.Fields(`一下 一个 一些 一点 一样 一次 这个 那个 这样 那样 这次 我们 你们 他们 什么 怎么 可以 就是 没有 不是
		帮我 给我 我想 想要 需要 现在 然后 还是 已经 如果 因为 所以 但是 自己 时候 的话 我的 你的 看看 看一 我看 知道 觉得 感觉 东西 问题
		有点 继续 好的 谢谢 麻烦 请你 你帮 帮忙 下这 这里 那里 一起 应该 能不 不能 能否
		the and for you are was were this that with from have has had not but can could would should will please help what when
		where which how why who about into your our their them they its let make get got just like need want also then than there here been some any all`) {
		stopTerms[term] = true
	}
}

// stopTerm 在 term 是常用词，或是单独一个汉字时为真。
func stopTerm(term string) bool {
	return stopTerms[term] || utf8.RuneCountInString(term) == 1
}

// stopRunes 是比较两句话像不像时不算的单字。
const stopRunes = "的了是我你他她它在和与也就都很这那个一吗呢吧啊把被给让对着过地得"

// simTerms 是比较两句话像不像时用的词：查询里的词，再加上中文的单字。常见的虚字不算。
func simTerms(text string) map[string]bool {
	lower := strings.ToLower(text)
	out := map[string]bool{}
	for _, term := range queryTerms(lower) {
		out[term] = true
	}
	for _, r := range lower {
		if cjk(r) && !strings.ContainsRune(stopRunes, r) {
			out[string(r)] = true
		}
	}
	return out
}

// overlap 返回两组词共有的个数，以及共有的部分各占两边的比例。
func overlap(a, b map[string]bool) (int, float64, float64) {
	shared := 0
	for term := range a {
		if b[term] {
			shared++
		}
	}
	if len(a) == 0 || len(b) == 0 {
		return shared, 0, 0
	}
	return shared, float64(shared) / float64(len(a)), float64(shared) / float64(len(b))
}
