// tools_task_child.go 给 task 子代理准备工具集和开场文本。
// runtime.Run 的 spawnTask 和桌面 agentloop/desk 的子代理都走这里，裁剪规则只写一份，两条循环上的子代理看到的工具面一致。

package runtime

import (
	"path/filepath"
	"strings"
)

// ChildTools 复制 parent，给子代理用。parent 为 nil 时返回 nil。
// 副本的工作区换成 ws，会话换成 childID，深度加一。Task 置空，由子循环按自己的深度重新挂上，深度到 MaxTaskDepth 后就挂不上了。
// ws 非空时，溢出目录放在 ws/.yoyo/context/task/<childID 的最后一段>，不和父会话的溢出文件混在一起。
// 计划模式、聊天覆盖层和验证提示都关掉；技能提示和自动加载清空，技能自动调用设为 off，并去掉包引导，子代理只做交给它的那一件事。
// profile 为 explore 或 qa 时只读，广告清单只留只读工具；过滤后为空时用 readOnlyAdvertised 的默认名单。并行上限沿用父工具集。
func ChildTools(parent *WorkspaceTools, childID, ws, profile string) *WorkspaceTools {
	if parent == nil {
		return nil
	}
	cp := parent.clone()
	cp.Workspace = ws
	cp.SessionID = childID
	cp.Depth = parent.Depth + 1
	cp.Task = nil
	if ws != "" {
		cp.Spill = NewSpill(filepath.Join(ws, ".yoyo", "context", "task", filepath.Base(childID)))
	}
	cp.PlanMode = false
	cp.ReadOnly = readOnlyTaskProfile(profile)
	cp.MaxParallel = parent.MaxParallel
	cp.ChatOverlay = false
	cp.VerifyHint = false
	if cp.ReadOnly {
		cp.Advertised = readOnlyAdvertised(cp.Advertised)
	}
	StripPackBootstrap(cp)
	cp.SkillHints = nil
	cp.Autoloaded = nil
	cp.SkillAutoInvoke = "off"
	return cp
}

// TaskUser 把 instruction 和 prompt 拼成子代理收到的第一条用户消息，中间空一行。
// instruction 去掉空白后为空时原样返回 prompt。
func TaskUser(instruction, prompt string) string {
	if strings.TrimSpace(instruction) == "" {
		return prompt
	}
	return instruction + "\n\n" + prompt
}

// TaskProfileInstruction 返回 profile 对应的角色说明。explore 返回 exploreInstruction，qa 返回 qaInstruction，大小写和首尾空白不影响判断，其余返回空串。
// runtime.Run 的子代理把它放进子循环的 TaskInstruction（见 childLoopFor）；agentloop 子代理没有循环预设，直接用 TaskUser 接在任务前面。
func TaskProfileInstruction(profile string) string {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "explore":
		return exploreInstruction
	case "qa":
		return qaInstruction
	default:
		return ""
	}
}
