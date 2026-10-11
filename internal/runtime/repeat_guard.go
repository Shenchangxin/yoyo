package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Repeat thresholds count identical calls whose results were also identical,
// with no progress in between. A changed result is a different call, so a
// poll that is making headway is never counted.
const (
	repeatNudgeHits = 3
	// repeatStopHits ends a run when a mutating call keeps returning the same
	// result. A read-only call only gets the nudge: asking again is cheap and
	// can be a legitimate check, so it ends a run only at repeatHardStopHits.
	repeatStopHits     = 8
	repeatHardStopHits = 16
)

const repeatNudgeEN = "You made the same %s call %d times and got the same result each time. Nothing has changed since, so the result you already have is current. Use it, or change what you do next."
const repeatNudgeZH = "同一个 %s 调用已重复 %d 次，每次结果都一样。之后没有任何变化，手上的结果就是最新的。直接用它，或换一种做法。"

// repeatTracker is the result-aware loop detector. The key covers the tool
// name, the exact argument text, and a digest of the result, so the detector
// judges what came back, not how often a tool was called.
type repeatTracker struct {
	counts   map[string]int
	max      int
	name     string
	readonly bool
}

func newRepeatTracker() *repeatTracker {
	return &repeatTracker{counts: map[string]int{}}
}

// note records one round. A round with progress clears every streak. It
// reports whether any call has reached the nudge threshold.
func (r *repeatTracker) note(calls []ToolCall, results []Message, progressed bool) bool {
	if r == nil {
		return false
	}
	if progressed {
		r.counts = map[string]int{}
		r.max = 0
		r.name = ""
		r.readonly = false
		return false
	}
	for i, tc := range calls {
		body := ""
		if i < len(results) {
			body = results[i].Content
		}
		sum := sha256.Sum256([]byte(body))
		key := tc.Name + "\x00" + tc.Arguments + "\x00" + hex.EncodeToString(sum[:8])
		r.counts[key]++
		if c := r.counts[key]; c > r.max {
			r.max = c
			r.name = tc.Name
			r.readonly = isReadonlyTool(tc.Name)
		}
	}
	return r.max >= repeatNudgeHits
}

// nudge returns the corrective message for the most-repeated call, or "" when
// no streak has reached the nudge threshold yet.
func (r *repeatTracker) nudge(req RunRequest) string {
	if r == nil || r.max < repeatNudgeHits {
		return ""
	}
	return fmt.Sprintf(voiceNudge(req, repeatNudgeEN, repeatNudgeZH), r.name, r.max)
}

// stuck reports that a call has returned the same result too many times for
// the run to be considered progressing.
func (r *repeatTracker) stuck() bool {
	if r == nil {
		return false
	}
	if r.readonly {
		return r.max >= repeatHardStopHits
	}
	return r.max >= repeatStopHits
}
