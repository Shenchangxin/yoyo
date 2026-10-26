// 短片镜头的生图、封面、发布文案，以及从网址或长视频里取素材。
// 不合成成片。

package video

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// shotPrompt 拼这一镜的生图提示。优先用画面说明，其次提示、字幕，再其次整段口播。风格、要保住的和不要出现的会包在外面。
func (e *Engine) shotPrompt(p StudioProject, shot StudioShot) string {
	visual := strings.TrimSpace(shot.Visual)
	if visual == "" {
		visual = strings.TrimSpace(shot.Prompt)
	}
	if visual == "" {
		visual = strings.TrimSpace(shot.Caption)
	}
	if visual == "" {
		visual = strings.TrimSpace(p.Script)
	}
	var b strings.Builder
	if p.Style != "" {
		fmt.Fprintf(&b, "Style: %s. ", p.Style)
	}
	if p.Identity.Preserve != "" {
		fmt.Fprintf(&b, "Keep: %s. ", p.Identity.Preserve)
	}
	b.WriteString(visual)
	if p.Identity.Prohibit != "" {
		fmt.Fprintf(&b, " Do not: %s.", p.Identity.Prohibit)
	}
	return strings.TrimSpace(b.String())
}

// GenerateStudioShot 给一镜排队生图。口播还没锁定，或找不到镜头，或提示是空的，就拒绝。镜头编号空着就用第一镜。
// 已有图就拿它做参考，否则用身份上的参考图。排队成功后把任务编号写上，并把这一镜标成过期，等结果回来再清掉。
// id 传给 GetStudioProject。
// shotID 与镜头 ID 比较，空串时匹配第一镜并改用该镜的 ID。
func (e *Engine) GenerateStudioShot(id, shotID string) (Job, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return Job{}, err
	}
	if p.ScriptStatus != ScriptApproved {
		return Job{}, fmt.Errorf("approve the script before generating visuals")
	}
	idx := -1
	for i, s := range p.Storyboard.Shots {
		if s.ID == shotID || (shotID == "" && i == 0) {
			idx = i
			shotID = s.ID
			break
		}
	}
	if idx < 0 {
		return Job{}, fmt.Errorf("shot not found")
	}
	shot := p.Storyboard.Shots[idx]
	prompt := e.shotPrompt(p, shot)
	if prompt == "" {
		return Job{}, fmt.Errorf("shot needs a visual description")
	}
	var refs []string
	if shot.ImageHash != "" {
		refs = append(refs, shot.ImageHash)
	} else if p.Identity.RefHash != "" {
		refs = append(refs, p.Identity.RefHash)
	}
	j, err := e.EnqueueImage(EnqueueImage{
		Prompt: prompt, Size: ImageSizeFor("scene", p.Aspect), AspectRatio: p.Aspect, Refs: refs,
		StudioProjectID: p.ID, StudioShotID: shot.ID, StudioKind: "shot",
	})
	if err != nil {
		return Job{}, err
	}
	p.Storyboard.Shots[idx].JobID = j.ID
	p.Storyboard.Shots[idx].Stale = true
	e.noteStudio(&p, "generate", "shot "+shot.ID)
	if _, err := e.saveStudio(p); err != nil {
		return j, err
	}
	return j, nil
}

// GenerateMissingStudioShots 给还没有可用画面的镜头排队生图。图还在且没过期、又没有视频的会跳过。视频还在且没过期的也会跳过。中途失败时返回已经排上的任务。
// id 传给 GetStudioProject，并与每个镜头的 ID 一起传给 GenerateStudioShot。
func (e *Engine) GenerateMissingStudioShots(id string) ([]Job, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return nil, err
	}
	var jobs []Job
	for _, s := range p.Storyboard.Shots {
		if s.ImageHash != "" && !s.Stale && s.VideoHash == "" {
			continue
		}
		if s.VideoHash != "" && !s.Stale {
			continue
		}
		j, err := e.GenerateStudioShot(id, s.ID)
		if err != nil {
			return jobs, err
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []Job{}
	}
	return jobs, nil
}

