// 按选题批量开短片。模板、时长和声音从选题文字里挑。
// 一次最多八条。已经做完的不重做。画幅跟着模板的预设走，不在挑模板时另定。

package video

import (
	"fmt"
	"strings"
)

const topicFilmBatchMax = 8

// topicFilmPlan 是从一张选题推出的模板、时长、受众、声音和备注。
type topicFilmPlan struct {
	TemplateID string
	Duration   int
	Audience   string
	Voice      string
	Brief      string
}

// planTopicFilm 按选题文字挑模板、时长、声音和受众。教程、测评、故事、科普和热榜各走不同模板。没有角度时受众写成「会刷到这条的人」。不在这里定画幅。
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

// hasAny 判断字符串里有没有其中任何一个片段。空片段不算命中。
func hasAny(s string, parts ...string) bool {
	for _, p := range parts {
		if p != "" && strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// StartTopicFilms 为每张选题准备一条短片，套上计划，再按顺序做还没完成的。立刻返回，进度看 auto_phase。
// 没勾选就拒绝。一次超过八条也拒绝。重复编号和空白会去掉。已经做完的原样返回，不重做。
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

// prepareTopicFilm 复用选题上已有的短片，没有就新建。还没套模板、还没有成片、也没在做时，才套上计划。
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

// applyTopicFilmPlan 套上模板，再写标题、受众、声音、时长和备注。适配写成 cover，分镜尺寸按画幅重算。
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

// filmNeedsRun 判断这条还要不要做。排队中、正在做或已经完成的返回假。失败的返回真。其余看还没有成片。
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

// filmPhaseBusy 判断一键成片正停在排队、口播、配音、画面或合成。完成和失败都不算正忙。
func filmPhaseBusy(phase string) bool {
	switch phase {
	case "queued", "script", "voice", "visuals", "compose":
		return true
	default:
		return false
	}
}

// enqueueTopicFilms 把编号排进待做队列，已在队列里的不重复。还没有人在抽队列时才启动。
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

// drainTopicFilms 按队列一条一条做。队列空了就停。
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

// runTopicFilm 把阶段写成 script 并开始一键成片。同一条已经在做就直接返回。工程读不到或保存失败就清掉占用。
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

// cleanIDs 去掉空白和重复编号，保留第一次出现的顺序。
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
