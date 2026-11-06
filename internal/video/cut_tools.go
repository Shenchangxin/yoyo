// cut_tools.go 列出剪辑工具和两条路由工具，并按名字调用。

package video

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CutToolNames 按 CutToolSpecs 的顺序返回剪辑工具名。
func CutToolNames() []string {
	specs := CutToolSpecs()
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Name)
	}
	return out
}

// ConductorToolNames 返回 video_route 和 provider_menu。
func ConductorToolNames() []string {
	return []string{"video_route", "provider_menu"}
}

// CutToolSpecs 返回剪辑工程从创建、分析、提议、拼接到字幕、配音和调色的工具规格。Desc 是给模型看的说明。
func CutToolSpecs() []ToolSpec {
	str := map[string]any{"type": "string"}
	num := map[string]any{"type": "number"}
	boolean := map[string]any{"type": "boolean"}
	arr := map[string]any{"type": "array", "items": str}
	obj := func(props map[string]any, req ...string) map[string]any {
		m := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 {
			m["required"] = req
		}
		return m
	}
	return []ToolSpec{
		{Name: "cut_create", Desc: "Open a footage project for ingest, analysis, and post. Bind it to this chat.", Params: obj(map[string]any{"title": str, "hash": str})},
		{Name: "cut_get", ReadOnly: true, Desc: "Read the bound footage project, or pass id.", Params: obj(map[string]any{"id": str})},
		{Name: "cut_ingest", Desc: "Attach a CAS video hash as the source. Analyze after ingest.", Params: obj(map[string]any{"id": str, "hash": str, "title": str}, "hash")},
		{Name: "cut_analyze", Desc: "Scene-detect and silencedetect the source. Does not spend a vendor key.", Params: obj(map[string]any{"id": str})},
		{Name: "cut_propose", Desc: "Score highlight windows from scene+speech energy and extract clips WITH audio. count default 3. window 15-30 or 60-90. The edit list becomes these clips.", Params: obj(map[string]any{"id": str, "count": num, "window": str})},
		{Name: "cut_keep", Desc: "Keep only these proposed clip ids in the edit list. Call cut_apply afterwards.", Params: obj(map[string]any{"id": str, "clip_ids": arr}, "clip_ids")},
		{Name: "cut_omit", Desc: "Toggle skipping one caption cue by index. A source cue cuts its span out of the edit list. Call cut_apply afterwards.", Params: obj(map[string]any{"id": str, "index": num}, "index")},
		{Name: "cut_apply", Desc: "Render the film from the edit list (or the proposed clips), keeping audio. Grade, enhance, burned captions, and dub are redone from the source on every render, so nothing stacks.", Params: obj(map[string]any{"id": str})},
		{Name: "cut_extract", Desc: "Dump the soundtrack of the result or source onto the original stem for download.", Params: obj(map[string]any{"id": str})},
		{Name: "cut_enhance", Desc: "Set the voice cleanup preset (voice, denoise, loudnorm; none clears) and re-render.", Params: obj(map[string]any{"id": str, "preset": str})},
		{Name: "cut_mix", Desc: "Put the dub (or voice) and music stems on the film and re-render. By default the original audio stays under the dub, ducked; replace=true drops it. duck=false keeps levels flat. The film keeps the picture length.", Params: obj(map[string]any{"id": str, "duck": boolean, "replace": boolean})},
		{Name: "cut_transcribe", Desc: "Local whisper.cpp transcript of the source (needs whisper-cli and a ggml model). Cue times are on the source timeline.", Params: obj(map[string]any{"id": str})},
		{Name: "cut_captions", Desc: "Lay caption text over the film timeline (or only set language) and store SRT/VTT hashes.", Params: obj(map[string]any{"id": str, "text": str, "language": str})},
		{Name: "cut_burnin", Desc: "Turn on burned captions and re-render. Source captions follow the edit list; burning twice does not stack.", Params: obj(map[string]any{"id": str})},
		{Name: "cut_dub", Desc: "TTS via the enabled speech connection in Settings, or local edge-tts. Writes the dub stem only; call cut_mix to put it on the film. Does not invent keys.", Params: obj(map[string]any{"id": str, "text": str, "voice": str})},
		{Name: "cut_grade", Desc: "Set a color preset (warm, cool, contrast, fade; none clears) and re-render from the source. Grades never stack.", Params: obj(map[string]any{"id": str, "preset": str})},
		{Name: "cut_set_stem", Desc: "Attach a CAS hash as music/sfx/voice/dub/original. Call cut_mix to hear it on the film.", Params: obj(map[string]any{"id": str, "kind": str, "hash": str}, "kind", "hash")},
	}
}

// ConductorToolSpecs 返回打开工作台的 video_route，以及列出已启用且有密钥的账号的 provider_menu。
func ConductorToolSpecs() []ToolSpec {
	str := map[string]any{"type": "string"}
	obj := func(props map[string]any, req ...string) map[string]any {
		m := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 {
			m["required"] = req
		}
		return m
	}
	return []ToolSpec{
		{Name: "video_route", Desc: "Open or bind a workshop. dest is studio (短片), drama (短剧), canvas (画布), or cut (胶片/创作). Creates a shorts or cut project when id is empty.", Params: obj(map[string]any{"dest": str, "id": str, "title": str, "intent": str}, "dest")},
		{Name: "provider_menu", ReadOnly: true, Desc: "List Settings accounts that are enabled and have a vault key. Missing vendors do not exist. Optional capability filter: speech, music, stock, asr, image, video.", Params: obj(map[string]any{"capability": str})},
	}
}