// applyStudioJob 把完成的任务写回短片。封面任务追加到候选，还没有选中封面时用这一张。镜头任务写下图或视频，并留下上一版，同时清掉过期。没有工程编号或没有结果就直接返回。
// j 的 ResultHash 为空时直接返回；封面任务把它追加到 Covers，镜头任务按 Type 写入视频或图，并用 ID 匹配镜头。
// params 的 StudioProjectID 为空时直接返回，否则传给 GetStudioProject，并用 StudioKind、StudioShotID 选择写封面还是镜头。
func (e *Engine) applyStudioJob(j Job, params JobParams) error {
	if params.StudioProjectID == "" || j.ResultHash == "" {
		return nil
	}
	p, err := e.GetStudioProject(params.StudioProjectID)
	if err != nil {
		return err
	}
	switch params.StudioKind {
	case "cover":
		p.Covers = append(p.Covers, j.ResultHash)
		if p.CoverHash == "" {
			p.CoverHash = j.ResultHash
		}
		e.noteStudio(&p, "cover", "candidate ready")
	default:
		found := false
		for i, s := range p.Storyboard.Shots {
			if s.ID != params.StudioShotID && s.JobID != j.ID {
				continue
			}
			if j.Type == "video" {
				if s.VideoHash != "" {
					p.Storyboard.Shots[i].PrevVideo = s.VideoHash
				}
				p.Storyboard.Shots[i].VideoHash = j.ResultHash
			} else {
				if s.ImageHash != "" {
					p.Storyboard.Shots[i].PrevImage = s.ImageHash
				}
				p.Storyboard.Shots[i].ImageHash = j.ResultHash
			}
			p.Storyboard.Shots[i].Stale = false
			p.Storyboard.Shots[i].JobID = j.ID
			found = true
			break
		}
		if found {
			e.noteStudio(&p, "shot", "media landed")
		}
	}
	_, err = e.saveStudio(p)
	return err
}

// markStudioJobFailed 记下镜头生图失败。封面任务不处理。这一镜已经有图或视频时清掉过期，避免旧画面被卡死。
// j 的 Params 传给 parseParams，ID 与参数里的 StudioShotID 一起用来匹配镜头。
func (e *Engine) markStudioJobFailed(j Job) {
	params := parseParams(j.Params)
	if params.StudioProjectID == "" || params.StudioKind == "cover" {
		return
	}
	p, err := e.GetStudioProject(params.StudioProjectID)
	if err != nil {
		return
	}
	for i, s := range p.Storyboard.Shots {
		if s.ID != params.StudioShotID && s.JobID != j.ID {
			continue
		}
		if strings.TrimSpace(s.ImageHash) != "" || strings.TrimSpace(s.VideoHash) != "" {
			p.Storyboard.Shots[i].Stale = false
		}
		e.noteStudio(&p, "shot", "generate failed")
		_, _ = e.saveStudio(p)
		return
	}
}

// GenerateStudioCover 按标题排队一张封面。有海报就用海报做参考，否则用第一张镜头图。
// id 传给 GetStudioProject。
func (e *Engine) GenerateStudioCover(id string) (Job, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return Job{}, err
	}
	title := strings.TrimSpace(p.Title)
	prompt := fmt.Sprintf("Thumbnail for a short video titled %q. Large readable title, one subject, high contrast, platform cover, no watermarks.", title)
	if p.Identity.Preserve != "" {
		prompt += " Keep: " + p.Identity.Preserve + "."
	}
	var refs []string
	if p.PosterHash != "" {
		refs = append(refs, p.PosterHash)
	} else {
		for _, s := range p.Storyboard.Shots {
			if s.ImageHash != "" {
				refs = append(refs, s.ImageHash)
				break
			}
		}
	}
	j, err := e.EnqueueImage(EnqueueImage{
		Prompt: prompt, Size: ImageSizeFor("scene", p.Aspect), AspectRatio: p.Aspect, Refs: refs,
		StudioProjectID: p.ID, StudioKind: "cover",
	})
	if err != nil {
		return Job{}, err
	}
	e.noteStudio(&p, "cover", "queued")
	_, _ = e.saveStudio(p)
	return j, nil
}

// PickStudioCover 把选中的封面编号写上工程。编号空着就拒绝。不检查它是不是候选里的一张。
// id 传给 GetStudioProject。
// hash 去掉首尾空白，空串时返回错误 cover hash required，否则写入 CoverHash。
func (e *Engine) PickStudioCover(id, hash string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return p, fmt.Errorf("cover hash required")
	}
	p.CoverHash = hash
	e.noteStudio(&p, "cover", "picked")
	return e.saveStudio(p)
}

// WriteStudioCopy 按平台把发布文案写上工程，并尽量把正文放进内容库。平台非空时改工程的平台。不编造经历或功效。
// id 传给 GetStudioProject。
// platform 非空时写入工程的 Platform。
func (e *Engine) WriteStudioCopy(id, platform string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	if platform != "" {
		p.Platform = platform
	}
	p.CopyText = defaultPublishCopy(p)
	if e.CAS != nil {
		if h, err := e.PutBytes([]byte(p.CopyText)); err == nil {
			p.CopyHash = h
		}
	}
	e.noteStudio(&p, "copy", p.Platform)
	return e.saveStudio(p)
}

