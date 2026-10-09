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
	if n > maxDiffDP || m > maxDiffDP || n*m > maxDiffDP*maxDiffDP {
		out := make([]diffOp, 0, n+m)
		for _, line := range a {
			out = append(out, diffOp{'-', line})
		}
		for _, line := range b {
			out = append(out, diffOp{'+', line})
		}
		return out
	}
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
