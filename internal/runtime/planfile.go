// 会话计划落在工作区 .yoyo/plans 下，文件名是 sanitizeID 之后的会话号加 .md。
// 不解释计划正文。session.go、session_ops.go、session_inventory.go 和 tools_codex.go 调用这里。

package runtime

import (
	"os"
	"path/filepath"
	"strings"
)

// PlanFilePath 返回计划文件路径。workspace 或 sessionID 为空时返回空串。
// 文件名只保留字母、数字、减号和下划线；滤完为空则用 x，长过 64 个字符则截断。
func PlanFilePath(workspace, sessionID string) string {
	if workspace == "" || sessionID == "" {
		return ""
	}
	return filepath.Join(workspace, ".yoyo", "plans", sanitizeID(sessionID)+".md")
}

// WritePlanFile 创建目录并把去掉首尾空白的正文加换行写进计划文件。路径为空或写入失败时直接返回。
func WritePlanFile(workspace, sessionID, text string) {
	path := PlanFilePath(workspace, sessionID)
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(strings.TrimSpace(text)+"\n"), 0o644)
}

// LoadPlanFile 读出计划并去掉首尾空白。路径为空或读失败时返回空串。
func LoadPlanFile(workspace, sessionID string) string {
	path := PlanFilePath(workspace, sessionID)
	if path == "" {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
