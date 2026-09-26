package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

func (t *WorkspaceTools) applyPatch(patch string) ToolResult {
	patch = strings.ReplaceAll(patch, "\r\n", "\n")
	if strings.TrimSpace(patch) == "" {
		return ToolResult{Err: fmt.Errorf("empty patch")}
	}
	var reports []string
	for _, block := range splitPatches(patch) {
		kind, path, body, err := parsePatchBlock(block)
		if err != nil {
			return ToolResult{Err: err}
		}
		switch kind {
		case "add":
			res := t.writeFile(path, body)
			if res.Err != nil {
				return res
			}
			reports = append(reports, "added "+path)
		case "update":
			res := t.applyUpdate(path, body)
			if res.Err != nil {
				return res
			}
			reports = append(reports, res.Content)
		default:
			return ToolResult{Err: fmt.Errorf("unknown patch op %s", kind)}
		}
	}
	if len(reports) == 0 {
		return ToolResult{Err: fmt.Errorf("no patch hunks")}
	}
	return ToolResult{
		Content:    strings.Join(reports, "\n"),
		FileChange: &FileChange{Paths: pathsFromReports(reports), Patch: patch},
	}
}

func pathsFromReports(reports []string) []string {
	var out []string
	for _, r := range reports {
		fields := strings.Fields(r)
		if len(fields) > 0 {
			out = append(out, fields[len(fields)-1])
		}
	}
	return out
}

func (t *WorkspaceTools) applyUpdate(rel, body string) ToolResult {
	p, err := t.resolve(rel)
	if err != nil {
		return ToolResult{Err: err}
	}
	if err := t.check(capability.WriteWorkspace, "apply_patch", p, ""); err != nil {
		return ToolResult{Err: err}
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return ToolResult{Err: err}
	}
	text := string(raw)
	old, new, err := hunkReplace(body)
	if err != nil {
		return ToolResult{Err: err}
	}
	if old == "" {
		return ToolResult{Err: fmt.Errorf("empty old hunk")}
	}
	n := strings.Count(text, old)
	if n == 0 {
		return ToolResult{Err: fmt.Errorf("hunk not found in %s\n%s", rel, fileContextHint(text, old))}
	}
	if n > 1 {
		return ToolResult{Err: fmt.Errorf("hunk matched %d times in %s; add more context", n, rel)}
	}
	next := strings.Replace(text, old, new, 1)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return ToolResult{Err: err}
	}
	t.snapshotBeforeWrite(rel, p)
	if err := os.WriteFile(p, []byte(next), 0o644); err != nil {
		return ToolResult{Err: err}
	}
	return ToolResult{Content: "updated " + rel}
}

func splitPatches(s string) []string {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "*** Begin Patch") {
		return []string{s}
	}
	parts := strings.Split(s, "*** Begin Patch")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if i := strings.Index(p, "*** End Patch"); i >= 0 {
			p = strings.TrimSpace(p[:i])
		}
		out = append(out, p)
	}
	return out
}

func parsePatchBlock(block string) (kind, path, body string, err error) {
	lines := strings.Split(strings.TrimSpace(block), "\n")
	if len(lines) == 0 {
		return "", "", "", fmt.Errorf("empty patch block")
	}
	head := strings.TrimSpace(lines[0])
	switch {
	case strings.HasPrefix(head, "*** Add File:"):
		kind = "add"
		path = strings.TrimSpace(strings.TrimPrefix(head, "*** Add File:"))
		var b strings.Builder
		for _, ln := range lines[1:] {
			if strings.HasPrefix(ln, "+") {
				b.WriteString(strings.TrimPrefix(ln, "+"))
				b.WriteByte('\n')
			} else if !strings.HasPrefix(ln, "***") && !strings.HasPrefix(ln, "@@") {
				b.WriteString(ln)
				b.WriteByte('\n')
			}
		}
		return kind, path, strings.TrimSuffix(b.String(), "\n"), nil
	case strings.HasPrefix(head, "*** Update File:"):
		kind = "update"
		path = strings.TrimSpace(strings.TrimPrefix(head, "*** Update File:"))
		return kind, path, strings.Join(lines[1:], "\n"), nil
	default:
		return "", "", "", fmt.Errorf("patch must start with *** Add File or *** Update File")
	}
}

func hunkReplace(body string) (old, new string, err error) {
	var oldB, newB strings.Builder
	for _, ln := range strings.Split(body, "\n") {
		if strings.HasPrefix(ln, "@@") || strings.HasPrefix(ln, "***") {
			continue
		}
		if strings.HasPrefix(ln, "+") {
			newB.WriteString(strings.TrimPrefix(ln, "+"))
			newB.WriteByte('\n')
			continue
		}
		if strings.HasPrefix(ln, "-") {
			oldB.WriteString(strings.TrimPrefix(ln, "-"))
			oldB.WriteByte('\n')
			continue
		}
		if strings.HasPrefix(ln, " ") {
			line := strings.TrimPrefix(ln, " ")
			oldB.WriteString(line)
			oldB.WriteByte('\n')
			newB.WriteString(line)
			newB.WriteByte('\n')
		}
	}
	return strings.TrimSuffix(oldB.String(), "\n"), strings.TrimSuffix(newB.String(), "\n"), nil
}

func fileContextHint(text, old string) string {
	needle := firstNonEmptyLine(old)
	lines := strings.Split(text, "\n")
	if needle != "" {
		for i, ln := range lines {
			if strings.Contains(ln, needle) {
				start := i - 4
				if start < 0 {
					start = 0
				}
				end := i + 5
				if end > len(lines) {
					end = len(lines)
				}
				return "closest existing lines:\n" + strings.Join(lines[start:end], "\n")
			}
		}
	}
	if len(lines) > 16 {
		lines = lines[:16]
	}
	return "file starts with:\n" + strings.Join(lines, "\n")
}

func firstNonEmptyLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return t
		}
	}
	return ""
}
