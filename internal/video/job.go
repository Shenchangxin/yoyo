package video

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const maxJobWorkers = 8

func (e *Engine) GetJob(id string) (Job, error) {
	return e.scanJob(e.DB.QueryRow(`SELECT id, type, status, provider, model, vault_key, remote_id, prompt, params, result_hash, poster_hash, error, drama_id, episode_id, storyboard_id, character_id, scene_id, prop_id, created_at, updated_at, completed_at FROM jobs WHERE id = ?`, id))
}

func (e *Engine) ListJobs(episodeID string) ([]Job, error) {
	q := `SELECT id, type, status, provider, model, vault_key, remote_id, prompt, params, result_hash, poster_hash, error, drama_id, episode_id, storyboard_id, character_id, scene_id, prop_id, created_at, updated_at, completed_at FROM jobs`
	var args []any
	if episodeID != "" {
		q += ` WHERE episode_id = ?`
		args = append(args, episodeID)
	}
	q += ` ORDER BY created_at DESC LIMIT 200`
	rows, err := e.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Job
	for rows.Next() {
		j, err := scanJobRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	if out == nil {
		out = []Job{}
	}
	return out, nil
}

func (e *Engine) scanJob(row *sql.Row) (Job, error) {
	var j Job
	err := row.Scan(&j.ID, &j.Type, &j.Status, &j.Provider, &j.Model, &j.VaultKey, &j.RemoteID, &j.Prompt, &j.Params, &j.ResultHash, &j.PosterHash, &j.Error, &j.DramaID, &j.EpisodeID, &j.StoryboardID, &j.CharacterID, &j.SceneID, &j.PropID, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt)
	return j, err
}

func scanJobRows(rows *sql.Rows) (Job, error) {
	var j Job
	err := rows.Scan(&j.ID, &j.Type, &j.Status, &j.Provider, &j.Model, &j.VaultKey, &j.RemoteID, &j.Prompt, &j.Params, &j.ResultHash, &j.PosterHash, &j.Error, &j.DramaID, &j.EpisodeID, &j.StoryboardID, &j.CharacterID, &j.SceneID, &j.PropID, &j.CreatedAt, &j.UpdatedAt, &j.CompletedAt)
	return j, err
}

func (e *Engine) insertJob(j Job) error {
	_, err := e.DB.Exec(`INSERT INTO jobs(id, type, status, provider, model, vault_key, remote_id, prompt, params, result_hash, poster_hash, error, drama_id, episode_id, storyboard_id, character_id, scene_id, prop_id, created_at, updated_at, completed_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		j.ID, j.Type, j.Status, j.Provider, j.Model, j.VaultKey, j.RemoteID, j.Prompt, j.Params, j.ResultHash, j.PosterHash, j.Error, j.DramaID, j.EpisodeID, j.StoryboardID, j.CharacterID, j.SceneID, j.PropID, j.CreatedAt, j.UpdatedAt, j.CompletedAt)
	return err
}

func (e *Engine) saveJob(j Job) error {
	j.UpdatedAt = Now()
	_, err := e.DB.Exec(`UPDATE jobs SET status=?, remote_id=?, prompt=?, params=?, result_hash=?, poster_hash=?, error=?, completed_at=?, updated_at=? WHERE id=?`,
		j.Status, j.RemoteID, j.Prompt, j.Params, j.ResultHash, j.PosterHash, j.Error, j.CompletedAt, j.UpdatedAt, j.ID)
	e.emit(j)
	return err
}

func (e *Engine) EnqueueImage(in EnqueueImage) (Job, error) {
	p, err := e.ActiveProvider("image", in.ProviderID)
	if err != nil {
		return Job{}, err
	}
	now := Now()
	j := Job{
		ID: NewID(), Type: "image", Status: "queued",
		Provider: p.Provider, Model: ResolveProviderModel(p, in.Model), VaultKey: p.VaultKey,
		Prompt: in.Prompt, Params: marshalJSON(JobParams{Size: first(in.Size, "1920x1080"), AspectRatio: in.AspectRatio, ReferenceImages: in.Refs}),
		DramaID: in.DramaID, EpisodeID: in.EpisodeID, CharacterID: in.CharacterID, SceneID: in.SceneID, PropID: in.PropID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := e.insertJob(j); err != nil {
		return j, err
	}
	e.emit(j)
	return j, nil
}

func (e *Engine) EnqueueVideo(in EnqueueVideo) (Job, error) {
	p, err := e.ActiveProvider("video", in.ProviderID)
	if err != nil {
		return Job{}, err
	}
	now := Now()
	dur := ClampProviderDuration(in.Duration, p.Provider)
	j := Job{
		ID: NewID(), Type: "video", Status: "queued",
		Provider: p.Provider, Model: ResolveProviderModel(p, in.Model), VaultKey: p.VaultKey,
		Prompt: in.Prompt,
		Params: marshalJSON(JobParams{
			Duration: dur, AspectRatio: first(in.AspectRatio, "16:9"),
			Resolution:    NormalizeVideoResolution(in.Resolution, p.Provider),
			GenerateAudio: in.Audio, ReferenceImageURLs: in.Refs,
		}),
		DramaID: in.DramaID, EpisodeID: in.EpisodeID, StoryboardID: in.ShotID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := e.insertJob(j); err != nil {
		return j, err
	}
	e.emit(j)
	return j, nil
}

func (e *Engine) EnqueueMerge(episodeID, dramaID string, paths []string, shotIDs []string) (Job, error) {
	now := Now()
	j := Job{
		ID: NewID(), Type: "merge", Status: "queued", Provider: "ffmpeg", Model: "concat-h264",
		Params:  marshalJSON(JobParams{ShotIDs: shotIDs, ReferenceVideoURLs: paths}),
		DramaID: dramaID, EpisodeID: episodeID, CreatedAt: now, UpdatedAt: now,
	}
	if err := e.insertJob(j); err != nil {
		return j, err
	}
	e.emit(j)
	_ = e.patchPipeline(episodeID, "merge", "running", "")
	return j, nil
}

func (e *Engine) CancelJob(id string) error {
	j, err := e.GetJob(id)
	if err != nil {
		return err
	}
	if j.Status == "succeeded" {
		return fmt.Errorf("already finished")
	}
	j.Status = "cancelled"
	j.CompletedAt = Now()
	if v, ok := e.inflight.Load(id); ok {
		if cancel, ok := v.(context.CancelFunc); ok {
			cancel()
		}
	}
	return e.saveJob(j)
}

func (e *Engine) RetryJob(id string) (Job, error) {
	j, err := e.GetJob(id)
	if err != nil {
		return j, err
	}
	j.Status = "queued"
	j.Error = ""
	j.RemoteID = ""
	j.CompletedAt = ""
	if err := e.saveJob(j); err != nil {
		return j, err
	}
	return j, nil
}

func (e *Engine) resumeJobs() {
	rows, err := e.DB.Query(`SELECT id FROM jobs WHERE status IN ('running','polling')`)
	if err != nil {
		return
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		j, err := e.GetJob(id)
		if err != nil {
			continue
		}
		if j.RemoteID != "" {
			j.Status = "polling"
		} else {
			j.Status = "queued"
		}
		_ = e.saveJob(j)
	}
}

func (e *Engine) loop(ctx context.Context) {
	t := time.NewTicker(800 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			e.pump(ctx)
		}
	}
}

func (e *Engine) inflightN() int {
	n := 0
	e.inflight.Range(func(_, _ any) bool {
		n++
		return true
	})
	return n
}

func (e *Engine) pump(ctx context.Context) {
	slots := maxJobWorkers - e.inflightN()
	if slots <= 0 {
		return
	}
	rows, err := e.DB.Query(`SELECT id FROM jobs WHERE status IN ('queued','polling') ORDER BY created_at ASC LIMIT ?`, slots)
	if err != nil {
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		jobCtx, cancel := context.WithTimeout(ctx, 12*time.Minute)
		if _, loaded := e.inflight.LoadOrStore(id, cancel); loaded {
			cancel()
			continue
		}
		go e.runJob(jobCtx, id, cancel)
	}
}

func (e *Engine) runJob(ctx context.Context, id string, cancel context.CancelFunc) {
	defer func() {
		cancel()
		e.inflight.Delete(id)
	}()
	j, err := e.GetJob(id)
	if err != nil || j.Status == "cancelled" {
		return
	}
	switch j.Status {
	case "queued":
		_ = e.runSubmit(ctx, j)
	case "polling":
		_ = e.runPoll(ctx, j)
	}
}

func (e *Engine) abandoned(id string) bool {
	j, err := e.GetJob(id)
	return err != nil || j.Status == "cancelled"
}

func (e *Engine) runSubmit(ctx context.Context, j Job) error {
	if ctx.Err() != nil || e.abandoned(j.ID) {
		return nil
	}
	j.Status = "running"
	_ = e.saveJob(j)
	var err error
	switch j.Type {
	case "image":
		err = e.submitImage(ctx, &j)
	case "video":
		err = e.submitVideo(ctx, &j)
	case "merge":
		err = e.runMerge(&j)
	default:
		err = fmt.Errorf("unknown job type")
	}
	if e.abandoned(j.ID) {
		return nil
	}
	if err != nil {
		j.Status = "failed"
		j.Error = err.Error()
		j.CompletedAt = Now()
		return e.saveJob(j)
	}
	return e.saveJob(j)
}

func (e *Engine) runPoll(ctx context.Context, j Job) error {
	if ctx.Err() != nil || e.abandoned(j.ID) {
		return nil
	}
	var err error
	switch j.Type {
	case "image":
		err = e.pollImage(ctx, &j)
	case "video":
		err = e.pollVideo(ctx, &j)
	default:
		j.Status = "failed"
		j.Error = "nothing to poll"
	}
	if e.abandoned(j.ID) {
		return nil
	}
	if err != nil {
		j.Status = "failed"
		j.Error = err.Error()
		j.CompletedAt = Now()
	}
	return e.saveJob(j)
}

func (e *Engine) finishMedia(j *Job, raw []byte, video bool) error {
	h, err := e.PutBytes(raw)
	if err != nil {
		return err
	}
	j.ResultHash = h
	if video {
		path := e.FilePath(h)
		if b, err := e.Poster(path); err == nil {
			if ph, err := e.PutBytes(b); err == nil {
				j.PosterHash = ph
			}
		}
	} else if th, err := e.Thumb(h); err == nil {
		j.PosterHash = th
	}
	j.Status = "succeeded"
	j.CompletedAt = Now()
	return e.writeBack(*j)
}

func (e *Engine) runMerge(j *Job) error {
	var p JobParams
	_ = json.Unmarshal([]byte(j.Params), &p)
	var files []string
	for _, h := range p.ReferenceVideoURLs {
		if strings.HasPrefix(h, "/") || len(h) > 8 && (h[1] == ':' || strings.Contains(h, `\`)) {
			files = append(files, h)
			continue
		}
		fp := e.FilePath(h)
		if fp == "" {
			return fmt.Errorf("clip missing")
		}
		if _, err := os.Stat(fp); err != nil {
			return fmt.Errorf("clip file missing")
		}
		files = append(files, fp)
	}
	if len(files) == 0 {
		return fmt.Errorf("no clips to stitch")
	}
	raw, err := e.Concat(files)
	if err != nil {
		return err
	}
	if err := e.finishMedia(j, raw, true); err != nil {
		return err
	}
	if j.EpisodeID != "" {
		_, _ = e.DB.Exec(`UPDATE episodes SET video_hash = ?, poster_hash = ?, updated_at = ? WHERE id = ?`, j.ResultHash, j.PosterHash, Now(), j.EpisodeID)
		_ = e.patchPipeline(j.EpisodeID, "merge", "done", "")
	}
	return nil
}

func first(v, fb string) string {
	if strings.TrimSpace(v) == "" {
		return fb
	}
	return v
}

func parseParams(raw string) JobParams {
	var p JobParams
	_ = json.Unmarshal([]byte(raw), &p)
	return p
}
