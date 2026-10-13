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
)

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

func (e *Engine) fetchMediaURL(rawURL string) (hash, mime string, err error) {
	if ytdlp, lookErr := exec.LookPath("yt-dlp"); lookErr == nil {
		tmp, terr := os.MkdirTemp(e.Dir, "ingest-*")
		if terr != nil {
			return "", "", terr
		}
		defer os.RemoveAll(tmp)
		out := filepath.Join(tmp, "media.%(ext)s")
		cmd := exec.Command(ytdlp, "-f", "mp4/best", "-o", out, "--no-playlist", rawURL)
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

type StudioClip struct {
	ID       string  `json:"id"`
	Hash     string  `json:"hash"`
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Title    string  `json:"title,omitempty"`
	Scores   map[string]int `json:"scores,omitempty"`
	MediaURL string  `json:"media_url,omitempty"`
}

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
	out := make([]StudioClip, 0, count)
	step := total / float64(count)
	if step < span {
		span = step
	}
	for i := 0; i < count; i++ {
		start := float64(i) * step
		if start+span > total {
			span = total - start
		}
		if span < 3 {
			break
		}
		tag := hash
		if len(tag) > 8 {
			tag = tag[:8]
		}
		clip := filepath.Join(e.Dir, fmt.Sprintf("clip-%s-%d.mp4", tag, i))
		cmd := exec.Command(e.FFMPEG, "-y", "-ss", fmt.Sprintf("%.2f", start), "-t", fmt.Sprintf("%.2f", span),
			"-i", src, "-c:v", "libx264", "-pix_fmt", "yuv420p", "-an", clip)
		if b, err := cmd.CombinedOutput(); err != nil {
			_ = os.Remove(clip)
			return out, fmt.Errorf("%s", ffmpegErr(b, err))
		}
		raw, err := os.ReadFile(clip)
		_ = os.Remove(clip)
		if err != nil {
			return out, err
		}
		h, err := e.PutBytes(raw)
		if err != nil {
			return out, err
		}
		complete := 5
		if span < 15 {
			complete = 3
		}
		out = append(out, StudioClip{
			ID: NewID(), Hash: h, Start: start, End: start + span,
			Title: fmt.Sprintf("clip %d", i+1),
			Scores: map[string]int{"hook": 3, "information": 3, "emotion": 3, "completeness": complete},
		})
	}
	if out == nil {
		out = []StudioClip{}
	}
	return out, nil
}

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

func probeFileSeconds(ffprobe, path string) (float64, error) {
	cmd := exec.Command(ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", path)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %s", strings.TrimSpace(string(b)))
	}
	s := strings.TrimSpace(string(b))
	n, err := strconv.ParseFloat(s, 64)
	return n, err
}

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
