package app

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/schedule"
	"github.com/Shenchangxin/yoyo/internal/session"
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
	eng.OnHub = a.onCanvasHub
	a.Video = eng
	a.seedConnections()
	if a.Observe != nil {
		a.Observe.SetEndpoint(a.otelEndpoint())
	}
	return nil
}

func (a *App) onCanvasHub(kind string, payload map[string]any) {
	if a.Hub == nil {
		return
	}
	a.Hub.Publish(trace.Event{
		TS: time.Now().UTC(), Type: trace.TypeSystem, Source: "video", ItemKind: kind, Payload: payload,
	})
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

func (a *App) attachCanvasTools(tools *runtime.WorkspaceTools, sessionID string) {
	if tools == nil || a.Video == nil {
		return
	}
	if _, ok := a.Video.CanvasSessionBind(sessionID); !ok {
		return
	}
	if tools.Extra == nil {
		tools.Extra = map[string]runtime.ExtraTool{}
	}
	mediaAsk := map[string]bool{
		"generate_media": true, "image_layer_split": true, "image_annotation_render": true,
	}
	for _, spec := range video.CanvasToolSpecs() {
		spec := spec
		openWorld := mediaAsk[spec.Name]
		tools.Extra[spec.Name] = runtime.ExtraTool{
			JSON: runtime.ToolJSON{Type: "function", Function: map[string]any{
				"name": spec.Name, "description": spec.Desc, "parameters": spec.Params,
			}},
			ReadOnly:  spec.ReadOnly,
			OpenWorld: openWorld,
			Exclusive: openWorld,
			Call: func(argsJSON string) runtime.ToolResult {
				if spec.Name == "plan_update" {
					return tools.Call("update_plan", argsJSON)
				}
				if spec.Name == "ask_user" {
					return tools.Call("ask_user", argsJSON)
				}
				if mediaAsk[spec.Name] && a.Caps != nil {
					if err := a.Caps.Check(capability.Request{
						Level: capability.Network, Action: spec.Name, SessionID: sessionID, ForceAsk: true,
					}); err != nil {
						return runtime.ToolResult{Err: err}
					}
				}
				var args map[string]any
				if argsJSON != "" {
					_ = json.Unmarshal([]byte(argsJSON), &args)
				}
				out, err := a.Video.CanvasTool(sessionID, spec.Name, args)
				return runtime.ToolResult{Content: out, Err: err}
			},
		}
	}
	tools.ExtraEnabled = uniqueStrings(tools.ExtraEnabled, video.CanvasToolNames())
	if !canvasExposeMCP(a.Video) {
		stripCanvasMCP(tools)
	}
}

var studioSkillNames = []string{"studio-script", "studio-storyboard", "studio-assemble", "studio-talking-head"}

const studioBootstrap = "Do not inspect the git workspace. Open the bound shorts project with studio_get, then write the spoken script with studio_update. If shots have no media, stop after the script."

const studioRuntimePin = "This is a Shorts turn. Use studio_* tools on the bound shorts project. Do not inspect the git workspace. Do not call shell, list_dir, glob, or grep. Write the spoken script with studio_update. Do not invent CAS hashes. If shots have no media, stop after the script and tell the operator to upload stills or clips on the Shorts assemble pane."

var studioDeniedTools = []string{"list_dir", "glob", "grep", "shell", "git_status", "git_diff", "git_commit"}

func (a *App) studioLane(sessionID string) bool {
	if a == nil || a.Video == nil || strings.TrimSpace(sessionID) == "" {
		return false
	}
	if _, ok := a.Video.StudioSessionBind(sessionID); ok {
		return true
	}
	if _, ok := a.Video.SessionBind(sessionID); ok {
		return false
	}
	if _, ok := a.Video.CanvasSessionBind(sessionID); ok {
		return false
	}
	if meta, err := a.GetSession(sessionID); err == nil && session.NormalizeChannel(meta.Channel) == session.ChannelVideo {
		return true
	}
	return false
}

func (a *App) attachStudioTools(tools *runtime.WorkspaceTools, sessionID string) {
	if tools == nil || a.Video == nil {
		return
	}
	_, studioBound := a.Video.StudioSessionBind(sessionID)
	if !studioBound {
		if _, ok := a.Video.SessionBind(sessionID); ok {
			return
		}
		if _, ok := a.Video.CanvasSessionBind(sessionID); ok {
			return
		}
		videoChan := false
		if meta, err := a.GetSession(sessionID); err == nil && session.NormalizeChannel(meta.Channel) == session.ChannelVideo {
			videoChan = true
		}
		if !videoChan {
			return
		}
	}
	if tools.Extra == nil {
		tools.Extra = map[string]runtime.ExtraTool{}
	}
	askNet := map[string]bool{"studio_speak": true, "trends_fetch": true, "studio_remotion_install": true}
	askBrowser := map[string]bool{"studio_publish_open": true, "studio_publish_analytics": true}
	for _, spec := range video.StudioToolSpecs() {
		spec := spec
		openWorld := askNet[spec.Name] || askBrowser[spec.Name]
		tools.Extra[spec.Name] = runtime.ExtraTool{
			JSON: runtime.ToolJSON{Type: "function", Function: map[string]any{
				"name": spec.Name, "description": spec.Desc, "parameters": spec.Params,
			}},
			ReadOnly:  spec.ReadOnly,
			OpenWorld: openWorld,
			Exclusive: openWorld,
			Call: func(argsJSON string) runtime.ToolResult {
				var args map[string]any
				if argsJSON != "" {
					_ = json.Unmarshal([]byte(argsJSON), &args)
				}
				if askBrowser[spec.Name] && a.Caps != nil {
					if err := a.Caps.Check(capability.Request{
						Level: capability.Browser, Action: spec.Name, SessionID: sessionID, ForceAsk: true,
					}); err != nil {
						return runtime.ToolResult{Err: err}
					}
				} else if askNet[spec.Name] && a.Caps != nil {
					if err := a.Caps.Check(capability.Request{
						Level: capability.Network, Action: spec.Name, SessionID: sessionID, ForceAsk: true,
					}); err != nil {
						return runtime.ToolResult{Err: err}
					}
				}
				if spec.Name == "studio_publish_open" || spec.Name == "studio_publish_analytics" {
					out, err := a.OpenStudioPublish(strAnyFrom(args, "publish_id", "id"), spec.Name == "studio_publish_analytics")
					if err != nil {
						return runtime.ToolResult{Err: err}
					}
					b, _ := json.Marshal(out)
					return runtime.ToolResult{Content: string(b)}
				}
				if spec.Name == "studio_persona_create" {
					out, err := a.CreateStudioPersona(strAnyFrom(args, "name", "title"), strAnyFrom(args, "topic_id"))
					if err != nil {
						return runtime.ToolResult{Err: err}
					}
					b, _ := json.Marshal(out)
					return runtime.ToolResult{Content: string(b)}
				}
				out, err := a.Video.StudioTool(sessionID, spec.Name, args)
				return runtime.ToolResult{Content: out, Err: err}
			},
		}
	}
	tools.ExtraEnabled = uniqueStrings(tools.ExtraEnabled, video.StudioToolNames())
	tools.MarkSkillsLoaded(studioSkillNames...)
	a.pinStudioLane(tools)
	if meta, err := a.GetSession(sessionID); err == nil {
		next := uniqueStrings(meta.PinnedSkills, studioSkillNames)
		if len(next) > len(meta.PinnedSkills) {
			_, _ = a.SetSessionPinnedSkills(sessionID, next)
		}
	}
}

func (a *App) pinStudioLane(tools *runtime.WorkspaceTools) {
	if tools == nil {
		return
	}
	deny := map[string]bool{}
	for _, n := range studioDeniedTools {
		deny[n] = true
	}
	var advertised []string
	for _, n := range tools.Advertised {
		if deny[n] {
			continue
		}
		advertised = append(advertised, n)
	}
	tools.Advertised = advertised
	tools.AllowedTools = uniqueStrings(advertised, video.StudioToolNames(), []string{
		"load_skill", "list_skills", "tool_search", "recall_context", "update_plan", "ask_user", "read_skill_file",
	})
}

func canvasExposeMCP(eng *video.Engine) bool {
	if eng == nil {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(eng.Setting("canvas_expose_mcp", "")))
	return v == "1" || v == "true" || v == "on"
}

func stripCanvasMCP(tools *runtime.WorkspaceTools) {
	if tools == nil {
		return
	}
	if tools.Extra != nil {
		next := map[string]runtime.ExtraTool{}
		for k, v := range tools.Extra {
			if strings.HasPrefix(k, "mcp__") {
				continue
			}
			next[k] = v
		}
		tools.Extra = next
	}
	enabled := make([]string, 0, len(tools.ExtraEnabled))
	for _, n := range tools.ExtraEnabled {
		if strings.HasPrefix(n, "mcp__") {
			continue
		}
		enabled = append(enabled, n)
	}
	tools.ExtraEnabled = enabled
}

func (a *App) RunDramaStage(sessionID, episodeID, stage, dramaID string) error {
	eng, err := a.requireVideo()
	if err != nil {
		return err
	}
	sessionID = strings.TrimSpace(sessionID)
	stage = strings.TrimSpace(stage)
	if sessionID == "" {
		return fmt.Errorf("session required")
	}
	if dramaID == "" && episodeID != "" {
		if ep, err := eng.GetEpisode(episodeID); err == nil {
			dramaID = ep.DramaID
		}
	}
	if stage == "outline" || stage == "bible" {
		if dramaID == "" {
			return fmt.Errorf("series required")
		}
		if err := eng.BindSessionDrama(sessionID, dramaID); err != nil {
			return err
		}
	} else {
		if err := eng.BindSession(sessionID, episodeID); err != nil {
			return err
		}
	}
	prompt, skills, err := eng.StagePrompt(stage, episodeID, dramaID)
	if err != nil {
		return err
	}
	if meta, err := a.GetSession(sessionID); err == nil {
		pinned := uniqueStrings(meta.PinnedSkills, skills)
		_, _ = a.SetSessionPinnedSkills(sessionID, pinned)
	}
	a.journalVideo("drama.stage", map[string]any{"session": sessionID, "episode_id": episodeID, "drama_id": dramaID, "stage": stage})
	return a.StartSendOpts(sessionID, prompt, false, nil)
}

func (a *App) DramaBind(sessionID, episodeID, dramaID string) error {
	eng, err := a.requireVideo()
	if err != nil {
		return err
	}
	if strings.TrimSpace(episodeID) != "" {
		return eng.BindSession(sessionID, episodeID)
	}
	if strings.TrimSpace(dramaID) != "" {
		return eng.BindSessionDrama(sessionID, dramaID)
	}
	return fmt.Errorf("episode or series required")
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
	case "video.ffmpeg.install":
		return eng.InstallFFmpeg(), nil
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
	case "connections.list":
		if eng.Conn == nil {
			return []connection.Connection{}, nil
		}
		return eng.Conn.List(str("capability"))
	case "connections.get":
		if eng.Conn == nil {
			return nil, fmt.Errorf("connection registry missing")
		}
		return eng.Conn.Get(str("id"))
	case "connections.upsert":
		if eng.Conn == nil {
			return nil, fmt.Errorf("connection registry missing")
		}
		var c connection.Connection
		b, _ := json.Marshal(params["connection"])
		if len(b) == 0 || string(b) == "null" {
			b, _ = json.Marshal(params)
		}
		_ = json.Unmarshal(b, &c)
		return eng.Conn.Upsert(c, str("api_key"))
	case "connections.delete":
		if eng.Conn == nil {
			return map[string]any{"ok": true}, nil
		}
		return map[string]any{"ok": true}, eng.Conn.Delete(str("id"))
	case "connections.test":
		if eng.Conn == nil {
			return map[string]any{"ok": false, "error": "registry missing"}, nil
		}
		return eng.Conn.Test(str("id")), nil
	case "connections.defaults":
		if eng.Conn == nil {
			return connection.Defaults{}, nil
		}
		return eng.Conn.Defaults(), nil
	case "connections.setDefault":
		if eng.Conn == nil {
			return nil, fmt.Errorf("connection registry missing")
		}
		return map[string]any{"ok": true}, eng.Conn.SetDefault(str("capability"), str("id"))
	case "connections.templates":
		return connection.Templates(), nil
	case "video.styles":
		return eng.ListStyles()
	case "video.styles.all":
		return eng.ListAllStyles()
	case "video.styles.upsert":
		var s video.Style
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &s)
		return eng.UpsertStyle(s)
	case "video.styles.delete":
		return map[string]any{"ok": true}, eng.DeleteStyle(str("id"))
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
	case "video.jobs.apply":
		return eng.ApplyJob(str("id"))
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
		return eng.PatchEpisode(str("id"), params)
	case "drama.episode.delete":
		return map[string]any{"ok": true}, eng.DeleteEpisode(str("id"))
	case "drama.bundle":
		b, err := eng.Bundle(str("episode_id"))
		if err != nil {
			return nil, err
		}
		return a.enrichBundle(b), nil
	case "drama.bind":
		return map[string]any{"ok": true}, a.DramaBind(str("session_id"), str("episode_id"), str("drama_id"))
	case "drama.stage.run":
		return map[string]any{"ok": true}, a.RunDramaStage(str("session_id"), str("episode_id"), str("stage"), str("drama_id"))
	case "drama.source.ingest":
		return eng.IngestSource(str("drama_id"), str("text"), str("title"))
	case "drama.source.ingest_file":
		return eng.IngestSourceFile(str("drama_id"), str("path"))
	case "drama.source.get":
		src, err := eng.GetSource(str("drama_id"))
		if err != nil {
			return nil, err
		}
		return src, nil
	case "drama.plan.get":
		plan, err := eng.GetPlan(str("drama_id"))
		if err != nil {
			return map[string]any{}, nil
		}
		return plan, nil
	case "drama.plan.propose":
		items := decodePlanEpisodes(params["episodes"])
		return eng.ProposeEpisodes(str("drama_id"), items)
	case "drama.plan.patch":
		items := decodePlanEpisodes(params["episodes"])
		return eng.PatchPlan(str("drama_id"), items)
	case "drama.plan.split":
		return eng.SplitPlanEpisode(str("drama_id"), intArgApp(params["n"]))
	case "drama.plan.merge":
		return eng.MergePlanEpisodes(str("drama_id"), intArgApp(params["n"]))
	case "drama.plan.commit":
		return eng.CommitEpisodes(str("drama_id"))
	case "drama.plan.queue_scripts":
		plan, err := eng.StartScriptQueue(str("drama_id"))
		if err != nil {
			return nil, err
		}
		ep, ok := eng.NextScriptEpisode(str("drama_id"))
		if !ok {
			return map[string]any{"ok": true, "plan": plan, "done": true}, nil
		}
		if err := a.RunDramaStage(str("session_id"), ep.ID, "rewrite", str("drama_id")); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true, "plan": plan, "episode_id": ep.ID}, nil
	case "drama.jobs":
		return eng.ListDramaJobs(str("drama_id"))
	case "drama.bible":
		id := str("drama_id")
		return map[string]any{
			"characters": eng.DramaCharacters(id),
			"scenes":     eng.DramaScenes(id),
			"props":      eng.DramaProps(id),
		}, nil
	case "drama.assets.save":
		return map[string]any{"ok": true}, eng.UpdateAsset(str("kind"), str("id"), params)
	case "drama.assets.create":
		return eng.CreateAsset(str("kind"), str("episode_id"), params)
	case "drama.assets.delete":
		return map[string]any{"ok": true}, eng.DeleteAsset(str("kind"), str("id"))
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
		id := str("id")
		if id == "" {
			id = str("storyboard_id")
		}
		return eng.PatchShot(id, params)
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
	case "drama.skip_rewrite":
		return map[string]any{"ok": true}, eng.SkipRewrite(str("episode_id"))
	case "media.url":
		return map[string]any{"url": a.mediaURL(str("hash"))}, nil
	case "canvas.bind":
		return map[string]any{"ok": true}, a.CanvasBind(str("session_id"), firstNonEmptyApp(str("project_id"), str("canvas_id")))
	case "video.history":
		return eng.ProjectHistory()
	case "video.session.project":
		return eng.SessionProject(str("session_id")), nil
	case "video.trends.fetch":
		return eng.FetchTrends(splitCSVApp(str("platforms")), intArgApp(params["limit"]))
	case "video.topic.list":
		return eng.ListTopics(str("status")), nil
	case "video.topic.get":
		return eng.GetTopic(str("id"))
	case "video.topic.save":
		var t video.Topic
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &t)
		return eng.SaveTopic(t)
	case "video.topic.delete":
		return map[string]any{"ok": true}, eng.DeleteTopic(str("id"))
	case "video.topic.from_trend":
		return eng.TopicFromTrend(str("platform"), str("title"), str("hot"))
	case "video.topic.schedule":
		t, err := eng.GetTopic(str("id"))
		if err != nil {
			return nil, err
		}
		proj, err := eng.StudioFromTopic(t.ID, str("aspect"))
		if err != nil {
			return nil, err
		}
		sid := str("session_id")
		if sid != "" {
			if meta, err := a.GetSession(sid); err != nil || session.NormalizeChannel(meta.Channel) != session.ChannelVideo {
				sid = ""
			}
		}
		if sid != "" {
			_ = eng.BindStudioSession(sid, proj.ID)
			if meta, err := a.GetSession(sid); err == nil {
				_, _ = a.SetSessionPinnedSkills(sid, uniqueStrings(meta.PinnedSkills, studioSkillNames))
			}
		}
		if latest, err := eng.GetTopic(t.ID); err == nil {
			t = latest
		}
		job := a.ScheduleCreate(schedule.Job{
			Kind:      schedule.KindOnce,
			Spec:      time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339),
			Prompt:    studioJobPrompt(t, proj),
			Workspace: a.VideoWorkspace(),
			SessionID: sid,
			Channel:   session.ChannelVideo,
			TopicID:   t.ID,
			StudioID:  proj.ID,
		})
		return map[string]any{"ok": true, "job": job, "topic": t, "studio": a.enrichStudio(proj)}, nil
	case "video.studio.list":
		rows := eng.ListStudioProjects()
		out := make([]map[string]any, 0, len(rows))
		for _, p := range rows {
			out = append(out, a.enrichStudio(p))
		}
		return out, nil
	case "video.studio.create":
		var p video.StudioProject
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &p)
		var out video.StudioProject
		var cerr error
		if str("topic_id") != "" && strings.TrimSpace(p.Title) == "" {
			out, cerr = eng.StudioFromTopic(str("topic_id"), str("aspect"))
		} else {
			out, cerr = eng.CreateStudioProject(p)
		}
		if cerr != nil {
			return nil, cerr
		}
		if sid := str("session_id"); sid != "" {
			_ = eng.BindStudioSession(sid, out.ID)
		}
		return a.enrichStudio(out), nil
	case "video.studio.get":
		p, err := eng.GetStudioProject(str("id"))
		if err != nil {
			return nil, err
		}
		return a.enrichStudio(p), nil
	case "video.studio.update":
		var p video.StudioProject
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &p)
		if p.ID == "" {
			p.ID = str("id")
		}
		if shots := params["shots"]; shots != nil && len(p.Storyboard.Shots) == 0 {
			sb, _ := json.Marshal(shots)
			_ = json.Unmarshal(sb, &p.Storyboard.Shots)
		}
		out, err := eng.UpdateStudioProject(p)
		if err != nil {
			return nil, err
		}
		if _, ok := params["script"]; ok {
			out.Script = str("script")
			if strings.TrimSpace(out.Script) == "" {
				out.ScriptHash = ""
			} else if h, herr := eng.PutBytes([]byte(out.Script)); herr == nil {
				out.ScriptHash = h
			}
			out.UpdatedAt = video.Now()
			if err := eng.PutStudioProject(out); err != nil {
				return nil, err
			}
		}
		return a.enrichStudio(out), nil
	case "video.studio.delete":
		return map[string]any{"ok": true}, eng.DeleteStudioProject(str("id"))
	case "video.studio.bind":
		return map[string]any{"ok": true}, eng.BindStudioSession(str("session_id"), firstNonEmptyApp(str("id"), str("project_id")))
	case "video.studio.assemble":
		p, err := eng.AssembleStudio(str("id"))
		if err != nil {
			return a.enrichStudio(p), err
		}
		return a.enrichStudio(p), nil
	case "video.studio.speak":
		p, err := eng.SpeakStudio(str("id"), str("text"))
		if err != nil {
			return nil, err
		}
		return a.enrichStudio(p), nil
	case "video.studio.ops.reframe":
		ref, err := eng.Reframe(str("hash"), str("aspect"))
		if err != nil {
			return nil, err
		}
		return a.enrichMedia(ref), nil
	case "video.studio.ops.burnin":
		ref, err := eng.BurnInCaptions(str("hash"), str("subtitle_hash"))
		if err != nil {
			return nil, err
		}
		return a.enrichMedia(ref), nil
	case "video.studio.ops.slideshow":
		var hashes []string
		b, _ := json.Marshal(params["hashes"])
		_ = json.Unmarshal(b, &hashes)
		ref, err := eng.Slideshow(hashes, str("aspect"), floatArgApp(params["duration"]))
		if err != nil {
			return nil, err
		}
		return a.enrichMedia(ref), nil
	case "video.studio.media.put":
		raw, err := decodeB64(firstNonEmptyApp(str("data_b64"), str("data")))
		if err != nil {
			return nil, err
		}
		hash, err := eng.PutBytes(raw)
		if err != nil {
			return nil, err
		}
		kind := str("kind")
		mime := str("mime")
		if kind == "" {
			if strings.HasPrefix(mime, "video/") {
				kind = "video"
			} else {
				kind = "image"
			}
		}
		return a.enrichMedia(video.MediaRef{Kind: kind, CASHash: hash, MIME: mime, Source: "studio-upload"}), nil
	case "video.studio.ops.wrap":
		ref, err := eng.WrapTalkingHead(str("hash"), firstNonEmptyApp(str("title"), str("text")))
		if err != nil {
			return nil, err
		}
		return a.enrichMedia(ref), nil
	case "video.studio.publish.platforms":
		return video.PublishPlatforms(), nil
	case "video.studio.publish.list":
		rows := eng.ListPublishes(str("project_id"), str("platform"))
		out := make([]map[string]any, 0, len(rows))
		for _, p := range rows {
			out = append(out, a.enrichPublish(p))
		}
		return out, nil
	case "video.studio.publish.get":
		p, err := eng.GetPublish(str("id"))
		if err != nil {
			return nil, err
		}
		return a.enrichPublish(p), nil
	case "video.studio.publish.draft":
		var p video.Publish
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &p)
		out, err := eng.SavePublish(p)
		if err != nil {
			return nil, err
		}
		return a.enrichPublish(out), nil
	case "video.studio.publish.export":
		out, err := eng.ExportPublish(firstNonEmptyApp(str("id"), str("publish_id")))
		if err != nil {
			return nil, err
		}
		return a.enrichPublish(out), nil
	case "video.studio.publish.open":
		return a.OpenStudioPublish(firstNonEmptyApp(str("id"), str("publish_id")), false)
	case "video.studio.publish.analytics":
		return a.OpenStudioPublish(firstNonEmptyApp(str("id"), str("publish_id")), true)
	case "video.studio.publish.mark":
		out, err := eng.MarkPublish(
			firstNonEmptyApp(str("id"), str("publish_id")),
			str("status"),
			firstNonEmptyApp(str("posted_url"), str("url")),
			str("note"),
			intArgApp(params["reads"]), intArgApp(params["likes"]), intArgApp(params["comments"]),
		)
		if err != nil {
			return nil, err
		}
		return a.enrichPublish(out), nil
	case "video.studio.remotion.status":
		return eng.RemotionSnapshot(), nil
	case "video.studio.remotion.install":
		return eng.InstallRemotion(), nil
	case "video.studio.persona.template":
		return map[string]any{"markdown": video.PersonaTemplate(firstNonEmptyApp(str("name"), str("title")))}, nil
	case "video.studio.persona.create":
		return a.CreateStudioPersona(firstNonEmptyApp(str("name"), str("title")), str("topic_id"))
	default:
		if strings.HasPrefix(method, "canvas.") {
			return eng.CanvasCall(method, params)
		}
		return nil, fmt.Errorf("unknown video method %s", method)
	}
}

