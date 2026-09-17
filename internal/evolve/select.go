package evolve

import (
	"math/rand/v2"
)

// SelectParent implements DGM-style archive sampling: weight by score /
// (1+children). The evaluator itself is never a parent — only harness snapshots.
func SelectParent(nodes []Node, active string) string {
	return SelectParentSeeded(nodes, active, rand.Uint64())
}

func SelectParentSeeded(nodes []Node, active string, seed uint64) string {
	if len(nodes) == 0 {
		return active
	}
	children := map[string]int{}
	for _, n := range nodes {
		if n.Parent != "" {
			children[n.Parent]++
		}
	}
	type row struct {
		id     string
		weight float64
	}
	var rows []row
	var sum float64
	for _, n := range nodes {
		if n.Note == "online-ace-stage" && n.Metrics.HeldInTotal+n.Metrics.HeldOutTotal == 0 {
			continue
		}
		w := ParentWeight(n, children[n.ID])
		if w <= 0 {
			continue
		}
		rows = append(rows, row{id: n.ID, weight: w})
		sum += w
	}
	if len(rows) == 0 || sum <= 0 {
		return active
	}
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)).Float64() * sum
	acc := 0.0
	for _, row := range rows {
		acc += row.weight
		if r <= acc {
			return row.id
		}
	}
	return rows[len(rows)-1].id
}

func ParentWeight(n Node, childCount int) float64 {
	pass := float64(n.Metrics.HeldInPass + n.Metrics.HeldOutPass)
	if pass == 0 && !n.Accepted {
		pass = 0.1
	}
	score := pass / float64(1+childCount)
	if n.Accepted {
		score += 0.25
	}
	if n.TransferFail {
		score *= 0.3
	}
	if n.ManifestoMiss > n.ManifestoHit {
		score *= 0.5
	}
	if n.Surface == "prompt_fragment" && n.TransferFail {
		score *= 0.5
	}
	return score
}

// BestParent is the greedy argmax used by tests that need a stable winner.
func BestParent(nodes []Node, active string) string {
	if len(nodes) == 0 {
		return active
	}
	children := map[string]int{}
	for _, n := range nodes {
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
		score := ParentWeight(n, children[n.ID])
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
