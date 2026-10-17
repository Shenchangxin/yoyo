package game

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/game/ir"
)

//go:embed embed/runtime.js
var runtimeJS []byte

//go:embed embed/index.html
var indexHTML []byte

func (e *Engine) Compile(sessionID, id string) (Project, error) {
	p, err := e.resolve(sessionID, id)
	if err != nil {
		return p, err
	}
	playable, inter, corr, err := e.buildPlayable(p)
	if err != nil {
		return p, err
	}
	if err := e.writeIR(p, "interactions.json", inter); err != nil {
		return p, err
	}
	if err := e.writeIR(p, "correspondence.json", corr); err != nil {
		return p, err
	}
	dir, err := e.projectDir(p)
	if err != nil {
		return p, err
	}
	if err := writeJSON(dir, "playable/game.json", playable); err != nil {
		return p, err
	}
	if err := os.WriteFile(filepath.Join(dir, "playable", "runtime.js"), runtimeJS, 0o644); err != nil {
		return p, err
	}
	html, err := renderIndex(playable)
	if err != nil {
		return p, err
	}
	if err := os.WriteFile(filepath.Join(dir, "playable", "index.html"), html, 0o644); err != nil {
		return p, err
	}
	p.Stage = StageBuilt
	p.Status = "compiled"
	p.Error = ""
	p.NeedCollect = playable.NeedCollect
	return e.Update(p)
}

func renderIndex(p ir.Playable) ([]byte, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	html := bytes.Replace(indexHTML, []byte("__YOYO_GAME_JSON__"), raw, 1)
	html = bytes.Replace(html, []byte(`<script src="runtime.js"></script>`), append(append([]byte("<script>\n"), runtimeJS...), []byte("\n</script>")...), 1)
	return html, nil
}

func (e *Engine) buildPlayable(p Project) (ir.Playable, []ir.Interaction, []ir.Correspondence, error) {
	var scene ir.ScenePacket
	var rules ir.RulePlan
	var place ir.PlacementPlan
	var assets ir.AssetPlan
	if err := e.readIR(p, "scene.json", &scene); err != nil {
		return ir.Playable{}, nil, nil, fmt.Errorf("scene required: %w", err)
	}
	if err := e.readIR(p, "rules.json", &rules); err != nil {
		return ir.Playable{}, nil, nil, fmt.Errorf("rules required: %w", err)
	}
	if err := e.readIR(p, "placement.json", &place); err != nil {
		return ir.Playable{}, nil, nil, fmt.Errorf("placement required: %w", err)
	}
	_ = e.readIR(p, "assets.json", &assets)
	assetByPlace := map[string]ir.Asset{}
	for _, a := range assets.Assets {
		for _, pid := range a.PlacementIDs {
			assetByPlace[pid] = a
		}
	}
	var player ir.PlayableActor
	player.Radius, player.Height, player.Speed = 0.42, 1.7, 7
	var els []ir.PlayableElement
	var inter []ir.Interaction
	var corr []ir.Correspondence
	need := 0
	for _, pl := range place.Elements {
		t := ir.NormalizeType(pl.ElementType)
		a := assetByPlace[pl.PlacementID]
		pos := pl.WorldXYZ
		pos[1] += pl.HeightOffset
		if t == "player_spawn" {
			player.Position = pos
		}
		el := ir.PlayableElement{
			ID: pl.PlacementID, Type: t, Position: pos,
			Size: a.SizeMeters, Color: a.Color, Primitive: a.Primitive,
		}
		if el.Size == (ir.Vec3{}) {
			el.Size = ir.Vec3{1, 1, 1}
		}
		if el.Color == "" {
			el.Color = "#bbbbbb"
		}
		switch t {
		case "collectible", "item_pickup":
			el.Trigger = "collect"
			need++
			inter = append(inter, ir.Interaction{
				ID: "logic_" + pl.PlacementID, Participants: []string{pl.PlacementID},
				Trigger: "collect", Delta: ir.StateDelta{Score: 1, Inventory: t},
			})
		case "goal_area":
			el.Trigger = "enter"
			inter = append(inter, ir.Interaction{
				ID: "logic_" + pl.PlacementID, Participants: []string{pl.PlacementID},
				Trigger: "enter", Precondition: "all_collected", Delta: ir.StateDelta{Win: true, Score: 5},
			})
		case "hazard_zone":
			el.Trigger = "enter"
			inter = append(inter, ir.Interaction{
				ID: "logic_" + pl.PlacementID, Participants: []string{pl.PlacementID},
				Trigger: "enter", Delta: ir.StateDelta{HP: -3, Fail: true},
			})
		case "obstacle":
			el.Solid = true
		}
		if !ir.LogicalOnly(t) {
			els = append(els, el)
		}
		corr = append(corr, ir.Correspondence{
			ID: pl.PlacementID, Kappa: t,
			SceneRef: pl.CandidateID, RuntimeRef: "playable.elements." + pl.PlacementID,
			InteractionIDs: []string{"logic_" + pl.PlacementID},
			LogicIDs:       []string{"logic_" + pl.PlacementID},
		})
	}
	inter = append(inter, ir.Interaction{
		ID: "logic_water", Participants: []string{"water"}, Trigger: "enter",
		Delta: ir.StateDelta{Fail: true},
	})
	play := ir.Playable{
		SchemaVersion: ir.SchemaVersion,
		Title:         preferWorldTitle(p.Title, rules.Title, p.Intent),
		Genre:         p.Genre,
		Objective:     rules.Objective,
		Win:           rules.Win,
		Fail:          rules.Fail,
		TimeLimit:     rules.TimeLimit,
		NeedCollect:   need,
		Camera:        p.Camera,
		Player:        player,
		Scene:         scene.Terrain,
		Elements:      els,
		Interactions:  inter,
	}
	return play, inter, corr, nil
}

func (e *Engine) PlayPath(p Project) string {
	return filepath.ToSlash(filepath.Join(p.Rel(), "playable", "index.html"))
}

func (e *Engine) VerifyPlan(p Project) map[string]any {
	open := e.PlayPath(p)
	return map[string]any{
		"open":     open,
		"root":     filepath.ToSlash(p.Rel()),
		"terminal": "designed-outcome",
		"restart":  "initial-state",
		"steps": []map[string]any{
			{"op": "wait", "selector": "#hud", "check": "render", "ms": 800},
			{"op": "click", "selector": "#stage", "check": "input"},
			{"op": "type", "selector": "#stage", "text": "wwwwwwww", "check": "coreLoop"},
			{"op": "wait", "selector": "#outcome", "contains": "WIN", "check": "outcome", "ms": 2500},
			{"op": "click", "selector": "#restart", "check": "restart"},
		},
	}
}

func PlayableLooksSealed(html string) bool {
	return strings.Contains(html, "__yoyoGame") && strings.Contains(html, "WASD")
}
