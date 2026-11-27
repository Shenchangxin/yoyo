// layout.go 是画布节点排布的确定性几何实现，供画布 Agent 的整理和归类使用。前端的对齐、分带和新增落位只在前端实现，没有移植到这里。
// 这里与前端 web/src/lib/canvas/canvas-layout.ts 保持同一套常量与算法语义，使前端自动整理和 Agent 整理得到同样的结果。修改任一侧常量或算法时，另一侧要同步，两侧各自有黄金用例。
// 本文件只做几何，不读取画布文档、不判断权限、不写库。

package layout

import "sort"

// 这些间距与前端 canvas-layout.ts 的常量相同。
const (
	// FlowColumnGap 是依赖分层时列与列的间距，120。
	FlowColumnGap = 120
	// LaneGap 是泳道之间的间距，96。
	LaneGap = 96
	// LaneNodeGap 是泳道内节点的间距，36。
	LaneNodeGap = 36
	// GridNodeGap 是网格里节点的间距，32。
	GridNodeGap = 32
)

// LaneOrder 是泳道的稳定顺序，决定从上到下的分区次序。
var LaneOrder = []string{"text", "image", "video", "audio"}

// Node 是排布所需的节点快照。Width/Height 为 0 时按默认尺寸补齐。
type Node struct {
	ID      string
	Type    string
	Lane    string
	X, Y    float64
	Width   float64
	Height  float64
	Locked  bool
	NoMove  bool // 容器节点（背板/文件夹）本身不参与整理
	Movable bool
}

// Position 是排布结果。
type Position struct {
	X float64
	Y float64
}

// Edge 是一条有向引用连线（来源 → 目标），用于依赖方向分层。
type Edge struct {
	From string
	To   string
}

// Mode 是整理模式。
type Mode string

// 整理模式的取值。
const (
	// ModeAuto 是自动整理 auto。
	ModeAuto Mode = "auto"
	// ModeFlow 是按依赖从左到右 flow。
	ModeFlow Mode = "flow"
	// ModeByLane 是按媒体类型分泳道 byType。
	ModeByLane Mode = "byType"
	// ModeRow 是单行 row。
	ModeRow Mode = "row"
	// ModeColumn 是单列 column。
	ModeColumn Mode = "column"
	// ModeGrid 是方阵 grid。
	ModeGrid Mode = "grid"
)

// width 在宽度大于 0 时用原值，否则用 340。
func (n Node) width() float64 {
	if n.Width > 0 {
		return n.Width
	}
	return 340
}

// height 在高度大于 0 时用原值，否则用 240。
func (n Node) height() float64 {
	if n.Height > 0 {
		return n.Height
	}
	return 240
}

// minX 返回节点里最小的 X。调用时节点不能为空。
// nodes 取第 0 个的 X，再与后面每个节点的 X 取更小值。
func minX(nodes []Node) float64 {
	value := nodes[0].X
	for _, node := range nodes[1:] {
		if node.X < value {
			value = node.X
		}
	}
	return value
}

// minY 返回节点里最小的 Y。调用时节点不能为空。
// nodes 取第 0 个的 Y，再与后面每个节点的 Y 取更小值。
func minY(nodes []Node) float64 {
	value := nodes[0].Y
	for _, node := range nodes[1:] {
		if node.Y < value {
			value = node.Y
		}
	}
	return value
}

// maxWidth 返回节点宽度的最大值，空切片返回 0。
// nodes 逐个比较 width，保留更大的，初值是 0。
func maxWidth(nodes []Node) float64 {
	value := 0.0
	for _, node := range nodes {
		if node.width() > value {
			value = node.width()
		}
	}
	return value
}

// maxHeight 返回节点高度的最大值，空切片返回 0。
// nodes 逐个比较 height，保留更大的，初值是 0。
func maxHeight(nodes []Node) float64 {
	value := 0.0
	for _, node := range nodes {
		if node.height() > value {
			value = node.height()
		}
	}
	return value
}

