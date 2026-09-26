package app

import "math"

var uiScaleSteps = []float64{1, 1.25, 1.5}

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
