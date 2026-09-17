package runtime

import "unicode"

const defaultToolResultRunes = 16_000

// estimateTokens approximates cl100k: CJK is 1 token/rune, latin words
// split on punctuation, Harbor still uses this same function so eval is stable.
func CountTokens(s string) int { return estimateTokens(s) }

func TokenizerName(model string) string {
	_ = model
	return "approx-cl100k"
}

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

func messageTokens(m Message) int {
	n := estimateTokens(m.Content) + 8
	for _, tc := range m.ToolCalls {
		n += estimateTokens(tc.Name) + estimateTokens(tc.Arguments) + 8
	}
	return n
}

func messagesTokens(msgs []Message) int {
	n := 0
	for _, m := range msgs {
		n += messageTokens(m)
	}
	return n
}

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
