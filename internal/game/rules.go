package game

import (
	"encoding/json"
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/game/ir"
)

func (e *Engine) PlanRules(sessionID, id, intent string) (Project, error) {
	p, err := e.resolve(sessionID, id)
	if err != nil {
		return p, err
	}
	if intent != "" {
		p.Intent = intent
	}
	var scene ir.ScenePacket
	_ = e.readIR(p, "scene.json", &scene)
	plan, err := e.buildRules(p.Intent, scene)
	if err != nil {
		return p, err
	}
	if err := ir.ValidateRules(plan); err != nil {
		return p, err
	}
	if err := e.writeIR(p, "rules.json", plan); err != nil {
		return p, err
	}
	p.NeedCollect = collectCount(plan)
	p.Stage = StageRules
	p.Title = preferWorldTitle(p.Title, plan.Title, p.Intent)
	return e.Update(p)
}

func (e *Engine) buildRules(intent string, scene ir.ScenePacket) (ir.RulePlan, error) {
	if e.Complete != nil {
		if plan, err := e.rulesFromModel(intent, scene); err == nil && ir.ValidateRules(plan) == nil {
			return plan, nil
		}
	}
	return heuristicRules(intent, scene), nil
}

func (e *Engine) rulesFromModel(intent string, scene ir.ScenePacket) (ir.RulePlan, error) {
	payload := map[string]any{
		"intent": intent,
		"scene":  ir.SanitizeForModel(scene),
		"types":  ir.ElementTypes,
	}
	raw, _ := json.Marshal(payload)
	sys := `Return one JSON object: title, description, core_loop, objective, win_condition, fail_condition, time_limit_seconds, allowed_actions, randomness_policy{free_navigation,fixed_path_required,random_choice_allowed}, required_gameplay_elements_for_placement[{element_type,purpose,required_count}]. Task-based win. free_navigation true. fixed_path_required false. Include four cardinal moves. Do not use existing trees or rocks as gameplay objects.`
	text, err := e.Complete(sys, string(raw))
	if err != nil {
		return ir.RulePlan{}, err
	}
	var rawPlan map[string]any
	if err := json.Unmarshal([]byte(extractJSON(text)), &rawPlan); err != nil {
		return ir.RulePlan{}, err
	}
	b, _ := json.Marshal(rawPlan)
	var plan ir.RulePlan
	if err := json.Unmarshal(b, &plan); err != nil {
		return ir.RulePlan{}, err
	}
	if plan.Win == "" {
		plan.Win = fmt.Sprint(rawPlan["win_condition"])
	}
	if plan.Fail == "" {
		plan.Fail = fmt.Sprint(rawPlan["fail_condition"])
	}
	plan.Status = ir.Status{Usable: true, Confidence: 0.8}
	return plan, nil
}

func heuristicRules(intent string, scene ir.ScenePacket) ir.RulePlan {
	title := "Relic run"
	obj := "Collect the relics and reach the shrine."
	win := "Collect every relic and enter the goal area."
	fail := "Fall in water or hit a hazard, or run out of time."
	need := 3
	if hasWord(intent, "hunt", "狩猎", "monster", "creature") {
		title = "Track and claim"
		obj = "Follow the signs, defeat the mark, and claim the trophy."
		win = "Reach the trophy after collecting the tracks."
	}
	if hasWord(intent, "ski", "滑雪") {
		title = "Downhill flags"
		obj = "Pass the flags and finish at the lodge."
	}
	if hasWord(intent, "神庙", "temple", "逃亡", "escape", "跑酷", "parkour") {
		title = "神庙逃亡"
		obj = "收集神庙钥匙碎片，躲开石柱与断墙，冲到出口祭坛逃脱。"
		win = "集齐碎片并抵达出口祭坛。"
		fail = "落入陷阱、撞上危险区，或被追上。"
	}
	if n := parseCount(intent); n > 0 {
		need = n
	}
	_ = scene
	req := []ir.ElementReq{
		{ElementType: "player_spawn", Purpose: "start", RequiredCount: 1},
		{ElementType: "collectible", Purpose: "relic", PlacementHint: "open ground ahead of spawn", RequiredCount: need},
		{ElementType: "goal_area", Purpose: "shrine", PlacementHint: "far walkable ground", RequiredCount: 1},
		{ElementType: "hazard_zone", Purpose: "danger", PlacementHint: "beside the route, not blocking spawn", RequiredCount: 1},
	}
	return ir.RulePlan{
		Status:         ir.Status{Usable: true, Confidence: 0.72, Reason: "heuristic from intent"},
		Title:          title,
		Description:    firstNonEmpty(intent, obj),
		CoreLoop:       "Move freely, collect relics, avoid hazards, reach the goal.",
		Role:           "player",
		Objective:      obj,
		Win:            win,
		Fail:           fail,
		TimeLimit:      90,
		AllowedActions: ir.CardinalMoves(),
		Randomness:     ir.Randomness{FreeNavigation: true, RandomChoice: true, Description: "player chooses the order of relics"},
		Required:       req,
		Scoring:        []string{"+1 per relic", "win bonus at the shrine"},
	}
}

func collectCount(p ir.RulePlan) int {
	n := 0
	for _, r := range p.Required {
		if r.ElementType == "collectible" || r.ElementType == "item_pickup" {
			n += r.RequiredCount
		}
	}
	return n
}

func parseCount(s string) int {
	for _, w := range []string{"three", "三", "3"} {
		if hasWord(s, w) {
			return 3
		}
	}
	for _, w := range []string{"two", "两", "二", "2"} {
		if hasWord(s, w) {
			return 2
		}
	}
	return 0
}
