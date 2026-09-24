package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

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

func (e *Engine) runCanvasTranscribe(j *Job) error {
	if e.WHISPER == "" {
		return fmt.Errorf("whisper is not available — set WHISPER_BIN or install whisper.cpp")
	}
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
	out := filepath.Join(tmp, "out")
	cmd := exec.Command(e.WHISPER, "-f", path, "-otxt", "-of", out)
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("whisper: %s", strings.TrimSpace(string(b)))
	}
	text, _ := os.ReadFile(out + ".txt")
	p.TextDraft = strings.TrimSpace(string(text))
	p.ResultState = "READY"
	j.Params = marshalJSON(p)
	j.Status = "succeeded"
	j.CompletedAt = Now()
	return nil
}
