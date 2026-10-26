// 序号环的连续尾部、空洞、分页和空 Hub。

package app

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// publishN 连续发布 n 条用户事件。
func publishN(h *Hub, sid string, n int) {
	for i := 0; i < n; i++ {
		h.Publish(trace.Event{SessionID: sid, Type: trace.TypeUser})
	}
}

// TestBatchReturnsContiguousTailWithoutGap 确认游标还在环内时按会话返回后续事件，且不标空洞。
func TestBatchReturnsContiguousTailWithoutGap(t *testing.T) {
	h := NewHub()
	publishN(h, "a", 3) // seq 1..3
	publishN(h, "b", 2) // seq 4..5

	b := h.Batch(0, "a")
	if b.Gap || b.Head != 5 || len(b.Events) != 3 {
		t.Fatalf("full tail for a: gap=%v head=%d n=%d", b.Gap, b.Head, len(b.Events))
	}
	b = h.Batch(2, "a")
	if b.Gap || len(b.Events) != 1 || b.Events[0].Seq != 3 {
		t.Fatalf("after=2 for a: gap=%v events=%+v", b.Gap, b.Events)
	}
	b = h.Batch(5, "")
	if b.Gap || len(b.Events) != 0 || b.Head != 5 {
		t.Fatalf("caught up: gap=%v n=%d head=%d", b.Gap, len(b.Events), b.Head)
	}
}

// TestBatchFlagsGapWhenRingRolledPastCursor 确认游标落在环已覆盖的序号上时标成空洞，贴着最老序号则不是。
func TestBatchFlagsGapWhenRingRolledPastCursor(t *testing.T) {
	h := NewHub()
	// 环滚动之后只剩最后一条属于 a，用来把空洞和按会话过滤区分开。
	publishN(h, "b", liveLogCap+9)
	publishN(h, "a", 1)

	if b := h.Batch(0, "a"); !b.Gap {
		t.Fatal("cursor at 0 must be a gap once the ring rolled over")
	}
	oldest := int64(11)
	if b := h.Batch(oldest-1, "a"); b.Gap || len(b.Events) != 1 {
		t.Fatalf("cursor exactly at oldest-1 is complete: gap=%v n=%d", b.Gap, len(b.Events))
	}
	if b := h.Batch(oldest-2, "a"); !b.Gap {
		t.Fatal("cursor one before that must be a gap")
	}
}

// TestBatchPagesOldestAndSetsMore 确认超出一页时保留最老的一页并标 More，下一拉拿到余下的。
func TestBatchPagesOldestAndSetsMore(t *testing.T) {
	h := NewHub()
	publishN(h, "a", liveSinceCap+5)

	b := h.Batch(0, "a")
	if b.Gap {
		t.Fatal("a full ring must page, not report a hole")
	}
	if !b.More {
		t.Fatal("truncated page must set More")
	}
	if len(b.Events) != liveSinceCap {
		t.Fatalf("kept %d events, want %d", len(b.Events), liveSinceCap)
	}
	if b.Events[0].Seq != 1 {
		t.Fatalf("page must keep the oldest first; seq=%d", b.Events[0].Seq)
	}
	if last := b.Events[len(b.Events)-1].Seq; last != int64(liveSinceCap) {
		t.Fatalf("first page last=%d want %d", last, liveSinceCap)
	}
	b2 := h.Batch(int64(liveSinceCap), "a")
	if b2.Gap || b2.More || len(b2.Events) != 5 {
		t.Fatalf("second page: gap=%v more=%v n=%d", b2.Gap, b2.More, len(b2.Events))
	}
}

// TestBatchFlagsCursorAheadOfHead 确认游标超过 head 时标成空洞且没有事件。
func TestBatchFlagsCursorAheadOfHead(t *testing.T) {
	h := NewHub()
	publishN(h, "a", 2)
	// 游标超过 head，表示调用方看到的序号空间已经重新开始。
	if b := h.Batch(999, "a"); !b.Gap || len(b.Events) != 0 {
		t.Fatalf("cursor ahead of head: gap=%v n=%d", b.Gap, len(b.Events))
	}
}

// TestBatchNilHubIsEmptyAndComplete 确认 nil Hub 返回空页，且不标空洞。
func TestBatchNilHubIsEmptyAndComplete(t *testing.T) {
	var h *Hub
	b := h.Batch(0, "a")
	if b.Gap || b.Head != 0 || b.Events == nil || len(b.Events) != 0 {
		t.Fatalf("nil hub: %+v", b)
	}
}

// TestSinceKeepsLegacyShape 确认 Since 只返回序号大于游标的事件。
func TestSinceKeepsLegacyShape(t *testing.T) {
	h := NewHub()
	publishN(h, "a", 3)
	if got := h.Since(1, "a"); len(got) != 2 {
		t.Fatalf("Since(1) returned %d events", len(got))
	}
}
