// 从工作区 .yoyo/hooks.json 读取确定性的文件钩子，按工具名做全名或前缀匹配。
// 不是模型策略。session.go 调用 LoadHookFile。loop.go 把 pre_tool 交给 applyFileHooks。
// pre_compact 只被读出；压缩路径把匹配名放进事件载荷，不把 Deny 交给 applyFileHooks。

package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// maxInlineBytes 是内联正文的上限，256 KiB。tools.go 和 skill_file.go 用它截断读入内容。
const maxInlineBytes = 256 * 1024

// FileHook 是一条工作区钩子。Match 为工具名、空串、* 或以前缀加星号结尾。Deny 为真时拒绝该工具。
type FileHook struct {
	Match  string `json:"match"`
	Deny   bool   `json:"deny"`
	Reason string `json:"reason,omitempty"`
}

// hookFile 是 .yoyo/hooks.json 的形状。PreCompact 只供观察，本文件不执行它的 Deny。
type hookFile struct {
	PreTool    []FileHook `json:"pre_tool"`
	Stop       []FileHook `json:"stop"`
	PreCompact []FileHook `json:"pre_compact"`
}

// LoadFileHooks 只返回 pre_tool 规则。工作区为空、文件不存在或 JSON 无效时返回 nil。
// workspace 传给 LoadHookFile。
func LoadFileHooks(workspace string) []FileHook {
	pre, _, _ := LoadHookFile(workspace)
	return pre
}

// LoadHookFile 返回 pre_tool、stop 和 pre_compact 三组规则。读失败或 JSON 无效时三组都是 nil。
// workspace 为空时返回三组 nil，否则读取其下的 .yoyo/hooks.json。
func LoadHookFile(workspace string) (pre, stop, preCompact []FileHook) {
	if workspace == "" {
		return nil, nil, nil
	}
	b, err := os.ReadFile(filepath.Join(workspace, ".yoyo", "hooks.json"))
	if err != nil {
		return nil, nil, nil
	}
	var hf hookFile
	if json.Unmarshal(b, &hf) != nil {
		return nil, nil, nil
	}
	return hf.PreTool, hf.Stop, hf.PreCompact
}

// applyFileHooks 按顺序匹配规则。第一条 Deny 生效后返回，Reason 为空时写成 denied by .yoyo/hooks.json。
// rules 按顺序匹配，第一条 Deny 生效后返回。
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

// hookMatch 在 pattern 为空、* 或与 name 相同时命中。以 * 结尾时按去掉星号后的前缀匹配。
func hookMatch(pattern, name string) bool {
	if pattern == "" || pattern == "*" || pattern == name {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, strings.TrimSuffix(pattern, "*"))
	}
	return false
}
