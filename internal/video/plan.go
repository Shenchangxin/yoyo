package video

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func (e *Engine) GetPlan(dramaID string) (EpisodePlan, error) {
	for _, p := range loadCol[EpisodePlan](e, colPlans) {
		if p.DramaID == dramaID {
			return p, nil
		}
	}
	return EpisodePlan{}, fmt.Errorf("plan not found")
}

func (e *Engine) savePlan(p EpisodePlan) (EpisodePlan, error) {
	p.UpdatedAt = Now()
	if p.CreatedAt == "" {
		p.CreatedAt = p.UpdatedAt
	}
	if p.ID == "" {
		p.ID = NewID()
	}
	return p, e.putDoc(colPlans, p.ID, p)
}

func (e *Engine) ProposeEpisodes(dramaID string, proposed []PlanEpisode) (EpisodePlan, error) {
	src, err := e.GetSource(dramaID)
	if err != nil {
		return EpisodePlan{}, err
	}
	if cur, err := e.GetPlan(dramaID); err == nil && cur.Status == "committed" {
		return EpisodePlan{}, fmt.Errorf("plan already committed")
	}
	text, err := e.SourceText(src)
	if err != nil {
		return EpisodePlan{}, err
	}
	seams := CandidateSeams(text)
	n := utf8.RuneCountInString(text)
	var eps []PlanEpisode
	if len(proposed) == 0 {
		eps = heuristicEpisodes(text)
	} else {
		for _, pe := range proposed {
			pe.Start = snapToSeam(seams, pe.Start)
			pe.End = snapToSeam(seams, pe.End)
			if pe.End <= pe.Start {
				continue
			}
			if pe.End > n {
				pe.End = n
			}
			if pe.Start < 0 {
				pe.Start = 0
			}
			eps = append(eps, pe)
		}
		if len(eps) == 0 {
			eps = heuristicEpisodes(text)
		}
	}
	eps = clampPlanEpisodes(text, eps, seams)
	now := Now()
	plan := EpisodePlan{
		DramaID: dramaID, Status: "draft", SourceID: src.ID,
		Episodes: eps, CreatedAt: now, UpdatedAt: now,
	}
	if old, err := e.GetPlan(dramaID); err == nil {
		plan.ID = old.ID
		plan.CreatedAt = old.CreatedAt
	} else {
		plan.ID = NewID()
	}
	return e.savePlan(plan)
}

func (e *Engine) PatchPlan(dramaID string, episodes []PlanEpisode) (EpisodePlan, error) {
	plan, err := e.GetPlan(dramaID)
	if err != nil {
		return EpisodePlan{}, err
	}
	if plan.Status == "committed" {
		return EpisodePlan{}, fmt.Errorf("plan already committed")
	}
	src, err := e.GetSource(dramaID)
	if err != nil {
		return EpisodePlan{}, err
	}
	text, err := e.SourceText(src)
	if err != nil {
		return EpisodePlan{}, err
	}
	seams := CandidateSeams(text)
	plan.Episodes = clampPlanEpisodes(text, episodes, seams)
	return e.savePlan(plan)
}

func (e *Engine) SplitPlanEpisode(dramaID string, n int) (EpisodePlan, error) {
	plan, err := e.GetPlan(dramaID)
	if err != nil {
		return EpisodePlan{}, err
	}
	if plan.Status == "committed" {
		return EpisodePlan{}, fmt.Errorf("plan already committed")
	}
	src, err := e.GetSource(dramaID)
	if err != nil {
		return EpisodePlan{}, err
	}
	text, err := e.SourceText(src)
	if err != nil {
		return EpisodePlan{}, err
	}
	seams := CandidateSeams(text)
	var next []PlanEpisode
	for _, pe := range plan.Episodes {
		if pe.N != n {
			next = append(next, pe)
			continue
		}
		mid := snapToSeam(seams, (pe.Start+pe.End)/2)
		if mid <= pe.Start+minEpisodeRunes/2 || mid >= pe.End-minEpisodeRunes/2 {
			next = append(next, pe)
			continue
		}
		a := pe
		a.End = mid
		a.Title = strings.TrimSpace(a.Title + " · 上")
		a.TargetSeconds = filmTarget(a.End - a.Start)
		b := pe
		b.Start = mid
		b.Title = strings.TrimSpace(strings.TrimSuffix(pe.Title, " · 上") + " · 下")
		b.Logline = ""
		b.Hook = pe.Hook
		b.TargetSeconds = filmTarget(b.End - b.Start)
		next = append(next, a, b)
	}
	plan.Episodes = renumberPlan(next)
	return e.savePlan(plan)
}

