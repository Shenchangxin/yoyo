// 短剧、集，以及生成任务写回角色、场景、道具和镜头。
// 不切分集，也不合成短片。

package video

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/filestore"
)

// writeBack 在任务有结果时写回。带了短片工程编号就交给 applyStudioJob。否则把图写回角色、场景或道具，把视频和海报写回镜头，再看这一集的阶段能不能收口。
func (e *Engine) writeBack(j Job) error {
	params := parseParams(j.Params)
	if params.StudioProjectID != "" {
		return e.applyStudioJob(j, params)
	}
	now := Now()
	var err error
	switch {
	case j.CharacterID != "":
		rec, getErr := getDoc[characterRec](e, colCharacters, j.CharacterID)
		if getErr != nil {
			return getErr
		}
		rec.ImageHash = j.ResultHash
		rec.UpdatedAt = now
		err = e.putDoc(colCharacters, rec.ID, rec)
	case j.SceneID != "":
		rec, getErr := getDoc[sceneRec](e, colScenes, j.SceneID)
		if getErr != nil {
			return getErr
		}
		rec.ImageHash = j.ResultHash
		rec.UpdatedAt = now
		err = e.putDoc(colScenes, rec.ID, rec)
	case j.PropID != "":
		rec, getErr := getDoc[propRec](e, colProps, j.PropID)
		if getErr != nil {
			return getErr
		}
		rec.ImageHash = j.ResultHash
		rec.UpdatedAt = now
		err = e.putDoc(colProps, rec.ID, rec)
	case j.StoryboardID != "" && j.Type == "video":
		rec, getErr := getDoc[shotRec](e, colShots, j.StoryboardID)
		if getErr != nil {
			return getErr
		}
		rec.VideoHash = j.ResultHash
		rec.PosterHash = j.PosterHash
		rec.Status = "ready"
		rec.UpdatedAt = now
		err = e.putDoc(colShots, rec.ID, rec)
	}
	if err == nil && j.EpisodeID != "" {
		e.maybeFinishStage(j)
	}
	return err
}

// maybeFinishStage 在这一集的静帧都齐了时把 assets 标成完成，镜头视频都齐了时把 gen 标成完成。没有集编号就什么都不做。
func (e *Engine) maybeFinishStage(j Job) {
	if j.EpisodeID == "" {
		return
	}
	switch {
	case j.Type == "image" || j.CharacterID != "" || j.SceneID != "" || j.PropID != "":
		if e.missingStills(j.EpisodeID) == 0 {
			_ = e.patchPipeline(j.EpisodeID, "assets", "done", "")
		}
	case j.Type == "video" && j.StoryboardID != "":
		if e.missingClips(j.EpisodeID) == 0 {
			_ = e.patchPipeline(j.EpisodeID, "gen", "done", "")
		}
	}
}

// patchPipeline 给这一集的流水线阶段写上状态、错误和更新时间。原来的流水线解不开就从空对象写起。
func (e *Engine) patchPipeline(episodeID, stage, status, errMsg string) error {
	rec, err := getDoc[episodeRec](e, colEpisodes, episodeID)
	if err != nil {
		return err
	}
	m := map[string]any{}
	_ = json.Unmarshal([]byte(rec.Pipeline), &m)
	if m == nil {
		m = map[string]any{}
	}
	m[stage] = map[string]any{"status": status, "error": errMsg, "updated_at": Now()}
	rec.Pipeline = marshalJSON(m)
	rec.UpdatedAt = Now()
	return e.putDoc(colEpisodes, rec.ID, rec)
}

