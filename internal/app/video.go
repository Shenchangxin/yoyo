package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/video"
)

func (a *App) openVideo() error {
	if a.Home == nil || a.CAS == nil || a.Vault == nil {
		return fmt.Errorf("video: home, cas, and vault are required")
	}
	eng, err := video.OpenDefault(a.Home.Video(), a.CAS, a.Vault)
	if err != nil {
		return err
	}
	if err := eng.StartMediaServer(); err != nil {
		diaglog.Warn("video media server failed", "component", "video", "err", err)
	}
	eng.OnJob = a.onVideoJob
	a.Video = eng
	return nil
}

func (a *App) onVideoJob(j video.Job) {
	if a.Hub == nil {
		return
	}
	ev := trace.Event{
		TS:        time.Now().UTC(),
		Type:      trace.TypeSystem,
		Source:    "video",
		SessionID: "",
		ItemKind:  "video_job",
		Payload: map[string]any{
			"job":         j,
			"id":          j.ID,
			"type":        j.Type,
			"status":      j.Status,
			"episode_id":  j.EpisodeID,
			"error":       j.Error,
			"result_hash": j.ResultHash,
			"poster_hash": j.PosterHash,
			"media_url":   a.mediaURL(j.ResultHash),
			"poster_url":  a.mediaURL(j.PosterHash),
		},
	}
	a.Hub.Publish(ev)
	if j.Status == "succeeded" || j.Status == "failed" {
		a.journalVideo("video.job."+j.Status, map[string]any{
			"id": j.ID, "kind": j.Type, "provider": j.Provider, "episode_id": j.EpisodeID, "error": j.Error,
		})
	}
}

func (a *App) mediaURL(hash string) string {
	if a.Video == nil || hash == "" {
		return ""
	}
	return a.Video.MediaURL(hash)
}

func (a *App) journalVideo(kind string, payload map[string]any) {
	if a.Journal == nil {
		return
	}
	_, _ = a.Journal.Append(kind, payload)
}

func (a *App) requireVideo() (*video.Engine, error) {
	if a.Video == nil {
		return nil, fmt.Errorf("video engine is not open")
	}
	return a.Video, nil
}

func (a *App) VideoStatus() map[string]any {
	eng, err := a.requireVideo()
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	out := eng.Snapshot()
	out["ok"] = true
	return out
}

func (a *App) attachDramaTools(tools *runtime.WorkspaceTools, sessionID string) {
	if tools == nil || a.Video == nil {
		return
	}
	if _, ok := a.Video.SessionBind(sessionID); !ok {
		return
	}
	if tools.Extra == nil {
		tools.Extra = map[string]runtime.ExtraTool{}
	}
	for _, spec := range video.DramaToolSpecs() {
		spec := spec
		tools.Extra[spec.Name] = runtime.ExtraTool{
			JSON: runtime.ToolJSON{Type: "function", Function: map[string]any{
				"name": spec.Name, "description": spec.Desc, "parameters": spec.Params,
			}},
			ReadOnly: spec.ReadOnly,
			Call: func(argsJSON string) runtime.ToolResult {
				var args map[string]any
				if argsJSON != "" {
					_ = json.Unmarshal([]byte(argsJSON), &args)
				}
				out, err := a.Video.Tool(sessionID, spec.Name, args)
				return runtime.ToolResult{Content: out, Err: err}
			},
		}
	}
	tools.ExtraEnabled = uniqueStrings(tools.ExtraEnabled, video.DramaToolNames())
}

func (a *App) RunDramaStage(sessionID, episodeID, stage string) error {
	eng, err := a.requireVideo()
	if err != nil {
		return err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session required")
	}
	if err := eng.BindSession(sessionID, episodeID); err != nil {
		return err
	}
	prompt, skills, err := eng.StagePrompt(stage, episodeID)
	if err != nil {
		return err
	}
	if meta, err := a.GetSession(sessionID); err == nil {
		pinned := uniqueStrings(meta.PinnedSkills, skills)
		_, _ = a.SetSessionPinnedSkills(sessionID, pinned)
	}
	a.journalVideo("drama.stage", map[string]any{"session": sessionID, "episode_id": episodeID, "stage": stage})
	return a.StartSendOpts(sessionID, prompt, false, nil)
}

