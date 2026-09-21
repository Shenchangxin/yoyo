package video

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (e *Engine) writeBack(j Job) error {
	now := Now()
	switch {
	case j.CharacterID != "":
		_, err := e.DB.Exec(`UPDATE characters SET image_hash = ?, updated_at = ? WHERE id = ?`, j.ResultHash, now, j.CharacterID)
		return err
	case j.SceneID != "":
		_, err := e.DB.Exec(`UPDATE scenes SET image_hash = ?, updated_at = ? WHERE id = ?`, j.ResultHash, now, j.SceneID)
		return err
	case j.PropID != "":
		_, err := e.DB.Exec(`UPDATE props SET image_hash = ?, updated_at = ? WHERE id = ?`, j.ResultHash, now, j.PropID)
		return err
	case j.StoryboardID != "" && j.Type == "video":
		_, err := e.DB.Exec(`UPDATE storyboards SET video_hash = ?, poster_hash = ?, status = 'ready', updated_at = ? WHERE id = ?`, j.ResultHash, j.PosterHash, now, j.StoryboardID)
		return err
	}
	return nil
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
	if d.AspectRatio != "9:16" {
		d.AspectRatio = "16:9"
	}
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
	cur.UpdatedAt = Now()
	_, err = e.DB.Exec(`UPDATE dramas SET title=?, description=?, genre=?, style=?, status=?, updated_at=? WHERE id=?`,
		cur.Title, cur.Description, cur.Genre, cur.Style, cur.Status, cur.UpdatedAt, cur.ID)
	return cur, err
}

func (e *Engine) DeleteDrama(id string) error {
	now := Now()
	_, err := e.DB.Exec(`UPDATE dramas SET deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	return err
}

func (e *Engine) ListEpisodes(dramaID string) ([]Episode, error) {
	rows, err := e.DB.Query(`SELECT id, drama_id, episode_number, title, content, script_content, status, video_hash, poster_hash, image_provider_id, video_provider_id, resolution, pipeline, created_at, updated_at FROM episodes WHERE drama_id = ? AND deleted_at = '' ORDER BY episode_number`, dramaID)
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

func scanEpisode(rows interface {
	Scan(dest ...any) error
}) (Episode, error) {
	var ep Episode
	err := rows.Scan(&ep.ID, &ep.DramaID, &ep.EpisodeNumber, &ep.Title, &ep.Content, &ep.ScriptContent, &ep.Status, &ep.VideoHash, &ep.PosterHash, &ep.ImageProviderID, &ep.VideoProviderID, &ep.Resolution, &ep.Pipeline, &ep.CreatedAt, &ep.UpdatedAt)
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
	now := Now()
	ep := Episode{
		ID: NewID(), DramaID: dramaID, EpisodeNumber: n + 1, Title: first(title, fmt.Sprintf("Episode %d", n+1)),
		Content: content, Status: "draft", ImageProviderID: img.ID, VideoProviderID: vid.ID, Resolution: "720p",
		Pipeline: "{}", CreatedAt: now, UpdatedAt: now,
	}
	_, err = e.DB.Exec(`INSERT INTO episodes(id, drama_id, episode_number, title, content, script_content, status, video_hash, poster_hash, image_provider_id, video_provider_id, resolution, pipeline, created_at, updated_at, deleted_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'')`,
		ep.ID, ep.DramaID, ep.EpisodeNumber, ep.Title, ep.Content, ep.ScriptContent, ep.Status, ep.VideoHash, ep.PosterHash, ep.ImageProviderID, ep.VideoProviderID, ep.Resolution, ep.Pipeline, ep.CreatedAt, ep.UpdatedAt)
	_ = d
	return ep, err
}

func (e *Engine) GetEpisode(id string) (Episode, error) {
	row := e.DB.QueryRow(`SELECT id, drama_id, episode_number, title, content, script_content, status, video_hash, poster_hash, image_provider_id, video_provider_id, resolution, pipeline, created_at, updated_at FROM episodes WHERE id = ? AND deleted_at = ''`, id)
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
	if ep.Status != "" {
		cur.Status = ep.Status
	}
	cur.UpdatedAt = Now()
	_, err = e.DB.Exec(`UPDATE episodes SET title=?, content=?, script_content=?, status=?, resolution=?, image_provider_id=?, video_provider_id=?, updated_at=? WHERE id=?`,
		cur.Title, cur.Content, cur.ScriptContent, cur.Status, cur.Resolution, cur.ImageProviderID, cur.VideoProviderID, cur.UpdatedAt, cur.ID)
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

func (e *Engine) Bundle(episodeID string) (EpisodeBundle, error) {
	ep, err := e.GetEpisode(episodeID)
	if err != nil {
		return EpisodeBundle{}, err
	}
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
