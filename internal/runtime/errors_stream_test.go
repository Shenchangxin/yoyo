package runtime

import (
	"errors"
	"testing"
)

// A model stream cut off mid-answer surfaced as "Something went wrong" with no
// cause. It is a dropped connection, and the card should say so.
func TestClassifyError_CutStreamIsConnectionDrop(t *testing.T) {
	for _, msg := range []string{
		"stream: unexpected EOF",
		"read tcp 10.0.0.2:443: connection reset by peer",
		"write: broken pipe",
		"http2: response body closed",
	} {
		info := ClassifyError(errors.New(msg))
		if info.Kind != ErrKindProvider || info.Title != "The model connection dropped" || !info.Retryable {
			t.Fatalf("%q: got kind=%s title=%q retryable=%v", msg, info.Kind, info.Title, info.Retryable)
		}
	}
}

func TestClassifyError_UnknownStaysUnknown(t *testing.T) {
	info := ClassifyError(errors.New("something odd happened in the planner"))
	if info.Kind != ErrKindUnknown || info.Title != "Something went wrong" {
		t.Fatalf("unrelated errors must keep the generic card: %+v", info)
	}
}
