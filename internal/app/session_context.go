// 按轨迹和材料估算会话已经占用的上下文。

package app

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// measureSessionContext 按会话的 harness、技能、历史和窗口估算上下文形状。材料读失败时只叠历史和现场形状。
// sessionID 传给 GetSession、ReadTrace、liveShape、BindSpill、extraTools、attachDramaTools、attachCanvasTools、attachStudioTools 和 sessionProfile。
func (a *App) measureSessionContext(sessionID string) runtime.ShapeReport {
	meta, _ := a.GetSession(sessionID)
	ws := meta.Workspace
	if ws == "" {
		ws = a.Workspace()
	}
	model := strings.TrimSpace(meta.Model)
	if model == "" {
		model = a.Config.Model
	}
	window := runtime.EffectiveModelWindow(model, a.Config.ContextWindow)
	hash := session.ResolveHarness(session.NormalizePolicy(meta.HarnessPolicy), meta.Harness, a.ActiveHash())
	if hash == "" {
		hash = a.ActiveHash()
	}
	loop, frags, pb, skills, _, pol, err := a.Materials(hash)
	if err != nil {
		loop, frags, pb, skills, _, pol, err = a.Materials(a.ActiveHash())
	}
	evs, _ := a.ReadTrace(sessionID)
	if err != nil {
		return overlayPrompt(runtime.ShapeFromEvents(evs, window), a.liveShape(sessionID))
	}
	loop = runtime.ApplyChatHorizon(loop)
	disk := runtime.LoadSkillDirs(a.skillRoots(ws)...)
	skills = runtime.MergeSkills(skills, disk)
	skillBodies, _, _ := runtime.MapSkills(skills)
	spill := runtime.BindSpill(a.Home.Root, ws, sessionID)
	tools := &runtime.WorkspaceTools{
		Workspace:          ws,
		SessionID:          sessionID,
		Caps:               a.Caps,
		Skills:             skillBodies,
		Loaded:             append([]string(nil), meta.LoadedSkills...),
		Policy:             pol,
		PlanMode:           loop.PlanMode,
		Extra:              a.extraTools(sessionID),
		Spill:              spill,
		PlanText:           meta.PlanText,
		Packs:              a.packSessions(ws),
		SkillAutoInvoke:    runtime.NormalizeAutoInvoke(a.Config.Skills.AutoInvoke),
		SkillCatalogBudget: a.Config.Skills.CatalogBudgetChars,
		CatalogLane:        session.NormalizeChannel(meta.Channel),
	}
	a.attachPersonal(tools)
	a.attachDramaTools(tools, sessionID)
	a.attachCanvasTools(tools, sessionID)
	a.attachStudioTools(tools, sessionID)
	runtime.ApplyPackBootstrap(tools)
	profile := ""
	if a.Memory != nil {
		if p := a.sessionProfile(sessionID); p == nil || p.Memory {
			profile = a.Memory.ProfilePin(profilePinBytes)
		}
	}
	pageCtx := ""
	if meta.PageID != "" && a.Pages != nil {
		if pg, err := a.Pages.Get("", meta.PageID); err == nil {
			pageCtx = runtime.PageInject(pg.Title, pg.Content, 3200)
		}
	}
	rep := runtime.MeasureContext(runtime.MeasureOpts{
		Loop:          loop,
		Fragments:     frags,
		Playbook:      pb,
		Skills:        skills,
		Workspace:     ws,
		History:       runtime.MessagesFromEventsOpts(evs, spill),
		Tools:         tools,
		Spill:         spill,
		ModelWindow:   window,
		PlanText:      meta.PlanText,
		PageContext:   pageCtx,
		ProfileMemory: profile,
	})
	if pp := promptFromEvents(evs); pp > 0 {
		rep.ProviderPrompt = pp
	} else if live := a.liveShape(sessionID); live.ProviderPrompt > 0 {
		rep.ProviderPrompt = live.ProviderPrompt
	}
	return rep
}

// liveShape 返回该会话上次记住的形状。
// sessionID 用作 lastShape 的键。
func (a *App) liveShape(sessionID string) runtime.ShapeReport {
	a.shapeMu.Lock()
	defer a.shapeMu.Unlock()
	return a.lastShape[sessionID]
}

// promptFromEvents 从最近的事件里取 provider_prompt，没有则返回 0。
// evs 从最后一条往前找 Payload 里的 provider_prompt。
func promptFromEvents(evs []trace.Event) int {
	for i := len(evs) - 1; i >= 0; i-- {
		if n := payloadInt(evs[i].Payload, "provider_prompt"); n > 0 {
			return n
		}
	}
	return 0
}

// overlayPrompt 在估算结果没有供应商提示量时，用现场形状里的值补上。
// rep 在 ProviderPrompt 为 0 且 live 的大于 0 时，改成 live 的 ProviderPrompt。
// live 只在自身 ProviderPrompt 大于 0 且 rep 的为 0 时写入 rep。
func overlayPrompt(rep, live runtime.ShapeReport) runtime.ShapeReport {
	if rep.ProviderPrompt == 0 && live.ProviderPrompt > 0 {
		rep.ProviderPrompt = live.ProviderPrompt
	}
	return rep
}

// payloadInt 把载荷里的整数或浮点读成 int，没有则返回 0。
// key 用作 p 的查找键。
func payloadInt(p map[string]any, key string) int {
	if p == nil {
		return 0
	}
	switch v := p[key].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	}
	return 0
}
