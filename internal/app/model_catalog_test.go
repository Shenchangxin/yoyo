package app

import "testing"

func TestCapOutputTokensLargeWindowFloor(t *testing.T) {
	if got := capOutputTokens(8192, 1_000_000); got != chatLargeWindowOutput {
		t.Fatalf("1M window 8k catalog: %d", got)
	}
	if got := capOutputTokens(8192, 128_000); got != 8192 {
		t.Fatalf("small window stays catalogued: %d", got)
	}
	if got := capOutputTokens(0, 1_000_000); got != 0 {
		t.Fatalf("unknown catalog must keep provider default: %d", got)
	}
	if got := capOutputTokens(64_000, 1_000_000); got != chatMaxOutputTokens {
		t.Fatalf("ceiling: %d", got)
	}
}
