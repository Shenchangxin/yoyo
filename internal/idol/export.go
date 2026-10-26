// 本文件把状态上的小写方法包成给包外调用的名字，并保存偶像、圈子、材料和服务设置。
// 不调用模型。NormalizeIdol 补上缺的名字、语言、颜色和时间。

package idol

import (
	"strings"
	"time"
)

// Idol 按 id 返回偶像。没有则 ok 为 false。
func (st State) Idol(id string) (Idol, bool) { return st.idol(id) }

// Pinned 返回当前钉选且未隐藏的偶像。没有则 ok 为 false。
func (st State) Pinned() (Idol, bool) { return pinnedOf(st) }

// EnsureDaily 在跨过用户时区的一天时补抽卡次数并换上当天的任务。
func (st *State) EnsureDaily(now time.Time) { st.ensureDaily(now) }

// ForceNew 把这位偶像当前对话标成过去，并另开一段空的当前对话。
func (st *State) ForceNew(idolID string, now time.Time) Conversation {
	return st.forceNew(idolID, now)
}

// MarkTask 按 kind 记下当天的说话、通话或送礼，并在首次完成时加一次奖励抽卡。
func (st *State) MarkTask(now time.Time, kind string) {
	st.markTask(now, st.User.Location(), kind)
}

// Unlock 按现有对话、礼物和好感补成就，返回这次新加上的 id。
func (st *State) Unlock(now time.Time) { st.unlock(now) }

// MemoriesFor 返回记忆。idolID 为空时返回全部；否则返回这位偶像自己的，以及她所在圈子里的。
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

// HistoryView 把一位偶像的对话分成当前一段和过去的几段。
type HistoryView struct {
	Current *Conversation  `json:"current,omitempty"`
	Past    []Conversation `json:"past"`
}

// History 按偶像 id 收集对话。Current 为真的放进 Current，其余放进 Past。
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

// SaveIdol 写入或替换一位偶像。已有 id 时保留钉选、空的工作会话、空的创建时间，以及原先合法的 OutfitID。
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

// SaveCircle 写入或替换一个圈子。名字空白时写成「圈子」，id 空白时用 newID。
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

// SaveLore 写入或替换一篇材料。标题空白时写成「材料」，id 空白时用 newID。
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

// SaveServices 整份替换 Services。
func (s *Store) SaveServices(svc Services) error {
	return s.Update(func(st *State) error {
		st.Services = svc
		return nil
	})
}

// NormalizeIdol 收掉名字空白，补上缺的语言、颜色、id 和时间。颜色不是 # 加 6 位时改成 #c47b8a，名字空白时写成「偶像」。
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
