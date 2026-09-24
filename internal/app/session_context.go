package app

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

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
	evs, _ := a.Trajectory(sessionID)
	if err != nil {
		return overlayPrompt(runtime.ShapeFromEvents(evs, window), a.liveShape(sessionID))
	}
	loop = runtime.ApplyChatHorizon(loop)
	skillBodies := map[string]string{}
	for _, s := range skills {
		skillBodies[s.Name] = s.Body
	}
	disk := runtime.LoadSkillDirs(runtime.SkillRoots(a.Home.Root, ws, bundledSkillsDir(a.BundledEvals))...)
	skills = runtime.MergeSkills(skills, disk)
	for _, s := range disk {
		skillBodies[s.Name] = s.Body
	}
	spill := runtime.BindSpill(a.Home.Root, ws, sessionID)
	tools := &runtime.WorkspaceTools{
		Workspace: ws,
		SessionID: sessionID,
		Caps:      a.Caps,
		Skills:    skillBodies,
		Loaded:    append([]string(nil), meta.LoadedSkills...),
		Policy:    pol,
		PlanMode:  loop.PlanMode,
		Extra:     a.extraTools(sessionID),
		Spill:     spill,
		PlanText:  meta.PlanText,
	}
	a.attachPersonal(tools)
	a.attachDramaTools(tools, sessionID)
	a.attachCanvasTools(tools, sessionID)
	rep := runtime.MeasureContext(runtime.MeasureOpts{
		Loop:        loop,
		Fragments:   frags,
		Playbook:    pb,
		Skills:      skills,
		Workspace:   ws,
		History:     runtime.MessagesFromEventsOpts(evs, spill),
		Tools:       tools,
		Spill:       spill,
		ModelWindow: window,
		PlanText:    meta.PlanText,
	})
	if pp := promptFromEvents(evs); pp > 0 {
		rep.ProviderPrompt = pp
	} else if live := a.liveShape(sessionID); live.ProviderPrompt > 0 {
		rep.ProviderPrompt = live.ProviderPrompt
	}
	return rep
}

func (a *App) liveShape(sessionID string) runtime.ShapeReport {
	a.shapeMu.Lock()
	defer a.shapeMu.Unlock()
	return a.lastShape[sessionID]
}

func promptFromEvents(evs []trace.Event) int {
	for i := len(evs) - 1; i >= 0; i-- {
		if n := payloadInt(evs[i].Payload, "provider_prompt"); n > 0 {
			return n
		}
	}
	return 0
}

func overlayPrompt(rep, live runtime.ShapeReport) runtime.ShapeReport {
	if rep.ProviderPrompt == 0 && live.ProviderPrompt > 0 {
		rep.ProviderPrompt = live.ProviderPrompt
	}
	return rep
}

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
