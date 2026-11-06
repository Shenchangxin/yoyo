// 本文件拥有检索用的切词和相关度打分，给事实分段、材料和记忆排序。不调用模型。
// 中日韩文字之间没有空格，按相邻两个字切成二元词，再补上不在停用表里的单字；字母和数字连成一个词。

package idol

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

const (
	// bigramWeight 是二元词和字母词在打分里的权重。
	bigramWeight = 1.0
	// unigramWeight 是单个汉字或假名的权重。单字太常见，只作补充。
	unigramWeight = 0.35
)

// stopUnigrams 是不参与打分的单字：语气词、代词和最常见的虚词。
var stopUnigrams = func() map[string]bool {
	m := map[string]bool{}
	for _, r := range "的了是我你他她它们在有和就不也都很吗呢吧啊呀哦嗯这那个么什要会说想把被给对到还没过着之与及或而且の" +
		"はがをにでとへもやかなねよだすたいうるれ" {
		m[string(r)] = true
	}
	return m
}()

// term 是切出来的一个词和它的权重。
type term struct {
	text   string
	weight float64
}

// terms 把文本切成检索用的词，同一个词只留一次。
// s 里连续的字母和数字成为一个词（转小写，至少两个字符）；连续的中日韩文字按相邻两字切成二元词，并补上不在停用表里的单字。
func terms(s string) []term {
	seen := map[string]bool{}
	var out []term
	add := func(t string, w float64) {
		if t == "" || seen[t] {
			return
		}
		seen[t] = true
		out = append(out, term{text: t, weight: w})
	}
	var word, han []rune
	flushWord := func() {
		if len(word) >= 2 {
			add(strings.ToLower(string(word)), bigramWeight)
		}
		word = word[:0]
	}
	flushHan := func() {
		for i := range han {
			if i+1 < len(han) {
				add(string(han[i:i+2]), bigramWeight)
			}
			if one := string(han[i]); !stopUnigrams[one] {
				add(one, unigramWeight)
			}
		}
		han = han[:0]
	}
	for _, r := range s {
		switch {
		case isCJK(r):
			flushWord()
			han = append(han, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			flushHan()
			word = append(word, r)
		default:
			flushWord()
			flushHan()
		}
	}
	flushWord()
	flushHan()
	return out
}

// isCJK 判断 r 是不是汉字、平假名、片假名或韩文。
func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r)
}

// hit 是一篇文档的下标和它对查询的得分。
type hit struct {
	index int
	score float64
}

// rank 按 query 给 docs 打分，只返回得分大于 0 的，分高的在前，同分时下标小的在前。
// 词的稀有度按这批文档算：只出现在少数文档里的词更有分量。长文档按长度略微压低，免得靠篇幅凑分。
func rank(query string, docs []string) []hit {
	q := terms(query)
	if len(q) == 0 || len(docs) == 0 {
		return nil
	}
	sets := make([]map[string]bool, len(docs))
	df := map[string]int{}
	total := 0
	for i, d := range docs {
		set := map[string]bool{}
		for _, t := range terms(d) {
			set[t.text] = true
		}
		sets[i] = set
		total += len(set)
		for t := range set {
			df[t]++
		}
	}
	avg := math.Max(1, float64(total)/float64(len(docs)))
	n := float64(len(docs))
	const k1, b = 1.2, 0.5
	var out []hit
	for i, set := range sets {
		norm := k1 * (1 - b + b*float64(len(set))/avg)
		score := 0.0
		for _, t := range q {
			if !set[t.text] {
				continue
			}
			idf := math.Log(1 + (n-float64(df[t.text])+0.5)/(float64(df[t.text])+0.5))
			score += t.weight * idf * (k1 + 1) / (1 + norm)
		}
		if score > 0 {
			out = append(out, hit{index: i, score: score})
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].score > out[b].score })
	return out
}

// clip 把 s 截到 n 个字符以内，截断时在末尾加省略号。
func clip(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
