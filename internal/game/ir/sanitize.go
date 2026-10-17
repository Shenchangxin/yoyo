package ir

import "encoding/json"

var sanitizeKeys = map[string]bool{
	"world_xyz": true, "anchor_world_xyz": true, "normal_xyz": true,
	"position": true, "world_position": true, "heights": true,
	"matrix_world": true, "teleport_to": true,
}

// SanitizeForModel strips host-owned coordinates so a planner cannot invent xyz.
func SanitizeForModel(v any) any {
	raw, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var tree any
	if err := json.Unmarshal(raw, &tree); err != nil {
		return v
	}
	return strip(tree)
}

func strip(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if sanitizeKeys[k] {
				continue
			}
			out[k] = strip(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = strip(val)
		}
		return out
	default:
		return v
	}
}

// RestorePlacement copies world coordinates from the chosen candidate.
func RestorePlacement(p Placement, c Candidate) Placement {
	p.CandidateID = c.ID
	p.WorldXYZ = c.WorldXYZ
	p.NormalXYZ = c.NormalXYZ
	if p.Mode == "" {
		p.Mode = "on_ground"
	}
	return p
}
