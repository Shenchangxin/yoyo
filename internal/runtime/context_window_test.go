// 锁住空模型名、未知名字、目录覆盖和若干家族的窗口数字。

package runtime

import "testing"

// TestModelContextWindowUnknownDefaults300k 确认空模型返回 0，对不上的名字用 UnknownModelWindow，点名的家族用各自的窗口。
func TestModelContextWindowUnknownDefaults300k(t *testing.T) {
	if got := ModelContextWindow(""); got != 0 {
		t.Fatalf("empty: %d", got)
	}
	if got := ModelContextWindow("local-flash-v4"); got != UnknownModelWindow {
		t.Fatalf("unlisted: %d", got)
	}
	if got := ModelContextWindow("gpt-4o"); got != 128_000 {
		t.Fatalf("catalog gpt-4o: %d", got)
	}
	if got := ModelContextWindow("anthropic/claude-sonnet-4-5"); got != 200_000 {
		t.Fatalf("openrouter claude: %d", got)
	}
	if got := ModelContextWindow("claude-sonnet-4-6"); got != 1_000_000 {
		t.Fatalf("claude 4.6: %d", got)
	}
	if got := ModelContextWindow("gpt-5.5"); got != 1_000_000 {
		t.Fatalf("gpt-5.5: %d", got)
	}
	if got := ModelContextWindow("gpt-5"); got != 200_000 {
		t.Fatalf("gpt-5: %d", got)
	}
}

// TestModelContextWindowDeepSeekV4Is1M 确认带 deepseek 的名字，包括自定义前缀，窗口是 100 万。接入点参数不改变结果。
func TestModelContextWindowDeepSeekV4Is1M(t *testing.T) {
	for _, id := range []string{
		"deepseek-v4-flash",
		"deepseek-v4.1-flash",
		"AIPC-deepseek-v4.1-flash",
		"custom/AIPC-deepseek-v4.1-flash",
		"deepseek-chat",
		"deepseek-reasoner",
	} {
		if got := ModelContextWindow(id); got != 1_000_000 {
			t.Fatalf("%s: %d", id, got)
		}
	}
	if got := ModelContextWindowFor("custom", "AIPC-deepseek-v4.1-flash"); got != 1_000_000 {
		t.Fatalf("custom provider: %d", got)
	}
}

// TestCatalogWindowOverridesHeuristic 确认 CatalogWindow 大于 0 时盖过启发式，返回 0 的名字仍用 UnknownModelWindow。
func TestCatalogWindowOverridesHeuristic(t *testing.T) {
	CatalogWindow = func(model string) int {
		if model == "gpt-4o" {
			return 999_000
		}
		return 0
	}
	t.Cleanup(func() { CatalogWindow = nil })
	if got := ModelContextWindow("gpt-4o"); got != 999_000 {
		t.Fatalf("override: %d", got)
	}
	if got := ModelContextWindow("local-flash-v4"); got != UnknownModelWindow {
		t.Fatalf("unlisted still heuristic: %d", got)
	}
}
