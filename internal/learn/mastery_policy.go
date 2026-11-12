// 下一步由引擎决定，不由模型的印象决定。顺序是：待答的题、到期的复习、前置都已过门的第一个未掌握知识点。
// 这里只读内存里的路径状态，不读写库。

package learn

import (
	"sort"
	"time"
)

const (
	// ActionOutline 表示路径还没有知识点，先定大纲。
	ActionOutline = "outline"
	// ActionAnswer 表示有一道题在等学习者作答。
	ActionAnswer = "answer_pending"
	// ActionReview 表示有已掌握的知识点到了复习时间。
	ActionReview = "review"
	// ActionProbe 表示新知识点先用一道题摸底。
	ActionProbe = "probe"
	// ActionTeach 表示新知识点先讲解。学习者选了先讲再测时用它。
	ActionTeach = "teach"
	// ActionPractice 表示记忆或步骤型知识点还要再练。
	ActionPractice = "practice"
	// ActionAssess 表示概念或设计型知识点要请学习者用自己的话讲一遍。
	ActionAssess = "assess"
	// ActionComplete 表示全部掌握，也没有到期的复习。
	ActionComplete = "complete"
)

// NextStep 是引擎决定的下一步。Question 只在 answer_pending 时有，里面没有标准答案。
type NextStep struct {
	Action   string          `json:"action"`
	PointID  string          `json:"point_id,omitempty"`
	Name     string          `json:"name,omitempty"`
	Kind     string          `json:"kind,omitempty"`
	ModuleID string          `json:"module_id,omitempty"`
	Reason   string          `json:"reason,omitempty"`
	Question *PublicQuestion `json:"question,omitempty"`
}

// reviews 列出已掌握知识点的复习，到期的在前，按现在还记得的概率从低到高。limit 小于 1 时不截断。
// now 用来判断到期和估计概率。暂缓的知识点不在里面。
func (st *pathState) reviews(now time.Time, limit int) []ReviewItem {
	out := []ReviewItem{}
	for _, p := range st.Points {
		if !p.mastered() || p.Deferred || p.Due.IsZero() || !p.Mem.Seen() {
			continue
		}
		out = append(out, ReviewItem{
			PointID: p.ID, Name: p.Name, Kind: p.Kind, DueAt: formatTime(p.Due),
			Due:            !p.Due.After(now),
			Retrievability: Retrievability(now.Sub(p.Last).Hours()/24, p.Mem.Stability),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Due != out[j].Due {
			return out[i].Due
		}
		if out[i].Due {
			return out[i].Retrievability < out[j].Retrievability
		}
		return out[i].DueAt < out[j].DueAt
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// next 决定下一步。now 用来判断复习是否到期。
// 前置知识点没过门的先跳过；全都被挡住时退回大纲顺序里的第一个，避免路径卡死。暂缓的排在最后。
func (st *pathState) next(now time.Time) NextStep {
	if st.Pending != nil {
		q := st.Pending.public()
		step := NextStep{Action: ActionAnswer, PointID: st.Pending.PointID, Reason: "a question is waiting for the learner's answer", Question: &q}
		for _, p := range st.Points {
			if p.ID == st.Pending.PointID {
				step.Name, step.Kind, step.ModuleID = p.Name, p.Kind, p.ModuleID
			}
		}
		return step
	}
	if len(st.Points) == 0 {
		return NextStep{Action: ActionOutline, Reason: "the path has no knowledge points yet"}
	}
	if due := st.reviews(now, 1); len(due) > 0 && due[0].Due {
		for _, p := range st.Points {
			if p.ID == due[0].PointID {
				return NextStep{Action: ActionReview, PointID: p.ID, Name: p.Name, Kind: p.Kind, ModuleID: p.ModuleID, Reason: "a mastered point is due for review"}
			}
		}
	}
	done := st.masteredSet()
	var ready, held, deferred *pointRow
	for i := range st.Points {
		p := &st.Points[i]
		if p.mastered() {
			continue
		}
		switch {
		case p.Deferred:
			if deferred == nil {
				deferred = p
			}
		case blocked(*p, done):
			if held == nil {
				held = p
			}
		default:
			if ready == nil {
				ready = p
			}
		}
	}
	pick, reason := ready, "next unmastered point whose prerequisites are cleared"
	if pick == nil {
		pick, reason = held, "every remaining point waits on a prerequisite; start with the first one"
	}
	if pick == nil {
		pick, reason = deferred, "only deferred points remain"
	}
	if pick == nil {
		return NextStep{Action: ActionComplete, Reason: "everything is mastered and nothing is due"}
	}
	action := ActionPractice
	switch {
	case pick.Status == StatusNew && st.Teaching == TeachFirst:
		action = ActionTeach
	case pick.Status == StatusNew:
		action = ActionProbe
	case qualitativeKind(pick.Kind):
		action = ActionAssess
	}
	return NextStep{Action: action, PointID: pick.ID, Name: pick.Name, Kind: pick.Kind, ModuleID: pick.ModuleID, Reason: reason}
}
