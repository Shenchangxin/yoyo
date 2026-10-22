package idol

import (
	"strings"
	"time"
)

func (st State) Idol(id string) (Idol, bool) { return st.idol(id) }

func (st State) Pinned() (Idol, bool) { return pinnedOf(st) }

func (st *State) EnsureDaily(now time.Time) { st.ensureDaily(now) }

func (st *State) ForceNew(idolID string, now time.Time) Conversation {
	return st.forceNew(idolID, now)
}

func (st *State) MarkTask(now time.Time, kind string) {
	st.markTask(now, st.User.Location(), kind)
}

func (st *State) Unlock(now time.Time) { st.unlock(now) }

func (st State) MemoriesFor(idolID string) []Memory {
	if strings.TrimSpace(idolID) == "" {
		return append([]Memory{}, st.Memories...)
	}
	idol, ok := st.idol(idolID)
	var out []Memory
	for _, m := range st.Memories {
		if m.SourceIdol == idolID || (ok && idol.CircleID != "" && m.CircleID == idol.CircleID) {
			out = append(out, m)
		}
	}
	return out
}

type HistoryView struct {
	Current *Conversation  `json:"current,omitempty"`
	Past    []Conversation `json:"past"`
}

func (st State) History(idolID string) HistoryView {
	out := HistoryView{Past: []Conversation{}}
	for _, c := range st.Conversations {
		if c.IdolID != idolID {
			continue
		}
		cp := c
		if cp.Current {
			out.Current = &cp
			continue
		}
		out.Past = append(out.Past, cp)
	}
	return out
}

func (s *Store) SaveIdol(in Idol, now time.Time) (Idol, error) {
	var out Idol
	err := s.Update(func(st *State) error {
		if in.ID != "" {
			if prev, ok := st.idol(in.ID); ok {
				if in.WorkSessionID == "" {
					in.WorkSessionID = prev.WorkSessionID
				}
				if in.CreatedAt.IsZero() {
					in.CreatedAt = prev.CreatedAt
				}
				if strings.TrimSpace(in.OutfitID) == "" && validOutfitID(prev.OutfitID) {
					in.OutfitID = prev.OutfitID
				}
				in.Pinned = prev.Pinned
			}
		}
		out = NormalizeIdol(in, now)
		st.putIdol(out)
		return nil
	})
	return out, err
}

func (s *Store) SaveCircle(c Circle) (Circle, error) {
	if strings.TrimSpace(c.Name) == "" {
		c.Name = "圈子"
	}
	if c.ID == "" {
		c.ID = newID()
	}
	err := s.Update(func(st *State) error {
		replaced := false
		for i := range st.Circles {
			if st.Circles[i].ID == c.ID {
				st.Circles[i] = c
				replaced = true
			}
		}
		if !replaced {
			st.Circles = append(st.Circles, c)
		}
		return nil
	})
	return c, err
}

func (s *Store) SaveLore(doc Lore) (Lore, error) {
	if doc.ID == "" {
		doc.ID = newID()
	}
	if strings.TrimSpace(doc.Title) == "" {
		doc.Title = "材料"
	}
	err := s.Update(func(st *State) error {
		for i := range st.Lore {
			if st.Lore[i].ID == doc.ID {
				st.Lore[i] = doc
				return nil
			}
		}
		st.Lore = append(st.Lore, doc)
		return nil
	})
	return doc, err
}

func (s *Store) SaveServices(svc Services) error {
	return s.Update(func(st *State) error {
		st.Services = svc
		return nil
	})
}

func NormalizeIdol(in Idol, now time.Time) Idol {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		in.Name = "偶像"
	}
	in.SpokenLang = lang(in.SpokenLang)
	in.SubtitleLang = lang(in.SubtitleLang)
	in.Color = strings.TrimSpace(in.Color)
	if len(in.Color) != 7 || !strings.HasPrefix(in.Color, "#") {
		in.Color = "#c47b8a"
	}
	if in.ID == "" {
		in.ID = newID()
		in.CreatedAt = now
	}
	in.UpdatedAt = now
	return in
}
