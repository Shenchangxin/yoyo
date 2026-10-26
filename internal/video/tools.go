// tools.go 按名字调用短剧工具，并给出各阶段交给模型的提示。

package video

import (
	"encoding/json"
	"fmt"
	"strings"
)

// seriesTool 判断名字是不是剧级工具：摄入、读原文、读方案、提议分集、确认分集和读设定。
// name 与 drama_ingest_source、drama_read_source、drama_read_plan、drama_propose_episodes、drama_commit_episodes、drama_read_bible 比较。
func seriesTool(name string) bool {
	switch name {
	case "drama_ingest_source", "drama_read_source", "drama_read_plan",
		"drama_propose_episodes", "drama_commit_episodes", "drama_read_bible":
		return true
	default:
		return false
	}
}

// Tool 调用已绑定短剧的工具。没有剧也没有集时要求先打开短剧。剧级工具，以及保存角色、场景、道具时没有集标识，走 seriesTool。保存剧本后若目标超过 120 秒或段数超过 16，结果里带上 split_episode。保存分镜时每调用最多留下 8 条。未知名字返回 unknown drama tool。
// sessionID 传给 SessionBind。
// name 传给 seriesTool，并选择集级分支，对不上时写进 unknown drama tool。
// args 供 strMap、decodeList、boolArg 和 intArg 取出剧集、剧本、资产和分镜字段。
func (e *Engine) Tool(sessionID, name string, args map[string]any) (string, error) {
	b, ok := e.SessionBind(sessionID)
	episodeID := first(strMap(args, "episode_id"), b.EpisodeID)
	dramaID := first(strMap(args, "drama_id"), b.DramaID)
	if dramaID == "" && episodeID != "" {
		if ep, err := e.GetEpisode(episodeID); err == nil {
			dramaID = ep.DramaID
		}
	}
	if !ok && episodeID == "" && dramaID == "" {
		return "", fmt.Errorf("no series bound to this thread — open Short drama first")
	}
	if seriesTool(name) || ((name == "drama_save_characters" || name == "drama_save_scenes" || name == "drama_save_props") && episodeID == "") {
		return e.seriesTool(name, dramaID, args)
	}
	if episodeID == "" {
		return "", fmt.Errorf("no episode bound — confirm the episode map first")
	}
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
			"note":     "this is a committed slice, not the whole series. Use drama_read_source for neighbouring windows.",
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
		plan := PlanDuration(script)
		out := map[string]any{"ok": true, "plan": plan}
		if plan.TargetSeconds > 120 || plan.SegmentCount > 16 {
			out["split_episode"] = true
			out["warning"] = "screenplay exceeds a single short-drama episode; split this window instead of packing more than 8 shots"
		}
		return marshalJSON(out), nil
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
		cov, done := e.storyboardCoverage(episodeID)
		if done {
			_ = e.patchPipeline(episodeID, "storyboard", "done", "")
		}
		return marshalJSON(map[string]any{"shots": out, "coverage": cov, "continue": !done}), nil
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

// seriesTool 执行剧级读写。确认分集直接返回错误，说明这是桌面上的操作者动作。方案不存在时返回 status 为 missing。
// name 选择摄入、读取、提议、确认、设定或保存分支，对不上时写进错误文本。
// dramaID 为空时返回错误 no series bound，否则传给对应的读或写函数。
// args 在摄入原文、按窗口读取、提议分集和保存设定时取出 text、title、offset、length、slice_index、query、episodes、characters、scenes 或 props。
func (e *Engine) seriesTool(name, dramaID string, args map[string]any) (string, error) {
	if dramaID == "" {
		return "", fmt.Errorf("no series bound")
	}
	switch name {
	case "drama_ingest_source":
		src, err := e.IngestSource(dramaID, strMap(args, "text"), strMap(args, "title"))
		if err != nil {
			return "", err
		}
		return marshalJSON(src), nil
	case "drama_read_source":
		out, err := e.ReadSource(dramaID, intArg(args["offset"]), intArg(args["length"]), intArg(args["slice_index"]), strMap(args, "query"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "drama_read_plan":
		plan, err := e.GetPlan(dramaID)
		if err != nil {
			return marshalJSON(map[string]any{"status": "missing"}), nil
		}
		return marshalJSON(plan), nil
	case "drama_propose_episodes":
		items := decodeList[PlanEpisode](args["episodes"])
		plan, err := e.ProposeEpisodes(dramaID, items)
		if err != nil {
			return "", err
		}
		return marshalJSON(plan), nil
	case "drama_commit_episodes":
		return "", fmt.Errorf("commit is an operator action on the desk — do not create episodes from a tool")
	case "drama_read_bible":
		return marshalJSON(map[string]any{
			"characters": e.DramaCharacters(dramaID),
			"scenes":     e.DramaScenes(dramaID),
			"props":      e.DramaProps(dramaID),
		}), nil
	case "drama_save_characters":
		out, err := e.SaveBibleCharacters(dramaID, decodeList[Character](args["characters"]))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "drama_save_scenes":
		out, err := e.SaveBibleScenes(dramaID, decodeList[Scene](args["scenes"]))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "drama_save_props":
		out, err := e.SaveBibleProps(dramaID, decodeList[Prop](args["props"]))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	default:
		return "", fmt.Errorf("unknown drama tool %s", name)
	}
}

// storyboardCoverage 用已有镜头秒数和条数对照时长计划。达到 MinTotal 或段数即算完成。目标超过 120 秒且段数超过 8 时，未完成的结果带 split_episode。读不到集时 continue 为真且完成标志为假。
// episodeID 传给 ListShots 和 GetEpisode。
func (e *Engine) storyboardCoverage(episodeID string) (map[string]any, bool) {
	shots, _ := e.ListShots(episodeID)
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return map[string]any{"continue": true}, false
	}
	src := ep.ScriptContent
	if src == "" {
		src = ep.Content
	}
	plan := PlanDuration(src)
	have := 0
	for _, s := range shots {
		have += s.Duration
	}
	done := (plan.MinTotal > 0 && have >= plan.MinTotal) || (plan.SegmentCount > 0 && len(shots) >= plan.SegmentCount)
	tooLong := plan.TargetSeconds > 120 && plan.SegmentCount > 8
	return map[string]any{
		"have_seconds":  have,
		"need_seconds":  plan.MinTotal,
		"have_shots":    len(shots),
		"need_shots":    plan.SegmentCount,
		"continue":      !done,
		"split_episode": tooLong && !done,
		"plan":          plan,
	}, done
}

// StagePrompt 按阶段返回一段提示和要加载的技能名。outline 和 bible 要剧标识。rewrite、extract、prompts、storyboard、video_prompts 以及分项提取会把对应流水线标成 running。未知阶段返回 unknown stage。
// episodeID 在 outline 和 bible 中，仅当 dramaID 为空且自身非空时传给 GetEpisode；其他阶段传给 GetEpisode，并在若干阶段传给 patchPipeline。
// dramaID 为空且 episodeID 非空时，outline 和 bible 会改用集上的 DramaID；改完仍为空则返回错误。
func (e *Engine) StagePrompt(stage, episodeID, dramaID string) (string, []string, error) {
	lang := e.ContentLanguage()
	langLine := contentLangDirective(lang)
	if stage == "outline" {
		if dramaID == "" && episodeID != "" {
			if ep, err := e.GetEpisode(episodeID); err == nil {
				dramaID = ep.DramaID
			}
		}
		if dramaID == "" {
			return "", nil, fmt.Errorf("series required for outline")
		}
		return strings.Join([]string{
			"Load the drama-series-mapper skill, then drama-episode-outline.",
			"A heuristic episode map is already stored. Call drama_read_plan, then drama_read_source on the candidate seams (slice_index or offset). Peek only.",
			"Refine titles, loglines, hooks, and target_seconds (45–90s of finished film) with drama_propose_episodes. Snap cuts to existing seams. Do not invent cut points.",
			"Do not call drama_commit_episodes. Do not write Episode.content. Do not generate images or video.",
			"When the map is ready, tell the operator to confirm it on the desk.",
			langLine,
		}, "\n"), []string{"drama-series-mapper", "drama-episode-outline"}, nil
	}
	if stage == "bible" {
		if dramaID == "" && episodeID != "" {
			if ep, err := e.GetEpisode(episodeID); err == nil {
				dramaID = ep.DramaID
			}
		}
		if dramaID == "" {
			return "", nil, fmt.Errorf("series required for bible")
		}
		return strings.Join([]string{
			"Load the drama-extractor skill.",
			"This is a series-level bible pass. Call drama_read_bible, then drama_read_source for peeks of each planned episode. Merge by normalized name.",
			"Save with drama_save_characters / drama_save_scenes / drama_save_props without an episode_id so they stay on the series.",
			"Set first_seen_episode when you know it. Do not generate stills. Do not narrate.",
			langLine,
		}, "\n"), []string{"drama-extractor"}, nil
	}
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return "", nil, err
	}
	switch stage {
	case "rewrite":
		_ = e.patchPipeline(episodeID, "rewrite", "running", "")
		return strings.Join([]string{
			"Load the drama-script-rewriter skill.",
			"Call drama_read_episode. That text is already a committed slice of the series, not the whole novel. Do not pull the full source.",
			"Rewrite into a formatted screenplay, then drama_save_script with the full script.",
			"If the saved plan is over 120 seconds, stop and say the slice must be split. Do not pack it into 8 shots.",
			"Do not narrate. Tool calls only until saved.",
			langLine,
			"Episode " + ep.Title + " is already bound.",
		}, "\n"), []string{"drama-script-rewriter"}, nil
	case "extract":
		_ = e.patchPipeline(episodeID, "extract", "running", "")
		return strings.Join([]string{
			"Load the drama-extractor skill.",
			"Read the screenplay, dedupe against existing assets (drama_read_assets and drama_read_bible), then save characters, scenes, and props with the drama_save_* tools.",
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
			"Read context, then save segments with drama_save_storyboards. First batch replace_existing true. At most 8 per batch.",
			"Honor coverage.continue: keep calling until continue is false. Cover this episode's duration plan, not the whole novel.",
			"If coverage.split_episode is true, stop and say the slice must be split. Do not truncate the rest of the episode.",
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

// contentLangDirective 在语言为空、zh 或 zh-CN 时要求保存的创作文本用简体中文，否则要求用给定语言。
// lang 为空、zh 或 zh-CN 时返回简体中文优先的那句，否则把 lang 拼进另一句。
func contentLangDirective(lang string) string {
	if lang == "" || lang == "zh" || lang == "zh-CN" {
		return "HIGHEST PRIORITY: write all saved creative text in Simplified Chinese."
	}
	return "HIGHEST PRIORITY: write all saved creative text in " + lang + ". Ignore any skill line that says to output Chinese only."
}

// decodeList 把值经 JSON 解成切片。空值或解不开时返回 nil。
// v 为 nil 时返回 nil；否则经 json.Marshal 再 json.Unmarshal 成切片，任一步失败也返回 nil。
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

// boolArg 接受布尔、字符串 true 或 1，以及非 0 的 float64。其他值返回假。
// v 为 bool 时原样返回，为字符串 true 或 1 时返回真，为 float64 且不等于 0 时返回真，其余返回假。
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

// intArg 从 float64、int 或 json.Number 取出整数。其他值返回 0。
// v 为 float64 或 int 时转成 int，为 json.Number 时取 Int64 再转成 int，其余返回 0。
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
