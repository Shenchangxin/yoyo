package runtime

const defaultToolResultRunes = 16_000

// estimateTokens is a tokenizer-free budget heuristic (~4 bytes/token).
func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	n := len(s)
	return (n + 3) / 4
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

// capText keeps head and tail of oversized tool output so the model still
// sees errors and structure without blowing the context window.
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
