package runtime

import "testing"

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
}

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