// CutTool 按名字调用剪辑或路由。参数里没有工程标识时用会话绑定。cut_mix 在 duck 缺省时为真，字符串 0 或 false 时为假。未知名字返回 unknown cut tool。
// sessionID 在参数里没有工程标识时传给 CutSessionBind，非空时传给 BindCutSession，并传给 cutOf 和 VideoRoute。
// name 与 cut_create、cut_get、cut_ingest、cut_analyze、cut_propose、cut_apply、cut_extract、cut_enhance、cut_mix、cut_transcribe、cut_captions、cut_burnin、cut_dub、cut_grade、cut_set_stem、video_route、provider_menu 比较，其他值返回 unknown cut tool。
// args 为 nil 时换成空 map；cut_mix 在没有 duck 键时为真，该键为字符串 0 或 false 时为假，其余分支读取 id、project_id、title、hash、count、window、duration、preset、text、language、lang、voice、kind、dest、to、intent、capability、cap。
func (e *Engine) CutTool(sessionID, name string, args map[string]any) (string, error) {
	if args == nil {
		args = map[string]any{}
	}
	id := strAnyMap(args, "id", "project_id")
	if id == "" {
		id, _ = e.CutSessionBind(sessionID)
	}
	switch name {
	case "cut_create":
		// 会话已经绑着一条有源片的工程、又没给新源片时就用那一条，不另开空工程把绑定抢走。
		if strAnyMap(args, "hash") == "" && id != "" {
			if cur, err := e.GetCutProject(id); err == nil && cur.SourceHash != "" {
				return marshalJSON(cur), nil
			}
		}
		p, err := e.CreateCutProject(CutProject{Title: strAnyMap(args, "title")})
		if err != nil {
			return "", err
		}
		if h := strAnyMap(args, "hash"); h != "" {
			p, err = e.CutIngest(p.ID, h, p.Title)
			if err != nil {
				return "", err
			}
		}
		if sessionID != "" {
			_ = e.BindCutSession(sessionID, p.ID)
		}
		return marshalJSON(p), nil
	case "cut_get":
		p, err := e.cutOf(sessionID, id)
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_ingest":
		p, err := e.CutIngest(id, strAnyMap(args, "hash"), strAnyMap(args, "title"))
		if err != nil {
			return "", err
		}
		if sessionID != "" {
			_ = e.BindCutSession(sessionID, p.ID)
		}
		return marshalJSON(p), nil
	case "cut_analyze":
		p, err := e.CutAnalyze(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_propose":
		p, err := e.CutPropose(id, intAny(args["count"]), strAnyMap(args, "window", "duration"))
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_keep":
		var ids []string
		b, _ := json.Marshal(args["clip_ids"])
		_ = json.Unmarshal(b, &ids)
		p, err := e.CutKeep(id, ids)
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_omit":
		p, err := e.CutOmitCue(id, intAny(args["index"]))
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_apply":
		p, err := e.CutApply(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_extract":
		p, err := e.CutExtractAudio(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_enhance":
		p, err := e.CutEnhance(id, strAnyMap(args, "preset"))
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_mix":
		duck := true
		if v, ok := args["duck"]; ok {
			switch t := v.(type) {
			case bool:
				duck = t
			case string:
				duck = t != "0" && !strings.EqualFold(t, "false")
			}
		}
		replace := false
		switch t := args["replace"].(type) {
		case bool:
			replace = t
		case string:
			replace = t == "1" || strings.EqualFold(t, "true")
		}
		p, err := e.CutMix(id, duck, replace)
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_transcribe":
		p, err := e.CutTranscribe(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_captions":
		p, err := e.CutCaptions(id, strAnyMap(args, "text"), strAnyMap(args, "language", "lang"))
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_burnin":
		p, err := e.CutBurnin(id)
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_dub":
		p, err := e.CutDub(id, strAnyMap(args, "text"), strAnyMap(args, "voice"))
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_grade":
		p, err := e.CutGrade(id, strAnyMap(args, "preset"))
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "cut_set_stem":
		p, err := e.CutSetStem(id, strAnyMap(args, "kind"), strAnyMap(args, "hash"))
		if err != nil {
			return "", err
		}
		return marshalJSON(p), nil
	case "video_route":
		out, err := e.VideoRoute(sessionID, strAnyMap(args, "dest", "to"), strAnyMap(args, "id"), strAnyMap(args, "title"), strAnyMap(args, "intent"))
		if err != nil {
			return "", err
		}
		return marshalJSON(out), nil
	case "provider_menu":
		return marshalJSON(e.ProviderMenu(strAnyMap(args, "capability", "cap"))), nil
	default:
		return "", fmt.Errorf("unknown cut tool %s", name)
	}
}
