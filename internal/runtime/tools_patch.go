// 应用 *** Add File 和 *** Update File 这种补丁。更新是整段唯一匹配后替换，不是 git apply。路径经 resolve，写权限是 WriteWorkspace。

package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// applyPatch 把补丁按 *** Begin Patch 切开。Add File 交给 writeFile，因此同样受 writeFilePartMax（32 * 1024 字节）限制。Update File 交给 applyUpdate。没有块则返回 no patch hunks。上下文占位补丁交给 resolveElidedWrite。
func (t *WorkspaceTools) applyPatch(patch string) ToolResult {
	patch = strings.ReplaceAll(patch, "\r\n", "\n")
	if looksLikeContextStub(patch) {
		return t.resolveElidedWrite("", "patch", "")
	}
	if strings.TrimSpace(patch) == "" {
		return ToolResult{Err: fmt.Errorf("empty patch")}
	}
	var reports []string
	var changes []*FileChange
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
			if res.FileChange != nil {
				changes = append(changes, res.FileChange)
			}
		case "update":
			res := t.applyUpdate(path, body)
			if res.Err != nil {
				return res
			}
			reports = append(reports, res.Content)
			if res.FileChange != nil {
				changes = append(changes, res.FileChange)
			}
		default:
			return ToolResult{Err: fmt.Errorf("unknown patch op %s", kind)}
		}
	}
	if len(reports) == 0 {
		return ToolResult{Err: fmt.Errorf("no patch hunks")}
	}
	added, removed := 0, 0
	created := false
	for _, r := range changes {
		added += r.Added
		removed += r.Removed
		created = created || r.Created
	}
	return ToolResult{
		Content: strings.Join(reports, "\n"),
		FileChange: &FileChange{
			Paths:   pathsFromReports(reports),
			Patch:   patch,
			Added:   added,
			Removed: removed,
			Created: created,
		},
	}
}

// pathsFromReports 取每条报告的最后一个字段当路径。
// reports 每一条按空白切开，取最后一个字段当路径。
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

// applyUpdate 经 resolve 和 WriteWorkspace 读取原文件。hunk 必须唯一匹配。写入前 snapshotBeforeWrite，成功后 absorbPathRules。
// rel 交给 resolve；匹配失败时写进错误文本，写入前交给 snapshotBeforeWrite。
// body 交给 hunkReplace，得到要替换的旧文本和新文本。
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
	t.absorbPathRules(rel)
	return ToolResult{Content: "updated " + rel, FileChange: fileChangeOf(rel, raw, []byte(next), false)}
}

// splitPatches 在没有 *** Begin Patch 时把整段当成一块。有标记时按它切开，并去掉 *** End Patch 之后的内容。
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

// parsePatchBlock 只接受 *** Add File: 和 *** Update File:。Add 收集加号行，以及不是 *** 或 @@ 的行。Update 把首行之后整段当正文。
// block 去掉空白后按行切开；第一行必须是 *** Add File: 或 *** Update File:。
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

// hunkReplace 把减号行和空格行收成旧文本，加号行和空格行收成新文本。@@ 和 *** 行跳过。
// body 按行切开；减号和空格收成旧文本，加号和空格收成新文本，@@ 和 *** 跳过。
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

// fileContextHint 用旧文本的第一行非空内容在文件里找最接近的窗口，前后各最多 4 行和 5 行。找不到则返回文件开头最多 16 行。
// old 交给 firstNonEmptyLine，得到的针在文件行里查找。
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

// firstNonEmptyLine 返回第一行去掉空白后非空的内容。没有则返回空串。
func firstNonEmptyLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return t
		}
	}
	return ""
}