func (a *App) DramaBind(sessionID, episodeID string) error {
	eng, err := a.requireVideo()
	if err != nil {
		return err
	}
	return eng.BindSession(sessionID, episodeID)
}

func (a *App) VideoCall(method string, params map[string]any) (any, error) {
	if params == nil {
		params = map[string]any{}
	}
	eng, err := a.requireVideo()
	if err != nil {
		return nil, err
	}
	str := func(k string) string { return strings.TrimSpace(fmt.Sprint(params[k])) }
	switch method {
	case "video.status":
		return a.VideoStatus(), nil
	case "video.modes":
		return video.Modes(), nil
	case "video.templates":
		return video.ProviderTemplates(), nil
	case "video.providers.list":
		return eng.ListProviders(str("service_type"))
	case "video.providers.upsert":
		var p video.Provider
		b, _ := json.Marshal(params["provider"])
		if len(b) == 0 {
			b, _ = json.Marshal(params)
		}
		_ = json.Unmarshal(b, &p)
		return eng.UpsertProvider(p, str("api_key"))
	case "video.providers.delete":
		return map[string]any{"ok": true}, eng.DeleteProvider(str("id"))
	case "video.providers.test":
		return eng.TestProvider(str("id")), nil
	case "video.styles":
		return eng.ListStyles()
	case "video.settings.get":
		return eng.SettingsMap(), nil
	case "video.settings.set":
		return map[string]any{"ok": true}, eng.SetSetting(str("key"), str("value"))
	case "video.jobs.list":
		return eng.ListJobs(str("episode_id"))
	case "video.jobs.get":
		return eng.GetJob(str("id"))
	case "video.jobs.cancel":
		return map[string]any{"ok": true}, eng.CancelJob(str("id"))
	case "video.jobs.retry":
		return eng.RetryJob(str("id"))
	case "drama.list":
		return eng.ListDramas()
	case "drama.create":
		var d video.Drama
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &d)
		return eng.CreateDrama(d)
	case "drama.get":
		return eng.GetDrama(str("id"))
	case "drama.update":
		var d video.Drama
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &d)
		return eng.UpdateDrama(d)
	case "drama.delete":
		return map[string]any{"ok": true}, eng.DeleteDrama(str("id"))
	case "drama.episodes":
		return eng.ListEpisodes(str("drama_id"))
	case "drama.episode.create":
		return eng.CreateEpisode(str("drama_id"), str("title"), str("content"))
	case "drama.episode.get":
		return eng.GetEpisode(str("id"))
	case "drama.episode.update":
		var ep video.Episode
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &ep)
		return eng.UpdateEpisode(ep)
	case "drama.episode.delete":
		return map[string]any{"ok": true}, eng.DeleteEpisode(str("id"))
	case "drama.bundle":
		b, err := eng.Bundle(str("episode_id"))
		if err != nil {
			return nil, err
		}
		return a.enrichBundle(b), nil
	case "drama.bind":
		return map[string]any{"ok": true}, a.DramaBind(str("session_id"), str("episode_id"))
	case "drama.stage.run":
		return map[string]any{"ok": true}, a.RunDramaStage(str("session_id"), str("episode_id"), str("stage"))
	case "drama.assets.save":
		return map[string]any{"ok": true}, eng.UpdateAsset(str("kind"), str("id"), params)
	case "drama.assets.generate":
		j, err := eng.GenerateAsset(str("kind"), str("id"), str("episode_id"))
		a.journalVideo("video.generate", map[string]any{"kind": "image", "asset": str("kind"), "id": str("id")})
		return j, err
	case "drama.assets.generate_missing":
		j, err := eng.GenerateMissingAssets(str("episode_id"))
		a.journalVideo("video.generate", map[string]any{"kind": "image_batch", "episode_id": str("episode_id"), "n": len(j)})
		return j, err
	case "drama.assets.upload":
		raw, err := decodeB64(str("data_b64"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, eng.AttachImage(str("kind"), str("id"), raw)
	case "drama.shots.save":
		var shots []video.Shot
		b, _ := json.Marshal(params["shots"])
		_ = json.Unmarshal(b, &shots)
		replace := false
		switch v := params["replace"].(type) {
		case bool:
			replace = v
		case string:
			replace = v == "true" || v == "1"
		}
		return eng.SaveShots(str("episode_id"), shots, replace)
	case "drama.shots.update":
		var s video.Shot
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &s)
		return eng.UpdateShot(s)
	case "drama.shots.delete":
		return map[string]any{"ok": true}, eng.DeleteShot(str("id"))
	case "drama.shots.generate":
		j, err := eng.GenerateShot(str("id"))
		a.journalVideo("video.generate", map[string]any{"kind": "video", "shot_id": str("id")})
		return j, err
	case "drama.shots.generate_missing":
		j, err := eng.GenerateMissingShots(str("episode_id"))
		a.journalVideo("video.generate", map[string]any{"kind": "video_batch", "episode_id": str("episode_id"), "n": len(j)})
		return j, err
	case "drama.merge":
		var ids []string
		b, _ := json.Marshal(params["shot_ids"])
		_ = json.Unmarshal(b, &ids)
		j, err := eng.MergeEpisode(str("episode_id"), ids)
		a.journalVideo("video.merge", map[string]any{"episode_id": str("episode_id"), "n": len(ids)})
		return j, err
	case "drama.import":
		return eng.ImportHuobao(str("db_path"), str("static_dir"))
	case "media.url":
		return map[string]any{"url": a.mediaURL(str("hash"))}, nil
	default:
		return nil, fmt.Errorf("unknown video method %s", method)
	}
}

