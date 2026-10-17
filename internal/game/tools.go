package game

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ToolSpec struct {
	Name     string
	Desc     string
	ReadOnly bool
	Params   map[string]any
}

func ToolNames() []string {
	specs := ToolSpecs()
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Name)
	}
	return out
}

func ToolSpecs() []ToolSpec {
	str := map[string]any{"type": "string"}
	obj := func(props map[string]any, req ...string) map[string]any {
		m := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 {
			m["required"] = req
		}
		return m
	}
	return []ToolSpec{
		{Name: "game_create", Desc: "Open a spatial game-world from a play intent. If a world is already bound to this session, apply the intent to THAT world — do not open a second world. Not a novel adaptation.", Params: obj(map[string]any{"title": str, "intent": str, "genre": str, "camera": str, "seed": str})},
		{Name: "game_list", ReadOnly: true, Desc: "List game-world projects.", Params: obj(map[string]any{})},
		{Name: "game_get", ReadOnly: true, Desc: "Read the bound game world, or pass id.", Params: obj(map[string]any{"id": str})},
		{Name: "game_update", Desc: "Patch title, intent, genre, or camera on the bound project.", Params: obj(map[string]any{"id": str, "title": str, "intent": str, "genre": str, "camera": str})},
		{Name: "game_analyze_scene", Desc: "Build a procedural scene packet. VLM/LLM must not invent coordinates.", Params: obj(map[string]any{"id": str, "intent": str})},
		{Name: "game_plan_rules", Desc: "Write a declarative rule plan. Task-based win, free navigation, four cardinal moves.", Params: obj(map[string]any{"id": str, "intent": str})},
		{Name: "game_extract_candidates", Desc: "Sample placement candidates from the scene heightfield. Host-owned geometry.", Params: obj(map[string]any{"id": str})},
		{Name: "game_place", Desc: "Pick candidate_id values for required elements. Coordinates are restored by the host.", Params: obj(map[string]any{"id": str})},
		{Name: "game_realize_assets", Desc: "Assign primitives (and optional image jobs) to visible placements. Logical types stay markers.", Params: obj(map[string]any{"id": str})},
		{Name: "game_compile", Desc: "Compile IR into playable/game.json and the sealed runtime. Do not rewrite runtime.js.", Params: obj(map[string]any{"id": str})},
		{Name: "game_play", Desc: "Return the workspace HTML path to browser_open. Do not start a static server.", Params: obj(map[string]any{"id": str})},
		{Name: "game_verify", Desc: "Return a browser_verify plan for the compiled playable. Host writes qa/verification.json.", Params: obj(map[string]any{"id": str})},
		{Name: "game_repair", Desc: "Localize failed QA checks via correspondence and recompile only the broken stages.", Params: obj(map[string]any{"id": str})},
	}
}

