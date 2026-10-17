package game

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/Shenchangxin/yoyo/internal/game/ir"
)

func (e *Engine) ExtractCandidates(sessionID, id string) (Project, error) {
	p, err := e.resolve(sessionID, id)
	if err != nil {
		return p, err
	}
	var scene ir.ScenePacket
	if err := e.readIR(p, "scene.json", &scene); err != nil {
		return p, fmt.Errorf("analyze the scene first: %w", err)
	}
	cands := sampleCandidates(scene.Terrain)
	if err := e.writeIR(p, "candidates.json", ir.CandidateSet{SchemaVersion: 3, Candidates: cands}); err != nil {
		return p, err
	}
	p.Stage = StageCands
	return e.Update(p)
}

func (e *Engine) Place(sessionID, id string) (Project, error) {
	p, err := e.resolve(sessionID, id)
	if err != nil {
		return p, err
	}
	var rules ir.RulePlan
	if err := e.readIR(p, "rules.json", &rules); err != nil {
		return p, fmt.Errorf("plan rules first: %w", err)
	}
	var set ir.CandidateSet
	if err := e.readIR(p, "candidates.json", &set); err != nil {
		p, err = e.ExtractCandidates(sessionID, p.ID)
		if err != nil {
			return p, err
		}
		if err := e.readIR(p, "candidates.json", &set); err != nil {
			return p, err
		}
	}
	plan, err := e.pickPlacements(rules, set.Candidates)
	if err != nil {
		return p, err
	}
	if err := ir.ValidateSpacing(plan.Elements); err != nil {
		return p, err
	}
	if err := e.writeIR(p, "placement.json", plan); err != nil {
		return p, err
	}
	p.Stage = StagePlaced
	return e.Update(p)
}

func sampleCandidates(t ir.Terrain) []ir.Candidate {
	var out []ir.Candidate
	if t.Width < 2 || t.Depth < 2 || t.CellSize <= 0 {
		return out
	}
	n := 0
	for z := 1; z < t.Depth-1; z += 2 {
		for x := 1; x < t.Width-1; x += 2 {
			i := z*t.Width + x
			if i < 0 || i >= len(t.Surfaces) || i >= len(t.Heights) {
				continue
			}
			surf := t.Surfaces[i]
			wx := (float64(x) - float64(t.Width)/2) * t.CellSize
			wz := (float64(z) - float64(t.Depth)/2) * t.CellSize
			wy := t.Heights[i]
			walk := "good"
			kind := "ground_surface"
			tags := []string{"walkable"}
			switch surf {
			case "water":
				walk, kind, tags = "poor", "blocked_surface", []string{"avoid"}
			case "rock":
				walk, kind, tags = "medium", "visual_anchor", []string{"landmark"}
			case "obstacle":
				walk, kind, tags = "poor", "occluder_anchor", []string{"blocked"}
			}
			cell := gridCell(x, z, t.Width, t.Depth)
			n++
			out = append(out, ir.Candidate{
				ID:            fmt.Sprintf("c_%03d", n),
				GridCell:      cell,
				WorldXYZ:      ir.Vec3{wx, wy, wz},
				NormalXYZ:     ir.Vec3{0, 1, 0},
				SurfaceType:   surf,
				Walkability:   walk,
				Kind:          kind,
				SupportRadius: 1.25,
				Tags:          tags,
				Distance:      math.Hypot(wx, wz),
			})
			if len(out) >= 80 {
				return out
			}
		}
	}
	return out
}

func gridCell(x, z, w, d int) string {
	col, row := "center", "center"
	if x < w/3 {
		col = "left"
	} else if x > 2*w/3 {
		col = "right"
	}
	if z < d/3 {
		row = "lower"
	} else if z > 2*d/3 {
		row = "upper"
	}
	if col == "center" && row == "center" {
		return "center"
	}
	return row + "_" + col
}

func (e *Engine) pickPlacements(rules ir.RulePlan, cands []ir.Candidate) (ir.PlacementPlan, error) {
	if e.Complete != nil {
		if plan, err := e.placeWithModel(rules, cands); err == nil {
			return plan, nil
		}
	}
	return placeHeuristic(rules, cands)
}

