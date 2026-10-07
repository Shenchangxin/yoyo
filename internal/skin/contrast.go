package skin

import (
	"math"
	"strconv"
	"strings"
)

type ContrastReport struct {
	Foreground float64 `json:"foreground"`
	Muted      float64 `json:"muted"`
	Pass       bool    `json:"pass"`
	Reason     string  `json:"reason,omitempty"`
}

func ContrastOf(tokens TokenMap) ContrastReport {
	bg, okBG := parseRGB(tokens["--background"])
	fg, okFG := parseRGB(tokens["--foreground"])
	mu, okMU := parseRGB(tokens["--muted"])
	rep := ContrastReport{Pass: true}
	if okBG && okFG {
		rep.Foreground = contrastRatio(bg, fg)
		if rep.Foreground < 4.5 {
			rep.Pass = false
			rep.Reason = "foreground/background contrast must be at least 4.5:1"
		}
	}
	if okBG && okMU {
		rep.Muted = contrastRatio(bg, mu)
		if rep.Muted < 3 {
			rep.Pass = false
			if rep.Reason == "" {
				rep.Reason = "muted/background contrast must be at least 3:1"
			}
		}
	}
	return rep
}

func parseRGB(s string) ([3]float64, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		return [3]float64{}, false
	}
	h := strings.TrimPrefix(s, "#")
	var r, g, b int64
	var err error
	switch len(h) {
	case 3:
		r, err = strconv.ParseInt(strings.Repeat(h[0:1], 2), 16, 64)
		if err != nil {
			return [3]float64{}, false
		}
		g, err = strconv.ParseInt(strings.Repeat(h[1:2], 2), 16, 64)
		if err != nil {
			return [3]float64{}, false
		}
		b, err = strconv.ParseInt(strings.Repeat(h[2:3], 2), 16, 64)
		if err != nil {
			return [3]float64{}, false
		}
	case 6, 8:
		r, err = strconv.ParseInt(h[0:2], 16, 64)
		if err != nil {
			return [3]float64{}, false
		}
		g, err = strconv.ParseInt(h[2:4], 16, 64)
		if err != nil {
			return [3]float64{}, false
		}
		b, err = strconv.ParseInt(h[4:6], 16, 64)
		if err != nil {
			return [3]float64{}, false
		}
	default:
		return [3]float64{}, false
	}
	return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}, true
}

func WindowRGB(tokens TokenMap) []int {
	rgb, ok := parseRGB(tokens["--background"])
	if !ok {
		return []int{28, 28, 26}
	}
	return []int{int(math.Round(rgb[0] * 255)), int(math.Round(rgb[1] * 255)), int(math.Round(rgb[2] * 255))}
}

func contrastRatio(a, b [3]float64) float64 {
	l1 := relLum(a)
	l2 := relLum(b)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func relLum(c [3]float64) float64 {
	lin := func(v float64) float64 {
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c[0]) + 0.7152*lin(c[1]) + 0.0722*lin(c[2])
}
