// 短片工具的名字、参数，以及在一场会话里怎么分发。
// 没带工程编号时用这个会话绑着的那条。打开发布页、数据页和创建人设不在这里做。

package video

import (
	"encoding/json"
	"fmt"
	"strings"
)

// StudioToolNames 按规格表返回工具名，顺序和 StudioToolSpecs 一致。
func StudioToolNames() []string {
	specs := StudioToolSpecs()
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Name)
	}
	return out
}

// StudioToolSpecs 返回短片工具的名字、说明和参数形状。说明文本是给模型看的，分发时不解释它们。
func StudioToolSpecs() []ToolSpec {
	str := map[string]any{"type": "string"}
	num := map[string]any{"type": "number"}
	obj := func(props map[string]any, req ...string) map[string]any {
		m := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 {
			m["required"] = req
		}
		return m
	}
	arr := map[string]any{"type": "array", "items": str}
	shots := map[string]any{"type": "array", "items": map[string]any{"type": "object"}}
	return []ToolSpec{
		{Name: "trends_fetch", ReadOnly: true, Desc: "Fetch live Chinese social hot lists (weibo/douyin/zhihu/bilibili/toutiao/baidu). Do not scrape platform homepages.", Params: obj(map[string]any{"platforms": str, "limit": num})},
		{Name: "topic_list", ReadOnly: true, Desc: "List shorts topic cards. Optional status: inbox, ready, in_production, done.", Params: obj(map[string]any{"status": str})},
		{Name: "topic_save", Desc: "Create or update a content topic card (title, source, angle, scores). Not a personal-OS idea.", Params: obj(map[string]any{
			"id": str, "title": str, "source_platform": str, "source_title": str, "angle": str, "format": str, "note": str, "status": str,
			"scores": map[string]any{"type": "object"},
		}, "title")},
		{Name: "topic_get", ReadOnly: true, Desc: "Read one topic card.", Params: obj(map[string]any{"id": str}, "id")},
		{Name: "studio_create", Desc: "Open a shorts project from a topic or a title. Confirm aspect with the operator first.", Params: obj(map[string]any{"title": str, "topic_id": str, "aspect": str, "script": str, "preset_id": str, "template_id": str})},
		{Name: "studio_get", ReadOnly: true, Desc: "Read the bound shorts project, or pass id.", Params: obj(map[string]any{"id": str})},
		{Name: "studio_update", Desc: "Patch script, aspect, style, or storyboard shots on the bound project.", Params: obj(map[string]any{
			"id": str, "title": str, "aspect": str, "style": str, "script": str, "shots": shots,
			"narration_hash": str, "subtitle_hash": str, "bgm_hash": str, "result_hash": str,
			"hook": str, "cta": str, "platform": str, "tone": str, "voice": str, "preset_id": str,
			"template_id": str, "gate": str, "audience": str, "target_duration": num, "voice_speed": num,
		})},
		{Name: "studio_catalog", ReadOnly: true, Desc: "List shorts presets, templates, character/style contracts, and TTS voices.", Params: obj(map[string]any{})},
		{Name: "studio_preflight", ReadOnly: true, Desc: "Report which adapters a gate will spend before running it.", Params: obj(map[string]any{"id": str, "gate": str})},
		{Name: "studio_approve_script", Desc: "Lock the spoken script (needs_review → approved) so TTS and image gen may spend.", Params: obj(map[string]any{"id": str})},
		{Name: "studio_skip_audio", Desc: "Skip TTS after the script is approved and continue to visuals.", Params: obj(map[string]any{"id": str})},
		{Name: "studio_apply_preset", Desc: "Lock 9:16 Douyin / 16:9 Bilibili (or other) output preset on the bound project.", Params: obj(map[string]any{"id": str, "preset_id": str}, "preset_id")},
		{Name: "studio_apply_template", Desc: "Start from a shorts template (explainer, UGC hook, product, news, story, howto).", Params: obj(map[string]any{"id": str, "template_id": str}, "template_id")},
		{Name: "studio_set_identity", Desc: "Set character/style identity contract (preserve/prohibit) used in shot prompts.", Params: obj(map[string]any{"id": str, "character_id": str, "style_id": str})},
		{Name: "studio_generate_shot", Desc: "Enqueue image generation for one shot. Writes a real CAS hash when the job finishes. Never invent hashes.", Params: obj(map[string]any{"id": str, "shot_id": str})},
		{Name: "studio_generate_missing", Desc: "Enqueue image generation for shots that lack media or are stale. Never invent hashes.", Params: obj(map[string]any{"id": str})},
		{Name: "studio_cover", Desc: "Enqueue a thumbnail candidate for the bound short.", Params: obj(map[string]any{"id": str})},
		{Name: "studio_pick_cover", Desc: "Choose a cover CAS hash from generated candidates.", Params: obj(map[string]any{"id": str, "hash": str}, "hash")},
		{Name: "studio_copy", Desc: "Write platform publish copy from the script. Does not fabricate experience or efficacy.", Params: obj(map[string]any{"id": str, "platform": str})},
		{Name: "studio_package", Desc: "Build the local delivery pack (mp4, srt, cover, copy) and mark publishable when complete. Does not post.", Params: obj(map[string]any{"id": str})},
		{Name: "studio_restore_version", Desc: "Restore a previous assembled version by number. Does not overwrite CAS objects.", Params: obj(map[string]any{"id": str, "n": num}, "n")},
		{Name: "studio_expand_topic", Desc: "Generate a few topic angles to pick from. Does not replace scores.", Params: obj(map[string]any{"topic_id": str}, "topic_id")},
		{Name: "studio_choose_option", Desc: "Lock one expanded topic option onto the card.", Params: obj(map[string]any{"topic_id": str, "option_id": str}, "topic_id", "option_id")},
		{Name: "studio_ingest", Desc: "Download a public URL (yt-dlp or direct) into a shot with a real CAS hash.", Params: obj(map[string]any{"id": str, "url": str}, "url")},
		{Name: "studio_autoclip", Desc: "Split a source CAS video into highlight clips. Returns hashes and scores. Does not concatenate.", Params: obj(map[string]any{"hash": str, "count": num, "duration": str}, "hash")},
		{Name: "studio_apply_timing", Desc: "Set shot durations from narration SRT / ffprobe after Speak.", Params: obj(map[string]any{"id": str})},
		{Name: "studio_assemble", Desc: "FFmpeg-assemble the bound storyboard into an mp4 and add it to the asset library. Requires shots with image_hash or video_hash.", Params: obj(map[string]any{"id": str})},
		{Name: "studio_auto", Desc: "One-click short: lock or draft the script, speak (edge-tts if no speech key), generate missing stills, then assemble. Spends chat/speech/image keys. Returns immediately; poll studio_get auto_phase until done or failed.", Params: obj(map[string]any{"id": str})},
		{Name: "studio_auto_topics", Desc: "One-click one or more topics. Picks aspect, template, voice, and length, then drafts, speaks, fills stills, and assembles. Does not post. Returns immediately; poll each auto_phase. Finished films land on delivery for the operator to choose a platform.", Params: obj(map[string]any{"topic_ids": arr}, "topic_ids")},
		{Name: "studio_reframe", Desc: "Pad/crop a CAS video to 9:16, 16:9, or 1:1.", Params: obj(map[string]any{"hash": str, "aspect": str}, "hash")},
		{Name: "studio_burnin", Desc: "Burn an SRT (CAS hash) into a video.", Params: obj(map[string]any{"hash": str, "subtitle_hash": str}, "hash", "subtitle_hash")},
		{Name: "studio_slideshow", Desc: "Ken Burns slideshow from image CAS hashes.", Params: obj(map[string]any{"hashes": arr, "aspect": str, "duration": num}, "hashes")},
		{Name: "studio_speak", Desc: "TTS via the configured speech connection. Writes audio CAS + optional SRT onto the bound project.", Params: obj(map[string]any{"text": str, "id": str}, "text")},
		{Name: "studio_publish_draft", Desc: "Create a social publish draft (xhs/douyin/kuaishou/channels/zhihu/bilibili). Does not post. Yoyo never clicks Publish.", Params: obj(map[string]any{
			"platform": str, "title": str, "caption": str, "tags": str, "id": str, "topic_id": str, "media_hash": str,
		}, "platform", "title")},
		{Name: "studio_publish_list", ReadOnly: true, Desc: "List shorts publish drafts and posted records.", Params: obj(map[string]any{"platform": str})},
		{Name: "studio_publish_export", Desc: "Write the assembled mp4 next to the video home so the operator can drop it on the creator page.", Params: obj(map[string]any{"publish_id": str}, "publish_id")},
		{Name: "studio_publish_open", Desc: "Export the mp4 and open the platform creator URL in isolated Chrome. Operator finishes login/upload. Never auto-clicks Publish.", Params: obj(map[string]any{"publish_id": str}, "publish_id")},
		{Name: "studio_publish_mark", Desc: "Record that the operator posted, failed, or pasted analytics. status: opened, posted, failed.", Params: obj(map[string]any{
			"publish_id": str, "status": str, "posted_url": str, "note": str, "reads": num, "likes": num, "comments": num,
		}, "publish_id", "status")},
		{Name: "studio_publish_analytics", Desc: "Open the platform analytics URL in isolated Chrome. Operator pastes numbers back. Yoyo does not scrape.", Params: obj(map[string]any{"publish_id": str}, "publish_id")},
		{Name: "studio_remotion_status", ReadOnly: true, Desc: "Report Remotion install state. bundled is always false. ready means the operator-installed composition under the video home can render.", Params: obj(map[string]any{})},
		{Name: "studio_remotion_install", Desc: "Start an opt-in Remotion install into the video home (~540 MB, needs Node.js 18+). Never part of the default installer. Poll studio_remotion_status until ready or error.", Params: obj(map[string]any{})},
		{Name: "studio_wrap", Desc: "Burn a title onto a CAS video. Uses Remotion TitleCard when ready, otherwise FFmpeg SRT burn-in.", Params: obj(map[string]any{"hash": str, "title": str}, "hash", "title")},
		{Name: "studio_persona_template", ReadOnly: true, Desc: "Return a creator-persona page template. Not a session Profile.", Params: obj(map[string]any{"name": str})},
		{Name: "studio_persona_create", Desc: "Create a Pages persona template and optionally link it to a topic. Not a session Profile.", Params: obj(map[string]any{"name": str, "topic_id": str})},
	}
}

