package evolve

// SelectParent implements DGM-style archive sampling: prefer high-scoring
// nodes that have been underexplored (few children). The evaluator itself
// is never a parent — only harness snapshots.
func SelectParent(nodes []Node, active string) string {
	if len(nodes) == 0 {
		return active
	}
	children := map[string]int{}
	byID := map[string]Node{}
	for _, n := range nodes {
		byID[n.ID] = n
		if n.Parent != "" {
			children[n.Parent]++
		}
	}
	bestID := active
	bestScore := -1.0
	for _, n := range nodes {
		if n.Note == "online-ace-stage" && n.Metrics.HeldInTotal+n.Metrics.HeldOutTotal == 0 {
			continue
		}
		pass := float64(n.Metrics.HeldInPass + n.Metrics.HeldOutPass)
		if pass == 0 && !n.Accepted {
			pass = 0.1
		}
		score := pass / float64(1+children[n.ID])
		if n.Accepted {
			score += 0.25
		}
		if score > bestScore {
			bestScore = score
			bestID = n.ID
		}
	}
	if bestID == "" {
		return active
	}
	return bestID
}
