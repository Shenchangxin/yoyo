package video

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/video/film"
)

func (e *Engine) lookFilmTools() {
	e.YTDLP = lookBin("YOYO_YTDLP", "yt-dlp")
	if e.YTDLP == "" {
		e.YTDLP = lookNamed(e, "yt-dlp")
	}
	e.StampBin = lookBin("YOYO_STAMP_BIN", "whisperx")
	e.RegionBin = lookBin("YOYO_REGION_BIN", "yoyo-region")
	e.BrowserBin = lookBrowser()
}

func lookNamed(e *Engine, name string) string {
	file := name
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(file), ".exe") {
		file += ".exe"
	}
	var dirs []string
	if e != nil && e.Dir != "" {
		dirs = append(dirs, filepath.Join(e.Dir, "bin"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".yoyo", "video", "bin"))
	}
	for _, dir := range dirs {
		path := filepath.Join(dir, file)
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path
		}
	}
	return ""
}

func lookBin(envKey, name string) string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return ""
}

func lookBrowser() string {
	if v := strings.TrimSpace(os.Getenv("YOYO_BROWSER")); v != "" {
		return v
	}
	for _, name := range []string{"chrome", "msedge", "google-chrome", "chromium"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	if runtime.GOOS != "windows" {
		return ""
	}
	cands := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
	}
	for _, path := range cands {
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path
		}
	}
	return ""
}

func (e *Engine) ListFilms() []film.Project {
	out := loadCol[film.Project](e, colFilms)
	if out == nil {
		return []film.Project{}
	}
	return out
}

func (e *Engine) GetFilm(id string) (film.Project, error) {
	p, err := getDoc[film.Project](e, colFilms, strings.TrimSpace(id))
	if err != nil {
		return film.Project{}, fmt.Errorf("film not found")
	}
	return p, nil
}

func (e *Engine) saveFilm(p film.Project) (film.Project, error) {
	if strings.TrimSpace(p.ID) == "" {
		p.ID = NewID()
	}
	if err := e.putDoc(colFilms, p.ID, p); err != nil {
		return film.Project{}, err
	}
	return p, nil
}

// OpenFilm reads a stored reference into a production project.
// A URL is downloaded when yt-dlp is on PATH. Scene marks need ffmpeg.
// Word marks prefer a stamp program, then whisper.cpp sentences split into tokens.
func (e *Engine) OpenFilm(in film.Input) (film.Project, error) {
	in.Hash = strings.TrimSpace(in.Hash)
	in.URL = strings.TrimSpace(in.URL)
	in = e.fetchFilmURL(in)
	in = e.readFilmMedia(in)
	if strings.TrimSpace(in.Structure) == "" && len(in.Cues) == 0 && len(in.Words) == 0 && strings.TrimSpace(in.Description) == "" {
		in.Structure = "free"
	}
	p := film.FromReference(in)
	if in.URL != "" && in.Hash == "" && strings.TrimSpace(e.YTDLP) == "" {
		p.Note = "还没有 yt-dlp，参考地址留在工程里，没有下载。"
	} else if in.Hash != "" && len(p.Words) == 0 {
		p.Note = "还没有词轴。本机有词级标注或 whisper.cpp 时会写成词。"
	}
	return e.saveFilm(p)
}

func (e *Engine) fetchFilmURL(in film.Input) film.Input {
	if in.Hash != "" || in.URL == "" || strings.TrimSpace(e.YTDLP) == "" {
		return in
	}
	dir, err := os.MkdirTemp(e.Dir, "film-dl-*")
	if err != nil {
		return in
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "ref.mp4")
	args := film.DownloadCommand(e.YTDLP, in.URL, out)
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Run(); err != nil {
		return in
	}
	raw, err := os.ReadFile(out)
	if err != nil || len(raw) == 0 {
		return in
	}
	hash, err := e.PutBytes(raw)
	if err != nil || hash == "" {
		return in
	}
	in.Hash = hash
	return in
}

