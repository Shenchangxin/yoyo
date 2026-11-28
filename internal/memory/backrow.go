// backrow.go 给记忆设上限和后排：很久没写过、也没用到的退到后排，前排超过上限时最久没动的也退到后排。
// 后排不删：还能搜到，记忆页还看得到，只是不进常驻画像、也不被召回。再用到（MarkUsed）就回到前排。
// 当事人亲口说的档案和偏好、没写来源的旧数据、本机清点从不退。后排每次现算，不写进 memory.json。
// 不拥有：哪几条这一轮真的带进了对话（app/memory_seen.go、memory_recall.go 调 MarkUsed）。

package memory

import (
	"sort"
	"strings"
	"time"
)

const (
	// coldAfter 是多久没写过、也没用到就退到后排。
	coldAfter = 180 * 24 * time.Hour
	// frontMax 是前排最多几条（不算从不退的那些）。多出来的按最久没动的退到后排。
	frontMax = 300
	// useEvery 是同一条记下「用到了」的最短间隔，免得每一轮都写一次盘。
	useEvery = 24 * time.Hour
)

// staysFront 在这一条从不退到后排时为真：本机清点，或者当事人说的（含没写来源的旧数据）档案和偏好。
func staysFront(it Item) bool {
	if machineItem(it) {
		return true
	}
	return (it.Kind == KindProfile || it.Kind == KindSemantic) && operatorSource(it.Source)
}

// lastActive 是这一条最近一次记下、改过、又说了一遍或被用到的时间。
func lastActive(it Item) time.Time {
	return maxTime(updatedOf(it), it.UsedAt)
}

// backSet 是现在在后排的条目编号。调用时要已持有锁。
func (s *Store) backSet(now time.Time) map[string]bool {
	back := map[string]bool{}
	var rest []Item
	for _, it := range s.items {
		if staysFront(it) || strings.TrimSpace(it.Text) == "" {
			continue
		}
		if at := lastActive(it); !at.IsZero() && now.Sub(at) > coldAfter {
			back[it.ID] = true
			continue
		}
		rest = append(rest, it)
	}
	if len(rest) > frontMax {
		sort.SliceStable(rest, func(i, j int) bool { return lastActive(rest[i]).After(lastActive(rest[j])) })
		for _, it := range rest[frontMax:] {
			back[it.ID] = true
		}
	}
	return back
}

// BackRow 返回现在在后排的条目编号。记忆页用它标「后排」。
func (s *Store) BackRow() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncLocked()
	return s.backSet(s.clock())
}

// MarkUsed 记下这几条这一轮带进了对话：后排的回到前排。同一条一天最多记一次；本机清点不记。有改动时才写盘，返回有没有改动。
// 用到不算改过：Touched 不变，整理建议攒新记忆时也不算它。
func (s *Store) MarkUsed(ids []string) bool {
	if len(ids) == 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.begin()()
	now := s.clock().UTC()
	changed := false
	for _, id := range ids {
		i := s.index(strings.TrimSpace(id))
		if i < 0 || machineItem(s.items[i]) || now.Sub(s.items[i].UsedAt) < useEvery {
			continue
		}
		s.items[i].UsedAt = now
		changed = true
	}
	if changed {
		_ = s.flush()
	}
	return changed
}
