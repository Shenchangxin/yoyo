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
