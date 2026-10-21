package idol

import "time"

const (
	TierCold     = "cold"
	TierDistant  = "distant"
	TierTrust    = "trust"
	TierClose    = "close"
	TierIntimate = "intimate"
)

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

func clampScore(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

func (st *State) affinityOf(idolID string) Affinity {
	for _, a := range st.Affinity {
		if a.IdolID == idolID {
			a.Tier = TierOf(a.Score)
			return a
		}
	}
	return Affinity{IdolID: idolID, Score: 50, Tier: TierTrust}
}

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

func (a Affinity) Cooling(now time.Time) bool {
	return !a.CooldownUntil.IsZero() && now.Before(a.CooldownUntil)
}

// ApplyDelta moves the score. A drop to the floor starts a cooldown.
// Natural recovery toward trust happens when the last contact is old; the
// caller passes hours since the last record, 0 during an active turn.
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