func (e *Engine) readFilmMedia(in film.Input) film.Input {
	if in.Hash == "" {
		return in
	}
	if d := e.probeCutDuration(in.Hash); d > 0 {
		in.Duration = d
	}
	e.refreshFFmpeg()
	if !e.FFmpegOK() {
		return in
	}
	src, err := e.materialize(in.Hash, ".mp4")
	if err != nil {
		return in
	}
	defer os.Remove(src)
	if bd, err := e.cutBin().Analyze(src); err == nil {
		in.Scenes = bd.Scenes
		if bd.Duration > 0 {
			in.Duration = bd.Duration
		}
	}
	if words := e.stampFilmWords(src); len(words) > 0 {
		in.Words = words
	}
	if len(in.Words) == 0 && strings.TrimSpace(e.WHISPER) != "" {
		if cues, _, err := e.cutBin().Transcribe(src, filepath.Join(e.Dir, "film-asr")); err == nil {
			for _, cue := range cues {
				in.Cues = append(in.Cues, film.Cue{Start: cue.Start, End: cue.End, Text: cue.Text})
			}
		}
	}
	if boxes := e.detectFilmRegions(src); len(boxes) > 0 {
		in.Regions = boxes
	}
	return in
}

func (e *Engine) stampFilmWords(src string) []film.Word {
	if strings.TrimSpace(e.StampBin) == "" {
		return nil
	}
	dir, err := os.MkdirTemp(e.Dir, "film-stamp-*")
	if err != nil {
		return nil
	}
	defer os.RemoveAll(dir)
	args := film.StampCommand(e.StampBin, src, dir)
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Run(); err != nil {
		return nil
	}
	stem := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	raw, err := os.ReadFile(filepath.Join(dir, stem+".json"))
	if err != nil {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.json"))
		if len(matches) == 0 {
			return nil
		}
		raw, err = os.ReadFile(matches[0])
		if err != nil {
			return nil
		}
	}
	words, err := film.WordsFromStamps(raw)
	if err != nil {
		return nil
	}
	return words
}

func (e *Engine) detectFilmRegions(src string) []film.Box {
	if strings.TrimSpace(e.RegionBin) != "" {
		if boxes := e.runRegionBin(src); len(boxes) > 0 {
			return boxes
		}
	}
	return e.cropFilmRegion(src)
}

func (e *Engine) runRegionBin(src string) []film.Box {
	out := src + ".regions.json"
	defer os.Remove(out)
	args := film.RegionCommand(e.RegionBin, src, out)
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Run(); err != nil {
		return nil
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return nil
	}
	boxes, err := film.ParseRegions(raw)
	if err != nil {
		return nil
	}
	return boxes
}

func (e *Engine) cropFilmRegion(src string) []film.Box {
	if strings.TrimSpace(e.FFMPEG) == "" || strings.TrimSpace(e.FFProbe) == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	probe, err := exec.CommandContext(ctx, e.FFProbe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0", src).Output()
	if err != nil {
		return nil
	}
	parts := strings.Split(strings.TrimSpace(string(probe)), ",")
	if len(parts) < 2 {
		return nil
	}
	width, werr := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, herr := strconv.Atoi(strings.TrimSpace(parts[1]))
	if werr != nil || herr != nil {
		return nil
	}
	combined, _ := exec.CommandContext(ctx, e.FFMPEG, "-i", src, "-vf", "cropdetect=24:16:0", "-f", "null", "-").CombinedOutput()
	last := ""
	for _, line := range strings.Split(string(combined), "\n") {
		if i := strings.Index(line, "crop="); i >= 0 {
			last = line[i:]
		}
	}
	box, ok := film.ParseCrop(last, width, height)
	if !ok {
		return nil
	}
	return []film.Box{box}
}

// DescribeFilm builds a structure from a description, with no reference file.
func (e *Engine) DescribeFilm(structure, title, description string) (film.Project, error) {
	structure = strings.TrimSpace(structure)
	if structure == "" {
		structure = "ranking"
	}
	if structure == "ranking" || structure == "podcast" || structure == "interview" {
		if strings.TrimSpace(description) == "" {
			p := film.FromTemplate(structure, title)
			return e.saveFilm(p)
		}
	}
	p := film.FromReference(film.Input{Title: title, Structure: structure, Description: description})
	return e.saveFilm(p)
}

// PlanFilm refreshes needs and quotes.
func (e *Engine) PlanFilm(id string) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	p = film.Plan(p)
	return e.saveFilm(p)
}