func (a *App) CanvasBind(sessionID, canvasID string) error {
	eng, err := a.requireVideo()
	if err != nil {
		return err
	}
	return eng.BindCanvasSession(sessionID, canvasID)
}

func firstNonEmptyApp(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
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

func decodePlanEpisodes(v any) []video.PlanEpisode {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out []video.PlanEpisode
	if json.Unmarshal(b, &out) != nil {
		return nil
	}
	return out
}

func intArgApp(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		n := 0
		fmt.Sscanf(t, "%d", &n)
		return n
	}
	return 0
}

func floatArgApp(v any) float64 {
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
		fmt.Sscanf(t, "%f", &n)
		return n
	}
	return 0
}

func studioTitleFromPrompt(prompt string) string {
	p := strings.TrimSpace(prompt)
	for _, prefix := range []string{"做成短片：", "做成短片:"} {
		if strings.HasPrefix(p, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(p, prefix))
		}
	}
	return ""
}

func studioJobPrompt(t video.Topic, p video.StudioProject) string {
	var b strings.Builder
	b.WriteString("做成短片。项目已经建好，用 studio_* 工具推进，不要检查代码仓库，不要调用 shell、list_dir 或 glob。\n")
	if p.ID != "" {
		fmt.Fprintf(&b, "studio_id: %s\n", p.ID)
	}
	if title := strings.TrimSpace(t.Title); title != "" {
		fmt.Fprintf(&b, "选题：%s\n", title)
	}
	if angle := strings.TrimSpace(t.Angle); angle != "" {
		fmt.Fprintf(&b, "角度：%s\n", angle)
	}
	if note := strings.TrimSpace(t.Note); note != "" {
		fmt.Fprintf(&b, "洞察：%s\n", note)
	}
	b.WriteString("先 studio_get，再把口播稿写入 studio_update 的 script。没有镜头素材时不要编造 CAS hash，写完脚本后停下来，让操作者在短片成片页上传静帧或成片。\n")
	return b.String()
}

