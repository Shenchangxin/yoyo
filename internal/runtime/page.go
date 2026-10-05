package runtime

import (
	"encoding/json"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TrajectoryPage is the renderer window: a UI-slimmed slice plus cursors.
// HeadSeq/TailSeq are store index seqs. HubSeq is the live bus head at
// the moment the page was built so the client can catch up without a gap.
type TrajectoryPage struct {
	Events  []trace.Event `json:"events"`
	HeadSeq int64         `json:"head_seq"`
	TailSeq int64         `json:"tail_seq"`
	Older   bool          `json:"older"`
	HubSeq  int64         `json:"hub_seq"`
}

// ProjectPage slims a store page and enforces a marshal budget by dropping
// the oldest user-turns. The live tail is never dropped.
func ProjectPage(raw trace.Page, hubSeq int64, byteCap int) TrajectoryPage {
	return ProjectPageKeep(raw, hubSeq, byteCap, 0)
}

// ProjectPageKeep is ProjectPage but will not drop the user-turn at keepSeq.
// Jump-around must keep the target even when the window is over budget.
func ProjectPageKeep(raw trace.Page, hubSeq int64, byteCap int, keepSeq int64) TrajectoryPage {
	ui := UITrajectory(raw.Events)
	older := raw.Older
	if byteCap > 0 {
		var trimmed bool
		ui, trimmed = capTrajectoryBytesKeep(ui, byteCap, keepSeq)
		if trimmed {
			older = true
		}
	}
	head := raw.HeadSeq
	tail := raw.TailSeq
	if len(ui) > 0 {
		if ui[0].Seq > 0 {
			head = ui[0].Seq
		}
		if ui[len(ui)-1].Seq > 0 {
			tail = ui[len(ui)-1].Seq
		}
	}
	return TrajectoryPage{
		Events:  ui,
		HeadSeq: head,
		TailSeq: tail,
		Older:   older,
		HubSeq:  hubSeq,
	}
}

func capTrajectoryBytes(evs []trace.Event, byteCap int) ([]trace.Event, bool) {
	return capTrajectoryBytesKeep(evs, byteCap, 0)
}

func capTrajectoryBytesKeep(evs []trace.Event, byteCap int, keepSeq int64) ([]trace.Event, bool) {
	if len(evs) == 0 {
		return evs, false
	}
	b, err := json.Marshal(evs)
	if err != nil || len(b) <= byteCap {
		return evs, false
	}
	trimmed := false
	for len(evs) > 1 {
		ranges := userRanges(evs)
		if len(ranges) <= 1 {
			break
		}
		drop := -1
		if keepSeq > 0 {
			keep := -1
			for i, r := range ranges {
				if rangeHasSeq(evs, r, keepSeq) {
					keep = i
					break
				}
			}
			if keep >= 0 && keep < len(ranges)-1 {
				drop = len(ranges) - 1
			} else if keep > 0 {
				drop = 0
			} else if keep == 0 && len(ranges) == 1 {
				break
			} else if keep < 0 {
				drop = 0
			} else {
				break
			}
		} else {
			drop = 0
		}
		if drop < 0 {
			break
		}
		r := ranges[drop]
		next := append([]trace.Event{}, evs[:r[0]]...)
		next = append(next, evs[r[1]:]...)
		if len(next) == 0 {
			break
		}
		evs = next
		trimmed = true
		b, err = json.Marshal(evs)
		if err != nil || len(b) <= byteCap {
			break
		}
	}
	return evs, trimmed
}

func userRanges(evs []trace.Event) [][2]int {
	var starts []int
	for i, ev := range evs {
		if ev.Type == trace.TypeUser && !trace.IsSteerUser(ev) {
			starts = append(starts, i)
		}
	}
	if len(starts) == 0 {
		return [][2]int{{0, len(evs)}}
	}
	out := make([][2]int, 0, len(starts))
	for i, s := range starts {
		e := len(evs)
		if i+1 < len(starts) {
			e = starts[i+1]
		}
		out = append(out, [2]int{s, e})
	}
	if starts[0] > 0 {
		out = append([][2]int{{0, starts[0]}}, out...)
	}
	return out
}

func rangeHasSeq(evs []trace.Event, r [2]int, seq int64) bool {
	for i := r[0]; i < r[1] && i < len(evs); i++ {
		if evs[i].Seq == seq {
			return true
		}
	}
	return false
}
