// 锁住大窗口输出下限、小窗口保持目录值、未知目录保持 0、以及 32000 上限。

package app

import "testing"

// TestCapOutputTokensLargeWindowFloor 锁住 100 万窗口把 8192 抬到 chatLargeWindowOutput，12.8 万窗口保持 8192，目录 0 保持 0，64000 截到 chatMaxOutputTokens。
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