func (a *App) runStudioJob(j schedule.Job) {
	eng, err := a.requireVideo()
	if err != nil {
		if a.Schedule != nil {
			a.Schedule.Record(j.ID, err)
		}
		return
	}
	sessionID := strings.TrimSpace(j.SessionID)
	if sessionID != "" {
		meta, err := a.GetSession(sessionID)
		if err != nil || session.NormalizeChannel(meta.Channel) != session.ChannelVideo {
			sessionID = ""
		}
	}
	if sessionID == "" {
		meta, err := a.NewSessionOn(a.VideoWorkspace(), session.ChannelVideo)
		if err != nil {
			if a.Schedule != nil {
				a.Schedule.Record(j.ID, err)
			}
			return
		}
		sessionID = meta.ID
		if title := titleFrom(j.Prompt); title != "" && title != "session" {
			_ = a.RenameSession(sessionID, title)
		}
	}
	studioID := strings.TrimSpace(j.StudioID)
	if studioID == "" && strings.TrimSpace(j.TopicID) != "" {
		if p, err := eng.StudioFromTopic(j.TopicID, ""); err == nil {
			studioID = p.ID
		}
	}
	if studioID == "" {
		if title := studioTitleFromPrompt(j.Prompt); title != "" {
			for _, t := range eng.ListTopics("") {
				if strings.TrimSpace(t.Title) == title {
					if p, err := eng.StudioFromTopic(t.ID, ""); err == nil {
						studioID = p.ID
					}
					break
				}
			}
		}
	}
	if studioID != "" {
		_ = eng.BindStudioSession(sessionID, studioID)
	}
	if meta, err := a.GetSession(sessionID); err == nil {
		_, _ = a.SetSessionPinnedSkills(sessionID, uniqueStrings(meta.PinnedSkills, studioSkillNames))
	}
	if a.Inbox != nil {
		a.Inbox.Push(inbox.Item{Kind: inbox.KindHeartbeat, Title: "Shorts " + j.ID, Body: j.Prompt, SessionID: sessionID})
	}
	err = a.StartSendOpts(sessionID, j.Prompt, false, nil)
	if err != nil && !errors.Is(err, ErrQueued) {
		if a.Schedule != nil {
			a.Schedule.Record(j.ID, err)
		}
		return
	}
	if a.Schedule != nil {
		a.Schedule.Record(j.ID, nil)
	}
}

func splitCSVApp(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (a *App) enrichStudio(p video.StudioProject) map[string]any {
	m := objJSON(p)
	m["media_url"] = a.mediaURL(p.ResultHash)
	m["poster_url"] = a.mediaURL(p.PosterHash)
	m["narration_url"] = a.mediaURL(p.NarrationHash)
	return m
}

func (a *App) enrichMedia(ref video.MediaRef) map[string]any {
	m := objJSON(ref)
	m["media_url"] = a.mediaURL(ref.CASHash)
	m["poster_url"] = a.mediaURL(ref.PosterHash)
	return m
}
