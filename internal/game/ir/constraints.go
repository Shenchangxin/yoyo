package ir

import "strings"

const (
	DefaultMinCenterDistance = 6.0
	DefaultMinFootprintGap   = 0.75
	MinimumSupportRadius     = 0.35
)

var ElementTypes = map[string]string{
	"collectible":        "visible item collected for score or objectives",
	"obstacle":           "visible object that blocks movement",
	"event_trigger":      "logical trigger for a scripted event",
	"goal_area":          "visible destination plus completion trigger",
	"hazard_zone":        "dangerous volume plus fail or damage",
	"safe_zone":          "area where danger is reduced",
	"landmark":           "orientation or spectacle asset",
	"player_spawn":       "initial player or respawn anchor",
	"enemy_spawn":        "hostile spawn anchor",
	"npc_spawn":          "friendly or neutral spawn anchor",
	"vehicle_spawn":      "vehicle spawn anchor",
	"item_pickup":        "inventory pickup",
	"objective_object":   "mission-critical object",
	"interaction_point":  "use, switch, or dialogue point",
	"cover_point":        "tactical cover anchor",
	"destructible":       "breakable object",
	"puzzle_element":     "switch or movable piece",
	"checkpoint":         "progression checkpoint",
	"capture_zone":       "contest or capture area",
	"traversal_point":    "jump, vault, or climb anchor",
	"navigation_anchor":  "path-planning anchor",
	"camera_anchor":      "camera preset anchor",
	"moving_platform":    "moving carrier",
}

var VisibleAssetTypes = map[string]bool{
	"collectible": true, "obstacle": true, "goal_area": true, "hazard_zone": true,
	"landmark": true, "item_pickup": true, "objective_object": true,
	"destructible": true, "puzzle_element": true, "moving_platform": true,
}

var LogicalOnlyTypes = map[string]bool{
	"player_spawn": true, "enemy_spawn": true, "npc_spawn": true,
	"vehicle_spawn": true, "navigation_anchor": true, "camera_anchor": true,
}

var OptionalAssetTypes = map[string]bool{
	"event_trigger": true, "safe_zone": true, "interaction_point": true,
	"cover_point": true, "checkpoint": true, "capture_zone": true, "traversal_point": true,
}

func NormalizeType(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if _, ok := ElementTypes[s]; ok {
		return s
	}
	return ""
}

func RequiresVisibleAsset(t string) bool {
	return VisibleAssetTypes[NormalizeType(t)]
}

func LogicalOnly(t string) bool {
	return LogicalOnlyTypes[NormalizeType(t)]
}

func CardinalMoves() []string {
	return []string{"move_forward", "move_backward", "move_left", "move_right"}
}

func HasCardinalMoves(actions []string) bool {
	have := map[string]bool{}
	for _, a := range actions {
		have[strings.ToLower(strings.TrimSpace(a))] = true
	}
	for _, need := range CardinalMoves() {
		if !have[need] {
			return false
		}
	}
	return true
}

func ValidateRules(r RulePlan) error {
	if !r.Randomness.FreeNavigation {
		return errf("randomness_policy.free_navigation must be true")
	}
	if r.Randomness.FixedPathRequired {
		return errf("randomness_policy.fixed_path_required must be false")
	}
	if !HasCardinalMoves(r.AllowedActions) {
		return errf("allowed_actions must include the four cardinal moves")
	}
	win := strings.ToLower(r.Win)
	if strings.Contains(win, "survive the timer") || strings.Contains(win, "survive the time") {
		return errf("win_condition must be task-based, not survive-the-timer")
	}
	if strings.TrimSpace(r.Objective) == "" {
		return errf("objective required")
	}
	if len(r.Required) == 0 {
		return errf("required_gameplay_elements_for_placement required")
	}
	for i, req := range r.Required {
		if NormalizeType(req.ElementType) == "" {
			return errf("required[%d]: unknown element_type %q", i, req.ElementType)
		}
		if req.RequiredCount < 1 {
			return errf("required[%d]: required_count must be >= 1", i)
		}
	}
	return nil
}

func ValidateAsset(a Asset) error {
	role := NormalizeType(a.Role)
	if role == "" {
		role = NormalizeType(a.Name)
	}
	if LogicalOnly(a.Role) && !a.SkipMesh {
		return errf("logical element %s must not receive a mesh", a.Role)
	}
	return nil
}

type constError string

func (e constError) Error() string { return string(e) }

func errf(format string, args ...any) error {
	if len(args) == 0 {
		return constError(format)
	}
	return wrapf(format, args...)
}
