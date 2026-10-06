package video

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const maxJobWorkers = 8

func (e *Engine) GetJob(id string) (Job, error) {
	j, err := getDoc[Job](e, colJobs, id)
	if err != nil {
		return Job{}, err
	}
	return j, nil
}

func (e *Engine) ListJobs(episodeID string) ([]Job, error) {
	all := loadCol[Job](e, colJobs)
	var out []Job
	for _, j := range all {
		if episodeID != "" && j.EpisodeID != episodeID {
			continue
		}
		out = append(out, j)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	if len(out) > 200 {
		out = out[:200]
	}
	if out == nil {
		out = []Job{}
	}
	return out, nil
}

func (e *Engine) ListDramaJobs(dramaID string) ([]Job, error) {
	if dramaID == "" {
		return e.ListJobs("")
	}
	all := loadCol[Job](e, colJobs)
	var out []Job
	for _, j := range all {
		if j.DramaID != dramaID {
			continue
		}
		out = append(out, j)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	if len(out) > 200 {
		out = out[:200]
	}
	if out == nil {
		out = []Job{}
	}
	return out, nil
}

func (e *Engine) insertJob(j Job) error {
	return e.putDoc(colJobs, j.ID, j)
}

func (e *Engine) saveJob(j Job) error {
	j.UpdatedAt = Now()
	if err := e.putDoc(colJobs, j.ID, j); err != nil {
		return err
	}
	e.emit(j)
	return nil
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
	for _, j := range loadCol[Job](e, colJobs) {
		if j.Status != "running" && j.Status != "polling" {
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
	var pending []Job
	for _, j := range loadCol[Job](e, colJobs) {
		if j.Status == "queued" || j.Status == "polling" {
			pending = append(pending, j)
		}
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].CreatedAt < pending[j].CreatedAt })
	if len(pending) > slots {
		pending = pending[:slots]
	}
	for _, j := range pending {
		jobCtx, cancel := context.WithTimeout(ctx, 12*time.Minute)
		if _, loaded := e.inflight.LoadOrStore(j.ID, cancel); loaded {
			cancel()
			continue
		}
		go e.runJob(jobCtx, j.ID, cancel)
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
	case "canvas":
		err = e.submitCanvas(ctx, &j)
	case "canvas_text":
		err = nil
		j.Status = "text_replay"
	case "canvas_timeline":
		err = e.runCanvasTimeline(&j)
	case "canvas_transcribe":
		err = e.runCanvasTranscribe(&j)
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
	case "canvas":
		err = e.pollCanvas(ctx, &j)
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
	if p := parseParams(j.Params); p.CanvasProjectID != "" || j.Type == "canvas" {
		if err := e.finishCanvasMedia(j, raw, video); err != nil {
			return err
		}
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
		if rec, err := getDoc[episodeRec](e, colEpisodes, j.EpisodeID); err == nil {
			rec.VideoHash = j.ResultHash
			rec.PosterHash = j.PosterHash
			rec.UpdatedAt = Now()
			_ = e.putDoc(colEpisodes, rec.ID, rec)
		}
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
