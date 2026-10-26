// job.go 排队、取消并推进图像、视频和拼接任务。模型循环仍是 runtime.Run，这里只按定时器领取已排队的任务。

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

// maxJobWorkers 是同时执行的任务上限，8 个。
const maxJobWorkers = 8

// GetJob 按标识读出一条任务。
// id 传给 getDoc。
func (e *Engine) GetJob(id string) (Job, error) {
	j, err := getDoc[Job](e, colJobs, id)
	if err != nil {
		return Job{}, err
	}
	return j, nil
}

// ListJobs 按创建时间从新到旧列出任务。episodeID 非空时只留该集。最多 200 条，没有任务时返回空切片。
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

// ListDramaJobs 列出一部剧的任务，按创建时间从新到旧，最多 200 条。dramaID 为空时改为列出全部任务。
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

// insertJob 把任务写入文档，不改更新时间。
// j 按 ID 传给 putDoc。
func (e *Engine) insertJob(j Job) error {
	return e.putDoc(colJobs, j.ID, j)
}

// saveJob 把 UpdatedAt 设为现在，写入文档，并在 OnJob 非空时交出去。
// j 写上 UpdatedAt 后 putDoc，并传给 emit。
func (e *Engine) saveJob(j Job) error {
	j.UpdatedAt = Now()
	if err := e.putDoc(colJobs, j.ID, j); err != nil {
		return err
	}
	e.emit(j)
	return nil
}