func (a *App) enrichBundle(b video.EpisodeBundle) map[string]any {
	urlOf := func(h string) string { return a.mediaURL(h) }
	chars := make([]map[string]any, 0, len(b.Characters))
	for _, c := range b.Characters {
		m := objJSON(c)
		m["image_url"] = urlOf(c.ImageHash)
		chars = append(chars, m)
	}
	scenes := make([]map[string]any, 0, len(b.Scenes))
	for _, s := range b.Scenes {
		m := objJSON(s)
		m["image_url"] = urlOf(s.ImageHash)
		scenes = append(scenes, m)
	}
	props := make([]map[string]any, 0, len(b.Props))
	for _, p := range b.Props {
		m := objJSON(p)
		m["image_url"] = urlOf(p.ImageHash)
		props = append(props, m)
	}
	shots := make([]map[string]any, 0, len(b.Shots))
	for _, s := range b.Shots {
		m := objJSON(s)
		m["video_url"] = urlOf(s.VideoHash)
		m["poster_url"] = urlOf(s.PosterHash)
		shots = append(shots, m)
	}
	ep := objJSON(b.Episode)
	ep["video_url"] = urlOf(b.Episode.VideoHash)
	ep["poster_url"] = urlOf(b.Episode.PosterHash)
	d := objJSON(b.Drama)
	d["thumbnail_url"] = urlOf(b.Drama.ThumbnailHash)
	jobs := make([]map[string]any, 0, len(b.Jobs))
	for _, j := range b.Jobs {
		m := objJSON(j)
		m["media_url"] = urlOf(j.ResultHash)
		m["poster_url"] = urlOf(j.PosterHash)
		jobs = append(jobs, m)
	}
	return map[string]any{
		"drama": d, "episode": ep, "characters": chars, "scenes": scenes, "props": props,
		"shots": shots, "jobs": jobs, "plan": b.Plan, "media_base": a.mediaURL(""),
		"status": a.VideoStatus(),
	}
}

func objJSON(v any) map[string]any {
	b, _ := json.Marshal(v)
	m := map[string]any{}
	_ = json.Unmarshal(b, &m)
	if m == nil {
		m = map[string]any{}
	}
	return m
}

func decodeB64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, ","); i >= 0 && strings.Contains(s[:i], "base64") {
		s = s[i+1:]
	}
	if s == "" {
		return nil, fmt.Errorf("empty upload")
	}
	return base64.StdEncoding.DecodeString(s)
}
