// edit.go 从剪辑表里挖掉或插回一段时间。

package cut

// Punch 从每个片段里去掉 [start, end)。窗口短于等于 0.05 秒时原表返回。被整段盖住的片段丢掉；被切到的片段清掉媒体哈希，中间切开时两段的标识也清掉，以便下次套用时从源片重新抽出。
// cuts 逐段与窗口比较，整段盖住的丢掉，被切到的清掉 Hash，从中间切开的两段再清掉 ID。
func Punch(cuts []Clip, start, end float64) []Clip {
	const eps = 0.05
	if end-start <= eps {
		return cuts
	}
	out := make([]Clip, 0, len(cuts)+1)
	for _, c := range cuts {
		if end <= c.Start+eps || start >= c.End-eps {
			out = append(out, c)
			continue
		}
		if start <= c.Start+eps && end >= c.End-eps {
			continue
		}
		if start <= c.Start+eps {
			c.Start = end
			c.Hash = ""
			out = append(out, c)
			continue
		}
		if end >= c.End-eps {
			c.End = start
			c.Hash = ""
			out = append(out, c)
			continue
		}
		left := c
		left.End = start
		left.Hash = ""
		left.ID = ""
		right := c
		right.Start = end
		right.Hash = ""
		right.ID = ""
		out = append(out, left, right)
	}
	return out
}

// InsertSpan 把 [start, end) 按开始时间插回剪辑表。窗口短于等于 0.05 秒时原表返回。新片段的标题是 restored。
// cuts 先复刻，再追加一段后按 Start 插入排序。
func InsertSpan(cuts []Clip, start, end float64) []Clip {
	if end-start <= 0.05 {
		return cuts
	}
	next := append(append([]Clip{}, cuts...), Clip{Start: start, End: end, Title: "restored"})
	for i := 1; i < len(next); i++ {
		j := i
		for j > 0 && next[j].Start < next[j-1].Start {
			next[j], next[j-1] = next[j-1], next[j]
			j--
		}
	}
	return next
}
