package video

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/filestore"
)

func (e *Engine) writeBack(j Job) error {
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

func (e *Engine) GetDrama(id string) (Drama, error) {
	r, err := getDoc[dramaRec](e, colDramas, id)
	if err != nil || r.DeletedAt != "" {
		return Drama{}, fmt.Errorf("drama not found")
	}
	d := r.Drama
	d.EpisodeCount = e.episodeCount(id)
	return d, nil
}

func (e *Engine) episodeCount(dramaID string) int {
	n := 0
	for _, ep := range loadCol[episodeRec](e, colEpisodes) {
		if ep.DramaID == dramaID && ep.DeletedAt == "" {
			n++
		}
	}
	return n
}

func (e *Engine) UpdateDrama(d Drama) (Drama, error) {
	cur, err := e.GetDrama(d.ID)
	if err != nil {
		return d, err
	}
	if d.Title != "" {
		cur.Title = d.Title
	}
	cur.Description = d.Description
	cur.Genre = d.Genre
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

func (e *Engine) CreateEpisode(dramaID, title, content string) (Episode, error) {
	return e.createEpisode(dramaID, title, content, 0, nil, nil, nil)
}

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

func (e *Engine) GetEpisode(id string) (Episode, error) {
	rec, err := getDoc[episodeRec](e, colEpisodes, id)
	if err != nil || rec.DeletedAt != "" {
		return Episode{}, fmt.Errorf("episode not found")
	}
	return rec.Episode, nil
}

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
