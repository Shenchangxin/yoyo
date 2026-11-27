// layout_test.go 锁住泳道顺序、依赖从左到右、分带、对齐和空位避让。不读画布文档。

package layout

import (
	"math"
	"testing"
)

// node 用标识、泳道和矩形造一个节点。
func node(id, lane string, x, y, w, h float64) Node {
	return Node{ID: id, Lane: lane, X: x, Y: y, Width: w, Height: h}
}

// assertPosition 确认该节点的坐标与期望相差不超过 0.001。
func assertPosition(t *testing.T, positions map[string]Position, id string, x, y float64) {
	t.Helper()
	position, ok := positions[id]
	if !ok {
		t.Fatalf("缺少节点 %s 的位置：%+v", id, positions)
	}
	if math.Abs(position.X-x) > 0.001 || math.Abs(position.Y-y) > 0.001 {
		t.Fatalf("节点 %s 位置 = (%v,%v)，期望 (%v,%v)", id, position.X, position.Y, x, y)
	}
}

// TestByLaneKeepsLaneOrderAndAvoidsOverlap 确认与前端 layoutCanvasAuto 的媒体泳道一致：text、image、video、audio 自上而下，且没有重叠。
func TestByLaneKeepsLaneOrderAndAvoidsOverlap(t *testing.T) {
	nodes := []Node{
		node("text-1", "text", 0, 0, 340, 240),
		node("image-1", "image", 900, 500, 720, 405),
		node("video-1", "video", 900, 1200, 720, 405),
		node("audio-1", "audio", 900, 1800, 340, 120),
	}
	positions := ByLane(nodes)

	assertPosition(t, positions, "text-1", 0, 0)
	// text 泳道一行高 240，之后是本泳道与下一泳道之间的 LaneGap
	assertPosition(t, positions, "image-1", 0, 240+LaneGap)
	// image 泳道一行高 405，其后同样是 LaneGap
	imageBottom := 240.0 + LaneGap + 405
	assertPosition(t, positions, "video-1", 0, imageBottom+LaneGap)
	assertPosition(t, positions, "audio-1", 0, imageBottom+LaneGap+405+LaneGap)

	if pairs := Overlaps(withPositions(nodes, positions), 0); len(pairs) != 0 {
		t.Fatalf("泳道排布出现重叠：%+v", pairs)
	}
}

// TestFlowLaysOutDependenciesLeftToRight 确认与前端 layoutCanvasFlow 一致：同一层横坐标相同，依赖节点在右侧，带回边时第一层节点仍在。
func TestFlowLaysOutDependenciesLeftToRight(t *testing.T) {
	nodes := []Node{
		node("text-1", "text", 0, 0, 340, 240),
		node("image-1", "image", 0, 400, 720, 405),
		node("video-1", "video", 0, 900, 720, 405),
	}
	edges := []Edge{{From: "text-1", To: "video-1"}, {From: "image-1", To: "video-1"}}
	positions := Flow(nodes, edges)

	textX := positions["text-1"].X
	imageX := positions["image-1"].X
	videoX := positions["video-1"].X
	if textX != imageX {
		t.Fatalf("同层起点应一致：text=%v image=%v", textX, imageX)
	}
	wantVideoX := textX + 720 + FlowColumnGap
	if math.Abs(videoX-wantVideoX) > 0.001 {
		t.Fatalf("下一层 x = %v，期望 %v", videoX, wantVideoX)
	}
	// 没有入边的自环也要落在第一层，不能死循环
	cyclic := Flow(nodes, append(edges, Edge{From: "video-1", To: "text-1"}))
	if _, ok := cyclic["text-1"]; !ok {
		t.Fatal("含环时第一层节点丢失")
	}
}

// TestAutoFallsBackToLanesWithoutConnections 确认没有有效连线时按泳道排在同一行，有连线时左右分开。
func TestAutoFallsBackToLanesWithoutConnections(t *testing.T) {
	nodes := []Node{node("image-1", "image", 0, 0, 720, 405), node("image-2", "image", 800, 0, 720, 405)}
	edges := []Edge{{From: "image-1", To: "missing"}} // 集合外的连线不算

	positions := Auto(nodes, edges)
	if positions["image-1"].Y != positions["image-2"].Y {
		t.Fatalf("无有效连线时应按泳道横排：%+v", positions)
	}

	connected := Auto(nodes, []Edge{{From: "image-1", To: "image-2"}})
	if connected["image-1"].X == connected["image-2"].X {
		t.Fatalf("有连线时应分层左右排开：%+v", connected)
	}
}

// withPositions 把算好的位置写回节点副本。
func withPositions(nodes []Node, positions map[string]Position) []Node {
	applied := make([]Node, 0, len(nodes))
	for _, item := range nodes {
		if position, ok := positions[item.ID]; ok {
			item.X, item.Y = position.X, position.Y
		}
		applied = append(applied, item)
	}
	return applied
}

// overlaps 判断两个节点（含额外空隙）是否相交。
// first 的 X、Y 分别与 second 的 X+width+gap、Y+height+gap 比较。
// second 的 X、Y 分别与 first 的 X+width+gap、Y+height+gap 比较。
// gap 加在双方宽高上再比坐标。
func overlaps(first, second Node, gap float64) bool {
	return first.X < second.X+second.width()+gap &&
		second.X < first.X+first.width()+gap &&
		first.Y < second.Y+second.height()+gap &&
		second.Y < first.Y+first.height()+gap
}

// Overlaps 返回相互重叠（含 gap 余量）的节点对，用于回执里的自查提示。
// nodes 两两传给 overlaps。
func Overlaps(nodes []Node, gap float64) [][2]string {
	pairs := make([][2]string, 0)
	for index := 0; index < len(nodes); index++ {
		for next := index + 1; next < len(nodes); next++ {
			if overlaps(nodes[index], nodes[next], gap) {
				pairs = append(pairs, [2]string{nodes[index].ID, nodes[next].ID})
			}
		}
	}
	return pairs
}
