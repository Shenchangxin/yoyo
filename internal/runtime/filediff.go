package runtime

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	maxDiffDP    = 800
	maxDiffHunk  = 120
	maxDiffRunes = 8_000
)

func fileChangeOf(rel string, prev, next []byte, created bool) *FileChange {
	rel = filepath.ToSlash(rel)
	added, removed, patch := unifiedLineDiff(rel, string(prev), string(next), maxDiffHunk)
	return &FileChange{
		Paths:   []string{rel},
		Patch:   patch,
		Added:   added,
		Removed: removed,
		Created: created,
	}
}

func unifiedLineDiff(path, oldS, newS string, maxHunk int) (added, removed int, patch string) {
	oldL := splitDiffLines(oldS)
	newL := splitDiffLines(newS)
	ops := diffLines(oldL, newL)
	for _, op := range ops {
		switch op.tag {
		case '+':
			added++
		case '-':
			removed++
		}
	}
	if added == 0 && removed == 0 {
		return 0, 0, ""
	}
	if maxHunk <= 0 {
		maxHunk = maxDiffHunk
	}
	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", path, path)
	oldStart, newStart := 1, 1
	i := 0
	emitted := 0
	truncated := false
	for i < len(ops) {
		if ops[i].tag == ' ' {
			oldStart++
			newStart++
			i++
			continue
		}
		j := i
		ctx := 2
		start := i
		for start > 0 && ctx > 0 && ops[start-1].tag == ' ' {
			start--
			ctx--
		}
		for j < len(ops) && ops[j].tag != ' ' {
			j++
		}
		end := j
		tail := 2
		for end < len(ops) && tail > 0 && ops[end].tag == ' ' {
			end++
			tail--
		}
		oldN, newN := 0, 0
		for _, op := range ops[start:end] {
			if op.tag != '+' {
				oldN++
			}
			if op.tag != '-' {
				newN++
			}
		}
		oldAt := oldStart - (i - start)
		newAt := newStart - (i - start)
		if oldAt < 1 {
			oldAt = 1
		}
		if newAt < 1 {
			newAt = 1
		}
		if emitted >= maxHunk {
			truncated = true
			break
		}
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", oldAt, oldN, newAt, newN)
		for _, op := range ops[start:end] {
			if emitted >= maxHunk {
				truncated = true
				break
			}
			switch op.tag {
			case ' ':
				b.WriteString(" ")
			case '+':
				b.WriteString("+")
			case '-':
				b.WriteString("-")
			}
			b.WriteString(op.line)
			b.WriteByte('\n')
			emitted++
		}
		for k := i; k < j; k++ {
			if ops[k].tag != '+' {
				oldStart++
			}
			if ops[k].tag != '-' {
				newStart++
			}
		}
		i = j
	}
	out := strings.TrimRight(b.String(), "\n")
	if truncated {
		out += "\n@@ … @@"
	}
	if utf8.RuneCountInString(out) > maxDiffRunes {
		r := []rune(out)
		out = string(r[:maxDiffRunes]) + "\n@@ … @@"
	}
	return added, removed, out
}

type diffOp struct {
	tag  byte
	line string
}

func splitDiffLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "")
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "\n")
}

func diffLines(a, b []string) []diffOp {
	return diffLinesDepth(a, b, 0)
}

const maxAnchorDepth = 32

func diffLinesDepth(a, b []string, depth int) []diffOp {
	n, m := len(a), len(b)
	if n == 0 && m == 0 {
		return nil
	}
	pre := 0
	for pre < n && pre < m && a[pre] == b[pre] {
		pre++
	}
	asuf, bsuf := n, m
	for asuf > pre && bsuf > pre && a[asuf-1] == b[bsuf-1] {
		asuf--
		bsuf--
	}
	mid := diffLinesMiddle(a[pre:asuf], b[pre:bsuf], depth)
	out := make([]diffOp, 0, pre+(n-asuf)+len(mid))
	for i := 0; i < pre; i++ {
		out = append(out, diffOp{' ', a[i]})
	}
	out = append(out, mid...)
	for i := asuf; i < n; i++ {
		out = append(out, diffOp{' ', a[i]})
	}
	return out
}

func diffLinesMiddle(a, b []string, depth int) []diffOp {
	n, m := len(a), len(b)
	if n == 0 && m == 0 {
		return nil
	}
	if n == 0 {
		out := make([]diffOp, m)
		for i, line := range b {
			out[i] = diffOp{'+', line}
		}
		return out
	}
	if m == 0 {
		out := make([]diffOp, n)
		for i, line := range a {
			out[i] = diffOp{'-', line}
		}
		return out
	}
	if lcsFits(n, m) {
		return lcsDiff(a, b)
	}
	if depth < maxAnchorDepth {
		if ai, bi, ok := uniqueAnchor(a, b); ok {
			left := diffLinesDepth(a[:ai], b[:bi], depth+1)
			right := diffLinesDepth(a[ai+1:], b[bi+1:], depth+1)
			out := make([]diffOp, 0, len(left)+1+len(right))
			out = append(out, left...)
			out = append(out, diffOp{' ', a[ai]})
			out = append(out, right...)
			return out
		}
	}
	return coarseReplace(a, b)
}

func lcsFits(n, m int) bool {
	if n > maxDiffDP || m > maxDiffDP {
		return false
	}
	return n*m <= maxDiffDP*maxDiffDP
}

func coarseReplace(a, b []string) []diffOp {
	out := make([]diffOp, 0, len(a)+len(b))
	for _, line := range a {
		out = append(out, diffOp{'-', line})
	}
	for _, line := range b {
		out = append(out, diffOp{'+', line})
	}
	return out
}

func uniqueAnchor(a, b []string) (ai, bi int, ok bool) {
	acount := make(map[string]int, len(a))
	bcount := make(map[string]int, len(b))
	apos := make(map[string]int, len(a))
	bpos := make(map[string]int, len(b))
	for i, line := range a {
		acount[line]++
		apos[line] = i
	}
	for i, line := range b {
		bcount[line]++
		bpos[line] = i
	}
	midA, midB := len(a)/2, len(b)/2
	best := -1
	bestDist := len(a) + len(b) + 1
	for line, c := range acount {
		if c != 1 || bcount[line] != 1 {
			continue
		}
		i, j := apos[line], bpos[line]
		dist := absInt(i-midA) + absInt(j-midB)
		if dist < bestDist {
			bestDist = dist
			best = i
			ai, bi = i, j
		}
	}
	return ai, bi, best >= 0
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func lcsDiff(a, b []string) []diffOp {
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	var out []diffOp
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			out = append(out, diffOp{' ', a[i]})
			i++
			j++
			continue
		}
		if lcs[i+1][j] >= lcs[i][j+1] {
			out = append(out, diffOp{'-', a[i]})
			i++
		} else {
			out = append(out, diffOp{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, diffOp{'-', a[i]})
	}
	for ; j < m; j++ {
		out = append(out, diffOp{'+', b[j]})
	}
	return out
}
