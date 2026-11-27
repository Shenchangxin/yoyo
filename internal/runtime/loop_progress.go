// Run 每批工具之后的收尾：把工作记忆写进 spill，以及按计划和验证提示附上技能正文。
// 桌面回合的停滞提醒和停止在 agentloop/desk/guard.go，不在这里。

package runtime

import (
	"strings"
)

// planFirstOverlay 在方法论模式返回空串。否则在计划仍开着时附上 plan-first 技能正文。
func planFirstOverlay(t *WorkspaceTools) string {
	if t != nil && t.Methodology {
		return ""
	}
	return skillOverlay(t, "plan-first", t != nil && PlanOpen(planTextOf(t)))
}

// verifyArtifactOverlay 在 VerifyHint 为真时附上 verify-artifact 技能正文。
func verifyArtifactOverlay(t *WorkspaceTools) string {
	return skillOverlay(t, "verify-artifact", t != nil && t.VerifyHint)
}

// skillOverlay 在聊天覆盖层打开、条件为真、技能尚未加载且正文非空时，返回带 Skill 标题的正文。否则返回空串。
// t 为 nil、ChatOverlay 为假或 Skills 为 nil 时返回空串。
// name 为空、已在 Loaded 里，或对应正文去掉空白后为空时返回空串，否则写进标题。
// cond 为假时返回空串。
func skillOverlay(t *WorkspaceTools, name string, cond bool) string {
	if t == nil || !t.ChatOverlay || !cond || name == "" {
		return ""
	}
	t.lock().Lock()
	defer t.lock().Unlock()
	for _, n := range t.Loaded {
		if n == name {
			return ""
		}
	}
	if t.Skills == nil {
		return ""
	}
	body := strings.TrimSpace(t.Skills[name])
	if body == "" {
		return ""
	}
	return "## Skill: " + name + "\n" + body
}
