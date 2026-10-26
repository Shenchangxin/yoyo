// cut_tools.go 列出剪辑工具和两条路由工具，并按名字调用。

package video

import (
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
		{Name: "cut_propose", Desc: "Score highlight windows from scene+speech energy and extract clips WITH audio. count default 3. window 15-30 or 60-90.", Params: obj(map[string]any{"id": str, "count": num, "window": str})},
		{Name: "cut_apply", Desc: "Concatenate the proposed (or edit_list) clips into a result mp4, keeping audio.", Params: obj(map[string]any{"id": str})},
		{Name: "cut_extract", Desc: "Dump the soundtrack of the source or result onto the original stem.", Params: obj(map[string]any{"id": str})},
		{Name: "cut_enhance", Desc: "Denoise/loudnorm a stem. preset: voice, denoise, loudnorm.", Params: obj(map[string]any{"id": str, "preset": str})},
		{Name: "cut_mix", Desc: "Mix dub/voice and music onto the picture. Duck music under speech when duck is true.", Params: obj(map[string]any{"id": str, "duck": map[string]any{"type": "boolean"}})},
		{Name: "cut_transcribe", Desc: "Local whisper.cpp to cues + SRT. Requires WHISPER_BIN.", Params: obj(map[string]any{"id": str})},
		{Name: "cut_captions", Desc: "Set caption text (or keep cues) and store SRT/VTT hashes.", Params: obj(map[string]any{"id": str, "text": str, "language": str})},
		{Name: "cut_burnin", Desc: "Burn stored SRT onto the result video.", Params: obj(map[string]any{"id": str})},
		{Name: "cut_dub", Desc: "TTS via the enabled speech connection in Settings. Writes the dub stem. Does not invent keys.", Params: obj(map[string]any{"id": str, "text": str, "voice": str})},
		{Name: "cut_grade", Desc: "FFmpeg color preset: warm, cool, contrast, fade.", Params: obj(map[string]any{"id": str, "preset": str})},
		{Name: "cut_set_stem", Desc: "Attach a CAS hash as music/sfx/voice/dub/original.", Params: obj(map[string]any{"id": str, "kind": str, "hash": str}, "kind", "hash")},
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
		p, err := e.CutMix(id, duck)
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