// byPosition 复刻前端的稳定次序：先按 y，再按 x。
// nodes 先复刻一份，再按 Y 升序、Y 相同则按 X 升序做稳定排序。
func byPosition(nodes []Node) []Node {
	sorted := append([]Node(nil), nodes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Y != sorted[j].Y {
			return sorted[i].Y < sorted[j].Y
		}
		return sorted[i].X < sorted[j].X
	})
	return sorted
}

// Auto 复刻 layoutCanvasAuto：节点之间有连线时按依赖分层，否则按媒体类型泳道。
// nodes 少于 2 个时返回空 map；connected 为真时传给 Flow，否则传给 ByLane。
// edges 传给 connected，为真时再传给 Flow。
func Auto(nodes []Node, edges []Edge) map[string]Position {
	if len(nodes) < 2 {
		return map[string]Position{}
	}
	if connected(nodes, edges) {
		return Flow(nodes, edges)
	}
	return ByLane(nodes)
}

// connected 在两端都落在这批节点里的连线至少有一条时为真。
// nodes 的 ID 收成集合。
// edges 里 From 和 To 都在该集合中时返回真。
func connected(nodes []Node, edges []Edge) bool {
	ids := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		ids[node.ID] = true
	}
	for _, edge := range edges {
		if ids[edge.From] && ids[edge.To] {
			return true
		}
	}
	return false
}

// Flow 复刻 layoutCanvasFlow：Kahn 拓扑分层从左到右，层内按泳道竖排。
// 环上的节点留在第 0 层，避免布局死循环。
// nodes 为空时返回空 map；否则按 ID 建入度，入度为 0 的放进第 0 层，循环结束仍没有层号的也写成 0。
// edges 只保留两端 ID 都在 nodes 索引里的，用来计入度和出边。
func Flow(nodes []Node, edges []Edge) map[string]Position {
	result := map[string]Position{}
	if len(nodes) == 0 {
		return result
	}
	index := make(map[string]Node, len(nodes))
	for _, node := range nodes {
		index[node.ID] = node
	}
	inbound := make(map[string]int, len(nodes))
	outbound := make(map[string][]string, len(nodes))
	for _, node := range nodes {
		inbound[node.ID] = 0
	}
	for _, edge := range edges {
		if _, ok := index[edge.From]; !ok {
			continue
		}
		if _, ok := index[edge.To]; !ok {
			continue
		}
		inbound[edge.To]++
		outbound[edge.From] = append(outbound[edge.From], edge.To)
	}

	queue := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if inbound[node.ID] == 0 {
			queue = append(queue, node.ID)
		}
	}
	layer := map[string]int{}
	for _, id := range queue {
		layer[id] = 0
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, target := range outbound[id] {
			if layer[target] < layer[id]+1 {
				layer[target] = layer[id] + 1
			}
			inbound[target]--
			if inbound[target] == 0 {
				queue = append(queue, target)
			}
		}
	}
	for _, node := range nodes {
		if _, ok := layer[node.ID]; !ok {
			layer[node.ID] = 0
		}
	}

	left, top := minX(nodes), minY(nodes)
	layerIDs := make([]int, 0, len(nodes))
	seen := map[int]bool{}
	for _, node := range nodes {
		if !seen[layer[node.ID]] {
			seen[layer[node.ID]] = true
			layerIDs = append(layerIDs, layer[node.ID])
		}
	}
	sort.Ints(layerIDs)
	layerX := map[int]float64{}
	x := left
	for _, value := range layerIDs {
		members := nodesInLayer(nodes, layer, value)
		layerX[value] = x
		x += maxWidth(members) + FlowColumnGap
	}

	laneTop := top
	for _, lane := range LaneOrder {
		lanes := laneNodes(nodes, lane)
		if len(lanes) == 0 {
			continue
		}
		stacks := map[int][]Node{}
		stackOrder := make([]int, 0, len(lanes))
		for _, node := range lanes {
			value := layer[node.ID]
			if _, ok := stacks[value]; !ok {
				stackOrder = append(stackOrder, value)
			}
			stacks[value] = append(stacks[value], node)
		}
		laneHeight := 0.0
		for _, stack := range stacks {
			height := 0.0
			for _, node := range stack {
				height += node.height()
			}
			height += float64(max(0, len(stack)-1)) * LaneNodeGap
			if height > laneHeight {
				laneHeight = height
			}
		}
		sort.Ints(stackOrder)
		for _, value := range stackOrder {
			stack := byPosition(stacks[value])
			y := laneTop
			for _, node := range stack {
				result[node.ID] = Position{X: layerX[value], Y: y}
				y += node.height() + LaneNodeGap
			}
		}
		laneTop += laneHeight + LaneGap
	}
	return result
}