// SwapFilm changes one brief field and keeps assets whose inputs did not change.
func (e *Engine) SwapFilm(id string, patch film.Brief) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	p = film.Swap(p, patch)
	if strings.TrimSpace(patch.Language) != "" {
		rewritten, err := e.translateFilm(p)
		if err != nil {
			p.Note = "语言已记下。接上对话连接后可以按这个语言重写每一行。"
		} else {
			p = rewritten
			p.Note = ""
		}
	}
	return e.saveFilm(p)
}

// ForkFilm stores a variant and leaves the original film untouched.
func (e *Engine) ForkFilm(id string, patch film.Brief) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	return e.saveFilm(film.Fork(p, patch))
}

// SetFilmBed attaches a soundtrack the operator already stored.
func (e *Engine) SetFilmBed(id, hash string) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	p.Bed = strings.TrimSpace(hash)
	return e.saveFilm(p)
}

func (e *Engine) translateFilm(p film.Project) (film.Project, error) {
	lines := film.LineTexts(p)
	if len(lines) == 0 {
		return p, nil
	}
	c, key, err := e.chatForFilm()
	if err != nil {
		return p, err
	}
	model := connection.ModelFor(c, connection.CapChat)
	if model == "" {
		model = "gpt-4.1-mini"
	}
	var body strings.Builder
	for _, line := range lines {
		body.WriteString(line.Text)
		body.WriteByte('\n')
	}
	raw, err := e.chatComplete(c.Endpoint, key, model,
		"你把口播改写成指定语言。保持行数和顺序，保留每行的口气。只输出改写后的正文，一行一句，不要编号，不要解释。",
		"语言："+p.Brief.Language+"\n"+body.String(),
	)
	if err != nil {
		return p, err
	}
	var parts []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts = append(parts, line)
	}
	if len(parts) != len(lines) {
		return p, fmt.Errorf("localize returned %d lines, want %d", len(parts), len(lines))
	}
	mapped := map[string]string{}
	for i, line := range lines {
		mapped[line.ID] = parts[i]
	}
	return film.Localize(p, mapped), nil
}

func (e *Engine) chatForFilm() (connection.Connection, string, error) {
	if e.Conn == nil {
		return connection.Connection{}, "", fmt.Errorf("no enabled connection for localize")
	}
	list, err := e.Conn.List(connection.CapChat)
	if err != nil {
		return connection.Connection{}, "", fmt.Errorf("no enabled connection for localize")
	}
	for _, c := range list {
		if !c.Active || !c.HasKey {
			continue
		}
		blob := strings.ToLower(c.ID + " " + c.Name + " " + c.Endpoint + " " + c.Vendor)
		if strings.Contains(blob, "hypihub") {
			continue
		}
		key, err := e.Conn.Lease(c)
		if err != nil || strings.TrimSpace(key) == "" {
			continue
		}
		return c, key, nil
	}
	return connection.Connection{}, "", fmt.Errorf("no enabled connection for localize")
}

// RewriteFilmLine replaces one line of words.
func (e *Engine) RewriteFilmLine(id, lineID, text string) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	return e.saveFilm(film.ReplaceLine(p, lineID, text))
}