// EnqueueImage 用当前图像供应商排一条 queued 任务。尺寸空着时用 1920x1080。模型用 ResolveProviderModel。写入后 emit。
// in 的 ProviderID 传给 ActiveProvider，Model 传给 ResolveProviderModel，Size 与 1920x1080 传给 first，Prompt、AspectRatio、Refs、StudioProjectID、StudioShotID、StudioKind、DramaID、EpisodeID、CharacterID、SceneID、PropID 写入任务。
func (e *Engine) EnqueueImage(in EnqueueImage) (Job, error) {
	p, err := e.ActiveProvider("image", in.ProviderID)
	if err != nil {
		return Job{}, err
	}
	now := Now()
	j := Job{
		ID: NewID(), Type: "image", Status: "queued",
		Provider: p.Provider, Model: ResolveProviderModel(p, in.Model), VaultKey: p.VaultKey,
		Prompt: in.Prompt, Params: marshalJSON(JobParams{
			Size: first(in.Size, "1920x1080"), AspectRatio: in.AspectRatio, ReferenceImages: in.Refs,
			StudioProjectID: in.StudioProjectID, StudioShotID: in.StudioShotID, StudioKind: in.StudioKind,
		}),
		DramaID: in.DramaID, EpisodeID: in.EpisodeID, CharacterID: in.CharacterID, SceneID: in.SceneID, PropID: in.PropID,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := e.insertJob(j); err != nil {
		return j, err
	}
	e.emit(j)
	return j, nil
}

// EnqueueVideo 用当前视频供应商排一条 queued 任务。时长经 ClampProviderDuration，画幅空着时用 16:9，分辨率按供应商归一。镜头标识写在 StoryboardID。
// in 的 ProviderID 传给 ActiveProvider，Model 传给 ResolveProviderModel，Duration 传给 ClampProviderDuration，AspectRatio 与 16:9 传给 first，Resolution 传给 NormalizeVideoResolution，Prompt、Audio、Refs、DramaID、EpisodeID 写入任务，ShotID 写入 StoryboardID。
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

// EnqueueMerge 排一条 ffmpeg、模型 concat-h264 的拼接任务，并把该集流水线的 merge 标成 running。
// episodeID 写入 EpisodeID，并传给 patchPipeline。
// dramaID 写入 DramaID。
// paths 写入 ReferenceVideoURLs。
// shotIDs 写入 ShotIDs。
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

// CancelJob 把未结束的任务标成 cancelled，并取消正在跑的上下文。succeeded、failed、cancelled 返回 already finished。
// id 传给 GetJob，并用来取出正在执行的取消函数。
func (e *Engine) CancelJob(id string) error {
	j, err := e.GetJob(id)
	if err != nil {
		return err
	}
	if j.Status == "succeeded" || j.Status == "failed" || j.Status == "cancelled" {
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

// RetryJob 只接受 failed 或 cancelled。状态改回 queued，并清掉错误、远程标识和完成时间。
// id 传给 GetJob。
func (e *Engine) RetryJob(id string) (Job, error) {
	j, err := e.GetJob(id)
	if err != nil {
		return j, err
	}
	if j.Status != "failed" && j.Status != "cancelled" {
		return j, fmt.Errorf("only failed or cancelled jobs can be retried")
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

// resumeJobs 在打开时处理 running 和 polling。已有 RemoteID 的改成 polling，没有的改回 queued。
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

// loop 每 800 毫秒调用一次 pump，直到上下文取消。它领取已排队的生成任务。
// ctx 取消时返回，否则传给 pump。
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

// inflightN 统计正在执行的任务数。
func (e *Engine) inflightN() int {
	n := 0
	e.inflight.Range(func(_, _ any) bool {
		n++
		return true
	})
	return n
}

// pump 按创建时间从旧到新领取 queued 和 polling，空位是 maxJobWorkers 减去正在执行的数量。每条任务超时 12 分钟。已经在执行的标识会跳过。
// ctx 加上 12 分钟超时后传给 runJob。
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

// runJob 在结束时取消上下文并移出正在执行的表。queued 走 runSubmit，polling 走 runPoll。读不到或已取消则返回。
// ctx 在状态为 queued 时传给 runSubmit，为 polling 时传给 runPoll。
// id 传给 GetJob，结束时从正在执行的表里删除。
// cancel 在结束时调用。
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

// abandoned 在任务读不到或状态已是 cancelled 时为真。
// id 传给 GetJob。
func (e *Engine) abandoned(id string) bool {
	j, err := e.GetJob(id)
	return err != nil || j.Status == "cancelled"
}

// runSubmit 把任务标成 running。image、video、merge、canvas、canvas_timeline、canvas_transcribe 分别提交；canvas_text 把状态写成 text_replay。未知类型失败。提交前或提交后若已取消则直接返回。失败时记下错误并标记短片任务失败。
// ctx 已取消时直接返回；类型为 image、video、merge、canvas 时传给对应提交函数。
// j 按 Type 提交，失败时把 Status 写成 failed。
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
		err = e.runMerge(ctx, &j)
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
		if saveErr := e.saveJob(j); saveErr != nil {
			return saveErr
		}
		e.markStudioJobFailed(j)
		return err
	}
	return e.saveJob(j)
}

// runPoll 轮询 image、video 和 canvas。其他类型把状态写成 failed，错误是 nothing to poll。已取消则直接返回。失败时记下错误并标记短片任务失败。
// ctx 已取消时直接返回；类型为 image、video、canvas 时传给对应轮询函数。
// j 按 Type 轮询，其他类型把 Status 写成 failed、Error 写成 nothing to poll。
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
		if saveErr := e.saveJob(j); saveErr != nil {
			return saveErr
		}
		e.markStudioJobFailed(j)
		return err
	}
	return e.saveJob(j)
}

// finishMedia 把字节写入 CAS。视频能抽出封面时写入 PosterHash，图像则尝试缩略图。带画布工程或类型为 canvas 时再交给 finishCanvasMedia。成功后状态为 succeeded，并 writeBack。
// j 写入 ResultHash 和 PosterHash，成功后传给 writeBack。
// raw 传给 PutBytes；带画布工程或类型为 canvas 时再传给 finishCanvasMedia。
// video 为真时用成片路径抽封面，否则尝试缩略图。
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

// runMerge 拼接 ReferenceVideoURLs。以 / 开头、像盘符路径或含反斜杠的条目当本地路径，其余当内容寻址哈希。没有片段、哈希没有文件或文件不在磁盘上时失败。成功后把成片哈希写回该集，并把 merge 标成 done。
// ctx 传给 ConcatContext。
// j 的 Params 解出片段，成功后把成片哈希写回 j.EpisodeID 对应的集。
func (e *Engine) runMerge(ctx context.Context, j *Job) error {
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
	raw, err := e.ConcatContext(ctx, files)
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

// first 在 v 去掉空白后为空时返回 fb。
func first(v, fb string) string {
	if strings.TrimSpace(v) == "" {
		return fb
	}
	return v
}

// parseParams 把 JSON 解成 JobParams。解不开时返回零值。
// raw 传给 json.Unmarshal。
func parseParams(raw string) JobParams {
	var p JobParams
	_ = json.Unmarshal([]byte(raw), &p)
	return p
}
