// 为一次文件改动生成 FileChange。行 diff 的计算在 internal/textdiff；把 diff 打回工作区在 hunks.go。

package runtime

import (
	"path/filepath"

	"github.com/Shenchangxin/yoyo/internal/textdiff"
)

// fileChangeOf 把路径改成斜杠，并用 textdiff.Unified 填 Added、Removed 和 Patch。created 原样写入。
// rel 转成斜杠后写入 Paths，并作为补丁里 --- 和 +++ 行的路径。
// prev 转成字符串后作为旧文本。
// next 转成字符串后作为新文本。
// 补丁行数按 textdiff.DefaultMaxHunk 截断。
func fileChangeOf(rel string, prev, next []byte, created bool) *FileChange {
	rel = filepath.ToSlash(rel)
	added, removed, patch := textdiff.Unified(rel, string(prev), string(next), textdiff.DefaultMaxHunk)
	return &FileChange{
		Paths:   []string{rel},
		Patch:   patch,
		Added:   added,
		Removed: removed,
		Created: created,
	}
}