// ExpandTopic 给选题写三个固定角度。不调用模型，也不改分数。
// id 传给 GetTopic。
func (e *Engine) ExpandTopic(id string) (Topic, error) {
	t, err := e.GetTopic(id)
	if err != nil {
		return Topic{}, err
	}
	title := strings.TrimSpace(t.Title)
	opts := []TopicOption{
		{ID: NewID(), Title: title, Angle: "把事实讲清楚，不编经历", Summary: strings.TrimSpace(t.Note)},
		{ID: NewID(), Title: title + "｜常见误区", Angle: "拆三件被忽略的细节", Summary: "只写可核验的点，不写亲身功效。"},
		{ID: NewID(), Title: "为什么" + title, Angle: "原因向，给一个可执行动作", Summary: "结尾一个具体下一步。"},
	}
	t.Options = opts
	return e.SaveTopic(t)
}

// ChooseTopicOption 把选中的角度写回标题、角度和备注。找不到这个角度就拒绝。
// id 传给 GetTopic。
// optionID 与选项 ID 比较，相等时在该选项的 Title、Angle、Summary 非空时写回选题。
func (e *Engine) ChooseTopicOption(id, optionID string) (Topic, error) {
	t, err := e.GetTopic(id)
	if err != nil {
		return Topic{}, err
	}
	for _, o := range t.Options {
		if o.ID == optionID {
			if o.Title != "" {
				t.Title = o.Title
			}
			if o.Angle != "" {
				t.Angle = o.Angle
			}
			if o.Summary != "" {
				t.Note = o.Summary
			}
			return e.SaveTopic(t)
		}
	}
	return t, fmt.Errorf("option not found")
}

// IngestStudioURL 把一个公开地址收成新的一镜。视频写成视频编号，其他写成图编号。地址空着就拒绝。
// id 传给 GetStudioProject。
// rawURL 去掉首尾空白，空串时返回错误 url required，否则传给 fetchMediaURL，并传给 noteStudio。
func (e *Engine) IngestStudioURL(id, rawURL string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return p, fmt.Errorf("url required")
	}
	hash, mime, err := e.fetchMediaURL(rawURL)
	if err != nil {
		return p, err
	}
	shot := StudioShot{ID: NewID(), Duration: 5, Motion: "ken-burns"}
	if strings.HasPrefix(mime, "video/") {
		shot.VideoHash = hash
	} else {
		shot.ImageHash = hash
	}
	p.Storyboard.Shots = append(p.Storyboard.Shots, shot)
	e.noteStudio(&p, "ingest", rawURL)
	return e.saveStudio(p)
}

// fetchMediaURL 优先用 PATH 里的 yt-dlp 下载。没有就直接 HTTP 下载，最多 80MB。空文件或状态码大于等于 300 都算失败。
// rawURL 传给 yt-dlp；找不到该程序时作为 GET 的地址。
func (e *Engine) fetchMediaURL(rawURL string) (hash, mime string, err error) {
	if ytdlp, lookErr := exec.LookPath("yt-dlp"); lookErr == nil {
		tmp, terr := os.MkdirTemp(e.Dir, "ingest-*")
		if terr != nil {
			return "", "", terr
		}
		defer os.RemoveAll(tmp)
		out := filepath.Join(tmp, "media.%(ext)s")
		cmd := quiet(exec.Command(ytdlp, "-f", "mp4/best", "-o", out, "--no-playlist", rawURL))
		if b, runErr := cmd.CombinedOutput(); runErr != nil {
			return "", "", fmt.Errorf("yt-dlp: %s", trimErr(b, runErr))
		}
		matches, _ := filepath.Glob(filepath.Join(tmp, "media.*"))
		if len(matches) == 0 {
			return "", "", fmt.Errorf("yt-dlp produced no file")
		}
		raw, rerr := os.ReadFile(matches[0])
		if rerr != nil {
			return "", "", rerr
		}
		h, herr := e.PutBytes(raw)
		return h, "video/mp4", herr
	}
	httpDo := e.HTTP
	if httpDo == nil {
		httpDo = &http.Client{Timeout: 2 * time.Minute}
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", err
	}
	res, err := httpDo.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return "", "", fmt.Errorf("download %d", res.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 80<<20))
	if err != nil {
		return "", "", err
	}
	if len(raw) < 32 {
		return "", "", fmt.Errorf("empty download")
	}
	h, err := e.PutBytes(raw)
	ct := res.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(raw)
	}
	return h, ct, err
}