func (e *Engine) MergePlanEpisodes(dramaID string, n int) (EpisodePlan, error) {
	plan, err := e.GetPlan(dramaID)
	if err != nil {
		return EpisodePlan{}, err
	}
	if plan.Status == "committed" {
		return EpisodePlan{}, fmt.Errorf("plan already committed")
	}
	var next []PlanEpisode
	for i := 0; i < len(plan.Episodes); i++ {
		pe := plan.Episodes[i]
		if pe.N == n && i+1 < len(plan.Episodes) {
			nxt := plan.Episodes[i+1]
			pe.End = nxt.End
			if nxt.Hook != "" {
				pe.Hook = nxt.Hook
			}
			if pe.Logline != "" && nxt.Logline != "" {
				pe.Logline = pe.Logline + " / " + nxt.Logline
			}
			pe.TargetSeconds = filmTarget(pe.End - pe.Start)
			pe.CharacterIDs = uniqueIDs(append(pe.CharacterIDs, nxt.CharacterIDs...))
			next = append(next, pe)
			i++
			continue
		}
		next = append(next, pe)
	}
	plan.Episodes = renumberPlan(next)
	return e.savePlan(plan)
}

func (e *Engine) CommitEpisodes(dramaID string) (EpisodePlan, error) {
	plan, err := e.GetPlan(dramaID)
	if err != nil {
		return EpisodePlan{}, err
	}
	if plan.Status == "committed" {
		return plan, nil
	}
	if len(plan.Episodes) == 0 {
		return EpisodePlan{}, fmt.Errorf("plan is empty")
	}
	src, err := e.GetSource(dramaID)
	if err != nil {
		return EpisodePlan{}, err
	}
	text, err := e.SourceText(src)
	if err != nil {
		return EpisodePlan{}, err
	}
	existing, _ := e.ListEpisodes(dramaID)
	if len(existing) > 0 {
		return EpisodePlan{}, fmt.Errorf("this series already has episodes — confirm on an empty series")
	}
	for i, pe := range plan.Episodes {
		content := runeSlice(text, pe.Start, pe.End)
		title := strings.TrimSpace(pe.Title)
		ep, err := e.createEpisode(dramaID, title, content, pe.N, pe.CharacterIDs, nil, nil)
		if err != nil {
			return EpisodePlan{}, err
		}
		pe.EpisodeID = ep.ID
		plan.Episodes[i] = pe
	}
	plan.Status = "committed"
	plan.QueueKind = ""
	plan.QueueIndex = 0
	return e.savePlan(plan)
}

func (e *Engine) NextScriptEpisode(dramaID string) (Episode, bool) {
	plan, err := e.GetPlan(dramaID)
	if err != nil || plan.Status != "committed" {
		return Episode{}, false
	}
	eps, _ := e.ListEpisodes(dramaID)
	byID := map[string]Episode{}
	for _, ep := range eps {
		byID[ep.ID] = ep
	}
	start := plan.QueueIndex
	if start < 0 {
		start = 0
	}
	for i := start; i < len(plan.Episodes); i++ {
		pe := plan.Episodes[i]
		ep, ok := byID[pe.EpisodeID]
		if !ok {
			continue
		}
		if strings.TrimSpace(ep.ScriptContent) == "" {
			return ep, true
		}
	}
	return Episode{}, false
}

func (e *Engine) AdvanceScriptQueue(dramaID, doneEpisodeID string) (EpisodePlan, error) {
	plan, err := e.GetPlan(dramaID)
	if err != nil {
		return EpisodePlan{}, err
	}
	if plan.QueueKind != "scripts" {
		return plan, nil
	}
	for i, pe := range plan.Episodes {
		if pe.EpisodeID == doneEpisodeID {
			plan.QueueIndex = i + 1
			break
		}
	}
	if plan.QueueIndex >= len(plan.Episodes) {
		plan.QueueKind = ""
	}
	return e.savePlan(plan)
}

func (e *Engine) StartScriptQueue(dramaID string) (EpisodePlan, error) {
	plan, err := e.GetPlan(dramaID)
	if err != nil {
		return EpisodePlan{}, err
	}
	if plan.Status != "committed" {
		return EpisodePlan{}, fmt.Errorf("confirm the episode map first")
	}
	plan.QueueKind = "scripts"
	plan.QueueIndex = 0
	return e.savePlan(plan)
}