func (e *Engine) placeWithModel(rules ir.RulePlan, cands []ir.Candidate) (ir.PlacementPlan, error) {
	type view struct {
		ID, Grid, Surface, Walk, Kind string
		Tags                          []string
		Support                       float64
	}
	views := make([]view, 0, len(cands))
	byID := map[string]ir.Candidate{}
	for _, c := range cands {
		byID[c.ID] = c
		views = append(views, view{c.ID, c.GridCell, c.SurfaceType, c.Walkability, c.Kind, c.Tags, c.SupportRadius})
	}
	payload := map[string]any{
		"required":   ir.SanitizeForModel(rules.Required),
		"candidates": views,
	}
	raw, _ := json.Marshal(payload)
	sys := "Pick existing candidate_id values only. Never invent coordinates. Return JSON {elements:[{placement_id,element_type,source_requirement_index,candidate_id,gameplay_purpose}]} matching required_count."
	text, err := e.Complete(sys, string(raw))
	if err != nil {
		return ir.PlacementPlan{}, err
	}
	var parsed struct {
		Elements []ir.Placement `json:"elements"`
	}
	if err := json.Unmarshal([]byte(extractJSON(text)), &parsed); err != nil {
		return ir.PlacementPlan{}, err
	}
	var out []ir.Placement
	used := map[string]bool{}
	for i, el := range parsed.Elements {
		c, ok := byID[el.CandidateID]
		if !ok || used[c.ID] {
			return ir.PlacementPlan{}, fmt.Errorf("invalid or reused candidate_id %q", el.CandidateID)
		}
		used[c.ID] = true
		el.PlacementID = fmt.Sprintf("placement_%03d", i)
		out = append(out, ir.RestorePlacement(el, c))
	}
	if err := matchCounts(rules.Required, out); err != nil {
		return ir.PlacementPlan{}, err
	}
	return ir.PlacementPlan{Status: ir.Status{Usable: true, Confidence: 0.8}, Elements: out}, nil
}

func placeHeuristic(rules ir.RulePlan, cands []ir.Candidate) (ir.PlacementPlan, error) {
	walk := filterCands(cands, func(c ir.Candidate) bool {
		return c.Walkability == "good" && c.SurfaceType == "ground"
	})
	if len(walk) < 2 {
		walk = cands
	}
	used := map[int]bool{}
	var out []ir.Placement
	n := 0
	take := func(preferFar bool, minDist float64) (ir.Candidate, bool) {
		best := -1e9
		bi := -1
		for i, c := range walk {
			if used[i] {
				continue
			}
			ok := true
			for _, o := range out {
				if distXZ(c.WorldXYZ, o.WorldXYZ) < minDist {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			score := c.Distance
			if !preferFar {
				score = -c.Distance
			}
			if bi < 0 || score > best {
				best, bi = score, i
			}
		}
		if bi < 0 {
			return ir.Candidate{}, false
		}
		used[bi] = true
		return walk[bi], true
	}
	for ri, req := range rules.Required {
		count := req.RequiredCount
		if count < 1 {
			count = 1
		}
		for k := 0; k < count; k++ {
			far := req.ElementType == "goal_area" || req.ElementType == "landmark"
			min := ir.DefaultMinCenterDistance
			if req.ElementType == "player_spawn" {
				min = 2
				far = false
			}
			c, ok := take(far, min)
			if !ok {
				c, ok = take(far, ir.DefaultMinFootprintGap)
			}
			if !ok {
				return ir.PlacementPlan{}, fmt.Errorf("not enough candidates for %s", req.ElementType)
			}
			out = append(out, ir.Placement{
				PlacementID: fmt.Sprintf("placement_%03d", n),
				ElementType: req.ElementType,
				SourceReq:   ri,
				CandidateID: c.ID,
				WorldXYZ:    c.WorldXYZ,
				NormalXYZ:   c.NormalXYZ,
				Mode:        "on_ground",
				Purpose:     req.Purpose,
				Reason:      "heuristic spacing on walkable ground",
			})
			n++
		}
	}
	return ir.PlacementPlan{Status: ir.Status{Usable: true, Confidence: 0.74, Reason: "heuristic"}, Elements: out}, nil
}

func matchCounts(req []ir.ElementReq, els []ir.Placement) error {
	got := map[string]int{}
	for _, el := range els {
		got[el.ElementType]++
	}
	for _, r := range req {
		if got[r.ElementType] != r.RequiredCount {
			return fmt.Errorf("%s count %d != %d", r.ElementType, got[r.ElementType], r.RequiredCount)
		}
	}
	return nil
}

func filterCands(in []ir.Candidate, keep func(ir.Candidate) bool) []ir.Candidate {
	var out []ir.Candidate
	for _, c := range in {
		if keep(c) {
			out = append(out, c)
		}
	}
	return out
}

func distXZ(a, b ir.Vec3) float64 {
	return math.Hypot(a[0]-b[0], a[2]-b[2])
}
