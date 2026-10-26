// 行为：时间线渲染和转写任务的创建与执行。渲染走 ffmpeg 拼接，转写走本机 whisper。

package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// LookWhisper 查找 whisper-cli 或 whisper，先看环境变量，再看可执行文件旁和用户目录。
func LookWhisper() string {
	if v := os.Getenv("WHISPER_BIN"); v != "" {
		return v
	}
	if v := os.Getenv("WHISPER_CPP"); v != "" {
		return v
	}
	name := "whisper-cli"
	if runtime.GOOS == "windows" {
		name = "whisper-cli.exe"
	}
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	cands := []string{dir, filepath.Join(dir, "bin"), filepath.Join(dir, "resources")}
	if home, err := os.UserHomeDir(); err == nil {
		cands = append(cands, filepath.Join(home, ".yoyo", "video", "bin"))
	}
	for _, d := range cands {
		p := filepath.Join(d, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
		alt := filepath.Join(d, strings.TrimSuffix(name, ".exe"))
		if runtime.GOOS == "windows" {
			alt += ".exe"
		}
		if st, err := os.Stat(alt); err == nil && !st.IsDir() {
			return alt
		}
	}
	if p, err := exec.LookPath("whisper-cli"); err == nil {
		return p
	}
	if p, err := exec.LookPath("whisper"); err == nil {
		return p
	}
	return ""
}

// createTimelineRender 把时间线片段收成一条排队中的 ffmpeg 拼接任务。
// req 读取 projectId、canvasId、clips 和 hashes；clips 里的 resourceId 或 id 能找到文件才收进片段，一段都没有时改用 hashes。
func (e *Engine) createTimelineRender(req map[string]any) (map[string]any, error) {
	now := Now()
	projectID := strAnyMap(req, "projectId", "canvasId")
	clips := []string{}
	for _, item := range asSlice(req["clips"]) {
		m := asMap(item)
		id := strAnyMap(m, "resourceId", "id")
		if id == "" {
			continue
		}
		if res, err := e.getCanvasResource(id); err == nil {
			if p := e.FilePath(res.CASHash); p != "" {
				clips = append(clips, p)
			}
		}
	}
	if len(clips) == 0 {
		for _, h := range asSlice(req["hashes"]) {
			p := e.FilePath(fmt.Sprint(h))
			if p != "" {
				clips = append(clips, p)
			}
		}
	}
	params := JobParams{CanvasProjectID: projectID, Mode: "video", ReferenceVideoURLs: clips, Operation: "timeline_render"}
	j := Job{ID: NewID(), Type: "canvas_timeline", Status: "queued", Provider: "ffmpeg", Model: "concat-h264", Params: marshalJSON(params), CreatedAt: now, UpdatedAt: now}
	if err := e.insertJob(j); err != nil {
		return nil, err
	}
	e.emit(j)
	return e.taskView(j), nil
}

// createTimelineTranscribe 把资源收成一条排队中的 whisper 转写任务。
// req 读取 resourceId 和 projectId，写入任务参数。
func (e *Engine) createTimelineTranscribe(req map[string]any) (map[string]any, error) {
	now := Now()
	resourceID := strAnyMap(req, "resourceId")
	projectID := strAnyMap(req, "projectId")
	params := JobParams{CanvasProjectID: projectID, ResourceID: resourceID, Mode: "audio", Operation: "transcribe"}
	j := Job{ID: NewID(), Type: "canvas_transcribe", Status: "queued", Provider: "whisper", Model: "whisper.cpp", Params: marshalJSON(params), CreatedAt: now, UpdatedAt: now}
	if err := e.insertJob(j); err != nil {
		return nil, err
	}
	e.emit(j)
	return e.taskView(j), nil
}

// runCanvasTimeline 按片段路径拼接视频并完成媒体登记。没有片段则返回错误。
// j 的 Params 解析出片段路径，拼完后和 j 一起传给 finishMedia。
func (e *Engine) runCanvasTimeline(j *Job) error {
	p := parseParams(j.Params)
	if len(p.ReferenceVideoURLs) == 0 {
		return fmt.Errorf("no clips to render")
	}
	raw, err := e.Concat(p.ReferenceVideoURLs)
	if err != nil {
		return err
	}
	return e.finishMedia(j, raw, true)
}

// runCanvasTranscribe 对资源文件做转写，把文本写进任务草稿并标成成功。
// j 的 Params 提供 ResourceID；转写文本写入 TextDraft，Status 写成 succeeded，并写回 Params。
func (e *Engine) runCanvasTranscribe(j *Job) error {
	p := parseParams(j.Params)
	res, err := e.getCanvasResource(p.ResourceID)
	if err != nil {
		return err
	}
	path := e.FilePath(res.CASHash)
	if path == "" {
		return fmt.Errorf("media missing")
	}
	tmp, err := os.MkdirTemp(e.Dir, "whisper-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	cues, text, err := e.cutBin().Transcribe(path, tmp)
	if err != nil {
		return err
	}
	if text == "" && len(cues) > 0 {
		var b strings.Builder
		for _, c := range cues {
			if c.Text != "" {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(c.Text)
			}
		}
		text = b.String()
	}
	p.TextDraft = strings.TrimSpace(text)
	p.ResultState = "READY"
	j.Params = marshalJSON(p)
	j.Status = "succeeded"
	j.CompletedAt = Now()
	return nil
}
