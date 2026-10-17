package game

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/game/ir"
)

func (e *Engine) RealizeAssets(sessionID, id string) (Project, error) {
	p, err := e.resolve(sessionID, id)
	if err != nil {
		return p, err
	}
	var place ir.PlacementPlan
	if err := e.readIR(p, "placement.json", &place); err != nil {
		return p, fmt.Errorf("place elements first: %w", err)
	}
	var existing ir.AssetPlan
	_ = e.readIR(p, "assets.json", &existing)
	reuse := map[string]ir.Asset{}
	for _, a := range existing.Assets {
		if a.TextureHash != "" {
			reuse[a.Role+"|"+a.Name] = a
		}
	}
	plan := ir.AssetPlan{}
	var jobs []string
	for i, el := range place.Elements {
		a := primitiveFor(el, i)
		if prev, ok := reuse[a.Role+"|"+a.Name]; ok {
			a.TextureHash = prev.TextureHash
		}
		if err := ir.ValidateAsset(a); err != nil {
			return p, err
		}
		if !e.SkipImage && e.Enqueue != nil && ir.RequiresVisibleAsset(el.ElementType) && a.TextureHash == "" && a.Prompt != "" {
			if jobID, err := e.Enqueue(a.Prompt, p.ID, a.AssetID); err == nil && jobID != "" {
				a.JobID = jobID
				jobs = append(jobs, jobID)
			}
		}
		plan.Assets = append(plan.Assets, a)
	}
	if err := e.writeIR(p, "assets.json", plan); err != nil {
		return p, err
	}
	p.JobIDs = jobs
	p.Stage = StageAssets
	return e.Update(p)
}

func primitiveFor(el ir.Placement, i int) ir.Asset {
	t := ir.NormalizeType(el.ElementType)
	a := ir.Asset{
		AssetID:      fmt.Sprintf("asset_%03d", i),
		Name:         t,
		Role:         t,
		PlacementIDs: []string{el.PlacementID},
		SizeMeters:   ir.Vec3{1, 1, 1},
		Primitive:    "box",
		Color:        "#c9a227",
	}
	switch t {
	case "collectible", "item_pickup":
		a.Primitive, a.Color, a.SizeMeters = "sphere", "#e4c14a", ir.Vec3{0.7, 0.7, 0.7}
		a.Prompt = "isolated gold relic on white studio background, no floor, game item"
	case "goal_area":
		a.Primitive, a.Color, a.SizeMeters = "cylinder", "#3d8bfd", ir.Vec3{2.2, 0.3, 2.2}
		a.Prompt = "isolated stone shrine marker, white studio, no floor"
	case "hazard_zone":
		a.Primitive, a.Color, a.SizeMeters = "box", "#c23b3b", ir.Vec3{2, 0.25, 2}
		a.Prompt = "isolated warning hazard plate, white studio, no floor"
	case "obstacle":
		a.Primitive, a.Color, a.SizeMeters = "box", "#6b6256", ir.Vec3{1.4, 1.6, 1.4}
	case "landmark":
		a.Primitive, a.Color, a.SizeMeters = "cylinder", "#8aa0b8", ir.Vec3{0.6, 3, 0.6}
	case "player_spawn", "enemy_spawn", "npc_spawn", "vehicle_spawn", "navigation_anchor", "camera_anchor":
		a.SkipMesh, a.Primitive, a.Color = true, "marker", "#888888"
	default:
		if ir.LogicalOnly(t) {
			a.SkipMesh, a.Primitive = true, "marker"
		}
	}
	return a
}
