package runtime

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// DiffHunk is one @@ hunk from a unified git diff. IDs are stable for a
// given raw diff so the UI can accept/reject without round-tripping git.
type DiffHunk struct {
	ID     string `json:"id"`
	File   string `json:"file"`
	Header string `json:"header"`
	Body   string `json:"body"`
}

// ParseDiffHunks splits a `git diff --no-color` into per-hunk records.
func ParseDiffHunks(diff string) []DiffHunk {
	diff = strings.ReplaceAll(diff, "\r\n", "\n")
	if strings.TrimSpace(diff) == "" || strings.HasPrefix(strings.TrimSpace(diff), "(no unstaged") {
		return nil
	}
	var out []DiffHunk
	file := ""
	inHunk := false
	var header, body strings.Builder
	flush := func() {
		if !inHunk || header.Len() == 0 {
			return
		}
		id := file + ":" + strconv.Itoa(len(out)+1)
		out = append(out, DiffHunk{
			ID:     id,
			File:   file,
			Header: strings.TrimRight(header.String(), "\n"),
			Body:   strings.TrimRight(body.String(), "\n"),
		})
		header.Reset()
		body.Reset()
		inHunk = false
	}
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flush()
			file = gitDiffPath(line)
		case strings.HasPrefix(line, "@@"):
			flush()
			inHunk = true
			header.WriteString(line)
			header.WriteByte('\n')
		case inHunk:
			if strings.HasPrefix(line, "diff --git ") {
				flush()
				continue
			}
			body.WriteString(line)
			body.WriteByte('\n')
		default:
			if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
				if n := fileFromPlusMinus(line); n != "" && !strings.HasPrefix(line, "---") {
					file = n
				}
				if strings.HasPrefix(line, "+++ ") {
					if n := fileFromPlusMinus(line); n != "" {
						file = n
					}
				}
			}
		}
	}
	flush()
	return out
}

func gitDiffPath(line string) string {
	// diff --git a/foo b/foo
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return ""
	}
	b := strings.TrimPrefix(fields[len(fields)-1], "b/")
	return b
}

func fileFromPlusMinus(line string) string {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "+++ ") {
		rest := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
		rest = strings.TrimPrefix(rest, "b/")
		if i := strings.Index(rest, "\t"); i >= 0 {
			rest = rest[:i]
		}
		if rest == "/dev/null" {
			return ""
		}
		return rest
	}
	return ""
}

// PatchFromHunks rebuilds a git-apply-able unified diff containing only ids.
func PatchFromHunks(diff string, ids []string) (string, error) {
	want := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			want[id] = true
		}
	}
	if len(want) == 0 {
		return "", fmt.Errorf("no hunks selected")
	}
	hunks := ParseDiffHunks(diff)
	byFile := map[string][]DiffHunk{}
	var order []string
	for _, h := range hunks {
		if !want[h.ID] {
			continue
		}
		if _, ok := byFile[h.File]; !ok {
			order = append(order, h.File)
		}
		byFile[h.File] = append(byFile[h.File], h)
		delete(want, h.ID)
	}
	if len(want) > 0 {
		var missing []string
		for id := range want {
			missing = append(missing, id)
		}
		return "", fmt.Errorf("unknown hunk ids: %s", strings.Join(missing, ", "))
	}
	var b strings.Builder
	for _, file := range order {
		hs := byFile[file]
		fmt.Fprintf(&b, "diff --git a/%s b/%s\n", file, file)
		fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", file, file)
		for _, h := range hs {
			b.WriteString(h.Header)
			b.WriteByte('\n')
			if h.Body != "" {
				b.WriteString(h.Body)
				b.WriteByte('\n')
			}
		}
	}
	return b.String(), nil
}

// ApplyHunks applies selected hunks from a workspace git diff. Paths are
// jailed to workspace; this is not a sandbox, just the same path jail as tools.
func ApplyHunks(workspace, diff string, ids []string) error {
	if workspace == "" {
		return fmt.Errorf("empty workspace")
	}
	patch, err := PatchFromHunks(diff, ids)
	if err != nil {
		return err
	}
	for _, h := range ParseDiffHunks(patch) {
		target := filepath.Join(workspace, filepath.FromSlash(h.File))
		if !capability.WithinWorkspace(workspace, target) {
			return fmt.Errorf("hunk path outside workspace: %s", h.File)
		}
	}
	cmd := exec.Command("git", "-C", workspace, "apply", "--unidiff-zero", "--whitespace=nowarn", "-")
	cmd.Stdin = strings.NewReader(patch)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	// Fallback: git apply without --unidiff-zero (normal unified hunks).
	cmd = exec.Command("git", "-C", workspace, "apply", "--whitespace=nowarn", "-")
	cmd.Stdin = strings.NewReader(patch)
	out2, err2 := cmd.CombinedOutput()
	if err2 == nil {
		return nil
	}
	return fmt.Errorf("git apply: %s %s", strings.TrimSpace(string(out)), strings.TrimSpace(string(out2)))
}
