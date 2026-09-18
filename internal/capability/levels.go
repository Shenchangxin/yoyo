package capability

import "strings"

const (
	ReadConnector  Level = "read_connector"
	WriteConnector Level = "write_connector"
	Browser        Level = "browser"
	ComputerUse    Level = "computer_use"
	SendAsYou      Level = "send_as_you"
	Schedule       Level = "schedule"
	MemoryWrite    Level = "memory_write"
)

// GateMode is the operator-facing approval policy.
const (
	GateManual   = "manual"
	GateAutoSafe = "autosafe"
	GateSkip     = "skip"
)

func IdentityLevel(l Level) bool {
	switch l {
	case SendAsYou, ComputerUse, HighRisk:
		return true
	default:
		return false
	}
}

// NeverAlways reports levels that must not stick as Always unless an L3
// policy pack explicitly opts in. Identity actions stay session-scoped.
func NeverAlways(l Level) bool {
	return IdentityLevel(l)
}

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

// AuthMode is the per-session approval posture. Settings GateMode remains
// the app ceiling when an action actually reaches the approver.
const (
	AuthDefault  = "default"
	AuthAutoEdit = "auto_edit"
	AuthFull     = "full"
	AuthAsk      = "ask"
)

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

// AuthModeLevels are session grants applied for a mode. Identity levels are
// never included; Default and Ask grant nothing extra (Default relies on the
// broker always-map, Ask skips that map).
func AuthModeLevels(mode string) []Level {
	switch ParseAuthMode(mode) {
	case AuthAutoEdit:
		return []Level{WriteWorkspace, SpecifiedPath, MemoryWrite}
	case AuthFull:
		return []Level{
			ReadWorkspace, WriteWorkspace, SpecifiedPath, MemoryWrite,
			Shell, Network, Browser, Schedule, ReadConnector, WriteConnector,
		}
	default:
		return nil
	}
}

func AuthModeCapStrings(mode string) []string {
	lv := DropSticky(AuthModeLevels(mode))
	out := make([]string, 0, len(lv))
	for _, l := range lv {
		out = append(out, string(l))
	}
	return out
}
