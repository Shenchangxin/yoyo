// 把 git diff 拆成可选择的 hunk，并用 git apply 打回工作区。失败不 reset。工作区隔离和整树回写在 worktree.go。

package runtime

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// DiffHunk 是统一 diff 里的一个 @@ 块。同一份原始 diff 上的 ID 稳定，界面可以按 ID 接受或拒绝，不必再跑一遍 git。
type DiffHunk struct {
	ID     string `json:"id"`
	File   string `json:"file"`
	Header string `json:"header"`
	Body   string `json:"body"`
}

// ParseDiffHunks 把 git diff --no-color 拆成逐 hunk 记录。空文本或以 (no unstaged 开头时返回 nil。ID 是文件路径加冒号再加序号。
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

// gitDiffPath 从 diff --git 行取最后一段，并去掉 b/ 前缀。字段不足四段时返回空串。
func gitDiffPath(line string) string {
	// diff --git 行至少四段，最后一段去掉 b/ 前缀就是路径。
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return ""
	}
	b := strings.TrimPrefix(fields[len(fields)-1], "b/")
	return b
}

// fileFromPlusMinus 只从 +++ 行取路径。去掉 b/ 和制表符之后的内容。/dev/null 返回空串。--- 行返回空串。
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

// PatchFromHunks 按 ids 重组一份 git apply 能读的统一 diff。没有选中、或有未知 ID 时返回错误。每个文件写成 diff --git、---、+++ 再接选中的 hunk。
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

// ApplyHunks 把所选 hunk 打进工作区。先 PatchFromHunks，再用 WithinWorkspace 检查每个路径。执行 git apply：先带 --unidiff-zero 和 --whitespace=nowarn，失败后再只带 --whitespace=nowarn。两次都失败就返回 git apply 错误，不执行 git reset。
func ApplyHunks(workspace, diff string, ids []string) error {
	return applyHunks(workspace, diff, ids, false)
}

// ReverseApplyHunks 用 applyHunks 的 reverse 路径撤销已应用的选择，git apply 会加上 -R。传入的 diff 仍须包含这些 hunk ID。
func ReverseApplyHunks(workspace, diff string, ids []string) error {
	return applyHunks(workspace, diff, ids, true)
}

// applyHunks 在 workspace 为空时返回错误。补丁从标准输入送给 git -C workspace apply -。reverse 为真时在参数里追加 -R。成功即返回，失败不 reset。
func applyHunks(workspace, diff string, ids []string, reverse bool) error {
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
	run := func(extra ...string) (string, error) {
		args := append([]string{"-C", workspace, "apply"}, extra...)
		if reverse {
			args = append(args, "-R")
		}
		args = append(args, "-")
		cmd := gitCmd(args...)
		cmd.Stdin = strings.NewReader(patch)
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := run("--unidiff-zero", "--whitespace=nowarn"); err == nil {
		return nil
	} else {
		if out2, err2 := run("--whitespace=nowarn"); err2 == nil {
			return nil
		} else {
			return fmt.Errorf("git apply: %s %s", out, out2)
		}
	}
}
