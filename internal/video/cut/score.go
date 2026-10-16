package cut

import (
	"fmt"
	"math"
	"sort"
)

// Propose picks up to count highlight windows using scene cuts, speech
// (inverse silence), and duration fitness. Equal-time slicing is only the
// last resort when the source has no scene or silence data.
func Propose(dur float64, scenes []float64, silences []Silence, count int, target float64) []Clip {
	if dur <= 0.5 {
		return []Clip{}
	}
	if count <= 0 {
		count = 3
	}
	if count > 12 {
		count = 12
	}
	if target <= 0 {
		target = 20
	}
	if target > dur {
		target = dur
	}
	cands := candidateWindows(dur, scenes, silences, target)
	if len(cands) == 0 {
		cands = equalFallback(dur, count, target)
	}
	for i := range cands {
		cands[i] = scoreClip(cands[i], target, silences)
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].Score > cands[j].Score })
	picked := greedyNonOverlap(cands, count, 0.6)
	sort.SliceStable(picked, func(i, j int) bool { return picked[i].Start < picked[j].Start })
	for i := range picked {
		picked[i].Title = fmt.Sprintf("clip %d", i+1)
	}
	return picked
}

func candidateWindows(dur float64, scenes []float64, silences []Silence, target float64) []Clip {
	bounds := []float64{0}
	for _, s := range scenes {
		if s > 0.35 && s < dur-0.35 {
			bounds = append(bounds, s)
		}
	}
	for _, sil := range silences {
		if sil.End > 0.35 && sil.End < dur-0.35 {
			bounds = append(bounds, sil.End)
		}
	}
	bounds = append(bounds, dur)
	sort.Float64s(bounds)
	uniq := bounds[:0]
	var last float64 = -1
	for _, t := range bounds {
		if last >= 0 && t-last < 0.25 {
			continue
		}
		uniq = append(uniq, t)
		last = t
	}
	bounds = uniq
	var out []Clip
	hop := math.Max(target*0.45, 4)
	for i := 0; i < len(bounds)-1; i++ {
		a, b := bounds[i], bounds[i+1]
		seg := b - a
		if seg < 2.4 {
			continue
		}
		if seg <= target*1.35 {
			out = append(out, Clip{Start: a, End: b, Reason: "scene"})
			continue
		}
		for t := a; t+2.4 < b; t += hop {
			end := t + target
			if end > b {
				end = b
			}
			if end-t < 2.4 {
				break
			}
			out = append(out, Clip{Start: t, End: end, Reason: "window"})
		}
	}
	return out
}

func equalFallback(dur float64, count int, target float64) []Clip {
	step := dur / float64(count)
	span := target
	if span > step {
		span = step
	}
	var out []Clip
	for i := 0; i < count; i++ {
		start := float64(i) * step
		end := start + span
		if end > dur {
			end = dur
		}
		if end-start < 2.4 {
			break
		}
		out = append(out, Clip{Start: start, End: end, Reason: "fallback"})
	}
	return out
}

func scoreClip(c Clip, target float64, silences []Silence) Clip {
	d := c.Dur()
	if d <= 0 {
		return c
	}
	speech := speechRatio(c.Start, c.End, silences)
	durFit := 1 - math.Min(1, math.Abs(d-target)/math.Max(target, 1))
	c.Score = 0.55*speech + 0.35*durFit + 0.10*math.Min(1, d/8)
	hook := clampScore(speech*5 + durFit)
	info := clampScore(3 + speech*2)
	emotion := clampScore(2 + speech*3)
	complete := 3
	if d >= 12 {
		complete = 5
	} else if d >= 6 {
		complete = 4
	}
	energy := clampScore(speech * 5)
	c.Scores = map[string]int{
		"hook":         hook,
		"information":  info,
		"emotion":      emotion,
		"completeness": complete,
		"energy":       energy,
	}
	if c.Reason == "" {
		c.Reason = "score"
	}
	return c
}

func speechRatio(start, end float64, silences []Silence) float64 {
	d := end - start
	if d <= 0 {
		return 0
	}
	if len(silences) == 0 {
		return 0.72
	}
	var muted float64
	for _, s := range silences {
		a := math.Max(start, s.Start)
		b := math.Min(end, s.End)
		if b > a {
			muted += b - a
		}
	}
	r := 1 - muted/d
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}

func greedyNonOverlap(cands []Clip, n int, minGap float64) []Clip {
	var out []Clip
	for _, c := range cands {
		ok := true
		for _, p := range out {
			if c.Start < p.End-minGap && c.End > p.Start+minGap {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		out = append(out, c)
		if len(out) >= n {
			break
		}
	}
	return out
}

func clampScore(v float64) int {
	n := int(math.Round(v))
	if n < 1 {
		return 1
	}
	if n > 5 {
		return 5
	}
	return n
}
