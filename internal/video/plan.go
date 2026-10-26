// 短剧的分集方案。按原文切出草稿、改边界、拆开、合并，确认后才写成集。
// 不改写剧本，也不生成画面。

package video

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// GetPlan 按剧编号读取分集方案。一部剧只认第一条。没有时返回 plan not found。
// dramaID 与每条方案的 DramaID 比较，相等就返回该条。
func (e *Engine) GetPlan(dramaID string) (EpisodePlan, error) {
	for _, p := range loadCol[EpisodePlan](e, colPlans) {
		if p.DramaID == dramaID {
			return p, nil
		}
	}
	return EpisodePlan{}, fmt.Errorf("plan not found")
}

// savePlan 写回方案并刷新更新时间。没有编号或创建时间时补上。
// p 写入 UpdatedAt；CreatedAt 为空时用 UpdatedAt，ID 为空时用 NewID，再传给 putDoc。
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

// ProposeEpisodes 按原文重写草稿方案。已经确认的方案拒绝再提案。
// 没给切分就按章节或窗口自己切。给了切分就吸到邻近的断点上，切完仍不合法就退回自己切。短的并进上一集，超长的再拆开。
// dramaID 传给 GetSource 和 GetPlan，并写入方案的 DramaID。
// proposed 长度为 0 时改用 heuristicEpisodes；否则逐条把 Start、End 交给 snapToSeam，End 小于等于 Start 的丢掉，End 大于原文 rune 数时收到该数，Start 小于 0 时写成 0。
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

// PatchPlan 用传入的集界替换草稿。已确认的方案拒绝修改。集界会吸到断点上并重新编号。
// dramaID 传给 GetPlan 和 GetSource。
// episodes 与原文、断点一起传给 clampPlanEpisodes，结果写入 Episodes。
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

// SplitPlanEpisode 把第 n 集从中间拆开，标题加上「上」「下」。中间切不开就保持原样。已确认的方案拒绝拆。
// dramaID 传给 GetPlan 和 GetSource。
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

// MergePlanEpisodes 把第 n 集和紧跟着的下一集合并。钩子、梗概和角色编号并到前一集。已确认就拒绝。没有紧跟着的下一集时，集界不变，仍写回。
// dramaID 传给 GetPlan。
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

// CommitEpisodes 把草稿写成真正的集。已经确认过就原样返回。方案是空的，或这部剧已经有集，就拒绝。
// 中途创建失败时，这次新建的集会删掉。成功后队列清空。
// dramaID 传给 GetPlan、GetSource、ListEpisodes 和 createEpisode。
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
	created := make([]string, 0, len(plan.Episodes))
	for i, pe := range plan.Episodes {
		content := runeSlice(text, pe.Start, pe.End)
		title := strings.TrimSpace(pe.Title)
		ep, err := e.createEpisode(dramaID, title, content, pe.N, pe.CharacterIDs, nil, nil)
		if err != nil {
			for _, id := range created {
				_ = e.DeleteEpisode(id)
			}
			return EpisodePlan{}, err
		}
		created = append(created, ep.ID)
		pe.EpisodeID = ep.ID
		plan.Episodes[i] = pe
	}
	plan.Status = "committed"
	plan.QueueKind = ""
	plan.QueueIndex = 0
	return e.savePlan(plan)
}

// NextScriptEpisode 从队列位置起找第一集还没有剧本的集。方案不存在或还没确认时返回假。
// dramaID 传给 GetPlan 和 ListEpisodes。
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

// AdvanceScriptQueue 在剧本队列里把指针移到刚写完的那一集之后。队列不是 scripts 时原样返回。走完最后一集就清空队列种类。
// dramaID 传给 GetPlan。
// doneEpisodeID 与各集 EpisodeID 比较，相等时把 QueueIndex 设为该下标加 1。
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

// StartScriptQueue 把已确认的方案放进剧本队列，指针回到第一集。还没确认就拒绝。
// dramaID 传给 GetPlan。
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

// heuristicEpisodes 在没有提案时自己切集。有两个以上章节就按章节切，否则按窗口切到断点上。太短的并进上一集，超长的再拆。标题优先用标题行，否则写成「第 n 集」。
// text 转成 rune，并传给 CandidateSeams 和 chapterStarts。
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

// clampPlanEpisodes 把集界吸到断点上，并掉过短的一段，拆开超长的一段，然后重新编号。目标秒数超出 1 到 120 就按字数重估。
// text 用来数 rune，并转成 rune 切片传给 headingAt。
// eps 逐条经 snapToSeam 调整 Start 和 End；Start 小于 0 时写成 0，End 超出原文 rune 数时收到该数；长度小于 minEpisodeRunes/2 且已有上一条时并进去；长度大于 maxEpisodeRunes 时按窗口拆开；TargetSeconds 小于等于 0 或大于 120 时改为 filmTarget 的结果。
// seams 传给 snapToSeam。
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

// renumberPlan 按顺序把集号写成 1 起。标题空着就写成「第 n 集」。
// eps 按顺序把 N 写成下标加 1，Title 去掉首尾空白后为空时写成「第」加该序号再加「集」。
func renumberPlan(eps []PlanEpisode) []PlanEpisode {
	for i := range eps {
		eps[i].N = i + 1
		if strings.TrimSpace(eps[i].Title) == "" {
			eps[i].Title = fmt.Sprintf("第 %d 集", i+1)
		}
	}
	return eps
}

// uniqueIDs 去掉空白和重复的编号，保留第一次出现的顺序。
// ids 逐条去掉首尾空白，空串或已经出现过的丢掉。
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
