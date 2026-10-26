// 把轨迹存储页收成界面用的窗口：瘦身后的事件，加上头尾序号和是否还有更早事件。
// 不读盘。调用方传入 trace.Page 和当时的总线序号。

package runtime

import (
	"encoding/json"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TrajectoryPage 是渲染窗口：瘦身后的事件，加上游标。
// HeadSeq 和 TailSeq 是存储序号。HubSeq 是组装这一页时的总线头，供客户端补齐空隙。
type TrajectoryPage struct {
	Events  []trace.Event `json:"events"`
	HeadSeq int64         `json:"head_seq"`
	TailSeq int64         `json:"tail_seq"`
	Older   bool          `json:"older"`
	HubSeq  int64         `json:"hub_seq"`
}

// ProjectPage 瘦身存储页，并按字节预算丢掉最旧的用户回合。默认不丢当前最后一段。
// raw 交给 ProjectPageKeep。
// hubSeq 交给 ProjectPageKeep。
// byteCap 交给 ProjectPageKeep。
func ProjectPage(raw trace.Page, hubSeq int64, byteCap int) TrajectoryPage {
	return ProjectPageKeep(raw, hubSeq, byteCap, 0)
}

// ProjectPageKeep 与 ProjectPage 相同，但不会丢掉含 keepSeq 的用户回合。
// 目标不在最后一段时，先删最后一段，直到目标变成最后一段，再删更早的段。
// raw 的 Events 交给 UITrajectory，Older、HeadSeq、TailSeq 填进返回页。
// hubSeq 写入返回页的 HubSeq。
// byteCap 大于 0 时和瘦身后的事件、keepSeq 一起交给 capTrajectoryBytesKeep。
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

// capTrajectoryBytes 按字节预算从最旧的用户回合删起。
// evs 交给 capTrajectoryBytesKeep。
// byteCap 交给 capTrajectoryBytesKeep。
func capTrajectoryBytes(evs []trace.Event, byteCap int) ([]trace.Event, bool) {
	return capTrajectoryBytesKeep(evs, byteCap, 0)
}

// capTrajectoryBytesKeep 在 JSON 超过 byteCap 时按用户回合整段删除，直到不超或只剩一段。keepSeq 为 0 时只删最旧段。
// evs 长度为 0 时原样返回；否则整段序列化，超长时按 userRanges 删掉若干用户回合。
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

// userRanges 按非插话的用户消息切回合。第一段用户消息之前的事件单独成段，并放在最前。
// evs 里类型是用户且不是插话的下标作为段起点；没有这种事件时返回从 0 到 len(evs) 的一段。
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

// rangeHasSeq 判断半开区间 r 里是否有序号 seq。
// evs 在下标落进 r 时取出，用事件的 Seq 和 seq 比较。
func rangeHasSeq(evs []trace.Event, r [2]int, seq int64) bool {
	for i := r[0]; i < r[1] && i < len(evs); i++ {
		if evs[i].Seq == seq {
			return true
		}
	}
	return false
}
