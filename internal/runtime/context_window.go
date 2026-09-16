package runtime

import "strings"

const (
	defaultOutputReserve = 16_000
	defaultWindowBuffer  = 13_000
	minShapeBudget       = 8_000
)

// ModelContextWindow is the best-effort catalog window for chat shaping.
// Harbor leaves ModelWindow unset so LoopPreset.CompactionTokens stays the eval budget.
func ModelContextWindow(model string) int {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case m == "":
		return 0
	case strings.Contains(m, "gpt-4.1"):
		return 1_047_576
	case strings.Contains(m, "o3"), strings.Contains(m, "o4"), strings.Contains(m, "gpt-5"):
		return 200_000
	case strings.Contains(m, "gpt-4o"), strings.Contains(m, "gpt-4-turbo"), strings.Contains(m, "gpt-4.1-mini"):
		return 128_000
	case strings.Contains(m, "claude"):
		return 200_000
	case strings.Contains(m, "gemini-2"), strings.Contains(m, "gemini-1.5"):
		return 1_000_000
	case strings.Contains(m, "gemini"):
		return 128_000
	case strings.Contains(m, "kimi"), strings.Contains(m, "moonshot"):
		return 128_000
	case strings.Contains(m, "deepseek"):
		return 128_000
	case strings.Contains(m, "qwen"):
		return 128_000
	default:
		if strings.Contains(m, "mini") || strings.Contains(m, "4o") {
			return 128_000
		}
		return 128_000
	}
}

func effectiveBudget(opts ShapeOpts) int {
	if opts.ModelWindow > 0 {
		reserve := opts.OutputReserve
		if reserve <= 0 {
			reserve = defaultOutputReserve
		}
		buf := defaultWindowBuffer
		b := opts.ModelWindow - reserve - buf
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
