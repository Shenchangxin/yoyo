package memory

import "strings"

// SetMuted 让 id 这一条不再带进对话（muted 为真），或者重新带进去。条目留着，记忆页还看得到。本机清点不改，找不到时返回假。
func (s *Store) SetMuted(id string, muted bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.begin()()
	i := s.index(strings.TrimSpace(id))
	if i < 0 || machineItem(s.items[i]) {
		return false
	}
	if s.items[i].Muted == muted {
		return true
	}
	s.items[i].Muted = muted
	_ = s.flush()
	return true
}

// Revert 把 id 这一条退回上一次的说法：历史里最近的那一句换回正文，来源跟着换回去。最初记下的时间不变。
// 给「记下了」回执上的撤销用：改口撤销就是退回旧说法。没有历史、找不到或是本机清点时返回假。
func (s *Store) Revert(id string) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.begin()()
	i := s.index(strings.TrimSpace(id))
	if i < 0 || machineItem(s.items[i]) || len(s.items[i].History) == 0 {
		return Item{}, false
	}
	now := s.clock()
	cur := s.items[i]
	n := len(cur.History)
	prev := cur.History[n-1]
	cur.History = append([]Revision(nil), cur.History[:n-1]...)
	text, until := anchorDates(prev.Text, now)
	cur.Text = text
	cur.ValidUntil = until
	if strings.TrimSpace(prev.Source) != "" {
		cur.Source = prev.Source
	}
	cur.UpdatedAt = now.UTC()
	s.items[i] = cur
	_ = s.flush()
	return cur, true
}