func heuristicEpisodes(text string) []PlanEpisode {
	runes := []rune(text)
	n := len(runes)
	seams := CandidateSeams(text)
	chapters := chapterStarts(text)
	var bounds [][2]int
	if len(chapters) >= 2 {
		if chapters[0] > minEpisodeRunes {
			bounds = append(bounds, [2]int{0, chapters[0]})
		} else if chapters[0] > 0 {
			chapters[0] = 0
		}
		for i, start := range chapters {
			end := n
			if i+1 < len(chapters) {
				end = chapters[i+1]
			}
			if start >= end {
				continue
			}
			bounds = append(bounds, [2]int{start, end})
		}
	} else {
		start := 0
		for start < n {
			target := start + episodeWindow
			if target >= n {
				bounds = append(bounds, [2]int{start, n})
				break
			}
			end := snapToSeam(seams, target)
			if end <= start {
				end = minInt(n, start+episodeWindow)
			}
			bounds = append(bounds, [2]int{start, end})
			start = end
		}
	}
	var packed [][2]int
	for _, b := range bounds {
		if len(packed) > 0 && b[1]-b[0] < minEpisodeRunes {
			packed[len(packed)-1][1] = b[1]
			continue
		}
		packed = append(packed, b)
	}
	var split [][2]int
	for _, b := range packed {
		if b[1]-b[0] <= maxEpisodeRunes {
			split = append(split, b)
			continue
		}
		start := b[0]
		for start < b[1] {
			target := minInt(b[1], start+episodeWindow)
			end := snapToSeam(seams, target)
			if end <= start {
				end = minInt(b[1], start+episodeWindow)
			}
			if end > b[1] {
				end = b[1]
			}
			split = append(split, [2]int{start, end})
			start = end
		}
	}
	out := make([]PlanEpisode, 0, len(split))
	for i, b := range split {
		chunk := string(runes[b[0]:minInt(len(runes), b[1])])
		title := headingAt(runes, b[0])
		if title == "" {
			title = fmt.Sprintf("第 %d 集", i+1)
		}
		logline := strings.TrimSpace(strings.ReplaceAll(chunk, "\n", " "))
		if utf8.RuneCountInString(logline) > 80 {
			logline = string([]rune(logline)[:80]) + "…"
		}
		out = append(out, PlanEpisode{
			N: i + 1, Title: title, Logline: logline,
			Start: b[0], End: b[1], TargetSeconds: filmTarget(b[1] - b[0]),
		})
	}
	return out
}

func clampPlanEpisodes(text string, eps []PlanEpisode, seams []int) []PlanEpisode {
	n := utf8.RuneCountInString(text)
	runes := []rune(text)
	var out []PlanEpisode
	for _, pe := range eps {
		pe.Start = snapToSeam(seams, pe.Start)
		pe.End = snapToSeam(seams, pe.End)
		if pe.Start < 0 {
			pe.Start = 0
		}
		if pe.End > n {
			pe.End = n
		}
		if pe.End-pe.Start < minEpisodeRunes/2 && len(out) > 0 {
			out[len(out)-1].End = pe.End
			continue
		}
		for pe.End-pe.Start > maxEpisodeRunes {
			mid := snapToSeam(seams, pe.Start+episodeWindow)
			if mid <= pe.Start {
				mid = minInt(pe.End, pe.Start+episodeWindow)
			}
			if mid >= pe.End {
				break
			}
			chunk := pe
			chunk.End = mid
			chunk.TargetSeconds = filmTarget(chunk.End - chunk.Start)
			if chunk.Title == "" {
				chunk.Title = headingAt(runes, chunk.Start)
			}
			out = append(out, chunk)
			pe.Start = mid
		}
		if pe.TargetSeconds <= 0 || pe.TargetSeconds > 120 {
			pe.TargetSeconds = filmTarget(pe.End - pe.Start)
		}
		if pe.Title == "" {
			pe.Title = headingAt(runes, pe.Start)
		}
		out = append(out, pe)
	}
	return renumberPlan(out)
}

func renumberPlan(eps []PlanEpisode) []PlanEpisode {
	for i := range eps {
		eps[i].N = i + 1
		if strings.TrimSpace(eps[i].Title) == "" {
			eps[i].Title = fmt.Sprintf("第 %d 集", i+1)
		}
	}
	return eps
}

func uniqueIDs(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
