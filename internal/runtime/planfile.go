package runtime

import (
	"os"
	"path/filepath"
	"strings"
)

func PlanFilePath(workspace, sessionID string) string {
	if workspace == "" || sessionID == "" {
		return ""
	}
	return filepath.Join(workspace, ".yoyo", "plans", sanitizeID(sessionID)+".md")
}

func WritePlanFile(workspace, sessionID, text string) {
	path := PlanFilePath(workspace, sessionID)
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(strings.TrimSpace(text)+"\n"), 0o644)
}

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
