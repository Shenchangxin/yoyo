// score.go 按场景切点和语音占比挑选不重叠的高光窗口。

package cut

import (
	"fmt"
	"math"
	"sort"
)

// Propose 挑出至多 count 个窗口。count 小于等于 0 时用 3，大于 12 时用 12。target 小于等于 0 时用 20 秒，并且不超过片长。片长小于等于 0.5 秒时返回空切片。没有场景也没有静音时才等分。按分数从高到低贪心选取，重叠间隙 0.6 秒，最后按开始时间排序，标题写成 clip 加序号。
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

// candidateWindows 用场景点和静音结束做边界。距两端 0.35 秒以内的点丢掉，相邻不足 0.25 秒的合并。短于 2.4 秒的段丢掉。不超过目标 1.35 倍的段整段保留，理由是 scene；更长的按步长滑窗，步长是目标的 0.45 倍和 4 秒中的较大者，理由是 window。
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

// equalFallback 把片长等分成 count 段，每段不超过目标和步长，短于 2.4 秒就停止。理由写成 fallback。
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

// scoreClip 用 0.55 的语音占比、0.35 的时长贴合和 0.10 的时长分给片段打分，并写入 hook、information、emotion、completeness、energy。时长达到 12 秒 completeness 为 5，达到 6 秒为 4，否则为 3。理由为空时写成 score。
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

// speechRatio 返回窗口里非静音的比例。没有静音数据时返回 0.72。时长小于等于 0 时返回 0。结果夹在 0 和 1 之间。
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

// greedyNonOverlap 按当前顺序取不重叠的片段，重叠判定留出 minGap，取满 n 个就停。
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

// clampScore 把数值四舍五入后夹到 1 和 5。
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
