// 其余能力级别、操作者侧策略字符串、会话授权模式，以及哪些级别和动作不能记成 Always。
// 不做路径判断，也不挂起审批。broker.go 的 CheckCtx、ApplyAuthMode、RestoreSession 调用这里。

package capability

import "strings"

const (
	// ReadConnector 是读取连接器。
	ReadConnector Level = "read_connector"
	// WriteConnector 是写入连接器。
	WriteConnector Level = "write_connector"
	// Browser 是浏览器操作。
	Browser Level = "browser"
	// ComputerUse 是操控本机界面。它是身份级别，不能记成 Always。
	ComputerUse Level = "computer_use"
	// SendAsYou 是以用户身份对外发送。它是身份级别，不能记成 Always。
	SendAsYou Level = "send_as_you"
	// Schedule 是定时任务。
	Schedule Level = "schedule"
	// MemoryWrite 是写入记忆。
	MemoryWrite Level = "memory_write"
	// Clipboard 是剪贴板。
	Clipboard Level = "clipboard"
)

// 操作者侧审批策略的取值。本文件只把别名收成这三个字符串。
const (
	// GateManual 是 manual。ParseGateMode 遇到不认识的字符串时返回它。
	GateManual = "manual"
	// GateAutoSafe 是 autosafe。ParseGateMode 也把 auto_safe 和 auto-safe 收成它。
	GateAutoSafe = "autosafe"
	// GateSkip 是 skip。ParseGateMode 也把 bypass 收成它。
	GateSkip = "skip"
)

// IdentityLevel 报告该级别是不是身份级别。SendAsYou、ComputerUse、HighRisk 返回 true，其余返回 false。这三档不能记成 Always。
func IdentityLevel(l Level) bool {
	switch l {
	case SendAsYou, ComputerUse, HighRisk:
		return true
	default:
		return false
	}
}

// NeverAlways 与 IdentityLevel 相同：SendAsYou、ComputerUse、HighRisk 不能记成 Always。
// CheckCtx 若收到 Always，只把这些级别写入本会话。
func NeverAlways(l Level) bool {
	return IdentityLevel(l)
}

// NeverAlwaysAction 判断动作本身能不能记成 Always。级别可以仍是 WriteWorkspace。
// 去掉首尾空白并转成小写后，review_page、page.save、page.edit 返回 true，其余返回 false。
// CheckCtx 对命中的动作不会写入始终表。审批返回 Session，或把 Always 改记本会话时，本会话仍然可以记住该级别。
// action 去掉首尾空白并转成小写后，与 review_page、page.save、page.edit 比较。
func NeverAlwaysAction(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "review_page", "page.save", "page.edit":
		return true
	default:
		return false
	}
}

// DropSticky 丢掉 HighRisk、SendAsYou、ComputerUse，其余级别按原顺序保留。
// levels 逐项检查，等于 HighRisk、SendAsYou 或 ComputerUse 的丢掉。
func DropSticky(levels []Level) []Level {
	out := make([]Level, 0, len(levels))
	for _, l := range levels {
		if l == HighRisk || l == SendAsYou || l == ComputerUse {
			continue
		}
		out = append(out, l)
	}
	return out
}

// ParseGateMode 把字符串收成 GateAutoSafe、GateSkip 或 GateManual。比较前去掉首尾空白并转成小写。
// auto_safe 与 auto-safe 收成 GateAutoSafe，bypass 收成 GateSkip。空串和其他文本收成 GateManual。
func ParseGateMode(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case GateAutoSafe, "auto_safe", "auto-safe":
		return GateAutoSafe
	case GateSkip, "bypass":
		return GateSkip
	default:
		return GateManual
	}
}

// 每个会话的审批姿态。设置里的 GateMode 仍是动作真正送到审批者时的应用上限。
const (
	// AuthDefault 是 default。AuthModeLevels 不额外授予级别。
	AuthDefault = "default"
	// AuthAutoEdit 是 auto_edit。AuthModeLevels 授予 WriteWorkspace、SpecifiedPath、MemoryWrite。
	AuthAutoEdit = "auto_edit"
	// AuthFull 是 full。AuthModeLevels 授予工作区读写、SpecifiedPath、MemoryWrite、Shell、Network、Browser、Schedule、连接器读写和 Clipboard，不含身份级别。
	AuthFull = "full"
	// AuthAsk 是 ask。AuthModeLevels 不额外授予；ApplyAuthMode 会把该会话标成 strict，从而跳过始终表。
	AuthAsk = "ask"
)

// ParseAuthMode 把字符串收成 AuthAsk、AuthAutoEdit、AuthFull 或 AuthDefault。
// 比较前去掉首尾空白，把连字符换成下划线，再转成小写。
// ask_every_time 与 strict 收成 AuthAsk，autoedit 收成 AuthAutoEdit，full_access 与 yolo 收成 AuthFull。
// auto-edit 因连字符折叠后等于 auto_edit，也收成 AuthAutoEdit。空串和其他文本收成 AuthDefault。
func ParseAuthMode(s string) string {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "-", "_")) {
	case AuthAsk, "ask_every_time", "strict":
		return AuthAsk
	case AuthAutoEdit, "autoedit":
		return AuthAutoEdit
	case AuthFull, "full_access", "yolo":
		return AuthFull
	default:
		return AuthDefault
	}
}

// AuthModeLevels 返回该模式要写入会话的级别。任何分支都不含 SendAsYou、ComputerUse、HighRisk。
// AuthAutoEdit 返回 WriteWorkspace、SpecifiedPath、MemoryWrite。
// AuthFull 返回 ReadWorkspace、WriteWorkspace、SpecifiedPath、MemoryWrite、Shell、Network、Browser、Schedule、ReadConnector、WriteConnector、Clipboard。
// AuthDefault 和 AuthAsk 都落到 default 并返回 nil：Default 随后能否直接通过取决于 Broker 里已有的始终表；Ask 跳过始终表靠的是 ApplyAuthMode 写上的 strict。
// mode 先交给 ParseAuthMode，再按 AuthAutoEdit、AuthFull 或其余分支返回级别。
func AuthModeLevels(mode string) []Level {
	switch ParseAuthMode(mode) {
	case AuthAutoEdit:
		return []Level{WriteWorkspace, SpecifiedPath, MemoryWrite}
	case AuthFull:
		return []Level{
			ReadWorkspace, WriteWorkspace, SpecifiedPath, MemoryWrite,
			Shell, Network, Browser, Schedule, ReadConnector, WriteConnector, Clipboard,
		}
	default:
		// AuthDefault 与 AuthAsk 都到这里，返回 nil。Ask 跳过始终表靠 strict，不靠这份返回值。
		return nil
	}
}

// AuthModeCapStrings 把 AuthModeLevels 的结果经 DropSticky 后收成字符串。
// mode 交给 AuthModeLevels。
func AuthModeCapStrings(mode string) []string {
	lv := DropSticky(AuthModeLevels(mode))
	out := make([]string, 0, len(lv))
	for _, l := range lv {
		out = append(out, string(l))
	}
	return out
}