// BuildFilm paints code shots locally and sends model needs to enabled connections.
func (e *Engine) BuildFilm(id string) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	p = film.Build(p, film.BuildEnv{
		Providers: e.filmEndpoints(),
		Put: func(b []byte) (string, error) {
			return e.PutBytes(b)
		},
		Dispatch: e.dispatchFilmNeed,
	})
	if strings.TrimSpace(p.Bed) == "" {
		for _, a := range p.Artifacts {
			if a.Name == "music" && a.Hash != "" {
				p.Bed = a.Hash
				break
			}
		}
	}
	if e.BrowserBin != "" {
		p = e.captureFilmBrowser(p)
	}
	p = e.assembleFilm(p)
	return e.saveFilm(p)
}

func (e *Engine) filmEndpoints() []film.Endpoint {
	var out []film.Endpoint
	for _, kind := range []string{"image", "video", "speech", "matting", "music"} {
		for _, row := range e.ProviderMenu(kind) {
			id := strAnyMap(row, "id")
			ep := film.Endpoint{
				ID: id, Name: strAnyMap(row, "name"), Service: kind, Vendor: strAnyMap(row, "vendor"),
				Model: firstModel(row["models"]), BaseURL: strAnyMap(row, "endpoint", "base_url"),
				Active: true, HasKey: true,
			}
			if e.Conn != nil && id != "" {
				if c, err := e.Conn.Get(id); err == nil {
					ep.BaseURL = c.Endpoint
					ep.VaultKey = c.VaultKey
					if ep.Vendor == "" {
						ep.Vendor = c.Vendor
					}
				}
			}
			out = append(out, ep)
		}
	}
	return out
}

func firstModel(v any) string {
	switch t := v.(type) {
	case []string:
		if len(t) > 0 {
			return t[0]
		}
	case []any:
		if len(t) > 0 {
			return fmt.Sprint(t[0])
		}
	}
	return ""
}

