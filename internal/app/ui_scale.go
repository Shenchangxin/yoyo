// 界面缩放档位。只认 1、1.25、1.5 三档，调用方在保存外观时使用。

package app

import "math"

// uiScaleSteps 是界面缩放的三档：1、1.25、1.5。
var uiScaleSteps = []float64{1, 1.25, 1.5}

// snapUIScale 把数值吸到最近的一档。小于等于 0、NaN 或 Inf 返回 1。
func snapUIScale(v float64) float64 {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 1
	}
	best := 1.0
	dist := math.Abs(v - best)
	for _, s := range uiScaleSteps {
		if d := math.Abs(v - s); d < dist {
			dist = d
			best = s
		}
	}
	return best
}
