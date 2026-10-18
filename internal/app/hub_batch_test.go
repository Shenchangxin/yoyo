package app

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

func publishN(h *Hub, sid string, n int) {
	for i := 0; i < n; i++ {
		h.Publish(trace.Event{SessionID: sid, Type: trace.TypeUser})
	}
}

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

func TestBatchFlagsGapWhenRingRolledPastCursor(t *testing.T) {
	h := NewHub()
	// Ring keeps seq 11..4106. Only seq 4106 belongs to "a", so the boundary
	// check is not confused with the per-session cap.
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

func TestBatchFlagsCursorAheadOfHead(t *testing.T) {
	h := NewHub()
	publishN(h, "a", 2)
	// A cursor past the head means the seq space restarted under the caller.
	if b := h.Batch(999, "a"); !b.Gap || len(b.Events) != 0 {
		t.Fatalf("cursor ahead of head: gap=%v n=%d", b.Gap, len(b.Events))
	}
}

func TestBatchNilHubIsEmptyAndComplete(t *testing.T) {
	var h *Hub
	b := h.Batch(0, "a")
	if b.Gap || b.Head != 0 || b.Events == nil || len(b.Events) != 0 {
		t.Fatalf("nil hub: %+v", b)
	}
}

func TestSinceKeepsLegacyShape(t *testing.T) {
	h := NewHub()
	publishN(h, "a", 3)
	if got := h.Since(1, "a"); len(got) != 2 {
		t.Fatalf("Since(1) returned %d events", len(got))
	}
}