func (e *Engine) dispatchFilmNeed(n film.Need) (string, error) {
	ctx := context.Background()
	switch n.Kind {
	case "matting":
		return e.dispatchFilmMatte(ctx, n)
	case "music":
		return e.dispatchFilmMusic(ctx, n)
	}
	j := &Job{
		Provider: n.Provider,
		VaultKey: n.VaultKey,
		Model:    n.Model,
		Prompt:   n.Prompt,
		Params:   `{"prompt":` + jsonString(n.Prompt) + `}`,
	}
	var err error
	switch n.Kind {
	case "image":
		err = e.submitImage(ctx, j)
	case "video":
		err = e.submitVideo(ctx, j)
	case "speech":
		raw, _, _, err2 := e.speakAudio(n.Prompt, "alloy", 1)
		if err2 != nil {
			return "", err2
		}
		return e.PutBytes(raw)
	default:
		return "", fmt.Errorf("unknown need %s", n.Kind)
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(j.ResultHash) != "" {
		return j.ResultHash, nil
	}
	if strings.TrimSpace(j.RemoteID) != "" {
		return "", film.Wait("生成还在进行：" + j.RemoteID)
	}
	return "", fmt.Errorf("need %s produced no asset", n.ID)
}

func (e *Engine) dispatchFilmMatte(ctx context.Context, n film.Need) (string, error) {
	j := &Job{Provider: n.Provider, VaultKey: n.VaultKey, Model: n.Model, Prompt: n.Prompt}
	p, err := e.providerForFilm(*j, "matting")
	if err != nil {
		return "", err
	}
	key, err := e.leaseProvider(p)
	if err != nil {
		return "", err
	}
	raw, err := e.matteSource(strings.TrimSpace(n.Prompt))
	if err != nil {
		return "", err
	}
	out, err := e.matteImage(ctx, p, e.protocolOf(p, "matting"), key, raw)
	if err != nil {
		var preset presetOnlyError
		if errors.As(err, &preset) {
			return "", film.Wait(preset.Error())
		}
		return "", err
	}
	return e.PutBytes(out)
}

func (e *Engine) dispatchFilmMusic(ctx context.Context, n film.Need) (string, error) {
	j := &Job{Provider: n.Provider, VaultKey: n.VaultKey, Model: n.Model, Prompt: n.Prompt}
	p, err := e.providerForFilm(*j, "music")
	if err != nil {
		return "", err
	}
	key, err := e.leaseProvider(p)
	if err != nil {
		return "", err
	}
	out, err := e.scoreMusic(ctx, p, e.protocolOf(p, "music"), key, n.Prompt, n.Model)
	if err != nil {
		var preset presetOnlyError
		if errors.As(err, &preset) {
			return "", film.Wait(preset.Error())
		}
		return "", err
	}
	return e.PutBytes(out)
}

func (e *Engine) providerForFilm(j Job, kind string) (Provider, error) {
	p, err := e.providerForJob(j, kind)
	if err == nil {
		return p, nil
	}
	if j.VaultKey != "" && e.Conn != nil {
		list, lerr := e.Conn.List("")
		if lerr == nil {
			for _, c := range list {
				if c.VaultKey == j.VaultKey && c.Active {
					return providerFromConn(c, kind, nil), nil
				}
			}
		}
	}
	return Provider{}, err
}

func (e *Engine) protocolOf(p Provider, cap string) string {
	if e.Conn != nil && p.ID != "" {
		if c, err := e.Conn.Get(p.ID); err == nil && strings.TrimSpace(c.Protocol) != "" {
			return c.Protocol
		}
	}
	return connection.InferProtocol(p.Provider, cap)
}

func (e *Engine) matteSource(hash string) ([]byte, error) {
	if hash == "" {
		return nil, film.Wait("没有可抠的画面")
	}
	raw, err := e.GetBytes(hash)
	if err != nil || len(raw) == 0 {
		return nil, film.Wait("没有可抠的画面")
	}
	return raw, nil
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func (e *Engine) captureFilmBrowser(p film.Project) film.Project {
	for _, shot := range p.Shots {
		if shot.Visual == "broll" {
			continue
		}
		html := film.RenderHTML(p, shot)
		dir, err := os.MkdirTemp(e.Dir, "film-html-*")
		if err != nil {
			return p
		}
		page := filepath.Join(dir, "shot.html")
		png := filepath.Join(dir, "shot.png")
		if err := os.WriteFile(page, []byte(html), 0o644); err != nil {
			os.RemoveAll(dir)
			continue
		}
		cmd := exec.Command(e.BrowserBin, film.BrowserCommand(e.BrowserBin, page, png)[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			os.RemoveAll(dir)
			continue
		}
		raw, err := os.ReadFile(png)
		os.RemoveAll(dir)
		if err != nil || len(raw) == 0 {
			continue
		}
		hash, err := e.PutBytes(raw)
		if err != nil {
			continue
		}
		p.Artifacts = append(p.Artifacts, film.Artifact{Name: "browser:" + shot.ID, Hash: hash, MediaType: "image/png"})
	}
	return p
}

// HandoffFilm opens the existing cut desk on the film's picture.
// The caller decides when to show that desk.
func (e *Engine) HandoffFilm(id, title string) (CutProject, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return CutProject{}, err
	}
	hash := ""
	for _, a := range p.Artifacts {
		if strings.HasPrefix(a.MediaType, "video/") && a.Hash != "" {
			hash = a.Hash
			break
		}
	}
	if hash == "" {
		for _, a := range p.Artifacts {
			if strings.HasPrefix(a.MediaType, "image/") && a.Hash != "" {
				hash = a.Hash
				break
			}
		}
	}
	if hash == "" {
		hash = p.Reference.Hash
	}
	if strings.TrimSpace(title) == "" {
		title = p.Title
	}
	if hash == "" {
		return CutProject{}, fmt.Errorf("film has no picture to hand off")
	}
	cut, err := e.CreateCutProject(CutProject{Title: title})
	if err != nil {
		return CutProject{}, err
	}
	return e.CutIngest(cut.ID, hash, title)
}
