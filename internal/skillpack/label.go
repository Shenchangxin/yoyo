// 把技能运行时工具调用显示成操作者看到的技能 id。

package skillpack

import (
	"encoding/json"
	"strings"
)

// CallLabel 给 load_skill、read_skill_file、run_skill_script、list_skills 抽出要显示的技能名。其他工具返回空串，调用方继续用宿主工具名。参数里没有名字时退回工具名本身。
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

// jsonString 从 JSON 对象里取出字符串字段。正文为空、不是对象或字段不是字符串时返回空串。
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