// StudioTool 按名字调用对应的短片操作，并返回 JSON。工程编号空着就用会话绑定。
// studio_publish_open、studio_publish_analytics 和 studio_persona_create 直接拒绝，交给应用层。未知名字也拒绝。不在这里点击发布。
func (e *Engine) StudioTool(sessionID, name string, args map[string]any) (string, error) {
	if args == nil {
		args = map[string]any{}
	}
	id := strAnyMap(args, "id", "project_id")
	if id == "" {
		id, _ = e.StudioSessionBind(sessionID)
	}
	switch name {
	case "trends_fetch":
		platforms := splitCSV(strAnyMap(args, "platforms"))
		limit := intAny(args["limit"])
		out, err := e.FetchTrends(platforms, limit)
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "topic_list":
		return marshalJSON(e.ListTopics(strAnyMap(args, "status"))), nil
	case "topic_get":
		t, err := e.GetTopic(strAnyMap(args, "id"))
		if err != nil {
			return "", err
		}
		return marshalJSON(t), nil
	case "topic_save":
		var t Topic
		b, _ := json.Marshal(args)
		_ = json.Unmarshal(b, &t)
		out, err := e.SaveTopic(t)
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_create":
		var p StudioProject
		b, _ := json.Marshal(args)
		_ = json.Unmarshal(b, &p)
		var out StudioProject
		var err error
		if p.TopicID != "" && p.Title == "" {
			out, err = e.StudioFromTopic(p.TopicID, p.Aspect)
		} else {
			out, err = e.CreateStudioProject(p)
		}
		if err != nil {
			return "", err
		}
		if tid := strAnyMap(args, "template_id"); tid != "" {
			out, err = e.ApplyStudioTemplate(out.ID, tid)
			if err != nil {
				return "", err
			}
		} else if pid := strAnyMap(args, "preset_id"); pid != "" {
			out, err = e.ApplyStudioPreset(out.ID, pid)
			if err != nil {
				return "", err
			}
		}
		if sessionID != "" {
			_ = e.BindStudioSession(sessionID, out.ID)
		}
		return marshalJSON(out), nil
	case "studio_get":
		if id == "" {
			return "", fmt.Errorf("no studio project bound — create one first")
		}
		p, err := e.GetStudioProject(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "studio_update":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		p, err := e.GetStudioProject(id)
		if err != nil {
			return "", err
		}
		p.Title = firstNonEmpty(strAnyMap(args, "title"), p.Title)
		p.Aspect = firstNonEmpty(strAnyMap(args, "aspect"), p.Aspect)
		p.Style = firstNonEmpty(strAnyMap(args, "style"), p.Style)
		p.Script = firstNonEmpty(strAnyMap(args, "script"), p.Script)
		p.ResultHash = firstNonEmpty(strAnyMap(args, "result_hash"), p.ResultHash)
		p.NarrationHash = firstNonEmpty(strAnyMap(args, "narration_hash"), p.NarrationHash)
		p.SubtitleHash = firstNonEmpty(strAnyMap(args, "subtitle_hash"), p.SubtitleHash)
		p.BGMHash = firstNonEmpty(strAnyMap(args, "bgm_hash"), p.BGMHash)
		p.Hook = firstNonEmpty(strAnyMap(args, "hook"), p.Hook)
		p.CTA = firstNonEmpty(strAnyMap(args, "cta"), p.CTA)
		p.Platform = firstNonEmpty(strAnyMap(args, "platform"), p.Platform)
		p.Tone = firstNonEmpty(strAnyMap(args, "tone"), p.Tone)
		p.Voice = firstNonEmpty(strAnyMap(args, "voice"), p.Voice)
		p.PresetID = firstNonEmpty(strAnyMap(args, "preset_id"), p.PresetID)
		p.TemplateID = firstNonEmpty(strAnyMap(args, "template_id"), p.TemplateID)
		p.Gate = firstNonEmpty(strAnyMap(args, "gate"), p.Gate)
		p.Audience = firstNonEmpty(strAnyMap(args, "audience"), p.Audience)
		if d := intAny(args["target_duration"]); d > 0 {
			p.TargetDuration = d
		}
		if sp := floatAny(args["voice_speed"]); sp > 0 {
			p.VoiceSpeed = sp
		}
		if shots := args["shots"]; shots != nil {
			b, _ := json.Marshal(shots)
			_ = json.Unmarshal(b, &p.Storyboard.Shots)
		}
		out, err := e.UpdateStudioProject(p)
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_assemble":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		p, err := e.AssembleStudio(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "studio_auto":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		p, err := e.StartStudioAuto(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "studio_auto_topics":
		var ids []string
		b, _ := json.Marshal(args["topic_ids"])
		_ = json.Unmarshal(b, &ids)
		rows, err := e.StartTopicFilms(ids)
		if err != nil {
			return "", err
		}
		return marshalJSON(rows), nil
	case "studio_reframe":
		ref, err := e.Reframe(strAnyMap(args, "hash"), strAnyMap(args, "aspect"))
		if err != nil {
			return "", err
		}
		return marshalJSON(ref), nil
	case "studio_burnin":
		ref, err := e.BurnInCaptions(strAnyMap(args, "hash"), strAnyMap(args, "subtitle_hash"))
		if err != nil {
			return "", err
		}
		return marshalJSON(ref), nil
	case "studio_slideshow":
		var hashes []string
		b, _ := json.Marshal(args["hashes"])
		_ = json.Unmarshal(b, &hashes)
		ref, err := e.Slideshow(hashes, strAnyMap(args, "aspect"), floatAny(args["duration"]))
		if err != nil {
			return "", err
		}
		return marshalJSON(ref), nil
	case "studio_speak":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		p, err := e.SpeakStudio(id, strAnyMap(args, "text"))
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "studio_publish_draft":
		var pub Publish
		b, _ := json.Marshal(args)
		_ = json.Unmarshal(b, &pub)
		if pub.ProjectID == "" {
			pub.ProjectID = strAnyMap(args, "project_id")
		}
		if pub.ProjectID == "" {
			if bound, ok := e.StudioSessionBind(sessionID); ok {
				pub.ProjectID = bound
			}
		}
		out, err := e.SavePublish(pub)
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_publish_list":
		projectID := strAnyMap(args, "project_id")
		if projectID == "" {
			projectID, _ = e.StudioSessionBind(sessionID)
		}
		return marshalJSON(e.ListPublishes(projectID, strAnyMap(args, "platform"))), nil
	case "studio_publish_export":
		out, err := e.ExportPublish(firstNonEmpty(strAnyMap(args, "publish_id", "id")))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_publish_open", "studio_publish_analytics", "studio_persona_create":
		return "", fmt.Errorf("%s is hosted by the app", name)
	case "studio_publish_mark":
		out, err := e.MarkPublish(
			firstNonEmpty(strAnyMap(args, "publish_id", "id")),
			strAnyMap(args, "status"),
			strAnyMap(args, "posted_url", "url"),
			strAnyMap(args, "note"),
			intAny(args["reads"]), intAny(args["likes"]), intAny(args["comments"]),
		)
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_remotion_status":
		return marshalJSON(e.RemotionSnapshot()), nil
	case "studio_remotion_install":
		return marshalJSON(e.InstallRemotion()), nil
	case "studio_wrap":
		ref, err := e.WrapTalkingHead(strAnyMap(args, "hash"), strAnyMap(args, "title", "text"))
		if err != nil {
			return "", err
		}
		return marshalJSON(ref), nil
	case "studio_persona_template":
		return PersonaTemplate(strAnyMap(args, "name", "title")), nil
	case "studio_catalog":
		return marshalJSON(StudioCatalog()), nil
	case "studio_preflight":
		out, err := e.StudioPreflight(id, strAnyMap(args, "gate"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_approve_script":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		out, err := e.ApproveStudioScript(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_skip_audio":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		out, err := e.SkipStudioAudio(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_apply_preset":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		out, err := e.ApplyStudioPreset(id, strAnyMap(args, "preset_id"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_apply_template":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		out, err := e.ApplyStudioTemplate(id, strAnyMap(args, "template_id"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_set_identity":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		out, err := e.SetStudioIdentity(id, strAnyMap(args, "character_id"), strAnyMap(args, "style_id"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_generate_shot":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		j, err := e.GenerateStudioShot(id, strAnyMap(args, "shot_id"))
		if err != nil {
			return "", err
		}
		return marshalJSON(j), nil
	case "studio_generate_missing":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		j, err := e.GenerateMissingStudioShots(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(j), nil
	case "studio_cover":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		j, err := e.GenerateStudioCover(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(j), nil
	case "studio_pick_cover":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		out, err := e.PickStudioCover(id, strAnyMap(args, "hash"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_copy":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		out, err := e.WriteStudioCopy(id, strAnyMap(args, "platform"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_package":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		out, err := e.PackageStudio(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_restore_version":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		out, err := e.RestoreStudioVersion(id, intAny(args["n"]))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_expand_topic":
		out, err := e.ExpandTopic(strAnyMap(args, "topic_id", "id"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_choose_option":
		out, err := e.ChooseTopicOption(strAnyMap(args, "topic_id", "id"), strAnyMap(args, "option_id"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_ingest":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		out, err := e.IngestStudioURL(id, strAnyMap(args, "url"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_autoclip":
		out, err := e.AutoClip(strAnyMap(args, "hash"), intAny(args["count"]), strAnyMap(args, "duration", "window"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "studio_apply_timing":
		if id == "" {
			return "", fmt.Errorf("no studio project bound")
		}
		out, err := e.ApplyStudioTiming(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	default:
		return "", fmt.Errorf("unknown studio tool %s", name)
	}
}

// splitCSV 按逗号拆开并去掉空白。整段是空的就返回 nil。
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// floatAny 把数字、整数、json.Number 或数字字符串收成 float64。认不出就返回 0。
func floatAny(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case json.Number:
		n, _ := t.Float64()
		return n
	case string:
		var n float64
		fmt.Sscanf(strings.TrimSpace(t), "%f", &n)
		return n
	default:
		return 0
	}
}
