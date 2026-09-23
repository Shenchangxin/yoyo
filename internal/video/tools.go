package video

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (e *Engine) Tool(sessionID, name string, args map[string]any) (string, error) {
	b, ok := e.SessionBind(sessionID)
	if !ok && strMap(args, "episode_id") == "" {
		return "", fmt.Errorf("no episode bound to this thread — open Short drama first")
	}
	episodeID := first(strMap(args, "episode_id"), b.EpisodeID)
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return "", err
	}
	switch name {
	case "drama_read_episode":
		return marshalJSON(map[string]any{
			"episode_id": ep.ID, "drama_id": ep.DramaID, "title": ep.Title,
			"content": ep.Content, "script_content": ep.ScriptContent,
			"language": e.ContentLanguage(),
		}), nil
	case "drama_save_script":
		script := strMap(args, "script")
		if script == "" {
			script = strMap(args, "script_content")
		}
		if script == "" {
			return "", fmt.Errorf("script is empty")
		}
		if err := e.SaveScript(episodeID, script); err != nil {
			return "", err
		}
		_ = e.patchPipeline(episodeID, "rewrite", "done", "")
		return "saved screenplay", nil
	case "drama_read_assets":
		bundle, err := e.Bundle(episodeID)
		if err != nil {
			return "", err
		}
		return marshalJSON(map[string]any{
			"characters": bundle.Characters, "scenes": bundle.Scenes, "props": bundle.Props,
		}), nil
	case "drama_save_characters":
		items := decodeList[Character](args["characters"])
		out, err := e.SaveCharacters(episodeID, items)
		if err != nil {
			return "", err
		}
		_ = e.patchPipeline(episodeID, "extract", "done", "")
		return marshalJSON(out), nil
	case "drama_save_scenes":
		items := decodeList[Scene](args["scenes"])
		out, err := e.SaveScenes(episodeID, items)
		if err != nil {
			return "", err
		}
		_ = e.patchPipeline(episodeID, "extract", "done", "")
		return marshalJSON(out), nil
	case "drama_save_props":
		items := decodeList[Prop](args["props"])
		out, err := e.SaveProps(episodeID, items)
		if err != nil {
			return "", err
		}
		_ = e.patchPipeline(episodeID, "extract", "done", "")
		return marshalJSON(out), nil
	case "drama_read_storyboard_context":
		bundle, err := e.Bundle(episodeID)
		if err != nil {
			return "", err
		}
		script := bundle.Episode.ScriptContent
		if script == "" {
			script = bundle.Episode.Content
		}
		return marshalJSON(map[string]any{
			"script": script, "characters": bundle.Characters, "scenes": bundle.Scenes, "props": bundle.Props,
			"existing_storyboards": bundle.Shots, "plan": bundle.Plan, "language": e.ContentLanguage(),
		}), nil
	case "drama_save_storyboards":
		shots := decodeList[Shot](args["storyboards"])
		if shots == nil {
			shots = decodeList[Shot](args["shots"])
		}
		replace := boolArg(args["replace_existing"]) || boolArg(args["replace"])
		if len(shots) > 8 {
			shots = shots[:8]
		}
		out, err := e.SaveShots(episodeID, shots, replace)
		if err != nil {
			return "", err
		}
		_ = e.patchPipeline(episodeID, "storyboard", "done", "")
		return marshalJSON(out), nil
	case "drama_update_storyboard":
		id := strMap(args, "storyboard_id")
		if id == "" {
			id = strMap(args, "id")
		}
		s, err := e.GetShot(id)
		if err != nil {
			return "", err
		}
		if v := strMap(args, "video_prompt"); v != "" {
			s.VideoPrompt = v
		}
		if v := strMap(args, "description"); v != "" {
			s.Description = v
		}
		if n := intArg(args["duration"]); n > 0 {
			s.Duration = n
		}
		s, err = e.UpdateShot(s)
		if err != nil {
			return "", err
		}
		if strMap(args, "video_prompt") != "" {
			_ = e.patchPipeline(episodeID, "video_prompts", "done", "")
		}
		return marshalJSON(s), nil
	case "drama_save_final_prompt":
		kind := strMap(args, "kind")
		id := strMap(args, "id")
		prompt := strMap(args, "prompt")
		d, _ := e.GetDrama(ep.DramaID)
		if err := e.SaveFinalPrompt(kind, id, prompt, d.Style); err != nil {
			return "", err
		}
		_ = e.patchPipeline(episodeID, "prompts", "done", "")
		return "saved final prompt", nil
	default:
		return "", fmt.Errorf("unknown drama tool %s", name)
	}
}