func (e *Engine) Tool(sessionID, name string, args map[string]any) (string, error) {
	if args == nil {
		args = map[string]any{}
	}
	str := func(keys ...string) string {
		for _, k := range keys {
			if v := strings.TrimSpace(fmt.Sprint(args[k])); v != "" && v != "<nil>" {
				return v
			}
		}
		return ""
	}
	switch name {
	case "game_create":
		p, err := e.openOrReuse(sessionID, Project{Title: str("title"), Intent: str("intent"), Genre: str("genre"), Camera: str("camera")})
		if err != nil {
			return "", err
		}
		if strings.EqualFold(str("seed"), "demo") && strings.TrimSpace(p.Intent) == "" {
			p.Intent = "Collect three relics and reach the shrine."
			p, _ = e.Update(p)
		}
		if strings.EqualFold(str("seed"), "demo") || strings.TrimSpace(p.Intent) != "" {
			p = e.runHostPipeline(sessionID, p)
		}
		return marshalTool(e.Receipt(p))
	case "game_list":
		rows := e.List()
		out := make([]map[string]any, 0, len(rows))
		for _, p := range rows {
			out = append(out, e.Receipt(p))
		}
		return marshalTool(out)
	case "game_get":
		p, err := e.resolve(sessionID, str("id"))
		if err != nil {
			return "", err
		}
		return marshalTool(e.Receipt(p))
	case "game_update":
		p, err := e.resolve(sessionID, str("id"))
		if err != nil {
			return "", err
		}
		p.Title = firstNonEmpty(str("title"), p.Title)
		p.Intent = firstNonEmpty(str("intent"), p.Intent)
		p.Genre = firstNonEmpty(str("genre"), p.Genre)
		p.Camera = firstNonEmpty(str("camera"), p.Camera)
		p, err = e.Update(p)
		if err != nil {
			return "", err
		}
		return marshalTool(e.Receipt(p))
	case "game_analyze_scene":
		p, err := e.AnalyzeScene(sessionID, str("id"), str("intent"))
		if err != nil {
			return "", err
		}
		return marshalTool(e.Receipt(p))
	case "game_plan_rules":
		p, err := e.PlanRules(sessionID, str("id"), str("intent"))
		if err != nil {
			return "", err
		}
		return marshalTool(e.Receipt(p))
	case "game_extract_candidates":
		p, err := e.ExtractCandidates(sessionID, str("id"))
		if err != nil {
			return "", err
		}
		return marshalTool(e.Receipt(p))
	case "game_place":
		p, err := e.Place(sessionID, str("id"))
		if err != nil {
			return "", err
		}
		return marshalTool(e.Receipt(p))
	case "game_realize_assets":
		p, err := e.RealizeAssets(sessionID, str("id"))
		if err != nil {
			return "", err
		}
		return marshalTool(e.Receipt(p))
	case "game_compile":
		p, err := e.Compile(sessionID, str("id"))
		if err != nil {
			return "", err
		}
		return marshalTool(e.Receipt(p))
	case "game_play":
		p, err := e.resolve(sessionID, str("id"))
		if err != nil {
			return "", err
		}
		if p.Stage != StageBuilt && p.Stage != StageChecked {
			p, err = e.Compile(sessionID, p.ID)
			if err != nil {
				return "", err
			}
		}
		return marshalTool(map[string]any{
			"open": e.PlayPath(p), "root": p.Rel(),
			"note": "browser_open this workspace HTML once, then browser_verify with the same open. Do not start a static server.",
		})
	case "game_verify":
		p, err := e.resolve(sessionID, str("id"))
		if err != nil {
			return "", err
		}
		if p.Stage != StageBuilt && p.Stage != StageChecked {
			p, err = e.Compile(sessionID, p.ID)
			if err != nil {
				return "", err
			}
		}
		return marshalTool(e.VerifyPlan(p))
	case "game_repair":
		p, err := e.Repair(sessionID, str("id"))
		if err != nil {
			return "", err
		}
		return marshalTool(e.Receipt(p))
	default:
		return "", fmt.Errorf("unknown game tool %s", name)
	}
}

func (e *Engine) openOrReuse(sessionID string, in Project) (Project, error) {
	if sessionID != "" {
		if id, ok := e.SessionBind(sessionID); ok {
			p, err := e.Get(id)
			if err != nil {
				return p, err
			}
			if strings.TrimSpace(in.Title) != "" {
				p.Title = preferWorldTitle(p.Title, in.Title, in.Intent)
			}
			if strings.TrimSpace(in.Intent) != "" {
				p.Intent = in.Intent
			}
			if strings.TrimSpace(in.Genre) != "" {
				p.Genre = in.Genre
			}
			if strings.TrimSpace(in.Camera) != "" {
				p.Camera = in.Camera
			}
			return e.Update(p)
		}
	}
	p, err := e.Create(in)
	if err != nil {
		return p, err
	}
	if sessionID != "" {
		_ = e.Bind(sessionID, p.ID)
	}
	return p, nil
}

func (e *Engine) runHostPipeline(sessionID string, p Project) Project {
	if _, err := e.AnalyzeScene(sessionID, p.ID, p.Intent); err != nil {
		return p
	}
	if _, err := e.PlanRules(sessionID, p.ID, p.Intent); err != nil {
		p, _ = e.Get(p.ID)
		return p
	}
	_, _ = e.ExtractCandidates(sessionID, p.ID)
	if _, err := e.Place(sessionID, p.ID); err != nil {
		p, _ = e.Get(p.ID)
		return p
	}
	_, _ = e.RealizeAssets(sessionID, p.ID)
	if got, err := e.Compile(sessionID, p.ID); err == nil {
		return got
	}
	p, _ = e.Get(p.ID)
	return p
}

func marshalTool(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
