// 锁住桌面投影在预算一半以下保留只读原文，压力上来后按 8 条一批换成回执，批与批之间已处理前缀的字节不变。

package runtime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// readHistory 构造 n 次 read_file，每次正文大约 1.8 KB，用来模仿桌面回合里反复读长文件。
func readHistory(n int) []Message {
	body := numberLines(strings.Repeat("const x = 1;\n", 90), 0, 0)
	var msgs []Message
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("c%d", i)
		msgs = append(msgs,
			Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: id, Name: "read_file", Arguments: fmt.Sprintf(`{"path":"src/f%d.js"}`, i)}}},
			Message{Role: RoleTool, Name: "read_file", ToolCallID: id, Content: body},
		)
	}
	return msgs
}

// TestReadsStayVerbatimBelowPressure 确认 100 万窗口下 30 次读取不会出现 reads 层，也不会写出回执。
func TestReadsStayVerbatimBelowPressure(t *testing.T) {
	msgs := readHistory(30)
	out, rep := projectChat(msgs, ShapeOpts{ModelWindow: 1_000_000})
	for _, layer := range rep.Layers {
		if layer == "reads" {
			t.Fatalf("reads cleared with no pressure: layers=%v", rep.Layers)
		}
	}
	for _, m := range out {
		if strings.HasPrefix(m.Content, "[cleared") {
			t.Fatal("receipt written below the read-clearing pressure")
		}
	}
}

// TestReadsClearUnderPressure 确认低压预算下出现 reads 层，30 次读取保留 12 条、按 8 条一批，得到 16 条回执。
func TestReadsClearUnderPressure(t *testing.T) {
	msgs := readHistory(30)
	// 预算 4000，30 次读取大约 1.3 万 token，超过 chatReadClearAt 的一半。
	out, rep := projectChat(msgs, ShapeOpts{Loop: artifact.LoopPreset{CompactionTokens: 4000}})
	var found bool
	for _, layer := range rep.Layers {
		found = found || layer == "reads"
	}
	if !found {
		t.Fatalf("expected reads layer under pressure: %v", rep.Layers)
	}
	var receipts int
	for _, m := range out {
		if strings.HasPrefix(m.Content, "[cleared read_file") {
			receipts++
		}
	}
	// 30 次读取保留 12 条，过期 18 条，按 8 取整是 16。
	if want := 16; receipts != want {
		t.Fatalf("receipts %d, want %d (30 reads, keep 12, batch 8)", receipts, want)
	}
}

// TestClearStaleReadsClearsOnlyWholeBatches 确认清除条数始终是 batch 的整数倍，不足一批时为 0。
func TestClearStaleReadsClearsOnlyWholeBatches(t *testing.T) {
	for n := chatKeepReads; n <= chatKeepReads+3*chatReadClearBatch; n++ {
		msgs := readHistory(n)
		got := clearStaleReads(msgs, chatKeepReads, chatReadClearBatch)
		want := 0
		if stale := n - chatKeepReads; stale >= chatReadClearBatch {
			want = stale / chatReadClearBatch * chatReadClearBatch
		}
		if got != want {
			t.Fatalf("n=%d cleared %d, want %d", n, got, want)
		}
	}
}

// TestClearStaleReadsKeepsPromptBytesBetweenBatches 确认从 20 次读到 27 次不跨过新的一批，已有消息字节保持不变。
func TestClearStaleReadsKeepsPromptBytesBetweenBatches(t *testing.T) {
	before := readHistory(20)
	clearStaleReads(before, chatKeepReads, chatReadClearBatch)
	after := readHistory(27)
	clearStaleReads(after, chatKeepReads, chatReadClearBatch)
	for i := range before {
		if before[i].Content != after[i].Content {
			t.Fatalf("message %d changed between batch boundaries", i)
		}
	}
}

// TestReadReceiptNamesLineSpan 确认回执带上 path 和所示行号范围。
func TestReadReceiptNamesLineSpan(t *testing.T) {
	body := numberLines(strings.Repeat("x\n", 80), 1, 40)
	msgs := []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "read_file", Arguments: `{"path":"app.js","offset":1,"limit":40}`}}},
		{Role: RoleTool, Name: "read_file", ToolCallID: "c1", Content: body},
	}
	got := readReceipt(msgs, 1)
	if !strings.Contains(got, "path=app.js lines=1-40 ") {
		t.Fatalf("receipt must name the shown span: %s", got)
	}
}

// TestNumberedSpanOnlyReadsNumberedRows 确认没有行号前缀时返回空串，有前缀时返回首尾行号。
func TestNumberedSpanOnlyReadsNumberedRows(t *testing.T) {
	if got := numberedSpan("a|b\n12:x\n"); got != "" {
		t.Fatalf("unnumbered rows gave span %q", got)
	}
	if got := numberedSpan("     7|x\n     9|y\n"); got != "lines=7-9" {
		t.Fatalf("span %q", got)
	}
}