// ListDramas 列出没删除的剧，并带上未删除的集数。按更新时间从新到旧。没有时返回空切片。
func (e *Engine) ListDramas() ([]Drama, error) {
	recs := loadCol[dramaRec](e, colDramas)
	eps := loadCol[episodeRec](e, colEpisodes)
	counts := map[string]int{}
	for _, ep := range eps {
		if ep.DeletedAt == "" {
			counts[ep.DramaID]++
		}
	}
	var out []Drama
	for _, r := range recs {
		if r.DeletedAt != "" {
			continue
		}
		d := r.Drama
		d.EpisodeCount = counts[d.ID]
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	if out == nil {
		out = []Drama{}
	}
	return out, nil
}

// CreateDrama 新建一部剧。画幅会规范化。风格空着写成 3d，状态空着写成 draft。
func (e *Engine) CreateDrama(d Drama) (Drama, error) {
	now := Now()
	d.ID = NewID()
	d.CreatedAt, d.UpdatedAt = now, now
	d.AspectRatio = NormalizeAspect(d.AspectRatio)
	if d.Style == "" {
		d.Style = "3d"
	}
	if d.Status == "" {
		d.Status = "draft"
	}
	d.Title = strings.TrimSpace(d.Title)
	return d, e.putDoc(colDramas, d.ID, dramaRec{Drama: d})
}

// GetDrama 读取一部没删除的剧，并带上集数。没有或已删除时返回 drama not found。
func (e *Engine) GetDrama(id string) (Drama, error) {
	r, err := getDoc[dramaRec](e, colDramas, id)
	if err != nil || r.DeletedAt != "" {
		return Drama{}, fmt.Errorf("drama not found")
	}
	d := r.Drama
	d.EpisodeCount = e.episodeCount(id)
	return d, nil
}

// episodeCount 数这部剧里还没删除的集。
func (e *Engine) episodeCount(dramaID string) int {
	n := 0
	for _, ep := range loadCol[episodeRec](e, colEpisodes) {
		if ep.DramaID == dramaID && ep.DeletedAt == "" {
			n++
		}
	}
	return n
}

// UpdateDrama 只覆盖标题、简介、类型、风格、画幅和状态里非空的字段。画幅会再规范化。
func (e *Engine) UpdateDrama(d Drama) (Drama, error) {
	cur, err := e.GetDrama(d.ID)
	if err != nil {
		return d, err
	}
	if d.Title != "" {
		cur.Title = d.Title
	}
	if d.Description != "" {
		cur.Description = d.Description
	}
	if d.Genre != "" {
		cur.Genre = d.Genre
	}
	if d.Style != "" {
		cur.Style = d.Style
	}
	if d.AspectRatio != "" {
		cur.AspectRatio = NormalizeAspect(d.AspectRatio)
	}
	if d.Status != "" {
		cur.Status = d.Status
	}
	cur.UpdatedAt = Now()
	r, _ := getDoc[dramaRec](e, colDramas, cur.ID)
	r.Drama = cur
	r.UpdatedAt = cur.UpdatedAt
	return cur, e.putDoc(colDramas, cur.ID, r)
}

// DeleteDrama 给剧、它的集、角色、场景和道具标上删除时间，不抹掉文档。
func (e *Engine) DeleteDrama(id string) error {
	now := Now()
	r, err := getDoc[dramaRec](e, colDramas, id)
	if err != nil {
		return err
	}
	r.DeletedAt = now
	r.UpdatedAt = now
	if err := e.putDoc(colDramas, id, r); err != nil {
		return err
	}
	for _, ep := range loadCol[episodeRec](e, colEpisodes) {
		if ep.DramaID == id && ep.DeletedAt == "" {
			ep.DeletedAt = now
			ep.UpdatedAt = now
			_ = e.putDoc(colEpisodes, ep.ID, ep)
		}
	}
	for _, c := range loadCol[characterRec](e, colCharacters) {
		if c.DramaID == id && c.DeletedAt == "" {
			c.DeletedAt = now
			c.UpdatedAt = now
			_ = e.putDoc(colCharacters, c.ID, c)
		}
	}
	for _, s := range loadCol[sceneRec](e, colScenes) {
		if s.DramaID == id && s.DeletedAt == "" {
			s.DeletedAt = now
			s.UpdatedAt = now
			_ = e.putDoc(colScenes, s.ID, s)
		}
	}
	for _, p := range loadCol[propRec](e, colProps) {
		if p.DramaID == id && p.DeletedAt == "" {
			p.DeletedAt = now
			p.UpdatedAt = now
			_ = e.putDoc(colProps, p.ID, p)
		}
	}
	return nil
}

// ListEpisodes 按集号列出这部剧里没删除的集。没有时返回空切片。
func (e *Engine) ListEpisodes(dramaID string) ([]Episode, error) {
	var out []Episode
	for _, rec := range loadCol[episodeRec](e, colEpisodes) {
		if rec.DramaID == dramaID && rec.DeletedAt == "" {
			out = append(out, rec.Episode)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EpisodeNumber < out[j].EpisodeNumber })
	if out == nil {
		out = []Episode{}
	}
	return out, nil
}

// CreateEpisode 在这部剧下新建一集。集号传 0，由 createEpisode 排到现有集数后面。
func (e *Engine) CreateEpisode(dramaID, title, content string) (Episode, error) {
	return e.createEpisode(dramaID, title, content, 0, nil, nil, nil)
}

// createEpisode 新建一集，并带上当前启用的图像、视频和语音提供商。分辨率先写成 720p。
// 剧必须还在。集号小于等于 0 时排到现有最大集号后面。没传入关联时，从分集方案和最近一集抄角色、场景和道具。
func (e *Engine) createEpisode(dramaID, title, content string, number int, chars, scenes, props []string) (Episode, error) {
	d, err := e.GetDrama(dramaID)
	if err != nil {
		return Episode{}, err
	}
	n := 0
	for _, rec := range loadCol[episodeRec](e, colEpisodes) {
		if rec.DramaID == dramaID && rec.EpisodeNumber > n {
			n = rec.EpisodeNumber
		}
	}
	if number <= 0 {
		number = n + 1
	}
	img, _ := e.ActiveProvider("image", "")
	vid, _ := e.ActiveProvider("video", "")
	tts, _ := e.ActiveProvider("tts", "")
	now := Now()
	ep := Episode{
		ID: NewID(), DramaID: dramaID, EpisodeNumber: number, Title: strings.TrimSpace(title),
		Content: content, Status: "draft",
		ImageProviderID: img.ID, VideoProviderID: vid.ID, TTSProviderID: tts.ID,
		ImageModel: img.Model, VideoModel: vid.Model, TTSModel: tts.Model,
		Resolution: "720p", Pipeline: "{}", CreatedAt: now, UpdatedAt: now,
	}
	if chars == nil && scenes == nil && props == nil {
		chars, scenes, props = e.inheritLinks(dramaID, number)
	}
	_ = d
	return ep, e.putDoc(colEpisodes, ep.ID, episodeRec{
		Episode: ep, CharacterIDs: chars, SceneIDs: scenes, PropIDs: props,
	})
}

// inheritLinks 给新集抄关联。方案里这一集写了角色就用方案的角色。场景、道具，以及方案没写的角色，抄最近一集里有关联的那一集。
func (e *Engine) inheritLinks(dramaID string, episodeNumber int) (chars, scenes, props []string) {
	if plan, err := e.GetPlan(dramaID); err == nil {
		for _, pe := range plan.Episodes {
			if pe.N == episodeNumber && len(pe.CharacterIDs) > 0 {
				chars = append([]string{}, pe.CharacterIDs...)
				break
			}
		}
	}
	var at string
	var hit episodeRec
	for _, rec := range loadCol[episodeRec](e, colEpisodes) {
		if rec.DramaID != dramaID || rec.DeletedAt != "" {
			continue
		}
		if len(rec.CharacterIDs)+len(rec.SceneIDs)+len(rec.PropIDs) == 0 {
			continue
		}
		if rec.UpdatedAt > at {
			at = rec.UpdatedAt
			hit = rec
		}
	}
	if len(chars) == 0 {
		chars = append([]string{}, hit.CharacterIDs...)
	}
	scenes = append([]string{}, hit.SceneIDs...)
	props = append([]string{}, hit.PropIDs...)
	return
}

// GetEpisode 读取一集。没有或已删除时返回 episode not found。
func (e *Engine) GetEpisode(id string) (Episode, error) {
	rec, err := getDoc[episodeRec](e, colEpisodes, id)
	if err != nil || rec.DeletedAt != "" {
		return Episode{}, fmt.Errorf("episode not found")
	}
	return rec.Episode, nil
}

// UpdateEpisode 保存一集。标题空着就保留。正文和剧本按传入的整段替换，空字符串也会清掉。其余字段空着就保留。
func (e *Engine) UpdateEpisode(ep Episode) (Episode, error) {
	cur, err := e.GetEpisode(ep.ID)
	if err != nil {
		return ep, err
	}
	if strings.TrimSpace(ep.Title) != "" {
		cur.Title = ep.Title
	}
	cur.Content = ep.Content
	cur.ScriptContent = ep.ScriptContent
	if ep.Resolution != "" {
		cur.Resolution = ep.Resolution
	}
	if ep.ImageProviderID != "" {
		cur.ImageProviderID = ep.ImageProviderID
	}
	if ep.VideoProviderID != "" {
		cur.VideoProviderID = ep.VideoProviderID
	}
	if ep.TTSProviderID != "" {
		cur.TTSProviderID = ep.TTSProviderID
	}
	if ep.ImageModel != "" {
		cur.ImageModel = ep.ImageModel
	}
	if ep.VideoModel != "" {
		cur.VideoModel = ep.VideoModel
	}
	if ep.TTSModel != "" {
		cur.TTSModel = ep.TTSModel
	}
	if ep.Status != "" {
		cur.Status = ep.Status
	}
	if ep.VideoHash != "" {
		cur.VideoHash = ep.VideoHash
	}
	if ep.PosterHash != "" {
		cur.PosterHash = ep.PosterHash
	}
	return e.saveEpisode(cur)
}

// PatchEpisode 只改参数里出现的键。换了提供商又没带模型时，模型改成该提供商当前的模型。
func (e *Engine) PatchEpisode(id string, fields map[string]any) (Episode, error) {
	cur, err := e.GetEpisode(id)
	if err != nil {
		return cur, err
	}
	has := func(k string) bool { _, ok := fields[k]; return ok }
	if has("title") {
		cur.Title = strings.TrimSpace(strMap(fields, "title"))
	}
	if has("content") {
		cur.Content = strMap(fields, "content")
	}
	if has("script_content") {
		cur.ScriptContent = strMap(fields, "script_content")
	}
	if has("status") {
		if v := strMap(fields, "status"); v != "" {
			cur.Status = v
		}
	}
	if has("resolution") {
		if v := strMap(fields, "resolution"); v != "" {
			cur.Resolution = v
		}
	}
	if has("image_provider_id") {
		cur.ImageProviderID = strMap(fields, "image_provider_id")
		if !has("image_model") {
			if p, err := e.GetProvider(cur.ImageProviderID); err == nil {
				cur.ImageModel = p.Model
			}
		}
	}
	if has("image_model") {
		cur.ImageModel = strMap(fields, "image_model")
	}
	if has("video_provider_id") {
		cur.VideoProviderID = strMap(fields, "video_provider_id")
		if !has("video_model") {
			if p, err := e.GetProvider(cur.VideoProviderID); err == nil {
				cur.VideoModel = p.Model
			}
		}
	}
	if has("video_model") {
		cur.VideoModel = strMap(fields, "video_model")
	}
	if has("tts_provider_id") {
		cur.TTSProviderID = strMap(fields, "tts_provider_id")
		if !has("tts_model") {
			if p, err := e.GetProvider(cur.TTSProviderID); err == nil {
				cur.TTSModel = p.Model
			}
		}
	}
	if has("tts_model") {
		cur.TTSModel = strMap(fields, "tts_model")
	}
	return e.saveEpisode(cur)
}

// saveEpisode 写回一集并刷新更新时间。文档不存在时按这一集新建一条记录。
func (e *Engine) saveEpisode(cur Episode) (Episode, error) {
	rec, err := getDoc[episodeRec](e, colEpisodes, cur.ID)
	if err != nil {
		if err == filestore.ErrNotFound {
			rec = episodeRec{Episode: cur}
		} else {
			return cur, err
		}
	}
	cur.UpdatedAt = Now()
	rec.Episode = cur
	rec.UpdatedAt = cur.UpdatedAt
	return cur, e.putDoc(colEpisodes, cur.ID, rec)
}

// DeleteEpisode 给这一集标上删除时间，不抹掉文档。
func (e *Engine) DeleteEpisode(id string) error {
	rec, err := getDoc[episodeRec](e, colEpisodes, id)
	if err != nil {
		return err
	}
	now := Now()
	rec.DeletedAt = now
	rec.UpdatedAt = now
	return e.putDoc(colEpisodes, id, rec)
}

// SaveScript 写入剧本，并把改写阶段标成完成。
func (e *Engine) SaveScript(episodeID, script string) error {
	rec, err := getDoc[episodeRec](e, colEpisodes, episodeID)
	if err != nil {
		return err
	}
	rec.ScriptContent = script
	rec.UpdatedAt = Now()
	if err := e.putDoc(colEpisodes, rec.ID, rec); err != nil {
		return err
	}
	return e.patchPipeline(episodeID, "rewrite", "done", "")
}

// SkipRewrite 把原文抄成剧本。原文是空的就拒绝。
func (e *Engine) SkipRewrite(episodeID string) error {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return err
	}
	src := strings.TrimSpace(ep.Content)
	if src == "" {
		return fmt.Errorf("source is empty")
	}
	return e.SaveScript(episodeID, src)
}