// StudioClip 是从一条长视频里切出的一段。
type StudioClip struct {
	ID    string  `json:"id"`
	Hash  string  `json:"hash"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Title string  `json:"title,omitempty"`
	// Scores 是这段的能量、完整度之类的分数。切分没给时写成 3 和 3。
	Scores   map[string]int `json:"scores,omitempty"`
	MediaURL string         `json:"media_url,omitempty"`
}

// AutoClip 按场景和静音把一条长视频切成若干段，每段单独放进内容库。不把它们接成片。
// 条数默认 3，最多 12。时长窗口 15-30 按 20 秒，60-90 按 75 秒，其余按 45 秒。没有 ffmpeg 就拒绝。
// hash 去掉首尾空白，空串时返回错误 source hash required，否则传给 ProbeSeconds，并以 .mp4 传给 materialize。
// count 小于等于 0 时写成 3，大于 12 时写成 12，再传给 cut.Propose。
// window 等于 15-30 时跨度用 20，等于 60-90 时用 75，其余用 45。
func (e *Engine) AutoClip(hash string, count int, window string) ([]StudioClip, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return nil, fmt.Errorf("source hash required")
	}
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return nil, err
	}
	if count <= 0 {
		count = 3
	}
	if count > 12 {
		count = 12
	}
	span := 30.0
	switch window {
	case "15-30":
		span = 20
	case "60-90":
		span = 75
	default:
		span = 45
	}
	total, err := e.ProbeSeconds(hash)
	if err != nil || total <= 0 {
		total = float64(count) * span
	}
	src, err := e.materialize(hash, ".mp4")
	if err != nil {
		return nil, err
	}
	defer os.Remove(src)
	bd, _ := e.cutBin().Analyze(src)
	if bd.Duration <= 0 {
		bd.Duration = total
	}
	proposed := cut.Propose(bd.Duration, bd.Scenes, bd.Silences, count, span)
	tmp, err := os.MkdirTemp(e.Dir, "autoclip-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	out := make([]StudioClip, 0, len(proposed))
	for i, c := range proposed {
		clip := filepath.Join(tmp, fmt.Sprintf("c%d.mp4", i))
		if err := e.cutBin().Extract(src, c.Start, c.Dur(), clip); err != nil {
			return out, err
		}
		raw, err := os.ReadFile(clip)
		if err != nil {
			return out, err
		}
		h, err := e.PutBytes(raw)
		if err != nil {
			return out, err
		}
		scores := c.Scores
		if scores == nil {
			scores = map[string]int{"energy": 3, "completeness": 3}
		}
		out = append(out, StudioClip{
			ID: NewID(), Hash: h, Start: c.Start, End: c.End,
			Title: c.Title, Scores: scores,
		})
	}
	if out == nil {
		out = []StudioClip{}
	}
	return out, nil
}

// ListStudioJobs 返回参数里带着这个短片编号的任务。没有时返回空切片。
// projectID 去掉首尾空白后与任务参数的 StudioProjectID 比较。
func (e *Engine) ListStudioJobs(projectID string) []Job {
	projectID = strings.TrimSpace(projectID)
	all := loadCol[Job](e, colJobs)
	var out []Job
	for _, j := range all {
		p := parseParams(j.Params)
		if p.StudioProjectID == projectID {
			out = append(out, j)
		}
	}
	if out == nil {
		return []Job{}
	}
	return out
}

// probeFileSeconds 用指定的 ffprobe 读文件时长。输出不是数字就返回错误。
// path 传给 ffprobe。
func probeFileSeconds(ffprobe, path string) (float64, error) {
	cmd := quiet(exec.Command(ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", path))
	b, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %s", strings.TrimSpace(string(b)))
	}
	s := strings.TrimSpace(string(b))
	n, err := strconv.ParseFloat(s, 64)
	return n, err
}

// trimErr 把命令输出收成一段错误文本，最长 400 个字符。输出是空的就用错误本身。
// b 转成文本并去掉首尾空白，长于 400 时只留前 400 字节。
// err 在文本为空且非 nil 时调用 Error。
func trimErr(b []byte, err error) string {
	msg := strings.TrimSpace(string(b))
	if msg == "" && err != nil {
		return err.Error()
	}
	if len(msg) > 400 {
		msg = msg[:400]
	}
	return msg
}
