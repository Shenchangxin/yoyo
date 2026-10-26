// 模型窗口和整形预算。聊天按目录或启发式得到窗口后再扣预留；不填 ModelWindow 时预算留在 CompactionTokens。
// 只识别溢出错误文本，不数重试次数。调用方是投影、检查点和计量。

package runtime

import "strings"

const (
	// defaultWindowBuffer 是未设置 ContextBuffer 时，从模型窗口扣掉的 token。
	defaultWindowBuffer = 20_000
	// minOutputReserve 是按窗口十分之一估算输出预留时的下限。
	minOutputReserve = 8_000
	// maxOutputReserve 是按窗口十分之一估算输出预留时的上限。
	maxOutputReserve = 32_000
	// minShapeBudget 是按窗口算出的整形预算下限。
	minShapeBudget = 8_000
	// UnknownModelWindow 是模型名对不上内置家族、且目录也没有窗口时，聊天整形用的窗口。空模型名返回 0，不用这个值。
	UnknownModelWindow = 300_000
)

// ModelContextWindow 返回聊天整形用的目录窗口。自定义或本地 id 仍按模型名匹配，例如带 deepseek 的名字得到 100 万。
// 不填 ModelWindow 时，effectiveBudget 继续用 CompactionTokens。
func ModelContextWindow(model string) int {
	return ModelContextWindowFor("", model)
}

// ModelContextWindowFor 从模型 id 推断窗口。第一个参数是接入点，不参与推断。
// CatalogWindow 若返回大于 0 的值则优先，先查完整模型名，再查最后一段路径。都没有则走内置家族，仍没有则用 UnknownModelWindow。空模型名返回 0。
func ModelContextWindowFor(_, model string) int {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return 0
	}
	id := lastPathSegment(m)
	if CatalogWindow != nil {
		if w := CatalogWindow(model); w > 0 {
			return w
		}
		if w := CatalogWindow(id); w > 0 {
			return w
		}
	}
	if w := catalogishWindow(id); w > 0 {
		return w
	}
	return UnknownModelWindow
}

// CatalogWindow 是实时目录查询。返回 0 时 ModelContextWindowFor 继续用内置家族。
var CatalogWindow func(model string) int

// EffectiveModelWindow 在目录窗口之上应用调用方的覆盖值。覆盖值大于 0 时直接用它，否则用 ModelContextWindow。两者都没有时，预算不按窗口扣减。
func EffectiveModelWindow(model string, override int) int {
	if override > 0 {
		return override
	}
	return ModelContextWindow(model)
}

// lastPathSegment 取模型名最后一个斜杠后面的一段。没有斜杠则返回原文。
func lastPathSegment(m string) string {
	if i := strings.LastIndex(m, "/"); i >= 0 && i+1 < len(m) {
		return m[i+1:]
	}
	return m
}

// catalogishWindow 按内置家族给窗口。更长的型号写在更短的前面，避免短名字先命中。对不上返回 0。
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

// claudeMillion 在名字含 claude，并且含 4-6、4.6、sonnet-5、opus-5、opus-4-6、sonnet-4-6 或 1m 时返回真。
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

// nameHas 判断 token 是否作为独立片段出现。点号算边界，所以短名字要靠 catalogishWindow 的顺序让路。
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

// tokenBounded 要求 token 两侧都不是字母或数字。
func tokenBounded(s string, start, end int) bool {
	if start > 0 && isModelIdent(s[start-1]) {
		return false
	}
	if end < len(s) && isModelIdent(s[end]) {
		return false
	}
	return true
}

// isModelIdent 只把小写字母和数字当成模型名的内部字符。
func isModelIdent(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

// effectiveBudget 在 ModelWindow 大于 0 时用窗口减去输出预留和 ContextBuffer 里较大的那个，并不得低于 minShapeBudget。
// 预留缺省是窗口的十分之一，夹在 minOutputReserve 和 maxOutputReserve 之间。ModelWindow 为 0 时用 CompactionTokens，再没有则用 24000。
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

// IsContextOverflow 看错误文本里是否有上下文超长的短语。单独的 max_tokens 不算。
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
