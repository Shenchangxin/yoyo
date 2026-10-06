package skillpack

import (
	"encoding/json"
	"strings"
)

// CallLabel is the operator-facing skill id for a skill-runtime tool.
// Empty means the call is not a skill tool — callers keep the host name.
func CallLabel(tool, argsJSON string) string {
	tool = strings.TrimSpace(tool)
	name := jsonString(argsJSON, "name")
	skill := jsonString(argsJSON, "skill")
	path := jsonString(argsJSON, "path")
	script := jsonString(argsJSON, "script")
	switch tool {
	case "load_skill":
		if name != "" {
			return name
		}
		return tool
	case "read_skill_file":
		if skill != "" && path != "" {
			return skill + " · " + path
		}
		if skill != "" {
			return skill
		}
		return tool
	case "run_skill_script":
		if skill != "" && script != "" {
			return skill + " · " + script
		}
		if skill != "" {
			return skill
		}
		if script != "" {
			return script
		}
		return tool
	case "list_skills":
		return tool
	default:
		return ""
	}
}

func jsonString(raw, key string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var m map[string]any
	if json.Unmarshal([]byte(raw), &m) != nil {
		return ""
	}
	s, _ := m[key].(string)
	return strings.TrimSpace(s)
}