// nodesInLayer 取出分层号等于 value 的节点。
// layer 用节点 ID 取层号，只保留等于 value 的节点。
func nodesInLayer(nodes []Node, layer map[string]int, value int) []Node {
	subset := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		if layer[node.ID] == value {
			subset = append(subset, node)
		}
	}
	return subset
}

// laneNodes 取出泳道等于 lane 的节点。
// nodes 只保留 Lane 等于 lane 的。
func laneNodes(nodes []Node, lane string) []Node {
	subset := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		if node.Lane == lane {
			subset = append(subset, node)
		}
	}
	return subset
}

// ByLane 复刻 layoutCanvasNodesByMediaType：每个泳道内部按网格排布（最多 4 列）。
// nodes 为空时返回空 map；否则按 LaneOrder 分组，组内先 byPosition，列数取 4 和 ceilSqrt 结果的较小值。
func ByLane(nodes []Node) map[string]Position {
	result := map[string]Position{}
	if len(nodes) == 0 {
		return result
	}
	left, top := minX(nodes), minY(nodes)
	laneTop := top
	for _, lane := range LaneOrder {
		lanes := byPosition(laneNodes(nodes, lane))
		if len(lanes) == 0 {
			continue
		}
		columns := min(4, max(1, ceilSqrt(len(lanes))))
		cellWidth := maxWidth(lanes) + GridNodeGap
		cellHeight := maxHeight(lanes) + GridNodeGap
		for index, node := range lanes {
			result[node.ID] = Position{
				X: left + float64(index%columns)*cellWidth,
				Y: laneTop + float64(index/columns)*cellHeight,
			}
		}
		rows := float64(ceilDiv(len(lanes), columns))
		laneTop += rows*cellHeight - GridNodeGap + LaneGap
	}
	return result
}

// Linear 复刻 layoutCanvasNodes：单行 / 单列 / 方阵。
// nodes 为空时返回空 map；否则先 byPosition，起点用 minX 和 minY。
// mode 为 ModeRow 时横排，为 ModeColumn 时竖排，其余按方阵，列数用 ceilSqrt。
func Linear(nodes []Node, mode Mode) map[string]Position {
	result := map[string]Position{}
	if len(nodes) == 0 {
		return result
	}
	sorted := byPosition(nodes)
	left, top := minX(nodes), minY(nodes)
	switch mode {
	case ModeRow:
		x := left
		for _, node := range sorted {
			result[node.ID] = Position{X: x, Y: top}
			x += node.width() + GridNodeGap
		}
	case ModeColumn:
		y := top
		for _, node := range sorted {
			result[node.ID] = Position{X: left, Y: y}
			y += node.height() + GridNodeGap
		}
	default:
		columns := max(1, int(ceilSqrt(len(sorted))))
		cellWidth := maxWidth(sorted) + GridNodeGap
		cellHeight := maxHeight(sorted) + GridNodeGap
		for index, node := range sorted {
			result[node.ID] = Position{
				X: left + float64(index%columns)*cellWidth,
				Y: top + float64(index/columns)*cellHeight,
			}
		}
	}
	return result
}

// ceilSqrt 返回不小于平方根的整数。小于等于 0 时返回 0。
// value 小于等于 0 时返回 0，否则从 1 递增到平方不小于 value。
func ceilSqrt(value int) int {
	if value <= 0 {
		return 0
	}
	root := 1
	for root*root < value {
		root++
	}
	return root
}

// ceilDiv 做向上取整的整数除法。除数小于等于 0 时返回 0。
// value 加上 divisor 减 1 再除以 divisor。
// divisor 小于等于 0 时返回 0。
func ceilDiv(value, divisor int) int {
	if divisor <= 0 {
		return 0
	}
	return (value + divisor - 1) / divisor
}
