package runtime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// readHistory is n read_file round trips with a body of about 1.8 KB each,
// the shape of a desktop turn that reads a long file many times.
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

func TestReadsClearUnderPressure(t *testing.T) {
	msgs := readHistory(30)
	// Budget 4000 tokens puts the 30 reads (about 13k tokens) well past 50%.
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
	// 30 reads, keep 12: 18 are stale, which is two whole batches of 8.
	if want := 16; receipts != want {
		t.Fatalf("receipts %d, want %d (30 reads, keep 12, batch 8)", receipts, want)
	}
}

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

func TestClearStaleReadsKeepsPromptBytesBetweenBatches(t *testing.T) {
	// Going from 20 to 27 reads adds no new clearing: the earlier prompt must
	// stay byte-identical so the provider's prompt cache keeps hitting.
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

func TestNumberedSpanOnlyReadsNumberedRows(t *testing.T) {
	if got := numberedSpan("a|b\n12:x\n"); got != "" {
		t.Fatalf("unnumbered rows gave span %q", got)
	}
	if got := numberedSpan("     7|x\n     9|y\n"); got != "lines=7-9" {
		t.Fatalf("span %q", got)
	}
}
