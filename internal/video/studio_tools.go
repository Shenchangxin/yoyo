package video

import (
	"encoding/json"
	"fmt"
	"strings"
)

func StudioToolNames() []string {
	specs := StudioToolSpecs()
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Name)
	}
	return out
}

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
		{Name: "studio_create", Desc: "Open a shorts project from a topic or a title. Confirm aspect with the operator first.", Params: obj(map[string]any{"title": str, "topic_id": str, "aspect": str, "script": str})},
		{Name: "studio_get", ReadOnly: true, Desc: "Read the bound shorts project, or pass id.", Params: obj(map[string]any{"id": str})},
		{Name: "studio_update", Desc: "Patch script, aspect, style, or storyboard shots on the bound project.", Params: obj(map[string]any{
			"id": str, "title": str, "aspect": str, "style": str, "script": str, "shots": shots,
			"narration_hash": str, "subtitle_hash": str, "bgm_hash": str, "result_hash": str,
		})},
		{Name: "studio_assemble", Desc: "FFmpeg-assemble the bound storyboard into an mp4 and add it to the asset library. Requires shots with image_hash or video_hash.", Params: obj(map[string]any{"id": str})},
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
		if p.TopicID != "" && p.Title == "" {
			out, err := e.StudioFromTopic(p.TopicID, p.Aspect)
			if err != nil {
				return "", err
			}
			if sessionID != "" {
				_ = e.BindStudioSession(sessionID, out.ID)
			}
			return marshalJSON(out), nil
		}
		out, err := e.CreateStudioProject(p)
		if err != nil {
			return "", err
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
	default:
		return "", fmt.Errorf("unknown studio tool %s", name)
	}
}

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
