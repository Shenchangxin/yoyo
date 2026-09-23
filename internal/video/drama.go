package video

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (e *Engine) writeBack(j Job) error {
	now := Now()
	var err error
	switch {
	case j.CharacterID != "":
		_, err = e.DB.Exec(`UPDATE characters SET image_hash = ?, updated_at = ? WHERE id = ?`, j.ResultHash, now, j.CharacterID)
	case j.SceneID != "":
		_, err = e.DB.Exec(`UPDATE scenes SET image_hash = ?, updated_at = ? WHERE id = ?`, j.ResultHash, now, j.SceneID)
	case j.PropID != "":
		_, err = e.DB.Exec(`UPDATE props SET image_hash = ?, updated_at = ? WHERE id = ?`, j.ResultHash, now, j.PropID)
	case j.StoryboardID != "" && j.Type == "video":
		_, err = e.DB.Exec(`UPDATE storyboards SET video_hash = ?, poster_hash = ?, status = 'ready', updated_at = ? WHERE id = ?`, j.ResultHash, j.PosterHash, now, j.StoryboardID)
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
	var raw string
	_ = e.DB.QueryRow(`SELECT pipeline FROM episodes WHERE id = ?`, episodeID).Scan(&raw)
	m := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &m)
	m[stage] = map[string]any{"status": status, "error": errMsg, "updated_at": Now()}
	_, err := e.DB.Exec(`UPDATE episodes SET pipeline = ?, updated_at = ? WHERE id = ?`, marshalJSON(m), Now(), episodeID)
	return err
}

