// 锁住流被切断时的错误卡片：种类是 provider，标题是连接中断，可以重试。无关错误仍是 unknown。

package runtime

import (
	"errors"
	"testing"
)

// TestClassifyError_CutStreamIsConnectionDrop 确认 unexpected EOF、连接重置、broken pipe 和 http2 body closed 都是可重试的连接中断，标题是 The model connection dropped。
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

// TestClassifyError_UnknownStaysUnknown 确认对不上已知种类的错误仍是 unknown，标题是 Something went wrong。
func TestClassifyError_UnknownStaysUnknown(t *testing.T) {
	info := ClassifyError(errors.New("something odd happened in the planner"))
	if info.Kind != ErrKindUnknown || info.Title != "Something went wrong" {
		t.Fatalf("unrelated errors must keep the generic card: %+v", info)
	}
}
