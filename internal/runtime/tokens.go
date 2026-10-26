// 近似 cl100k 的 token 计数，以及按 rune 截断工具结果。评测和聊天调用同一个 CountTokens。
// 不访问模型，也不按模型名换词表。

package runtime

import "unicode"

// defaultToolResultRunes 是 16000。capText 在 maxRunes 小于等于 0 时用它。
const defaultToolResultRunes = 16_000

// CountTokens 转给 estimateTokens。模型名不参与计算。
func CountTokens(s string) int { return estimateTokens(s) }

// TokenizerName 忽略 model，固定返回 approx-cl100k。
func TokenizerName(model string) string {
	_ = model
	return "approx-cl100k"
}

// estimateTokens 按近似 cl100k 计数。空串返回 0。只含空白（rune 小于等于 32）时返回 1。
// 汉字、谚文、平假名、片假名各计 1。连续字母或数字按 (长度+3)/4 计。标点和其他字符各计 1，并先结算未结束的词。
func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	n := 0
	word := 0
	flush := func() {
		if word == 0 {
			return
		}
		n += (word + 3) / 4
		if n == 0 {
			n = 1
		}
		word = 0
	}
	for _, r := range s {
		switch {
		case r <= 32:
			flush()
		case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hangul, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r):
			flush()
			n++
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			word++
		default:
			flush()
			n++
		}
	}
	flush()
	if n <= 0 {
		return 1
	}
	return n
}

// messageTokens 是正文 token 再加 8，每条工具调用再加名称、参数的 token 和 8。
func messageTokens(m Message) int {
	n := estimateTokens(m.Content) + 8
	for _, tc := range m.ToolCalls {
		n += estimateTokens(tc.Name) + estimateTokens(tc.Arguments) + 8
	}
	return n
}

// messagesTokens 把每条消息的 messageTokens 相加。
func messagesTokens(msgs []Message) int {
	n := 0
	for _, m := range msgs {
		n += messageTokens(m)
	}
	return n
}

// capText 按 rune 截断。maxRunes 小于等于 0 时用 defaultToolResultRunes（16000）。长度不超过上限时返回原文和 false。
// 超过时头尾各留 keep 个 rune。keep 是 maxRunes 的一半，小于 256 时改成 256。中间标出丢掉的 rune 数。
func capText(s string, maxRunes int) (out string, truncated bool) {
	if maxRunes <= 0 {
		maxRunes = defaultToolResultRunes
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s, false
	}
	keep := maxRunes / 2
	if keep < 256 {
		keep = 256
	}
	head := string(r[:keep])
	tail := string(r[len(r)-keep:])
	return head + "\n...[truncated " + itoa(len(r)-keep*2) + " runes]...\n" + tail, true
}

// itoa 把非负整数写成十进制。n 小于 0 时返回 0。
func itoa(n int) string {
	if n < 0 {
		return "0"
	}
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