func (e *Engine) ListDramas() ([]Drama, error) {
	rows, err := e.DB.Query(`SELECT d.id, d.title, d.description, d.genre, d.style, d.aspect_ratio, d.status, d.thumbnail_hash, d.created_at, d.updated_at,
		(SELECT COUNT(*) FROM episodes e WHERE e.drama_id = d.id AND e.deleted_at = '') FROM dramas d WHERE d.deleted_at = '' ORDER BY d.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Drama
	for rows.Next() {
		var d Drama
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Genre, &d.Style, &d.AspectRatio, &d.Status, &d.ThumbnailHash, &d.CreatedAt, &d.UpdatedAt, &d.EpisodeCount); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
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
	if d.Title == "" {
		d.Title = "Untitled"
	}
	_, err := e.DB.Exec(`INSERT INTO dramas(id, title, description, genre, style, aspect_ratio, status, thumbnail_hash, created_at, updated_at, deleted_at)
		VALUES(?,?,?,?,?,?,?,?,?,?, '')`, d.ID, d.Title, d.Description, d.Genre, d.Style, d.AspectRatio, d.Status, d.ThumbnailHash, d.CreatedAt, d.UpdatedAt)
	return d, err
}

func (e *Engine) GetDrama(id string) (Drama, error) {
	var d Drama
	err := e.DB.QueryRow(`SELECT id, title, description, genre, style, aspect_ratio, status, thumbnail_hash, created_at, updated_at FROM dramas WHERE id = ? AND deleted_at = ''`, id).
		Scan(&d.ID, &d.Title, &d.Description, &d.Genre, &d.Style, &d.AspectRatio, &d.Status, &d.ThumbnailHash, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return d, fmt.Errorf("drama not found")
	}
	_ = e.DB.QueryRow(`SELECT COUNT(*) FROM episodes WHERE drama_id = ? AND deleted_at = ''`, id).Scan(&d.EpisodeCount)
	return d, nil
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
	_, err = e.DB.Exec(`UPDATE dramas SET title=?, description=?, genre=?, style=?, aspect_ratio=?, status=?, updated_at=? WHERE id=?`,
		cur.Title, cur.Description, cur.Genre, cur.Style, cur.AspectRatio, cur.Status, cur.UpdatedAt, cur.ID)
	return cur, err
}

func (e *Engine) DeleteDrama(id string) error {
	now := Now()
	_, err := e.DB.Exec(`UPDATE dramas SET deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	if err != nil {
		return err
	}
	_, _ = e.DB.Exec(`UPDATE episodes SET deleted_at = ?, updated_at = ? WHERE drama_id = ? AND deleted_at = ''`, now, now, id)
	_, _ = e.DB.Exec(`UPDATE characters SET deleted_at = ?, updated_at = ? WHERE drama_id = ? AND deleted_at = ''`, now, now, id)
	_, _ = e.DB.Exec(`UPDATE scenes SET deleted_at = ?, updated_at = ? WHERE drama_id = ? AND deleted_at = ''`, now, now, id)
	_, _ = e.DB.Exec(`UPDATE props SET deleted_at = ?, updated_at = ? WHERE drama_id = ? AND deleted_at = ''`, now, now, id)
	return nil
}

func (e *Engine) ListEpisodes(dramaID string) ([]Episode, error) {
	rows, err := e.DB.Query(`SELECT `+episodeCols+` FROM episodes WHERE drama_id = ? AND deleted_at = '' ORDER BY episode_number`, dramaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Episode
	for rows.Next() {
		ep, err := scanEpisode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ep)
	}
	if out == nil {
		out = []Episode{}
	}
	return out, nil
}

const episodeCols = `id, drama_id, episode_number, title, content, script_content, status, video_hash, poster_hash, image_provider_id, video_provider_id, image_model, video_model, tts_provider_id, tts_model, resolution, pipeline, created_at, updated_at`

func scanEpisode(rows interface {
	Scan(dest ...any) error
}) (Episode, error) {
	var ep Episode
	err := rows.Scan(&ep.ID, &ep.DramaID, &ep.EpisodeNumber, &ep.Title, &ep.Content, &ep.ScriptContent, &ep.Status, &ep.VideoHash, &ep.PosterHash, &ep.ImageProviderID, &ep.VideoProviderID, &ep.ImageModel, &ep.VideoModel, &ep.TTSProviderID, &ep.TTSModel, &ep.Resolution, &ep.Pipeline, &ep.CreatedAt, &ep.UpdatedAt)
	return ep, err
}

func (e *Engine) CreateEpisode(dramaID, title, content string) (Episode, error) {
	d, err := e.GetDrama(dramaID)
	if err != nil {
		return Episode{}, err
	}
	var n int
	_ = e.DB.QueryRow(`SELECT COALESCE(MAX(episode_number),0) FROM episodes WHERE drama_id = ?`, dramaID).Scan(&n)
	img, _ := e.ActiveProvider("image", "")
	vid, _ := e.ActiveProvider("video", "")
	tts, _ := e.ActiveProvider("tts", "")
	now := Now()
	ep := Episode{
		ID: NewID(), DramaID: dramaID, EpisodeNumber: n + 1, Title: first(title, fmt.Sprintf("Episode %d", n+1)),
		Content: content, Status: "draft",
		ImageProviderID: img.ID, VideoProviderID: vid.ID, TTSProviderID: tts.ID,
		ImageModel: img.Model, VideoModel: vid.Model, TTSModel: tts.Model,
		Resolution: "720p", Pipeline: "{}", CreatedAt: now, UpdatedAt: now,
	}
	_, err = e.DB.Exec(`INSERT INTO episodes(id, drama_id, episode_number, title, content, script_content, status, video_hash, poster_hash, image_provider_id, video_provider_id, image_model, video_model, tts_provider_id, tts_model, resolution, pipeline, created_at, updated_at, deleted_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'')`,
		ep.ID, ep.DramaID, ep.EpisodeNumber, ep.Title, ep.Content, ep.ScriptContent, ep.Status, ep.VideoHash, ep.PosterHash, ep.ImageProviderID, ep.VideoProviderID, ep.ImageModel, ep.VideoModel, ep.TTSProviderID, ep.TTSModel, ep.Resolution, ep.Pipeline, ep.CreatedAt, ep.UpdatedAt)
	_ = d
	return ep, err
}

func (e *Engine) GetEpisode(id string) (Episode, error) {
	row := e.DB.QueryRow(`SELECT `+episodeCols+` FROM episodes WHERE id = ? AND deleted_at = ''`, id)
	ep, err := scanEpisode(row)
	if err != nil {
		return ep, fmt.Errorf("episode not found")
	}
	return ep, nil
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
	return e.saveEpisode(cur)
}

func (e *Engine) PatchEpisode(id string, fields map[string]any) (Episode, error) {
	cur, err := e.GetEpisode(id)
	if err != nil {
		return cur, err
	}
	has := func(k string) bool { _, ok := fields[k]; return ok }
	if has("title") {
		if v := strings.TrimSpace(strMap(fields, "title")); v != "" {
			cur.Title = v
		}
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
	cur.UpdatedAt = Now()
	_, err := e.DB.Exec(`UPDATE episodes SET title=?, content=?, script_content=?, status=?, resolution=?, image_provider_id=?, video_provider_id=?, image_model=?, video_model=?, tts_provider_id=?, tts_model=?, updated_at=? WHERE id=?`,
		cur.Title, cur.Content, cur.ScriptContent, cur.Status, cur.Resolution, cur.ImageProviderID, cur.VideoProviderID, cur.ImageModel, cur.VideoModel, cur.TTSProviderID, cur.TTSModel, cur.UpdatedAt, cur.ID)
	return cur, err
}

func (e *Engine) DeleteEpisode(id string) error {
	now := Now()
	_, err := e.DB.Exec(`UPDATE episodes SET deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	return err
}

func (e *Engine) SaveScript(episodeID, script string) error {
	_, err := e.DB.Exec(`UPDATE episodes SET script_content = ?, updated_at = ? WHERE id = ?`, script, Now(), episodeID)
	if err != nil {
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
		changed := false
		if strings.TrimSpace(id) == "" {
			if p, err := e.ActiveProvider(kind, ""); err == nil && p.ID != "" {
				return p.ID, first(model, p.Model), true
			}
			return id, model, false
		}
		if strings.TrimSpace(model) == "" {
			if p, err := e.GetProvider(id); err == nil {
				return id, p.Model, p.Model != ""
			}
		}
		return id, model, changed
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