// hydrateEpisodeProviders 按当前启用的图像、视频和语音提供商补编号和模型。有变化才写回。
func (e *Engine) hydrateEpisodeProviders(ep *Episode) {
	fields := map[string]any{}
	fill := func(kind, id, model string) (string, string, bool) {
		p, err := e.ActiveProvider(kind, id)
		if err != nil || p.ID == "" {
			return id, model, false
		}
		nextModel := first(strings.TrimSpace(model), p.Model)
		changed := providerIDFromChannel(id) != p.ID || strings.TrimSpace(model) != nextModel
		return p.ID, nextModel, changed
	}
	if id, model, ok := fill("image", ep.ImageProviderID, ep.ImageModel); ok {
		fields["image_provider_id"] = id
		fields["image_model"] = model
	}
	if id, model, ok := fill("video", ep.VideoProviderID, ep.VideoModel); ok {
		fields["video_provider_id"] = id
		fields["video_model"] = model
	}
	if id, model, ok := fill("tts", ep.TTSProviderID, ep.TTSModel); ok {
		fields["tts_provider_id"] = id
		fields["tts_model"] = model
	}
	if len(fields) == 0 {
		return
	}
	if next, err := e.PatchEpisode(ep.ID, fields); err == nil {
		*ep = next
	}
}

// Bundle 拼出这一集的剧、集、角色、场景、道具、镜头和任务，并按剧本或原文估时长。剧本空着就用原文。
func (e *Engine) Bundle(episodeID string) (EpisodeBundle, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return EpisodeBundle{}, err
	}
	e.hydrateEpisodeProviders(&ep)
	d, err := e.GetDrama(ep.DramaID)
	if err != nil {
		return EpisodeBundle{}, err
	}
	chars, _ := e.EpisodeCharacters(episodeID)
	scenes, _ := e.EpisodeScenes(episodeID)
	props, _ := e.EpisodeProps(episodeID)
	shots, _ := e.ListShots(episodeID)
	jobs, _ := e.ListJobs(episodeID)
	src := ep.ScriptContent
	if src == "" {
		src = ep.Content
	}
	return EpisodeBundle{
		Drama: d, Episode: ep, Characters: chars, Scenes: scenes, Props: props, Shots: shots, Jobs: jobs,
		Plan: PlanDuration(src),
	}, nil
}
