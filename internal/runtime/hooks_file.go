package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const maxInlineBytes = 256 * 1024

// FileHook is a deterministic workspace policy (Claude/Cursor-style hooks),
// not an LLM policy. Loaded from .yoyo/hooks.json.
type FileHook struct {
	Match  string `json:"match"`
	Deny   bool   `json:"deny"`
	Reason string `json:"reason,omitempty"`
}

type hookFile struct {
	PreTool []FileHook `json:"pre_tool"`
}

func LoadFileHooks(workspace string) []FileHook {
	if workspace == "" {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(workspace, ".yoyo", "hooks.json"))
	if err != nil {
		return nil
	}
	var hf hookFile
	if json.Unmarshal(b, &hf) != nil {
		return nil
	}
	return hf.PreTool
}

func applyFileHooks(rules []FileHook, hook ToolHook) ToolHook {
	for _, r := range rules {
		if !hookMatch(r.Match, hook.Name) {
			continue
		}
		if r.Deny {
			hook.Deny = true
			hook.Reason = r.Reason
			if hook.Reason == "" {
				hook.Reason = "denied by .yoyo/hooks.json"
			}
			return hook
		}
	}
	return hook
}

func hookMatch(pattern, name string) bool {
	if pattern == "" || pattern == "*" || pattern == name {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, strings.TrimSuffix(pattern, "*"))
	}
	return false
}
