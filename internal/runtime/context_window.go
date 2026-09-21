package runtime

import "strings"

const (
	defaultWindowBuffer = 20_000
	minOutputReserve    = 8_000
	maxOutputReserve    = 32_000
	minShapeBudget      = 8_000
	// UnknownModelWindow is the chat shaping default when the model id
	// cannot be matched to a models.dev family.
	UnknownModelWindow = 300_000
)

// ModelContextWindow is the best-effort catalog window for chat shaping.
// Custom/local ids still match on the model name (AIPC-deepseek-v4.1-flash → 1M).
// Harbor leaves ModelWindow unset so LoopPreset.CompactionTokens stays the eval budget.
func ModelContextWindow(model string) int {
	return ModelContextWindowFor("", model)
}

// ModelContextWindowFor infers the window from the model id. Provider is an
// endpoint, not a catalog — a custom OpenAI-compatible host does not change the
// model's native window.
func ModelContextWindowFor(_, model string) int {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return 0
	}
	id := lastPathSegment(m)
	if w := catalogishWindow(id); w > 0 {
		return w
	}
	return UnknownModelWindow
}

// EffectiveModelWindow applies an operator override (config.context_window)
// on top of the catalog. Harbor leaves both unset.
func EffectiveModelWindow(model string, override int) int {
	if override > 0 {
		return override
	}
	return ModelContextWindow(model)
}

func lastPathSegment(m string) string {
	if i := strings.LastIndex(m, "/"); i >= 0 && i+1 < len(m) {
		return m[i+1:]
	}
	return m
}

func catalogishWindow(id string) int {
	switch {
	case nameHas(id, "gpt-4.1"):
		return 1_047_576
	case nameHas(id, "gpt-5.5"), nameHas(id, "gpt-5.6"), nameHas(id, "gpt-5.4"), nameHas(id, "gpt-5.2"):
		return 1_000_000
	case nameHas(id, "o3"), nameHas(id, "o4"), nameHas(id, "gpt-5"):
		return 200_000
	case nameHas(id, "gpt-4o"), nameHas(id, "gpt-4-turbo"):
		return 128_000
	case claudeMillion(id):
		return 1_000_000
	case nameHas(id, "claude"):
		return 200_000
	case nameHas(id, "gemini-2"), nameHas(id, "gemini-1.5"), nameHas(id, "gemini-3"):
		return 1_000_000
	case nameHas(id, "gemini"):
		return 128_000
	case nameHas(id, "kimi"), nameHas(id, "moonshot"):
		return 128_000
	case nameHas(id, "deepseek"):
		return 1_000_000
	case nameHas(id, "qwen"), nameHas(id, "qwq"):
		return 128_000
	default:
		return 0
	}
}

func claudeMillion(id string) bool {
	if !strings.Contains(id, "claude") {
		return false
	}
	for _, m := range []string{"4-6", "4.6", "sonnet-5", "opus-5", "opus-4-6", "sonnet-4-6", "1m"} {
		if strings.Contains(id, m) {
			return true
		}
	}
	return false
}

func nameHas(id, token string) bool {
	if id == token {
		return true
	}
	for i := 0; ; {
		j := strings.Index(id[i:], token)
		if j < 0 {
			return false
		}
		j += i
		if tokenBounded(id, j, j+len(token)) {
			return true
		}
		i = j + 1
	}
}

func tokenBounded(s string, start, end int) bool {
	if start > 0 && isModelIdent(s[start-1]) {
		return false
	}
	if end < len(s) && isModelIdent(s[end]) {
		return false
	}
	return true
}

func isModelIdent(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

func effectiveBudget(opts ShapeOpts) int {
	if opts.ModelWindow > 0 {
		reserve := opts.OutputReserve
		if reserve <= 0 {
			reserve = opts.Loop.OutputReserve
		}
		if reserve <= 0 {
			reserve = opts.ModelWindow / 10
			if reserve < minOutputReserve {
				reserve = minOutputReserve
			}
			if reserve > maxOutputReserve {
				reserve = maxOutputReserve
			}
		}
		buf := opts.Loop.ContextBuffer
		if buf <= 0 {
			buf = defaultWindowBuffer
		}
		b := opts.ModelWindow - max(reserve, buf)
		if b < minShapeBudget {
			b = minShapeBudget
		}
		return b
	}
	if opts.Loop.CompactionTokens > 0 {
		return opts.Loop.CompactionTokens
	}
	return 24_000
}

func IsContextOverflow(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	for _, n := range []string{
		"context_length_exceeded", "context_length", "context window",
		"too many tokens", "maximum context", "prompt is too long",
		"request too large", "string too long", "reduce the length of the messages",
		"max context length", "maximum context length",
		"context overflow",
	} {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}
