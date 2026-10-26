// 从档案里抽下一个父快照。权重看通过数、子节点数、是否接受、迁移失败和宣言命中。评估器本身不是父节点。不写 refs。

package evolve

import (
	"math/rand/v2"
)

// SelectParent 用随机种子调用 SelectParentSeeded。档案为空时返回 active。
func SelectParent(nodes []Node, active string) string {
	return SelectParentSeeded(nodes, active, rand.Uint64())
}

// SelectParentSeeded 按 ParentWeight 做加权抽样。Note 为 online-ace-stage 且 held-in、held-out 总数都是 0 的节点跳过。没有正权重时返回 active。
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

// ParentWeight 用 held-in 与 held-out 通过数之和除以 1 加子节点数。通过数为 0 且未接受时通过数按 0.1 计。接受则再加 0.25。
// TransferFail 乘 0.3。ManifestoMiss 大于 ManifestoHit 时乘 0.5。surface 为 prompt_fragment 且 TransferFail 时再乘 0.5。
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

// BestParent 取 ParentWeight 最大的节点编号。同样跳过未评估的 online-ace-stage。都没有时返回 active。
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
