// 本文件拥有好感分数、档位和冷却。ApplyDelta 在一轮对话或送礼时改分数。
// 不调用模型。档位文案 Feeling 由 buildPrompt 放进当轮提示。

package idol

import "time"

const (
	// TierCold 是分数不超过 19 的档位，值是 cold。落到 12 及以下会开始冷却。
	TierCold = "cold"
	// TierDistant 是分数 20 到 39 的档位，值是 distant。这一档不会主动开口。
	TierDistant = "distant"
	// TierTrust 是分数 40 到 59 的档位，值是 trust。没有记录时 affinityOf 用 50，落在这一档。
	TierTrust = "trust"
	// TierClose 是分数 60 到 79 的档位，值是 close。主动开口的间隔是 18 小时。
	TierClose = "close"
	// TierIntimate 是分数 80 及以上的档位，值是 intimate。解锁时多给一张重置卡，主动开口的间隔是 8 小时。
	TierIntimate = "intimate"
)

// TierOf 按分数返回档位名。19 及以下是 cold，39 及以下是 distant，59 及以下是 trust，79 及以下是 close，其余是 intimate。
// score 小于等于 19 返回 TierCold，小于等于 39 返回 TierDistant，小于等于 59 返回 TierTrust，小于等于 79 返回 TierClose，其余返回 TierIntimate。
func TierOf(score int) string {
	switch {
	case score <= 19:
		return TierCold
	case score <= 39:
		return TierDistant
	case score <= 59:
		return TierTrust
	case score <= 79:
		return TierClose
	default:
		return TierIntimate
	}
}

// clampScore 把分数限制在 0 到 100。
// n 小于 0 时返回 0，大于 100 时返回 100，否则原样返回。
func clampScore(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

// affinityOf 返回这位偶像的好感。没有记录时分数是 50，档位是 trust。
// idolID 与每条的 IdolID 比较。没有命中时返回的 IdolID 就是它，分数是 50，档位是 TierTrust。
func (st *State) affinityOf(idolID string) Affinity {
	for _, a := range st.Affinity {
		if a.IdolID == idolID {
			a.Tier = TierOf(a.Score)
			return a
		}
	}
	return Affinity{IdolID: idolID, Score: 50, Tier: TierTrust}
}

// setAffinity 先夹紧分数、重算档位，再按偶像 id 替换或追加。
// a 的 Score 先交给 clampScore，Tier 改成 TierOf 的结果，再按 IdolID 替换已有项或追加。
func (st *State) setAffinity(a Affinity) {
	a.Score = clampScore(a.Score)
	a.Tier = TierOf(a.Score)
	for i := range st.Affinity {
		if st.Affinity[i].IdolID == a.IdolID {
			st.Affinity[i] = a
			return
		}
	}
	st.Affinity = append(st.Affinity, a)
}

// Cooling 在 CooldownUntil 还没到时返回真。
// now 在 CooldownUntil 非零且 now.Before 该时间时使函数返回 true。
func (a Affinity) Cooling(now time.Time) bool {
	return !a.CooldownUntil.IsZero() && now.Before(a.CooldownUntil)
}

// ApplyDelta 改一位偶像的分数。分数掉到 12 及以下时，冷却到 now 之后 12 小时。
// idleHours 大于 0 且分数低于 50 时，每 12 小时向 50 回一点；一轮进行中的对话传入 0。冷却中的分数回到 28 及以上就清掉冷却。
// idolID 传给 affinityOf，取出要改的那条。
// delta 加进夹紧前的分数。
func (st *State) ApplyDelta(idolID string, delta int, now time.Time, idleHours int) Affinity {
	a := st.affinityOf(idolID)
	if idleHours > 0 && a.Score < 50 {
		recover := idleHours / 12
		if recover > 50-a.Score {
			recover = 50 - a.Score
		}
		a.Score += recover
	}
	a.Score = clampScore(a.Score + delta)
	if a.Score <= 12 {
		a.CooldownUntil = now.Add(12 * time.Hour)
	}
	if a.Cooling(now) && a.Score >= 28 {
		a.CooldownUntil = time.Time{}
	}
	st.setAffinity(a)
	return st.affinityOf(idolID)
}

// Feeling 按档位返回放进提示的一句态度。人设本身不改，冷淡档只是话少、保持距离。
// tier 与 TierCold、TierDistant、TierClose、TierIntimate 比较，其余返回平常信任那句。
func Feeling(tier string) string {
	switch tier {
	case TierCold:
		return "你现在对用户很冷淡，话少，不想亲近，但人设不变。"
	case TierDistant:
		return "你现在和用户有些疏远，礼貌，但保持距离。"
	case TierClose:
		return "你现在觉得和用户很亲近，会自然地高兴。"
	case TierIntimate:
		return "你现在觉得和用户很亲密，仍然是你自己，不夸张。"
	default:
		return "你现在对用户是平常的信任。"
	}
}