func (e *Engine) StagePrompt(stage, episodeID string) (string, []string, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return "", nil, err
	}
	lang := e.ContentLanguage()
	langLine := contentLangDirective(lang)
	switch stage {
	case "rewrite":
		_ = e.patchPipeline(episodeID, "rewrite", "running", "")
		return strings.Join([]string{
			"Load the drama-script-rewriter skill.",
			"Call drama_read_episode, rewrite the novel into a formatted screenplay, then drama_save_script with the full script.",
			"Do not narrate. Tool calls only until saved.",
			langLine,
			"Episode " + ep.Title + " is already bound.",
		}, "\n"), []string{"drama-script-rewriter"}, nil
	case "extract":
		_ = e.patchPipeline(episodeID, "extract", "running", "")
		return strings.Join([]string{
			"Load the drama-extractor skill.",
			"Read the screenplay, dedupe against existing assets, then save characters, scenes, and props with the drama_save_* tools.",
			"Props: only plot-critical objects that deserve their own still. Usually 0–3.",
			"Keep narrator/voice-over as a character row if they speak, but do not invent a face.",
			"Do not narrate. Tool calls only.",
			langLine,
		}, "\n"), []string{"drama-extractor"}, nil
	case "extract_characters":
		_ = e.patchPipeline(episodeID, "extract", "running", "")
		return strings.Join([]string{
			"Load the drama-extractor skill.",
			"Extract only on-screen characters. Save with drama_save_characters. Do not save scenes or props.",
			"Keep narrator/voice-over as a character if they speak, without inventing a face.",
			"Do not narrate. Tool calls only.",
			langLine,
		}, "\n"), []string{"drama-extractor"}, nil
	case "extract_scenes":
		_ = e.patchPipeline(episodeID, "extract", "running", "")
		return strings.Join([]string{
			"Load the drama-extractor skill.",
			"Extract only establishing locations. Save with drama_save_scenes. Do not save characters or props.",
			"Do not narrate. Tool calls only.",
			langLine,
		}, "\n"), []string{"drama-extractor"}, nil
	case "extract_props":
		_ = e.patchPipeline(episodeID, "extract", "running", "")
		return strings.Join([]string{
			"Load the drama-extractor skill.",
			"Extract only plot-critical props that need their own still. At most three. Save with drama_save_props. Do not save characters or scenes.",
			"Do not narrate. Tool calls only.",
			langLine,
		}, "\n"), []string{"drama-extractor"}, nil
	case "prompts":
		_ = e.patchPipeline(episodeID, "prompts", "running", "")
		return strings.Join([]string{
			"Load drama-prompt-character, drama-prompt-scene, and drama-prompt-prop.",
			"For every linked character, scene, and prop without a strong final_prompt, write one and save it with drama_save_final_prompt.",
			"Do not add style words; the tool prefixes the project style.",
			langLine,
		}, "\n"), []string{"drama-prompt-character", "drama-prompt-scene", "drama-prompt-prop"}, nil
	case "storyboard":
		_ = e.patchPipeline(episodeID, "storyboard", "running", "")
		return strings.Join([]string{
			"Load the drama-storyboard skill.",
			"Read context, then save segments with drama_save_storyboards. First batch replace_existing true. At most 8 per batch. Continue until the episode is fully covered.",
			"Each segment is 8–15 seconds, 2–4 subshots, one video job.",
			"Do not narrate. Tool calls only.",
			langLine,
		}, "\n"), []string{"drama-storyboard"}, nil
	case "video_prompts":
		_ = e.patchPipeline(episodeID, "video_prompts", "running", "")
		return strings.Join([]string{
			"Load drama-prompt-video.",
			"For shots missing video_prompt, write the prompt and drama_update_storyboard with only storyboard_id and video_prompt.",
			"First line is an info header listing on-screen people and the scene with @Name.",
			"Then one line per ~3 seconds. Align cuts with 【镜头N】 in description. Do not invent dialogue.",
			langLine,
		}, "\n"), []string{"drama-prompt-video"}, nil
	default:
		return "", nil, fmt.Errorf("unknown stage")
	}
}

func contentLangDirective(lang string) string {
	if lang == "" || lang == "zh" || lang == "zh-CN" {
		return "HIGHEST PRIORITY: write all saved creative text in Simplified Chinese."
	}
	return "HIGHEST PRIORITY: write all saved creative text in " + lang + ". Ignore any skill line that says to output Chinese only."
}

func decodeList[T any](v any) []T {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out []T
	if json.Unmarshal(b, &out) != nil {
		return nil
	}
	return out
}

func boolArg(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	case float64:
		return t != 0
	}
	return false
}

func intArg(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	}
	return 0
}
