package video

type ToolSpec struct {
	Name     string
	Desc     string
	ReadOnly bool
	Params   map[string]any
}

func DramaToolNames() []string {
	specs := DramaToolSpecs()
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Name)
	}
	return out
}

func DramaToolSpecs() []ToolSpec {
	str := map[string]any{"type": "string"}
	obj := func(props map[string]any, req ...string) map[string]any {
		m := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 {
			m["required"] = req
		}
		return m
	}
	item := map[string]any{"type": "object"}
	arr := map[string]any{"type": "array", "items": item}
	return []ToolSpec{
		{Name: "drama_read_episode", ReadOnly: true, Desc: "Read the bound episode novel, screenplay, and language.", Params: obj(map[string]any{"episode_id": str})},
		{Name: "drama_save_script", Desc: "Replace the episode screenplay with a fully formatted script.", Params: obj(map[string]any{"script": str, "episode_id": str}, "script")},
		{Name: "drama_read_assets", ReadOnly: true, Desc: "List characters, scenes, and props for the bound episode, including existing rows for dedupe.", Params: obj(map[string]any{"episode_id": str})},
		{Name: "drama_save_characters", Desc: "Insert or merge characters by normalized name and link them to this episode.", Params: obj(map[string]any{"characters": arr, "episode_id": str}, "characters")},
		{Name: "drama_save_scenes", Desc: "Insert or merge scenes by location+time and link them to this episode.", Params: obj(map[string]any{"scenes": arr, "episode_id": str}, "scenes")},
		{Name: "drama_save_props", Desc: "Insert or merge at most three plot-critical props and link them to this episode.", Params: obj(map[string]any{"props": arr, "episode_id": str}, "props")},
		{Name: "drama_read_storyboard_context", ReadOnly: true, Desc: "Screenplay, linked assets, duration plan, and existing shots for storyboarding.", Params: obj(map[string]any{"episode_id": str})},
		{Name: "drama_save_storyboards", Desc: "Write shot segments. First batch must set replace_existing true. At most 8 per call. Continue until the episode is covered.", Params: obj(map[string]any{
			"storyboards": arr, "shots": arr, "replace_existing": map[string]any{"type": "boolean"}, "episode_id": str,
		})},
		{Name: "drama_update_storyboard", Desc: "Patch one shot. Prefer storyboard_id plus only the fields that changed.", Params: obj(map[string]any{
			"storyboard_id": str, "id": str, "video_prompt": str, "description": str, "duration": map[string]any{"type": "integer"},
		})},
		{Name: "drama_save_final_prompt", Desc: "Store a style-prefixed still prompt. kind is character, scene, or prop. Do not put style words in prompt; the tool prefixes the project style.", Params: obj(map[string]any{"kind": str, "id": str, "prompt": str}, "kind", "id", "prompt")},
	}
}
