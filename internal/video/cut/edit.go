package cut

// Punch removes [start, end) from each clip. Trimmed clips drop their media
// hash so the next apply re-extracts from the source.
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

// InsertSpan puts [start, end) back into the edit list, in time order.
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
