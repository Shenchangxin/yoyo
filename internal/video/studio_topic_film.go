package video

import (
	"fmt"
	"strings"
)

const topicFilmBatchMax = 8

type topicFilmPlan struct {
	TemplateID string
	Duration   int
	Audience   string
	Voice      string
	Brief      string
}

// planTopicFilm picks aspect, template, voice, and length from the topic.
// The operator does not set these on the one-click path.
func planTopicFilm(t Topic) topicFilmPlan {
	blob := strings.TrimSpace(t.Title + "\n" + t.Angle + "\n" + t.Note)
	templateID := "ugc-hook"
	switch {
	case hasAny(blob, "教程", "怎么", "如何", "步骤", "攻略", "三步"):
		templateID = "howto"
	case hasAny(blob, "测评", "开箱", "产品", "好用吗", "推荐"):
		templateID = "product"
	case hasAny(blob, "故事", "剧情", "反转", "小说"):
		templateID = "story"
	case hasAny(blob, "为什么", "原理", "科普", "知识") || strings.EqualFold(t.SourcePlatform, "zhihu"):
		templateID = "explainer"
	case strings.TrimSpace(t.SourcePlatform) != "":
		templateID = "news-take"
	}
	tpl, _ := LookupStudioTemplate(templateID)
	voice := "alloy"
	switch tpl.Tone {
	case "energetic":
		voice = "nova"
	case "professional":
		voice = "onyx"
	case "storytelling":
		voice = "fable"
	}
	duration := 30
	switch templateID {
	case "howto", "explainer", "product", "story":
		duration = 45
	}
	audience := strings.TrimSpace(t.Angle)
	if audience == "" {
		audience = "会刷到这条的人"
	}
	return topicFilmPlan{
		TemplateID: templateID,
		Duration:   duration,
		Audience:   audience,
		Voice:      voice,
		Brief:      strings.TrimSpace(t.Note),
	}
}

func hasAny(s string, parts ...string) bool {
	for _, p := range parts {
		if p != "" && strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// StartTopicFilms creates or reuses a short per topic, applies the plan, and
// makes the films one after another. Poll auto_phase. Already finished films
// are returned and not remade.
func (e *Engine) StartTopicFilms(ids []string) ([]StudioProject, error) {
	ids = cleanIDs(ids)
	if len(ids) == 0 {
		return nil, fmt.Errorf("先勾选选题")
	}
	if len(ids) > topicFilmBatchMax {
		return nil, fmt.Errorf("一次最多 %d 条。先做这一批，成片会出现在交付里。", topicFilmBatchMax)
	}
	out := make([]StudioProject, 0, len(ids))
	var queue []string
	for _, id := range ids {
		p, err := e.prepareTopicFilm(id)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
		if filmNeedsRun(p) {
			queue = append(queue, p.ID)
		}
	}
	for _, id := range queue {
		e.markAuto(id, "queued")
	}
	e.enqueueTopicFilms(queue)
	for i, p := range out {
		if latest, err := e.GetStudioProject(p.ID); err == nil {
			out[i] = latest
		}
	}
	return out, nil
}

func (e *Engine) prepareTopicFilm(topicID string) (StudioProject, error) {
	t, err := e.GetTopic(topicID)
	if err != nil {
		return StudioProject{}, err
	}
	var p StudioProject
	if t.StudioProjectID != "" {
		if got, gerr := e.GetStudioProject(t.StudioProjectID); gerr == nil {
			p = got
		}
	}
	if p.ID == "" {
		created, cerr := e.CreateStudioProject(StudioProject{Title: t.Title, TopicID: t.ID})
		if cerr != nil {
			return StudioProject{}, cerr
		}
		p = created
	}
	if p.TemplateID == "" && p.ResultHash == "" && !filmPhaseBusy(p.AutoPhase) {
		planned, perr := e.applyTopicFilmPlan(p.ID, t)
		if perr != nil {
			return p, perr
		}
		p = planned
	}
	return p, nil
}

func (e *Engine) applyTopicFilmPlan(id string, t Topic) (StudioProject, error) {
	plan := planTopicFilm(t)
	p, err := e.ApplyStudioTemplate(id, plan.TemplateID)
	if err != nil {
		return StudioProject{}, err
	}
	title := strings.TrimSpace(t.Title)
	if title == "" {
		title = "短片"
	}
	p.Title = title
	p.Audience = plan.Audience
	p.Voice = plan.Voice
	p.VoiceSpeed = 1
	p.TargetDuration = plan.Duration
	p.Brief = plan.Brief
	p.Storyboard.Fit = "cover"
	w, h := aspectSize(p.Aspect)
	p.Storyboard.Size = fmt.Sprintf("%dx%d", w, h)
	e.noteStudio(&p, "plan", plan.TemplateID)
	return e.saveStudio(p)
}

func filmNeedsRun(p StudioProject) bool {
	switch p.AutoPhase {
	case "queued", "script", "voice", "visuals", "compose", "done":
		return false
	case "failed":
		return true
	default:
		return p.ResultHash == ""
	}
}

func filmPhaseBusy(phase string) bool {
	switch phase {
	case "queued", "script", "voice", "visuals", "compose":
		return true
	default:
		return false
	}
}

func (e *Engine) enqueueTopicFilms(ids []string) {
	if len(ids) == 0 {
		return
	}
	e.filmMu.Lock()
	for _, id := range ids {
		if id == "" {
			continue
		}
		dup := false
		for _, x := range e.filmPending {
			if x == id {
				dup = true
				break
			}
		}
		if !dup {
			e.filmPending = append(e.filmPending, id)
		}
	}
	start := !e.filmRunning
	e.filmRunning = true
	e.filmMu.Unlock()
	if start {
		go e.drainTopicFilms()
	}
}

func (e *Engine) drainTopicFilms() {
	for {
		e.filmMu.Lock()
		if len(e.filmPending) == 0 {
			e.filmRunning = false
			e.filmMu.Unlock()
			return
		}
		id := e.filmPending[0]
		e.filmPending = e.filmPending[1:]
		e.filmMu.Unlock()
		e.runTopicFilm(id)
	}
}

func (e *Engine) runTopicFilm(id string) {
	key := "studio-auto:" + id
	if _, loaded := e.inflight.LoadOrStore(key, true); loaded {
		return
	}
	p, err := e.GetStudioProject(id)
	if err != nil {
		e.inflight.Delete(key)
		return
	}
	p.AutoPhase = "script"
	p.AutoError = ""
	e.noteStudio(&p, "auto", "started")
	if _, err := e.saveStudio(p); err != nil {
		e.inflight.Delete(key)
		return
	}
	e.runStudioAuto(id)
}

func cleanIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
