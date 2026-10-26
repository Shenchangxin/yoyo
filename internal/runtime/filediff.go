// 为一次文件改动生成统一 diff，写入 FileChange。不做 git apply，那在 hunks.go。超长文件不用完整 LCS，改用锚点或整段替换。

package runtime

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// maxDiffDP、maxDiffHunk、maxDiffRunes 限制行 diff 的表格、输出行数和 rune 数。
const (
	// maxDiffDP 是 800。任一边超过它，或两边乘积超过 800*800，就不用完整 LCS。
	maxDiffDP = 800
	// maxDiffHunk 是 120。输出的上下文加增删行达到这个数就截断，并追加 @@ … @@。
	maxDiffHunk = 120
	// maxDiffRunes 是 8000。补丁 rune 数超过它就截断，并追加 @@ … @@。
	maxDiffRunes = 8_000
)

// fileChangeOf 把路径改成斜杠，并用 unifiedLineDiff 填 Added、Removed 和 Patch。created 原样写入。
// rel 转成斜杠后写入 Paths，并传给 unifiedLineDiff。
// prev 转成字符串后作为旧文本传给 unifiedLineDiff。
// next 转成字符串后作为新文本传给 unifiedLineDiff。
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

// unifiedLineDiff 按行做统一 diff。先把换行收成 \\n。增删都是 0 时补丁为空。每个改动块带最多 2 行上下文。
// path 写进 --- 和 +++ 行。
// oldS 交给 splitDiffLines，作为旧行。
// newS 交给 splitDiffLines，作为新行。
// maxHunk 小于等于 0 时改用 maxDiffHunk，已写出的行数达到它就截断。
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

// diffOp 是一行编辑。tag 为空格、加号或减号。
type diffOp struct {
	tag  byte
	line string
}

// splitDiffLines 去掉 \\r\\n 和单独的 \\r，再按 \\n 切开。空串返回 nil。只有换行时返回一个空行。
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

// diffLines 从深度 0 调用 diffLinesDepth。
// a 传给 diffLinesDepth。
// b 传给 diffLinesDepth。
func diffLines(a, b []string) []diffOp {
	return diffLinesDepth(a, b, 0)
}

// maxAnchorDepth 是 32。锚点递归超过这个深度就改为整段替换。
const maxAnchorDepth = 32

// diffLinesDepth 先剥掉两端相同的行，中间交给 diffLinesMiddle，再把两端标成未改。
// a 与 b 剥掉两端相同行，相同部分标成未改，中间切片传给 diffLinesMiddle。
// b 同样参与两端比较，中间切片传给 diffLinesMiddle。
// depth 传给 diffLinesMiddle。
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

// diffLinesMiddle 在一边为空时把另一边全部标成新增或删除。装得下 LCS 就用 lcsDiff。否则在深度允许时找唯一锚点再递归，找不到就 coarseReplace。
// b 在 a 为空时全部标成新增；两边都非空时参与 LCS、锚点或整段替换。
// depth 小于 maxAnchorDepth 时才找唯一锚点并加一再递归。
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

// lcsFits 在两边都不超过 maxDiffDP 且乘积不超过 maxDiffDP*maxDiffDP 时返回真。
// n 超过 maxDiffDP，或与另一边的乘积超过 maxDiffDP 的平方时返回 false。
func lcsFits(n, m int) bool {
	if n > maxDiffDP || m > maxDiffDP {
		return false
	}
	return n*m <= maxDiffDP*maxDiffDP
}

// coarseReplace 把 a 全部标成删除，b 全部标成新增。
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

// uniqueAnchor 找两边都只出现一次的行，选离两边中点距离之和最小的那一行当锚点。没有则 ok 为假。
// a 统计每行出现次数和位置，并参与中点距离。
// b 同样统计，只保留两边都只出现一次的行。
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

// absInt 返回非负整数。
func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// lcsDiff 用从后往前的 LCS 长度表回溯，相等行标成未改，否则按表选择删除或新增。
// a 与 b 逐行比较，相等标成未改，否则按表删除这一行。
// b 与 a 一起建长度表，a 用尽后剩余行标成新增。
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
